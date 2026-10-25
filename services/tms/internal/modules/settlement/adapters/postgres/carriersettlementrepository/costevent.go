package carriersettlementrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var costEventErrors = dberror.Mapping{Entity: "CarrierCostEvent"}

type costEventRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewCostEvent(p Params) carriersettlementports.CarrierCostEventRepository {
	return &costEventRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-cost-event-repository"),
	}
}

func (r *costEventRepository) List(
	ctx context.Context,
	req *carriersettlementports.ListCarrierCostEventsRequest,
) (*pagination.ListResult[*carriersettlement.CostEvent], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*carriersettlement.CostEvent], error) {
			ce := carriersettlementdb.CostEvents
			q := ce.Where(
				when(req.Filter.Query != "", ce.ProNumber.ContainsFold(req.Filter.Query)),
				when(!req.CarrierID.IsNil(), ce.CarrierID.Eq(req.CarrierID)),
				when(!req.ShipmentID.IsNil(), ce.ShipmentID.Eq(req.ShipmentID)),
				when(req.Status != "", ce.Status.Eq(req.Status)),
			).Load(ce.Carrier()).Order(ce.EventDate.Desc())

			result, err := listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
			if err != nil {
				return nil, fmt.Errorf("list carrier cost events: %w", err)
			}

			return result, nil
		},
	)
}

func (r *costEventRepository) ListConnection(
	ctx context.Context,
	req *carriersettlementports.ListCarrierCostEventConnectionRequest,
) (*pagination.CursorListResult[*carriersettlement.CostEvent], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*carriersettlement.CostEvent], error) {
			ce := carriersettlementdb.CostEvents
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				ce.Query(),
				&listquery.Request[carriersettlement.CostEvent]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[*carriersettlement.CostEvent],
					) *keel.Query[*carriersettlement.CostEvent] {
						return q.Load(ce.Carrier())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list carrier cost events", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *costEventRepository) GetByID(
	ctx context.Context,
	req carriersettlementports.GetCarrierCostEventByIDRequest,
) (*carriersettlement.CostEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*carriersettlement.CostEvent, error) {
		ce := carriersettlementdb.CostEvents
		entity, err := ce.Where(ce.ID.Eq(req.ID)).Load(ce.Carrier()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, costEventErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *costEventRepository) GetByIdempotencyKey(
	ctx context.Context,
	_ pagination.TenantInfo,
	key string,
) (*carriersettlement.CostEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*carriersettlement.CostEvent, error) {
		ce := carriersettlementdb.CostEvents
		entity, err := ce.Where(ce.IdempotencyKey.Eq(key)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, costEventErrors.Map(err, key)
		}

		return entity, nil
	})
}

func (r *costEventRepository) ListPendingByCarrier(
	ctx context.Context,
	req *carriersettlementports.ListPendingCostEventsRequest,
) ([]*carriersettlement.CostEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*carriersettlement.CostEvent, error) {
		ce := carriersettlementdb.CostEvents
		status := keel.And(
			ce.Status.Eq(carriersettlement.CostEventStatusPending),
			when(req.HoldUntilInvoiceMatched, invoiceMatched(ce)),
		)
		if !req.ReleasedFrom.IsNil() {
			status = keel.Or(status, keel.And(
				ce.Status.In(
					carriersettlement.CostEventStatusAttached,
					carriersettlement.CostEventStatusSettled,
				),
				ce.SettlementID.Eq(req.ReleasedFrom),
			))
		}

		items, err := ce.Where(
			ce.CarrierID.Eq(req.CarrierID),
			status,
			when(req.PeriodEnd > 0, ce.EventDate.Lt(req.PeriodEnd)),
		).Order(ce.EventDate.Asc()).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list pending carrier cost events: %w", err)
		}

		return items, nil
	})
}

func resolvedInvoiceMatch(
	ce *carriersettlementdb.CostEventsTable,
) *keel.Query[*carriersettlement.InvoiceMatch] {
	im := carriersettlementdb.InvoiceMatches
	return im.Where(
		im.CarrierAssignmentID.EqCol(ce.CarrierAssignmentID),
		im.Status.Eq(carriersettlement.InvoiceMatchStatusResolved),
	)
}

func invoiceMatched(ce *carriersettlementdb.CostEventsTable) keel.Pred {
	return keel.Or(ce.CarrierAssignmentID.IsNull(), keel.Exists(resolvedInvoiceMatch(ce)))
}

func (r *costEventRepository) CountAwaitingInvoiceMatch(
	ctx context.Context,
	_ pagination.TenantInfo,
	settlementID pulid.ID,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		ce := carriersettlementdb.CostEvents
		count, err := keel.IntoOne[int64](
			ctx,
			r.db.Keel(ctx),
			ce.Pick(ce.CarrierAssignmentID.CountDistinct()).Where(
				ce.SettlementID.Eq(settlementID),
				ce.Status.Ne(carriersettlement.CostEventStatusVoided),
				ce.CarrierAssignmentID.IsNotNull(),
				keel.NotExists(resolvedInvoiceMatch(ce)),
			),
		)
		if err != nil {
			return 0, fmt.Errorf("count carrier cost events awaiting an invoice match: %w", err)
		}

		return int(count), nil
	})
}

func (r *costEventRepository) ListCarrierIDsWithPendingEvents(
	ctx context.Context,
	req carriersettlementports.ListCarriersWithPendingEventsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		ce := carriersettlementdb.CostEvents
		ids, err := keel.Into[pulid.ID](
			ctx,
			r.db.Keel(ctx),
			ce.Pick(ce.CarrierID).
				Where(
					ce.Status.Eq(carriersettlement.CostEventStatusPending),
					ce.EventDate.Lt(req.PeriodEnd),
					when(req.HoldUntilInvoiceMatched, invoiceMatched(ce)),
				).
				Group(ce.CarrierID),
		)
		if err != nil {
			return nil, fmt.Errorf("list carriers with pending cost events: %w", err)
		}

		return ids, nil
	})
}

func (r *costEventRepository) ListByShipment(
	ctx context.Context,
	_ pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*carriersettlement.CostEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*carriersettlement.CostEvent, error) {
		ce := carriersettlementdb.CostEvents
		items, err := ce.Where(ce.ShipmentID.Eq(shipmentID)).
			Order(ce.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list carrier cost events by shipment: %w", err)
		}

		return items, nil
	})
}

func (r *costEventRepository) ListByMove(
	ctx context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
) ([]*carriersettlement.CostEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*carriersettlement.CostEvent, error) {
		ce := carriersettlementdb.CostEvents
		items, err := ce.Where(ce.MoveID.Eq(moveID)).
			Order(ce.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list carrier cost events by move: %w", err)
		}

		return items, nil
	})
}

func (r *costEventRepository) GetPendingSummary(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*carriersettlementports.CarrierCostEventSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*carriersettlementports.CarrierCostEventSummary, error) {
			ce := carriersettlementdb.CostEvents
			summary, err := keel.IntoOne[carriersettlementports.CarrierCostEventSummary](
				ctx,
				r.db.Keel(ctx),
				ce.Pick(
					keel.Count(),
					coalesceZero(ce.AmountMinor.Sum()),
					ce.CarrierID.CountDistinct(),
				).Where(ce.Status.Eq(carriersettlement.CostEventStatusPending)),
			)
			if err != nil {
				return nil, fmt.Errorf("get pending carrier cost event summary: %w", err)
			}

			return &summary, nil
		},
	)
}

func (r *costEventRepository) Create(
	ctx context.Context,
	entity *carriersettlement.CostEvent,
) (*carriersettlement.CostEvent, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*carriersettlement.CostEvent, error) {
		if err := carriersettlementdb.CostEvents.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, fmt.Errorf(
				"create carrier cost event: %w",
				costEventErrors.Map(err, entity.ID.String()),
			)
		}

		return entity, nil
	})
}

func (r *costEventRepository) Update(
	ctx context.Context,
	entity *carriersettlement.CostEvent,
) (*carriersettlement.CostEvent, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*carriersettlement.CostEvent, error) {
		ce := carriersettlementdb.CostEvents
		if err := ce.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			ce.Status,
			ce.AmountMinor,
			ce.Description,
			ce.ProNumber,
			ce.AssignmentVersion,
			ce.SettlementID,
			ce.VoidedAt,
			ce.VoidReason,
		); err != nil {
			return nil, costEventErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *costEventRepository) MarkAttached(
	ctx context.Context,
	_ pagination.TenantInfo,
	eventIDs []pulid.ID,
	settlementID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(eventIDs) == 0 {
			return nil
		}

		ce := carriersettlementdb.CostEvents
		rows, err := ce.Where(
			ce.ID.In(eventIDs...),
			ce.Status.Eq(carriersettlement.CostEventStatusPending),
		).Update(
			ctx,
			r.db.Keel(ctx),
			ce.Status.To(carriersettlement.CostEventStatusAttached),
			ce.SettlementID.To(settlementID),
		)
		if err != nil {
			return fmt.Errorf("mark carrier cost events attached: %w", err)
		}
		if rows == 0 {
			return errortypes.NewValidationError(
				"costEventIds",
				errortypes.ErrInvalidOperation,
				"The selected carrier cost events are no longer pending; they were attached or voided by a concurrent request",
			)
		}

		return nil
	})
}

func (r *costEventRepository) MarkSettled(
	ctx context.Context,
	_ pagination.TenantInfo,
	settlementID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ce := carriersettlementdb.CostEvents
		_, err := ce.Where(
			ce.SettlementID.Eq(settlementID),
			ce.Status.Eq(carriersettlement.CostEventStatusAttached),
		).Update(ctx, r.db.Keel(ctx), ce.Status.To(carriersettlement.CostEventStatusSettled))
		if err != nil {
			return fmt.Errorf("mark carrier cost events settled: %w", err)
		}

		return nil
	})
}

func (r *costEventRepository) ReleaseSettled(
	ctx context.Context,
	_ pagination.TenantInfo,
	settlementID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ce := carriersettlementdb.CostEvents
		_, err := ce.Where(
			ce.SettlementID.Eq(settlementID),
			ce.Status.In(
				carriersettlement.CostEventStatusAttached,
				carriersettlement.CostEventStatusSettled,
			),
		).Update(
			ctx,
			r.db.Keel(ctx),
			ce.Status.To(carriersettlement.CostEventStatusPending),
			ce.SettlementID.ToNull(),
		)
		if err != nil {
			return fmt.Errorf("release settled carrier cost events: %w", err)
		}

		return nil
	})
}
