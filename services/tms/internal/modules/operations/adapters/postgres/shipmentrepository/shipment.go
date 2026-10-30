package shipmentrepository

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentgraph"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentpredicate"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentprojection"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/chargeallocationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	shipmentEntity          = "Shipment"
	shipmentBusyMessage     = "Shipment is busy. Retry the request."
	previousRatesLimit      = 50
	maxDuplicateBOLResults  = 100
	defaultRatedByTemplates = 100
)

var shipmentErrors = dberror.Mapping{Entity: shipmentEntity}

type Params struct {
	fx.In

	DB                         *postgres.Connection
	Logger                     *zap.Logger
	Generator                  seqgen.Generator
	MoveRepository             shipmentmoveports.ShipmentMoveRepository
	AdditionalChargeRepository shipmentports.ShipmentAdditionalChargeRepository
	ChargeAllocationRepository chargeallocationports.ChargeAllocationRepository
	CommodityRepository        shipmentports.ShipmentCommodityRepository
	OrderRepository            orderports.OrderRepository
	Quota                      platformplanports.QuotaGuard `optional:"true"`
}

type repository struct {
	db                         *postgres.Connection
	l                          *zap.Logger
	generator                  seqgen.Generator
	moveRepository             shipmentmoveports.ShipmentMoveRepository
	additionalChargeRepository shipmentports.ShipmentAdditionalChargeRepository
	chargeAllocationRepository chargeallocationports.ChargeAllocationRepository
	commodityRepository        shipmentports.ShipmentCommodityRepository
	orderRepository            orderports.OrderRepository
	quota                      platformplanports.QuotaGuard
}

//nolint:gocritic // This is a constructor function
func New(p Params) shipmentports.ShipmentRepository {
	return &repository{
		db:                         p.DB,
		l:                          p.Logger.Named("postgres.shipment-repository"),
		generator:                  p.Generator,
		moveRepository:             p.MoveRepository,
		additionalChargeRepository: p.AdditionalChargeRepository,
		chargeAllocationRepository: p.ChargeAllocationRepository,
		commodityRepository:        p.CommodityRepository,
		orderRepository:            p.OrderRepository,
		quota:                      quotaservice.OrUnlimited(p.Quota),
	}
}

func busy(err error, id pulid.ID) error {
	if err == nil {
		return nil
	}

	return dberror.MapRetryableTransactionError(
		shipmentErrors.Map(err, id.String()),
		shipmentBusyMessage,
	)
}

func (r *repository) List(
	ctx context.Context,
	req *shipmentports.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*shipment.Shipment], error) {
			quick, err := shipmentpredicate.QuickFilterConditions(&req.ShipmentOptions)
			if err != nil {
				return nil, err
			}

			loads := shipmentLoads(&req.ShipmentOptions)

			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				shipmentdb.Shipments.
					Where(shipmentpredicate.OptionPredicates(&req.ShipmentOptions)...).
					Where(quick...),
				&listquery.Request[shipment.Shipment]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[*shipment.Shipment]) *keel.Query[*shipment.Shipment] {
						return q.Load(loads...)
					},
				},
			)
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		sp := shipmentdb.Shipments
		entity, err := sp.Where(sp.ID.Eq(req.ID)).
			Load(shipmentLoads(&req.ShipmentOptions)...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, shipmentErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req *shipmentports.GetShipmentsByIDsRequest,
) ([]*shipment.Shipment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipment.Shipment, error) {
		if len(req.ShipmentIDs) == 0 {
			return []*shipment.Shipment{}, nil
		}

		sp := shipmentdb.Shipments
		loads := []keel.Loader{sp.Customer(), sp.ServiceType()}
		if req.IncludeCharges {
			loads = append(loads, sp.BillToCustomer())
			loads = append(loads, chargeLoads()...)
		}

		entities, err := sp.Where(sp.ID.In(req.ShipmentIDs...)).
			Load(loads...).
			Limit(len(req.ShipmentIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, shipmentErrors.Map(err, "")
		}

		return entities, nil
	})
}

func (r *repository) ListRatedByFormulaTemplate(
	ctx context.Context,
	req *shipmentports.ListRatedByFormulaTemplateRequest,
) ([]*shipment.Shipment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipment.Shipment, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultRatedByTemplates
		}

		sp := shipmentdb.Shipments
		m := sp.Moves()
		ac := sp.AdditionalCharges()

		return sp.Where(sp.FormulaTemplateID.Eq(req.TemplateID)).
			Load(
				sp.Customer(),
				sp.ServiceType(),
				sp.ShipmentType(),
				sp.TractorType(),
				sp.TrailerType(),
				m.Order(m.Sequence.Asc()).Load(orderedStopsWithLocation(m.Stops())),
				ac.Load(ac.AccessorialCharge()),
				commoditiesWithHazmat(),
			).
			Order(sp.CreatedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *shipmentports.ShipmentSelectOptionsRequest,
) (*pagination.ListResult[*shipment.Shipment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*shipment.Shipment], error) {
			sp := shipmentdb.Shipments
			q := sp.Pick(sp.ID, sp.CreatedAt, sp.Status, sp.ProNumber, sp.BOL).
				Order(sp.CreatedAt.Desc())
			if !req.CustomerID.IsNil() {
				q = q.Where(sp.CustomerID.Eq(req.CustomerID))
			}
			if req.AttachableOnly {
				q = q.Where(
					sp.Status.NotIn(shipmentvocab.StatusCanceled, shipmentvocab.StatusInvoiced),
				)
			}
			if !req.ExcludeOrderID.IsNil() {
				q = q.Where(sp.OrderID.Ne(req.ExcludeOrderID))
			}

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(sp.ProNumber.ContainsFold(s), sp.BOL.ContainsFold(s))
				},
			)
		},
	)
}

func (r *repository) GetUnassigned(
	ctx context.Context,
	req *shipmentports.GetUnassignedShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*shipment.Shipment], error) {
			sp := shipmentdb.Shipments
			loads := []keel.Loader{sp.Customer()}
			if req.ShipmentOptions.ExpandShipmentDetails {
				loads = append(loads, movesWithDetails())
			}

			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				sp.Where(
					sp.Status.Eq(shipmentvocab.StatusNew),
					shipmentpredicate.UnassignedPredicate(),
				),
				&listquery.Request[shipment.Shipment]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[*shipment.Shipment]) *keel.Query[*shipment.Shipment] {
						return q.Load(loads...)
					},
				},
			)
		},
	)
}

func (r *repository) GetPreviousRates(
	ctx context.Context,
	req *shipmentports.GetPreviousRatesRequest,
) (*pagination.ListResult[*shipmentports.PreviousRateSummary], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*shipmentports.PreviousRateSummary], error) {
			sp := shipmentdb.Shipments
			q := sp.Where(
				sp.ShipmentTypeID.Eq(req.ShipmentTypeID),
				sp.ServiceTypeID.Eq(req.ServiceTypeID),
				sp.Status.Eq(shipmentvocab.StatusInvoiced),
				sp.ID.InQuery(previousRateOrigins(req.OriginLocationID)),
				sp.ID.InQuery(previousRateDestinations(req.DestinationLocationID)),
			)
			if req.CustomerID != nil {
				q = q.Where(sp.CustomerID.Eq(pulid.ConvertFromPtr(req.CustomerID)))
			}
			if req.ExcludeShipmentID != nil {
				q = q.Where(sp.ID.Ne(pulid.ConvertFromPtr(req.ExcludeShipmentID)))
			}

			items := q.Clone().
				Pick(
					sp.ID,
					sp.ProNumber,
					sp.CustomerID,
					sp.ServiceTypeID,
					sp.ShipmentTypeID,
					sp.FormulaTemplateID,
					sp.FreightChargeAmount,
					sp.OtherChargeAmount,
					sp.TotalChargeAmount,
					sp.Pieces,
					sp.Weight,
					sp.CreatedAt,
				).
				Order(sp.CreatedAt.Desc()).
				Limit(previousRatesLimit)

			var rows []shipmentports.PreviousRateSummary
			var total keel.RowCount
			if err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				keel.IntoSlice(items, &rows),
				keel.CountEstimateInto(q, pagination.ExactTotalUpTo, &total),
			); err != nil {
				return nil, err
			}

			entities := make([]*shipmentports.PreviousRateSummary, len(rows))
			for i := range rows {
				entities[i] = &rows[i]
			}

			return listquery.Result(entities, total), nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		quotaReq := shipmentQuotaRequest(entity.OrganizationID, entity.BusinessUnitID, 1)
		if err := quotaservice.Preflight(ctx, r.quota, &quotaReq); err != nil {
			return nil, err
		}

		locationCode, businessUnitCode, err := ResolveSequenceCodes(ctx, r.db.Keel(ctx), entity)
		if err != nil {
			return nil, err
		}

		proNumber, err := r.generator.GenerateShipmentProNumber(
			ctx,
			entity.OrganizationID,
			entity.BusinessUnitID,
			locationCode,
			businessUnitCode,
		)
		if err != nil {
			return nil, err
		}
		entity.ProNumber = proNumber

		newOrder, err := r.buildAutoOrder(ctx, entity, locationCode, businessUnitCode)
		if err != nil {
			return nil, err
		}

		if err = r.insertShipment(ctx, entity, newOrder, quotaReq); err != nil {
			return nil, busy(err, entity.ID)
		}

		return entity, nil
	})
}

func (r *repository) insertShipment(
	ctx context.Context,
	entity *shipment.Shipment,
	newOrder *order.Order,
	quotaReq platformplanports.QuotaRequest,
) error {
	if err := quotaservice.EnforceAll(ctx, r.quota, quotaReq); err != nil {
		return err
	}

	if newOrder != nil {
		if _, err := r.orderRepository.Create(ctx, newOrder); err != nil {
			return err
		}
		entity.OrderID = newOrder.ID
	}

	if err := shipmentdb.Shipments.Create(ctx, r.db.Keel(ctx), entity); err != nil {
		return err
	}

	if err := r.moveRepository.SyncForShipment(ctx, entity); err != nil {
		return err
	}

	if err := r.additionalChargeRepository.SyncForShipment(ctx, entity); err != nil {
		return err
	}

	if err := r.chargeAllocationRepository.SyncForShipment(ctx, entity); err != nil {
		return err
	}

	return r.commodityRepository.SyncForShipment(ctx, entity)
}

func (r *repository) buildAutoOrder(
	ctx context.Context,
	entity *shipment.Shipment,
	locationCode, businessUnitCode string,
) (*order.Order, error) {
	if !entity.OrderID.IsNil() {
		return nil, nil
	}

	orderNumber, err := r.generator.GenerateOrderNumber(
		ctx,
		entity.OrganizationID,
		entity.BusinessUnitID,
		locationCode,
		businessUnitCode,
	)
	if err != nil {
		return nil, err
	}

	return &order.Order{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		CustomerID:     entity.CustomerID,
		OwnerID:        entity.OwnerID,
		EnteredByID:    entity.EnteredByID,
		Status:         order.StatusConfirmed,
		OrderNumber:    orderNumber,
		CurrencyCode:   "USD",
		TotalAmount:    entity.TotalChargeAmount,
	}, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		err := func() error {
			sp := shipmentdb.Shipments
			if err := sp.SaveExcept(
				ctx,
				r.db.Keel(ctx),
				entity,
				sp.BillingTransferStatus,
				sp.TransferredToBillingAt,
				sp.IdempotencyKey,
			); err != nil {
				return err
			}

			if err := r.moveRepository.SyncForShipment(ctx, entity); err != nil {
				return err
			}

			if err := r.additionalChargeRepository.SyncForShipment(ctx, entity); err != nil {
				return err
			}

			if err := r.chargeAllocationRepository.SyncForShipment(ctx, entity); err != nil {
				return err
			}

			return r.commodityRepository.SyncForShipment(ctx, entity)
		}()
		if err != nil {
			return nil, busy(err, entity.ID)
		}

		return entity, nil
	})
}

func (r *repository) UpdateOperationalLifecycle(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		sp := shipmentdb.Shipments
		err := sp.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			sp.Status,
			sp.ActualShipDate,
			sp.ActualDeliveryDate,
		)
		if err == nil {
			err = r.moveRepository.SyncForShipment(ctx, entity)
		}
		if err != nil {
			return nil, busy(err, entity.ID)
		}

		return entity, nil
	})
}

func (r *repository) MarkReadyToInvoice(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		sp := shipmentdb.Shipments
		if err := sp.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			sp.Status,
			sp.MarkedReadyToBillAt,
		); err != nil {
			return nil, busy(err, entity.ID)
		}

		return entity, nil
	})
}

func (r *repository) UpdateDerivedState(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		sp := shipmentdb.Shipments
		err := sp.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			sp.Status,
			sp.ActualShipDate,
			sp.ActualDeliveryDate,
			sp.FormulaTemplateID,
			sp.FreightChargeAmount,
			sp.TariffInputs,
			sp.FreightLines,
			sp.OtherChargeAmount,
			sp.TotalChargeAmount,
			sp.RatingDetail,
			sp.MarkedReadyToBillAt,
			sp.BilledAt,
			sp.RateLocked,
		)
		if err == nil {
			err = r.additionalChargeRepository.SyncForShipment(ctx, entity)
		}
		if err == nil {
			err = r.chargeAllocationRepository.SyncForShipment(ctx, entity)
		}
		if err != nil {
			return nil, busy(err, entity.ID)
		}

		return entity, nil
	})
}

func (r *repository) updateOne(
	ctx context.Context,
	id pulid.ID,
	q *keel.Query[*shipment.Shipment],
	sets ...keel.Assignment,
) (*shipment.Shipment, error) {
	rows, err := q.UpdateReturning(ctx, r.db.Keel(ctx), sets...)
	if err != nil {
		return nil, shipmentErrors.Map(err, id.String())
	}
	if err = dberror.CheckAffected(int64(len(rows)), shipmentEntity, id.String()); err != nil {
		return nil, err
	}

	return rows[0], nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *shipmentports.UpdateShipmentStatusRequest,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		sp := shipmentdb.Shipments
		return r.updateOne(
			ctx,
			req.ShipmentID,
			sp.Where(sp.ID.Eq(req.ShipmentID), sp.Version.Eq(req.Version)),
			sp.Status.To(req.Status),
		)
	})
}

func (r *repository) Cancel(
	ctx context.Context,
	req *shipmentports.CancelShipmentRequest,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		sp := shipmentdb.Shipments
		entity, err := r.updateOne(
			ctx,
			req.ShipmentID,
			sp.Where(
				sp.ID.Eq(req.ShipmentID),
				sp.Version.Eq(req.ExpectedVersion),
				sp.Status.NotIn(shipmentvocab.StatusCanceled, shipmentvocab.StatusInvoiced),
			),
			sp.Status.To(shipmentvocab.StatusCanceled),
			sp.CanceledAt.To(req.CanceledAt),
			sp.CanceledByID.To(req.CanceledByID),
			sp.CancelReason.To(req.CancelReason),
		)
		if err == nil {
			err = r.cancelShipmentComponents(ctx, []pulid.ID{req.ShipmentID}, scopedTenant)
		}
		if err != nil {
			return nil, busy(err, req.ShipmentID)
		}

		return entity, nil
	})
}

func (r *repository) Uncancel(
	ctx context.Context,
	req *shipmentports.UncancelShipmentRequest,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		sp := shipmentdb.Shipments
		entity, err := r.updateOne(
			ctx,
			req.ShipmentID,
			sp.Where(
				sp.ID.Eq(req.ShipmentID),
				sp.Version.Eq(req.ExpectedVersion),
				sp.Status.Eq(shipmentvocab.StatusCanceled),
			),
			sp.Status.To(restoredShipmentStatus(req)),
			sp.CanceledAt.ToNull(),
			sp.CanceledByID.ToNull(),
			sp.CancelReason.ToNull(),
		)
		if err == nil {
			err = r.uncancelShipmentComponents(ctx, req)
		}
		if err != nil {
			return nil, busy(err, req.ShipmentID)
		}

		return entity, nil
	})
}

func (r *repository) TransferOwnership(
	ctx context.Context,
	req *shipmentports.TransferOwnershipRequest,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		sp := shipmentdb.Shipments
		return r.updateOne(
			ctx,
			req.ShipmentID,
			sp.Where(sp.ID.Eq(req.ShipmentID)),
			sp.OwnerID.To(req.OwnerID),
		)
	})
}

func (r *repository) FindByExternalReference(
	ctx context.Context,
	req *shipmentports.ExternalReferenceCheckRequest,
) (*shipmentports.DuplicateBOLResult, error) {
	reference := strings.TrimSpace(req.ExternalReference)
	if reference == "" || req.CustomerID.IsNil() {
		return nil, nil //nolint:nilnil // no reference never collides
	}
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentports.DuplicateBOLResult, error) {
			sp := shipmentdb.Shipments
			q := sp.Pick(sp.ID, sp.ProNumber).Where(
				sp.CustomerID.Eq(req.CustomerID),
				sp.ExternalReference.Lower().Eq(strings.ToLower(reference)),
				sp.Status.Ne(shipmentvocab.StatusCanceled),
			)
			if req.ShipmentID.IsNotNil() {
				q = q.Where(sp.ID.Ne(req.ShipmentID))
			}

			found, err := keel.Into[shipmentports.DuplicateBOLResult](
				ctx,
				r.db.Keel(ctx),
				q.Limit(1),
			)
			if err != nil {
				return nil, err
			}
			if len(found) == 0 {
				return nil, nil
			}

			return &found[0], nil
		},
	)
}

func (r *repository) FindIDByIdempotencyKey(
	ctx context.Context,
	req *shipmentports.IdempotencyKeyLookupRequest,
) (pulid.ID, error) {
	if req.IdempotencyKey == "" {
		return pulid.Nil, nil
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (pulid.ID, error) {
		sp := shipmentdb.Shipments
		ids, err := keel.Into[pulid.ID](
			ctx,
			r.db.Keel(ctx),
			sp.Pick(sp.ID).Where(sp.IdempotencyKey.Eq(req.IdempotencyKey)).Limit(1),
		)
		if err != nil {
			return pulid.Nil, err
		}
		if len(ids) == 0 {
			return pulid.Nil, nil
		}

		return ids[0], nil
	})
}

func (r *repository) CheckForDuplicateBOLs(
	ctx context.Context,
	req *shipmentports.DuplicateBOLCheckRequest,
) ([]*shipmentports.DuplicateBOLResult, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.DuplicateBOLResult, error) {
			sp := shipmentdb.Shipments
			q := sp.Pick(sp.ID, sp.ProNumber).Where(
				sp.BOL.Eq(req.BOL),
				sp.Status.Ne(shipmentvocab.StatusCanceled),
			)
			if req.ShipmentID != nil {
				q = q.Where(sp.ID.Ne(pulid.ConvertFromPtr(req.ShipmentID)))
			}

			rows, err := keel.Into[shipmentports.DuplicateBOLResult](
				ctx,
				r.db.Keel(ctx),
				q.Order(sp.CreatedAt.Desc()).Limit(maxDuplicateBOLResults),
			)
			if err != nil {
				return nil, err
			}

			duplicates := make([]*shipmentports.DuplicateBOLResult, len(rows))
			for i := range rows {
				duplicates[i] = &rows[i]
			}

			return duplicates, nil
		},
	)
}

func (r *repository) BulkDuplicate(
	ctx context.Context,
	req *shipmentports.BulkDuplicateShipmentRequest,
) ([]*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*shipment.Shipment, error) {
		quotaReq := shipmentQuotaRequest(
			req.TenantInfo.OrgID,
			req.TenantInfo.BuID,
			int64(max(req.Count, 0)),
		)
		if err := quotaservice.Preflight(ctx, r.quota, &quotaReq); err != nil {
			return nil, err
		}

		source, err := LoadGraphSource(ctx, r.db.Keel(ctx), req.TenantInfo, req.ShipmentID)
		if err != nil {
			return nil, err
		}

		locationCode, businessUnitCode, err := ResolveSequenceCodes(ctx, r.db.Keel(ctx), source)
		if err != nil {
			return nil, err
		}

		proNumbers, err := r.generator.GenerateBatch(ctx, &seqgen.GenerateRequest{
			Type:             sequenceconfig.SequenceTypeProNumber,
			OrgID:            req.TenantInfo.OrgID,
			BuID:             req.TenantInfo.BuID,
			Count:            req.Count,
			LocationCode:     locationCode,
			BusinessUnitCode: businessUnitCode,
		})
		if err != nil {
			return nil, err
		}

		graph := shipmentgraph.BuildBatch(
			source,
			proNumbers,
			req.DateAnchor(timeutils.NowUnix()),
			req.TenantInfo.UserID,
		)

		orderNumbers, err := r.generator.GenerateBatch(ctx, &seqgen.GenerateRequest{
			Type:             sequenceconfig.SequenceTypeOrder,
			OrgID:            req.TenantInfo.OrgID,
			BuID:             req.TenantInfo.BuID,
			Count:            len(graph.Shipments),
			LocationCode:     locationCode,
			BusinessUnitCode: businessUnitCode,
		})
		if err != nil {
			return nil, err
		}
		autoOrders := make([]*order.Order, 0, len(graph.Shipments))
		for idx, duplicated := range graph.Shipments {
			if duplicated == nil || idx >= len(orderNumbers) {
				continue
			}
			autoOrder := shipmentgraph.BuildAutoOrder(duplicated, orderNumbers[idx])
			duplicated.OrderID = autoOrder.ID
			autoOrders = append(autoOrders, autoOrder)
		}

		quotaReq.Quantity = int64(len(graph.Shipments))
		if err = r.insertGraph(ctx, graph, autoOrders, quotaReq); err != nil {
			return nil, busy(err, req.ShipmentID)
		}

		return graph.Shipments, nil
	})
}

func (r *repository) insertGraph(
	ctx context.Context,
	graph *shipmentgraph.Batch,
	autoOrders []*order.Order,
	quotaReq platformplanports.QuotaRequest,
) error {
	if err := quotaservice.EnforceAll(ctx, r.quota, quotaReq); err != nil {
		return err
	}

	db := r.db.Keel(ctx)
	if err := orderdb.Orders.CreateMany(ctx, db, autoOrders); err != nil {
		return err
	}
	if err := shipmentdb.Shipments.CreateMany(ctx, db, graph.Shipments); err != nil {
		return err
	}
	if err := shipmentdb.ShipmentMoves.CreateMany(ctx, db, graph.Moves); err != nil {
		return err
	}
	if err := shipmentdb.Stops.CreateMany(ctx, db, graph.Stops); err != nil {
		return err
	}
	if err := shipmentdb.AdditionalCharges.CreateMany(
		ctx,
		db,
		graph.AdditionalCharges,
	); err != nil {
		return err
	}
	if err := shipmentdb.ChargeAllocations.CreateMany(
		ctx,
		db,
		graph.ChargeAllocations,
	); err != nil {
		return err
	}
	if err := shipmentdb.ShipmentCommodities.CreateMany(ctx, db, graph.Commodities); err != nil {
		return err
	}

	return shipmentprojection.Refresh(ctx, db, shipmentIDsFromEntities(graph.Shipments)...)
}

func (r *repository) PlanDuplicate(
	ctx context.Context,
	req *shipmentports.BulkDuplicateShipmentRequest,
) (*shipmentports.ShipmentDuplicatePlan, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentports.ShipmentDuplicatePlan, error) {
			source, err := LoadGraphSource(
				ctx,
				r.db.Keel(ctx),
				req.TenantInfo,
				req.ShipmentID,
			)
			if err != nil {
				return nil, err
			}

			graph := shipmentgraph.BuildBatch(
				source,
				make([]string, req.Count),
				req.DateAnchor(timeutils.NowUnix()),
				req.TenantInfo.UserID,
			)

			return &shipmentports.ShipmentDuplicatePlan{
				Source: source,
				Copies: graph.Shipments,
			}, nil
		},
	)
}

// ResolveSequenceCodes resolves the location and business-unit codes feeding
// pro-number/order-number sequence generation for copies of the given shipment.
func ResolveSequenceCodes(
	ctx context.Context,
	db *keel.DB,
	entity *shipment.Shipment,
) (locationCode, businessUnitCode string, err error) {
	bu := tenantdb.BusinessUnits
	businessUnitCode, err = keel.IntoOne[string](
		ctx,
		db,
		bu.Pick(bu.Code).Where(bu.ID.Eq(entity.BusinessUnitID)),
	)
	if err != nil {
		return "", "", dberror.HandleNotFoundError(err, "Sequence")
	}

	if len(entity.Moves) == 0 || len(entity.Moves[0].Stops) == 0 {
		return "", businessUnitCode, nil
	}

	loc := locationdb.Locations
	q := loc.Pick(loc.Code).Where(loc.ID.Eq(entity.Moves[0].Stops[0].LocationID))
	if dbscope.IsSystem(ctx) {
		q = q.AllTenants().Where(
			loc.OrganizationID.Eq(entity.OrganizationID),
			loc.BusinessUnitID.Eq(entity.BusinessUnitID),
		)
	}

	locationCode, err = keel.IntoOne[string](ctx, db, q)
	if err != nil {
		return "", "", dberror.HandleNotFoundError(err, "Sequence")
	}

	return locationCode, businessUnitCode, nil
}

func shipmentQuotaRequest(orgID, buID pulid.ID, quantity int64) platformplanports.QuotaRequest {
	return platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		Meter:      platformcatalog.MeterShipmentsTotal,
		Quantity:   quantity,
	}
}

var errNoShipmentScope = errors.New("shipment graph source needs a tenant")

func tenantShipments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*keel.Query[*shipment.Shipment], error) {
	sp := shipmentdb.Shipments
	if !dbscope.IsSystem(ctx) {
		return sp.Query(), nil
	}
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return nil, errNoShipmentScope
	}

	return sp.AllTenants().Where(
		sp.OrganizationID.Eq(tenantInfo.OrgID),
		sp.BusinessUnitID.Eq(tenantInfo.BuID),
	), nil
}
