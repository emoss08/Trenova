package driversettlementrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var payEventErrors = dberror.Mapping{
	Entity: "PayEvent",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_driver_pay_events_idempotency",
			Field:   "idempotencyKey",
			Message: "Pay event with this idempotency key already exists",
		},
	},
}

type payEventRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPayEvent(p Params) driversettlementports.PayEventRepository {
	return &payEventRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.pay-event-repository"),
	}
}

func (r *payEventRepository) List(
	ctx context.Context,
	req *driversettlementports.ListPayEventsRequest,
) (*pagination.ListResult[*driversettlement.PayEvent], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*driversettlement.PayEvent], error) {
			pe := driversettlementdb.PayEvents
			q := pe.Where(
				when(req.Filter.Query != "", pe.ProNumber.ContainsFold(req.Filter.Query)),
				when(!req.WorkerID.IsNil(), pe.WorkerID.Eq(req.WorkerID)),
				when(!req.ShipmentID.IsNil(), pe.ShipmentID.Eq(req.ShipmentID)),
				when(req.Status != "", pe.Status.Eq(req.Status)),
			).Load(pe.Worker()).Order(pe.EventDate.Desc())

			result, err := listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
			if err != nil {
				return nil, fmt.Errorf("list pay events: %w", err)
			}

			return result, nil
		},
	)
}

func (r *payEventRepository) ListConnection(
	ctx context.Context,
	req *driversettlementports.ListPayEventConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.PayEvent], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*driversettlement.PayEvent], error) {
			pe := driversettlementdb.PayEvents
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				pe.Query(),
				&listquery.Request[driversettlement.PayEvent]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[driversettlement.PayEvent],
					) *keel.Query[driversettlement.PayEvent] {
						return q.Load(pe.Worker())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list pay events", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *payEventRepository) GetByID(
	ctx context.Context,
	req driversettlementports.GetPayEventByIDRequest,
) (*driversettlement.PayEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driversettlement.PayEvent, error) {
		pe := driversettlementdb.PayEvents
		entity, err := pe.Where(pe.ID.Eq(req.ID)).Load(pe.Worker()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, payEventErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *payEventRepository) GetByIdempotencyKey(
	ctx context.Context,
	_ pagination.TenantInfo,
	key string,
) (*driversettlement.PayEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driversettlement.PayEvent, error) {
		pe := driversettlementdb.PayEvents
		entity, err := pe.Where(pe.IdempotencyKey.Eq(key)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, payEventErrors.Map(err, key)
		}

		return entity, nil
	})
}

func (r *payEventRepository) ListAccruedForWorker(
	ctx context.Context,
	req *driversettlementports.ListAccruedPayEventsRequest,
) ([]*driversettlement.PayEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driversettlement.PayEvent, error) {
		pe := driversettlementdb.PayEvents
		status := pe.Status.Eq(driversettlement.PayEventStatusAccrued)
		if !req.ReleasedFrom.IsNil() {
			status = keel.Or(status, keel.And(
				pe.Status.Eq(driversettlement.PayEventStatusSettled),
				pe.SettlementID.Eq(req.ReleasedFrom),
			))
		}

		var scope keel.Pred
		switch {
		case len(req.EventIDs) > 0:
			scope = pe.ID.In(req.EventIDs...)
		case req.PeriodEnd > 0:
			scope = pe.EventDate.Lt(req.PeriodEnd)
		}

		items, err := pe.Where(
			pe.WorkerID.Eq(req.WorkerID),
			pe.OnHold.IsFalse(),
			status,
			scope,
		).Order(pe.EventDate.Asc()).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list accrued pay events: %w", err)
		}

		return items, nil
	})
}

func (r *payEventRepository) ListWorkerIDsWithAccruedEvents(
	ctx context.Context,
	req driversettlementports.ListWorkersWithAccruedEventsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		pe := driversettlementdb.PayEvents
		ids, err := keel.Into[pulid.ID](
			ctx,
			r.db.Keel(ctx),
			pe.Pick(pe.WorkerID).
				Where(
					pe.Status.Eq(driversettlement.PayEventStatusAccrued),
					pe.OnHold.IsFalse(),
					pe.EventDate.Lt(req.PeriodEnd),
				).
				Group(pe.WorkerID),
		)
		if err != nil {
			return nil, fmt.Errorf("list workers with accrued pay events: %w", err)
		}

		return ids, nil
	})
}

func (r *payEventRepository) GetAccruedTotalsForWorker(
	ctx context.Context,
	req driversettlementports.GetAccruedTotalsForWorkerRequest,
) (*driversettlementports.AccruedPayTotals, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*driversettlementports.AccruedPayTotals, error) {
			pe := driversettlementdb.PayEvents
			totals, err := keel.IntoOne[driversettlementports.AccruedPayTotals](
				ctx,
				r.db.Keel(ctx),
				pe.Pick(keel.Count(), coalesceZero(pe.GrossAmountMinor.Sum())).
					Where(
						pe.WorkerID.Eq(req.WorkerID),
						pe.Status.Eq(driversettlement.PayEventStatusAccrued),
					),
			)
			if err != nil {
				return nil, fmt.Errorf("get accrued totals for worker: %w", err)
			}

			return &totals, nil
		},
	)
}

func (r *payEventRepository) ListByShipment(
	ctx context.Context,
	_ pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*driversettlement.PayEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driversettlement.PayEvent, error) {
		pe := driversettlementdb.PayEvents
		items, err := pe.Where(pe.ShipmentID.Eq(shipmentID)).
			Order(pe.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list pay events by shipment: %w", err)
		}

		return items, nil
	})
}

func (r *payEventRepository) ListByMove(
	ctx context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
) ([]*driversettlement.PayEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driversettlement.PayEvent, error) {
		pe := driversettlementdb.PayEvents
		items, err := pe.Where(pe.MoveID.Eq(moveID)).
			Order(pe.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list pay events by move: %w", err)
		}

		return items, nil
	})
}

func (r *payEventRepository) ListByMovesForWorker(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
	moveIDs []pulid.ID,
) ([]*driversettlement.PayEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driversettlement.PayEvent, error) {
		if len(moveIDs) == 0 {
			return []*driversettlement.PayEvent{}, nil
		}

		pe := driversettlementdb.PayEvents
		items, err := pe.Where(
			pe.WorkerID.Eq(workerID),
			pe.MoveID.In(moveIDs...),
			pe.Status.Ne(driversettlement.PayEventStatusVoided),
		).Order(pe.CreatedAt.Asc()).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list pay events by moves for worker: %w", err)
		}

		return items, nil
	})
}

func (r *payEventRepository) ListUnsettledWorkerSummaries(
	ctx context.Context,
	req *driversettlementports.ListUnsettledWorkerSummariesRequest,
) ([]*driversettlementports.UnsettledWorkerSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*driversettlementports.UnsettledWorkerSummary, error) {
			pe := driversettlementdb.PayEvents
			w := pe.Worker()
			s := driversettlementdb.Settlements
			held := pe.OnHold.IsTrue()
			released := pe.OnHold.IsFalse()
			gross := coalesceZero(pe.GrossAmountMinor.Sum().Filter(released))

			rows, err := keel.Into[driversettlementports.UnsettledWorkerSummary](
				ctx,
				r.db.Keel(ctx),
				pe.Pick(
					pe.WorkerID,
					keel.DynFunc("trim", keel.DynOf(w.FirstName).Concat(" ").Concat(w.LastName)),
					keel.Count().Filter(released),
					gross,
					keel.Count().Filter(held),
					coalesceZero(pe.GrossAmountMinor.Sum().Filter(held)),
					keel.Exists(s.Where(
						s.WorkerID.EqCol(pe.WorkerID),
						s.PeriodStart.Eq(req.PeriodStart),
						s.PeriodEnd.Eq(req.PeriodEnd),
						s.Status.Ne(driversettlement.StatusVoided),
					)),
				).
					Where(
						w.ID.IsNotNull(),
						pe.Status.Eq(driversettlement.PayEventStatusAccrued),
						pe.EventDate.Lte(req.PeriodEnd),
					).
					Group(pe.WorkerID, w.FirstName, w.LastName).
					Order(keel.Desc(gross)),
			)
			if err != nil {
				return nil, fmt.Errorf("list unsettled worker summaries: %w", err)
			}

			summaries := make([]*driversettlementports.UnsettledWorkerSummary, len(rows))
			for i := range rows {
				summaries[i] = &rows[i]
			}

			return summaries, nil
		},
	)
}

func (r *payEventRepository) GetUnsettledSummary(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*driversettlementports.UnsettledPayEventSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*driversettlementports.UnsettledPayEventSummary, error) {
			pe := driversettlementdb.PayEvents
			held := pe.OnHold.IsTrue()
			released := pe.OnHold.IsFalse()
			summary, err := keel.IntoOne[driversettlementports.UnsettledPayEventSummary](
				ctx,
				r.db.Keel(ctx),
				pe.Pick(
					keel.Count().Filter(released),
					coalesceZero(pe.GrossAmountMinor.Sum().Filter(released)),
					keel.Count().Filter(held),
					coalesceZero(pe.GrossAmountMinor.Sum().Filter(held)),
					pe.WorkerID.CountDistinct(),
				).Where(pe.Status.Eq(driversettlement.PayEventStatusAccrued)),
			)
			if err != nil {
				return nil, fmt.Errorf("get unsettled pay event summary: %w", err)
			}

			return &summary, nil
		},
	)
}

func (r *payEventRepository) Create(
	ctx context.Context,
	entity *driversettlement.PayEvent,
) (*driversettlement.PayEvent, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driversettlement.PayEvent, error) {
		if err := driversettlementdb.PayEvents.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, payEventErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *payEventRepository) Update(
	ctx context.Context,
	entity *driversettlement.PayEvent,
) (*driversettlement.PayEvent, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driversettlement.PayEvent, error) {
		pe := driversettlementdb.PayEvents
		if err := pe.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			pe.Status,
			pe.GrossAmountMinor,
			pe.TotalMiles,
			pe.Components,
			pe.SettlementID,
			pe.SettlementLineID,
			pe.VoidedAt,
			pe.VoidReason,
			pe.OnHold,
			pe.HoldReason,
		); err != nil {
			return nil, payEventErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *payEventRepository) MarkSettled(
	ctx context.Context,
	_ pagination.TenantInfo,
	eventIDs []pulid.ID,
	settlementID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(eventIDs) == 0 {
			return nil
		}

		pe := driversettlementdb.PayEvents
		_, err := pe.Where(
			pe.ID.In(eventIDs...),
			pe.Status.Eq(driversettlement.PayEventStatusAccrued),
		).Update(
			ctx,
			r.db.Keel(ctx),
			pe.Status.To(driversettlement.PayEventStatusSettled),
			pe.SettlementID.To(settlementID),
		)
		if err != nil {
			return fmt.Errorf("mark pay events settled: %w", err)
		}

		return nil
	})
}

func (r *payEventRepository) ReleaseSettled(
	ctx context.Context,
	_ pagination.TenantInfo,
	settlementID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		pe := driversettlementdb.PayEvents
		if err := r.release(ctx, pe.SettlementID.Eq(settlementID)); err != nil {
			return fmt.Errorf("release settled pay events: %w", err)
		}

		return nil
	})
}

func (r *payEventRepository) ReleaseEvents(
	ctx context.Context,
	_ pagination.TenantInfo,
	eventIDs []pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(eventIDs) == 0 {
			return nil
		}

		pe := driversettlementdb.PayEvents
		if err := r.release(ctx, pe.ID.In(eventIDs...)); err != nil {
			return fmt.Errorf("release pay events: %w", err)
		}

		return nil
	})
}

func (r *payEventRepository) release(ctx context.Context, target keel.Pred) error {
	pe := driversettlementdb.PayEvents
	_, err := pe.Where(
		target,
		pe.Status.Eq(driversettlement.PayEventStatusSettled),
	).Update(
		ctx,
		r.db.Keel(ctx),
		pe.Status.To(driversettlement.PayEventStatusAccrued),
		pe.SettlementID.ToNull(),
		pe.SettlementLineID.ToNull(),
	)

	return err
}
