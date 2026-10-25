package detentionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type OccurrenceParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type occurrenceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewOccurrenceRepository(
	p OccurrenceParams,
) detentionports.DetentionOccurrenceRepository {
	return &occurrenceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.detention-occurrence-repository"),
	}
}

type occurrenceQuery = keel.Query[*detention.DetentionOccurrence]

func withOccurrenceNames(q *occurrenceQuery) *occurrenceQuery {
	o := detentiondb.DetentionOccurrences
	loc := locationdb.Locations
	cus := customerdb.Customers
	sp := shipmentdb.Shipments

	return q.
		LeftJoin(loc, loc.ID.EqCol(o.LocationID)).
		LeftJoin(cus, cus.ID.EqCol(o.CustomerID)).
		LeftJoin(sp, sp.ID.EqCol(o.ShipmentID)).
		Compute(o.LocationName, keel.DynOf(loc.Name).Coalesce("")).
		Compute(o.CustomerName, keel.DynOf(cus.Name).Coalesce("")).
		Compute(o.ShipmentProNumber, keel.DynOf(sp.ProNumber).Coalesce(""))
}

func namedOccurrences() *occurrenceQuery {
	return withOccurrenceNames(detentiondb.DetentionOccurrences.Query())
}

func optionalIDEq(col interface{ Eq(pulid.ID) keel.Pred }, id *pulid.ID) keel.Pred {
	if id == nil || id.IsNil() {
		return nil
	}

	return col.Eq(*id)
}

func (r *occurrenceRepository) List(
	ctx context.Context,
	req *detentionports.ListDetentionOccurrencesRequest,
) (*pagination.ListResult[*detention.DetentionOccurrence], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*detention.DetentionOccurrence], error) {
			o := detentiondb.DetentionOccurrences
			q := o.Where(
				optionalIDEq(o.ShipmentID, req.ShipmentID),
				optionalIDEq(o.CustomerID, req.CustomerID),
				optionalIDEq(o.LocationID, req.LocationID),
				keel.When(req.Status != "", o.Status.Eq(detention.OccurrenceStatus(req.Status))),
				keel.When(req.OpenOnly, o.IsOpen.IsTrue()),
			)

			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				q,
				&listquery.Request[detention.DetentionOccurrence]{
					Filter: req.Filter,
					Shape:  withOccurrenceNames,
				},
			)
			if err != nil {
				r.l.Error("failed to list detention occurrences", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *occurrenceRepository) GetByID(
	ctx context.Context,
	req *detentionports.GetDetentionOccurrenceByIDRequest,
) (*detention.DetentionOccurrence, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*detention.DetentionOccurrence, error) {
		o := detentiondb.DetentionOccurrences
		ev := o.Evidence()
		nt := o.Notices()

		q := namedOccurrences().Where(o.ID.Eq(req.OccurrenceID))
		if req.IncludeEvidence {
			q = q.Load(ev.Order(ev.Sequence.Asc()))
		}
		if req.IncludeNotices {
			q = q.Load(nt.Order(nt.CreatedAt.Asc()))
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to get detention occurrence",
				zap.String("id", req.OccurrenceID.String()),
				zap.Error(err),
			)
			return nil, occurrenceErrors.Map(err, req.OccurrenceID.String())
		}

		return entity, nil
	})
}

func (r *occurrenceRepository) GetByStop(
	ctx context.Context,
	req *detentionports.GetOccurrenceByStopRequest,
) (*detention.DetentionOccurrence, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*detention.DetentionOccurrence, error) {
		o := detentiondb.DetentionOccurrences
		entity, err := o.Where(o.StopID.Eq(req.StopID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			r.l.Error("failed to get detention occurrence by stop", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *occurrenceRepository) GetByShipment(
	ctx context.Context,
	req *detentionports.GetOccurrencesByShipmentRequest,
) ([]*detention.DetentionOccurrence, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detention.DetentionOccurrence, error) {
			o := detentiondb.DetentionOccurrences
			q := namedOccurrences().Where(o.ShipmentID.Eq(req.ShipmentID))
			if req.IncludeEvidence {
				ev := o.Evidence()
				q = q.Load(ev.Order(ev.Sequence.Asc()))
			}

			entities, err := q.Order(o.ClockStartAt.Asc()).All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list detention occurrences for shipment", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *occurrenceRepository) ListOpen(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*detention.DetentionOccurrence, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detention.DetentionOccurrence, error) {
			o := detentiondb.DetentionOccurrences
			entities, err := namedOccurrences().
				Where(o.IsOpen.IsTrue(), o.Status.Ne(detention.OccurrenceStatusBilled)).
				Order(o.FreeTimeExpiresAt.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list open detention occurrences", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func billingHold() []keel.Pred {
	o := detentiondb.DetentionOccurrences
	return []keel.Pred{
		o.Status.Eq(detention.OccurrenceStatusPending),
		o.RequiresApproval.IsTrue(),
	}
}

func (r *occurrenceRepository) ListBillingHoldsByShipments(
	ctx context.Context,
	req *detentionports.ListDetentionBillingHoldsRequest,
) ([]*detention.DetentionOccurrence, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detention.DetentionOccurrence, error) {
			if req == nil || len(req.ShipmentIDs) == 0 {
				return []*detention.DetentionOccurrence{}, nil
			}

			o := detentiondb.DetentionOccurrences
			entities, err := namedOccurrences().
				Where(billingHold()...).
				Where(o.ShipmentID.In(req.ShipmentIDs...)).
				Order(o.ShipmentID.Asc(), o.ClockStartAt.Asc(), o.ID.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list detention billing holds by shipment", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *occurrenceRepository) ListBillingHolds(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*detention.DetentionOccurrence, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detention.DetentionOccurrence, error) {
			o := detentiondb.DetentionOccurrences
			entities, err := namedOccurrences().
				Where(billingHold()...).
				Order(o.ClockStartAt.Asc(), o.ID.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list detention billing holds", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *occurrenceRepository) ListByAdditionalCharges(
	ctx context.Context,
	req *detentionports.ListOccurrencesByChargesRequest,
) ([]*detention.DetentionOccurrence, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detention.DetentionOccurrence, error) {
			if req == nil || len(req.ChargeIDs) == 0 || len(req.ShipmentIDs) == 0 ||
				len(req.Statuses) == 0 {
				return []*detention.DetentionOccurrence{}, nil
			}

			o := detentiondb.DetentionOccurrences
			entities, err := o.Where(
				o.ShipmentID.In(req.ShipmentIDs...),
				o.AdditionalChargeID.In(req.ChargeIDs...),
				o.Status.In(req.Statuses...),
			).
				Order(o.ShipmentID.Asc(), o.ClockStartAt.Asc(), o.ID.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list detention occurrences by charge", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

// ListNoticesDue powers the pre-breach alerting sweep: occurrences whose notice
// window has opened but whose notice has not gone out yet.
func (r *occurrenceRepository) ListNoticesDue(
	ctx context.Context,
	req *detentionports.ListNoticesDueRequest,
) ([]*detention.DetentionOccurrence, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detention.DetentionOccurrence, error) {
			o := detentiondb.DetentionOccurrences
			entities, err := namedOccurrences().
				Where(
					o.NotificationStatus.Eq(detention.NotificationStatusPending),
					o.NoticeDueAt.IsNotNull(),
					o.NoticeDueAt.Lte(req.Before),
				).
				Order(o.NoticeDueAt.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list detention notices due", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

// Upsert writes the occurrence keyed on its stop. Recalculation runs on every
// shipment save, so this is the hot path and must stay idempotent.
func (r *occurrenceRepository) Upsert(
	ctx context.Context,
	entity *detention.DetentionOccurrence,
) (*detention.DetentionOccurrence, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*detention.DetentionOccurrence, error) {
		existing, err := r.GetByStop(ctx, &detentionports.GetOccurrenceByStopRequest{
			StopID: entity.StopID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		})
		if err != nil {
			return nil, err
		}

		if existing == nil {
			err = detentiondb.DetentionOccurrences.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to insert detention occurrence",
					zap.String("stopId", entity.StopID.String()),
					zap.Error(err),
				)
				return nil, occurrenceErrors.Map(err, entity.ID.String())
			}
			return entity, nil
		}

		entity.ID = existing.ID
		entity.CreatedAt = existing.CreatedAt
		entity.Version = existing.Version

		return r.Update(ctx, entity)
	})
}

func (r *occurrenceRepository) Update(
	ctx context.Context,
	entity *detention.DetentionOccurrence,
) (*detention.DetentionOccurrence, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*detention.DetentionOccurrence, error) {
		err := detentiondb.DetentionOccurrences.Save(ctx, r.db.Keel(ctx), entity)
		if err != nil {
			r.l.Error(
				"failed to update detention occurrence",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, occurrenceErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

// ShipmentAccrued totals what detention has already been booked against a
// shipment so a per-shipment cap can be applied across stops.
func (r *occurrenceRepository) ShipmentAccrued(
	ctx context.Context,
	req *detentionports.GetOccurrencesByShipmentRequest,
) (decimal.Decimal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (decimal.Decimal, error) {
		o := detentiondb.DetentionOccurrences
		total, err := keel.IntoOne[decimal.NullDecimal](ctx, r.db.Keel(ctx), o.
			Pick(sumOrZero(o.BillableAmount)).
			Where(o.ShipmentID.Eq(req.ShipmentID)))
		if err != nil {
			r.l.Error("failed to total shipment detention accrual", zap.Error(err))
			return decimal.Zero, err
		}

		if !total.Valid {
			return decimal.Zero, nil
		}

		return total.Decimal, nil
	})
}

func (r *occurrenceRepository) LinkCharges(
	ctx context.Context,
	req *detentionports.LinkOccurrenceChargesRequest,
) error {
	if req == nil || req.ShipmentID.IsNil() {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		occurrences := make([]string, 0, len(req.ChargeOccurrences)*2)
		charges := make([]string, 0, len(req.ChargeOccurrences)*2)
		linked := make([]pulid.ID, 0, len(req.ChargeOccurrences)*2)
		for chargeID, occurrenceIDs := range req.ChargeOccurrences {
			if chargeID.IsNil() || len(occurrenceIDs) == 0 {
				continue
			}
			for _, occurrenceID := range occurrenceIDs {
				occurrences = append(occurrences, occurrenceID.String())
				charges = append(charges, chargeID.String())
				linked = append(linked, occurrenceID)
			}
		}

		db := r.db.Keel(ctx)
		o := detentiondb.DetentionOccurrences
		unchanged := []keel.Assignment{o.Version.ToExpr(o.Version), o.UpdatedAt.ToExpr(o.UpdatedAt)}

		if len(linked) > 0 {
			links := keel.Unnest("links", []string{"id", "charge_id"}, occurrences, charges)
			if _, err := o.With(links).
				Join(links, links.Col("id").EqCol(o.ID)).
				Where(o.ShipmentID.Eq(req.ShipmentID)).
				Update(
					ctx,
					db,
					append(unchanged, o.AdditionalChargeID.ToExpr(links.Col("charge_id")))...,
				); err != nil {
				return fmt.Errorf("link detention occurrences to charges: %w", err)
			}
		}

		if _, err := o.Where(
			o.ShipmentID.Eq(req.ShipmentID),
			o.AdditionalChargeID.IsNotNull(),
			keel.When(len(linked) > 0, o.ID.NotIn(linked...)),
		).Update(ctx, db, append(unchanged, o.AdditionalChargeID.ToNull())...); err != nil {
			return fmt.Errorf("release stale detention charge links: %w", err)
		}

		return nil
	})
}
