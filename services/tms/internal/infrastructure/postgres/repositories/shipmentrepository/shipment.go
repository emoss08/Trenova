package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB                         *postgres.Connection
	Logger                     *zap.Logger
	Generator                  seqgen.Generator
	MoveRepository             repositories.ShipmentMoveRepository
	AdditionalChargeRepository repositories.ShipmentAdditionalChargeRepository
	CommodityRepository        repositories.ShipmentCommodityRepository
	OrderRepository            repositories.OrderRepository
}

type repository struct {
	db                         *postgres.Connection
	l                          *zap.Logger
	generator                  seqgen.Generator
	moveRepository             repositories.ShipmentMoveRepository
	additionalChargeRepository repositories.ShipmentAdditionalChargeRepository
	commodityRepository        repositories.ShipmentCommodityRepository
	orderRepository            repositories.OrderRepository
}

//nolint:gocritic // This is a constructor function
func New(p Params) repositories.ShipmentRepository {
	return &repository{
		db:                         p.DB,
		l:                          p.Logger.Named("postgres.shipment-repository"),
		generator:                  p.Generator,
		moveRepository:             p.MoveRepository,
		additionalChargeRepository: p.AdditionalChargeRepository,
		commodityRepository:        p.CommodityRepository,
		orderRepository:            p.OrderRepository,
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	dba := r.db.DBForContext(ctx)

	total, err := dba.
		NewSelect().
		Model((*shipment.Shipment)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return countShipmentListQuery(sq, req)
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*shipment.Shipment]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(items *[]*shipment.Shipment) *bun.SelectQuery {
			return dba.NewSelect().
				Model(items).
				ColumnExpr(buncolgen.ShipmentTable.All())
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return cursorFilterQuery(sq, req)
		},
	})
	if err != nil {
		return nil, err
	}

	if req.ShipmentOptions.ExpandShipmentDetails {
		if err = r.hydrateMoves(ctx, result.Items); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	sp := buncolgen.ShipmentColumns
	entity := new(shipment.Shipment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentScopeTenant(sq, req.TenantInfo).
				Where(sp.ID.Eq(), req.ID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return standardShipmentFilter(sq, req.ShipmentOptions)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Shipment")
	}

	if req.ExpandShipmentDetails {
		if err = r.hydrateMoves(ctx, []*shipment.Shipment{entity}); err != nil {
			return nil, err
		}
	}

	return entity, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req *repositories.GetShipmentsByIDsRequest,
) ([]*shipment.Shipment, error) {
	sp := buncolgen.ShipmentColumns
	entities := make([]*shipment.Shipment, 0, len(req.ShipmentIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentScopeTenant(sq, req.TenantInfo).
				Where(sp.ID.In(), bun.List(req.ShipmentIDs))
		}).
		Relation(buncolgen.ShipmentRelations.Customer).
		Relation(buncolgen.ShipmentRelations.ServiceType).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Shipment")
	}

	return entities, nil
}

func (r *repository) ListRatedByFormulaTemplate(
	ctx context.Context,
	req *repositories.ListRatedByFormulaTemplateRequest,
) ([]*shipment.Shipment, error) {
	sp := buncolgen.ShipmentColumns
	sm := buncolgen.ShipmentMoveColumns
	stp := buncolgen.StopColumns
	entities := make([]*shipment.Shipment, 0, req.Limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentScopeTenant(sq, req.TenantInfo).
				Where(sp.FormulaTemplateID.Eq(), req.TemplateID)
		}).
		Relation(buncolgen.ShipmentRelations.Customer).
		Relation(buncolgen.ShipmentRelations.TractorType).
		Relation(buncolgen.ShipmentRelations.TrailerType).
		RelationWithOpts(buncolgen.ShipmentRelations.Moves, bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order(sm.Sequence.OrderAsc())
			},
		}).
		RelationWithOpts(
			buncolgen.Rel(buncolgen.ShipmentRelations.Moves, buncolgen.ShipmentMoveRelations.Stops),
			bun.RelationOpts{
				Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Order(stp.Sequence.OrderAsc())
				},
			},
		).
		RelationWithOpts(buncolgen.ShipmentRelations.AdditionalCharges, bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation(buncolgen.AdditionalChargeRelations.AccessorialCharge)
			},
		}).
		RelationWithOpts(buncolgen.ShipmentRelations.Commodities, bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation(buncolgen.ShipmentCommodityRelations.Commodity).
					Relation(buncolgen.Rel(
						buncolgen.ShipmentCommodityRelations.Commodity,
						buncolgen.CommodityRelations.HazardousMaterial))
			},
		}).
		Order(sp.CreatedAt.OrderDesc()).
		Limit(req.Limit).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.ShipmentSelectOptionsRequest,
) (*pagination.ListResult[*shipment.Shipment], error) {
	sp := buncolgen.ShipmentColumns

	return dbhelper.SelectOptions[*shipment.Shipment](
		ctx,
		r.db.DBForContext(ctx),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				sp.ID,
				sp.CreatedAt,
				sp.Status,
				sp.ProNumber,
				sp.BOL,
			},
			OrgColumnRef:     &sp.OrganizationID,
			BuColumnRef:      &sp.BusinessUnitID,
			SearchColumnRefs: []buncolgen.Column{sp.ProNumber, sp.BOL},
			EntityName:       "Shipment",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				if !req.CustomerID.IsNil() {
					q = q.Where(sp.CustomerID.Eq(), req.CustomerID)
				}
				return q.Order(sp.CreatedAt.OrderDesc())
			},
		},
	)
}

func (r *repository) GetUnassigned(
	ctx context.Context,
	req *repositories.GetUnassignedShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	dba := r.db.DBForContext(ctx)

	total, err := dba.
		NewSelect().
		Model((*shipment.Shipment)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			countReq := repositories.ListShipmentsRequest{
				Filter: req.Filter,
				ShipmentOptions: repositories.ShipmentOptions{
					Status: string(shipment.StatusNew),
				},
			}
			return countShipmentListQuery(sq, &countReq).
				Where("NOT EXISTS (?)", unassignedShipmentPredicate(dba))
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*shipment.Shipment]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(items *[]*shipment.Shipment) *bun.SelectQuery {
			return dba.NewSelect().
				Model(items).
				ColumnExpr(buncolgen.ShipmentTable.All())
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return unassignedShipmentListQuery(sq, dba, req)
		},
	})
	if err != nil {
		return nil, err
	}

	if req.ShipmentOptions.ExpandShipmentDetails {
		if err = r.hydrateMoves(ctx, result.Items); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (r *repository) GetPreviousRates(
	ctx context.Context,
	req *repositories.GetPreviousRatesRequest,
) (*pagination.ListResult[*repositories.PreviousRateSummary], error) {
	sp := buncolgen.ShipmentColumns
	entities := make([]*repositories.PreviousRateSummary, 0, 50)

	baseQuery := func(dba bun.IDB) *bun.SelectQuery {
		originCTE, destinationCTE := buildPreviousRatesCTEs(
			dba,
			req.OriginLocationID,
			req.DestinationLocationID,
		)

		query := dba.NewSelect().
			With("origin_shipments", originCTE).
			With("destination_shipments", destinationCTE).
			TableExpr(buncolgen.ShipmentTable.Name+" AS "+buncolgen.ShipmentTable.Alias).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ShipmentScopeTenant(sq, req.TenantInfo).
					Where(sp.ShipmentTypeID.Eq(), req.ShipmentTypeID).
					Where(sp.ServiceTypeID.Eq(), req.ServiceTypeID).
					Where(sp.Status.Eq(), shipment.StatusInvoiced).
					Where("sp.id IN (SELECT shipment_id FROM origin_shipments)").
					Where("sp.id IN (SELECT shipment_id FROM destination_shipments)")
			})

		if req.CustomerID != nil {
			query = query.Where(sp.CustomerID.Eq(), pulid.ConvertFromPtr(req.CustomerID))
		}

		if req.ExcludeShipmentID != nil {
			query = query.Where(sp.ID.Ne(), pulid.ConvertFromPtr(req.ExcludeShipmentID))
		}

		return query
	}

	countQuery := baseQuery(r.db.DBForContext(ctx)).
		ColumnExpr("COUNT(*)")

	total := 0
	if err := countQuery.Scan(ctx, &total); err != nil {
		return nil, err
	}

	itemsQuery := baseQuery(r.db.DBForContext(ctx)).
		ColumnExpr(sp.ID.As("shipment_id")).
		ColumnExpr(sp.ProNumber.Qualified()).
		ColumnExpr(sp.CustomerID.Qualified()).
		ColumnExpr(sp.ServiceTypeID.Qualified()).
		ColumnExpr(sp.ShipmentTypeID.Qualified()).
		ColumnExpr(sp.FormulaTemplateID.Qualified()).
		ColumnExpr(sp.FreightChargeAmount.Qualified()).
		ColumnExpr(sp.OtherChargeAmount.Qualified()).
		ColumnExpr(sp.TotalChargeAmount.Qualified()).
		ColumnExpr(sp.RatingUnit.Qualified()).
		ColumnExpr(sp.Pieces.Qualified()).
		ColumnExpr(sp.Weight.Qualified()).
		ColumnExpr(sp.CreatedAt.Qualified()).
		Order(sp.CreatedAt.OrderDesc()).
		Limit(50)

	if err := itemsQuery.Scan(ctx, &entities); err != nil {
		return nil, err
	}

	return &pagination.ListResult[*repositories.PreviousRateSummary]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	locationCode, businessUnitCode, err := r.resolveSequenceCodes(ctx, entity)
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

	// A shipment created outside a commercial order gets its own single-leg order so
	// everything has a commercial parent. The order number is minted before the tx,
	// consistent with the pro-number lifecycle (a rolled-back tx burns a number).
	newOrder, err := r.buildAutoOrder(ctx, entity, locationCode, businessUnitCode)
	if err != nil {
		return nil, err
	}

	err = r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		if newOrder != nil {
			if err = r.orderRepository.CreateInTx(c, tx, newOrder); err != nil {
				return err
			}
			entity.OrderID = newOrder.ID
		}

		if _, err = r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); err != nil {
			return err
		}

		if err = r.moveRepository.SyncForShipment(c, tx, entity); err != nil {
			return err
		}

		if err = r.additionalChargeRepository.SyncForShipment(c, tx, entity); err != nil {
			return err
		}

		return r.commodityRepository.SyncForShipment(c, tx, entity)
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Shipment is busy. Retry the request.",
		)
	}

	return entity, nil
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
	sp := buncolgen.ShipmentColumns
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		ov := entity.Version
		entity.Version++

		results, err := r.db.DBForContext(c).NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ShipmentScopeTenantUpdate(uq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).
					Where(sp.ID.Eq(), entity.ID).
					Where(sp.Version.Eq(), ov)
			}).
			Returning("*").
			Exec(c)
		if err != nil {
			return err
		}

		if err = dberror.CheckRowsAffected(results, "Shipment", entity.ID.String()); err != nil {
			return err
		}

		if err = r.moveRepository.SyncForShipment(c, tx, entity); err != nil {
			return err
		}

		if err = r.additionalChargeRepository.SyncForShipment(c, tx, entity); err != nil {
			return err
		}

		return r.commodityRepository.SyncForShipment(c, tx, entity)
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Shipment is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) UpdateOperationalLifecycle(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	sp := buncolgen.ShipmentColumns
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		ov := entity.Version
		entity.Version++

		results, err := r.db.DBForContext(c).NewUpdate().
			Model(entity).
			Column(
				sp.Status.Bare(),
				sp.ActualShipDate.Bare(),
				sp.ActualDeliveryDate.Bare(),
				sp.Version.Bare(),
				sp.UpdatedAt.Bare(),
			).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ShipmentScopeTenantUpdate(uq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).
					Where(sp.ID.Eq(), entity.ID).
					Where(sp.Version.Eq(), ov)
			}).
			Returning("*").
			Exec(c)
		if err != nil {
			return err
		}

		if err = dberror.CheckRowsAffected(results, "Shipment", entity.ID.String()); err != nil {
			return err
		}

		return r.moveRepository.SyncForShipment(c, tx, entity)
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Shipment is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) UpdateDerivedState(
	ctx context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	sp := buncolgen.ShipmentColumns
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		ov := entity.Version
		entity.Version++

		results, err := r.db.DBForContext(c).NewUpdate().
			Model(entity).
			Column(
				sp.Status.Bare(),
				sp.ActualShipDate.Bare(),
				sp.ActualDeliveryDate.Bare(),
				sp.FormulaTemplateID.Bare(),
				sp.FreightChargeAmount.Bare(),
				sp.BaseRate.Bare(),
				sp.OtherChargeAmount.Bare(),
				sp.TotalChargeAmount.Bare(),
				sp.RatingDetail.Bare(),
				sp.BillingTransferStatus.Bare(),
				sp.TransferredToBillingAt.Bare(),
				sp.MarkedReadyToBillAt.Bare(),
				sp.BilledAt.Bare(),
				sp.Version.Bare(),
				sp.UpdatedAt.Bare(),
			).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ShipmentScopeTenantUpdate(uq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).
					Where(sp.ID.Eq(), entity.ID).
					Where(sp.Version.Eq(), ov)
			}).
			Returning("*").
			Exec(c)
		if err != nil {
			return err
		}

		if err = dberror.CheckRowsAffected(results, "Shipment", entity.ID.String()); err != nil {
			return err
		}

		return r.additionalChargeRepository.SyncForShipment(c, tx, entity)
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Shipment is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdateShipmentStatusRequest,
) (*shipment.Shipment, error) {
	sp := buncolgen.ShipmentColumns
	entity := new(shipment.Shipment)

	results, err := r.db.DBForContext(ctx).NewUpdate().
		Model(entity).
		Set(sp.Status.Set(), req.Status).
		Set(sp.Version.Inc(1)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ShipmentScopeTenantUpdate(uq, req.TenantInfo).
				Where(sp.ID.Eq(), req.ShipmentID).
				Where(sp.Version.Eq(), req.Version)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Shipment", req.ShipmentID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Cancel(
	ctx context.Context,
	req *repositories.CancelShipmentRequest,
) (*shipment.Shipment, error) {
	sp := buncolgen.ShipmentColumns
	entity := new(shipment.Shipment)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		results, err := tx.NewUpdate().
			Model(entity).
			Set(sp.Status.Set(), shipment.StatusCanceled).
			Set(sp.CanceledAt.Set(), req.CanceledAt).
			Set(sp.CanceledByID.Set(), req.CanceledByID).
			Set(sp.CancelReason.Set(), req.CancelReason).
			Set(sp.Version.Inc(1)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ShipmentScopeTenantUpdate(uq, req.TenantInfo).
					Where(sp.ID.Eq(), req.ShipmentID)
			}).
			Returning("*").
			Exec(c)
		if err != nil {
			return err
		}

		if err = dberror.CheckRowsAffected(
			results, "Shipment",
			req.ShipmentID.String(),
		); err != nil {
			return err
		}

		return r.cancelShipmentComponents(c, tx, req.ShipmentID)
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Shipment is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) Uncancel(
	ctx context.Context,
	req *repositories.UncancelShipmentRequest,
) (*shipment.Shipment, error) {
	sp := buncolgen.ShipmentColumns
	entity := new(shipment.Shipment)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		results, err := tx.NewUpdate().
			Model(entity).
			Set(sp.Status.Set(), shipment.StatusNew).
			Set(sp.CanceledAt.Set(), nil).
			Set(sp.CanceledByID.Set(), pulid.Nil).
			Set(sp.CancelReason.Set(), "").
			Set(sp.Version.Inc(1)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ShipmentScopeTenantUpdate(uq, req.TenantInfo).
					Where(sp.ID.Eq(), req.ShipmentID)
			}).
			Returning("*").
			Exec(c)
		if err != nil {
			return err
		}

		if err = dberror.CheckRowsAffected(
			results,
			"Shipment",
			req.ShipmentID.String(),
		); err != nil {
			return err
		}

		return r.uncancelShipmentComponents(c, tx, req.ShipmentID)
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Shipment is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) TransferOwnership(
	ctx context.Context,
	req *repositories.TransferOwnershipRequest,
) (*shipment.Shipment, error) {
	sp := buncolgen.ShipmentColumns
	entity := new(shipment.Shipment)

	results, err := r.db.DBForContext(ctx).NewUpdate().
		Model(entity).
		Set(sp.OwnerID.Set(), req.OwnerID).
		Set(sp.Version.Inc(1)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ShipmentScopeTenantUpdate(uq, req.TenantInfo).
				Where(sp.ID.Eq(), req.ShipmentID)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	if err = dberror.CheckRowsAffected(
		results,
		"Shipment",
		req.ShipmentID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) CheckForDuplicateBOLs(
	ctx context.Context,
	req *repositories.DuplicateBOLCheckRequest,
) ([]*repositories.DuplicateBOLResult, error) {
	sp := buncolgen.ShipmentColumns
	duplicates := make([]*repositories.DuplicateBOLResult, 0)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Column(sp.ID.Bare(), sp.ProNumber.Bare()).
		Model((*shipment.Shipment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentScopeTenant(sq, req.TenantInfo).
				Where(sp.BOL.Eq(), req.BOL).
				Where(sp.Status.Ne(), shipment.StatusCanceled)
		})

	if req.ShipmentID != nil {
		query = query.Where(sp.ID.Ne(), pulid.ConvertFromPtr(req.ShipmentID))
	}

	if err := query.Scan(ctx, &duplicates); err != nil {
		return nil, err
	}

	return duplicates, nil
}

func (r *repository) BulkDuplicate(
	ctx context.Context,
	req *repositories.BulkDuplicateShipmentRequest,
) ([]*shipment.Shipment, error) {
	source, err := r.getDuplicateSource(ctx, req)
	if err != nil {
		return nil, err
	}

	locationCode, businessUnitCode, err := r.resolveSequenceCodes(ctx, source)
	if err != nil {
		return nil, err
	}

	proNumbers, err := r.generator.GenerateBatch(ctx, &seqgen.GenerateRequest{
		Type:             tenant.SequenceTypeProNumber,
		OrgID:            req.TenantInfo.OrgID,
		BuID:             req.TenantInfo.BuID,
		Count:            req.Count,
		LocationCode:     locationCode,
		BusinessUnitCode: businessUnitCode,
	})
	if err != nil {
		return nil, err
	}

	graph := buildDuplicatedShipmentGraph(
		source,
		proNumbers,
		req.OverrideDates,
		req.TenantInfo.UserID,
	)

	err = r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		if len(graph.shipments) > 0 {
			if _, insertErr := tx.
				NewInsert().
				Model(&graph.shipments).
				Returning("NULL").
				Exec(c); insertErr != nil {
				return insertErr
			}
		}

		if len(graph.moves) > 0 {
			if _, insertErr := tx.
				NewInsert().
				Model(&graph.moves).
				Returning("NULL").
				Exec(c); insertErr != nil {
				return insertErr
			}
		}

		if len(graph.stops) > 0 {
			if _, insertErr := tx.
				NewInsert().
				Model(&graph.stops).
				Returning("NULL").
				Exec(c); insertErr != nil {
				return insertErr
			}
		}

		if len(graph.additionalCharges) > 0 {
			if _, insertErr := tx.
				NewInsert().
				Model(&graph.additionalCharges).
				Returning("NULL").
				Exec(c); insertErr != nil {
				return insertErr
			}
		}

		if len(graph.commodities) > 0 {
			if _, insertErr := tx.
				NewInsert().
				Model(&graph.commodities).
				Returning("NULL").
				Exec(c); insertErr != nil {
				return insertErr
			}
		}

		return nil
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Shipment is busy. Retry the request.",
		)
	}

	return graph.shipments, nil
}

func (r *repository) getDuplicateSource(
	ctx context.Context,
	req *repositories.BulkDuplicateShipmentRequest,
) (*shipment.Shipment, error) {
	sp := buncolgen.ShipmentColumns
	sm := buncolgen.ShipmentMoveColumns
	stp := buncolgen.StopColumns
	entity := new(shipment.Shipment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentScopeTenant(sq, req.TenantInfo).
				Where(sp.ID.Eq(), req.ShipmentID)
		}).
		RelationWithOpts(buncolgen.ShipmentRelations.Moves, bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order(sm.Sequence.OrderAsc())
			},
		}).
		RelationWithOpts(buncolgen.Rel(buncolgen.ShipmentRelations.Moves, buncolgen.ShipmentMoveRelations.Stops), bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order(stp.Sequence.OrderAsc())
			},
		}).
		Relation(buncolgen.ShipmentRelations.AdditionalCharges).
		Relation(buncolgen.ShipmentRelations.Commodities).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Shipment")
	}

	return entity, nil
}

func (r *repository) resolveSequenceCodes(
	ctx context.Context,
	entity *shipment.Shipment,
) (locationCode, businessUnitCode string, err error) {
	bu := new(tenant.BusinessUnit)
	err = r.db.DBForContext(ctx).NewSelect().
		Model(bu).
		Column(buncolgen.BusinessUnitColumns.Code.Bare()).
		Where(buncolgen.BusinessUnitColumns.ID.Eq(), entity.BusinessUnitID).
		Scan(ctx)
	if err != nil {
		return "", "", dberror.HandleNotFoundError(err, "Sequence")
	}

	businessUnitCode = bu.Code

	if len(entity.Moves) > 0 && len(entity.Moves[0].Stops) > 0 {
		var code string
		err = r.db.DBForContext(ctx).NewSelect().
			TableExpr("locations").
			Column("code").
			Where("id = ?", entity.Moves[0].Stops[0].LocationID).
			Scan(ctx, &code)
		if err != nil {
			return "", "", dberror.HandleNotFoundError(err, "Sequence")
		}

		locationCode = code
	}

	return locationCode, businessUnitCode, nil
}
