package dispatchlearningrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchlearning"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice/dispatchplanpricedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent/shipmenteventdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchlearningports"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultDecisionLimit = 200
	maxDecisionLimit     = 1_000
	planFreshnessSeconds = int64(24 * 3600)
)

func (r *repository) ListDecisions(
	ctx context.Context,
	req *dispatchlearningports.ListDecisionsRequest,
) (*dispatchlearningports.DecisionPage, error) {
	limit := intutils.Clamp(
		intutils.WithDefault(max(req.Limit, 0), defaultDecisionLimit),
		1,
		maxDecisionLimit,
	)

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*dispatchlearningports.DecisionPage, error) {
			events, err := r.assignmentEvents(ctx, req.After, limit)
			if err != nil {
				return nil, err
			}
			page := &dispatchlearningports.DecisionPage{
				Decisions: []*dispatchlearning.Decision{},
				Scanned:   len(events),
				Last:      req.After,
			}
			if len(events) == 0 {
				return page, nil
			}
			last := events[len(events)-1]
			page.Last = dispatchlearningports.DecisionCursor{At: last.OccurredAt, ID: last.ID}

			moveIDs := make([]pulid.ID, 0, len(events))
			assignmentIDs := make([]pulid.ID, 0, len(events))
			for _, event := range events {
				moveIDs = append(moveIDs, event.MoveID)
				assignmentIDs = append(assignmentIDs, event.AssignmentID)
			}

			prices, err := r.pricesByMove(ctx, moveIDs)
			if err != nil {
				return nil, err
			}
			workers, err := r.workersByAssignment(ctx, assignmentIDs)
			if err != nil {
				return nil, err
			}

			for _, event := range events {
				decision, ok := decisionOf(event, prices[event.MoveID], workers[event.AssignmentID])
				if ok {
					page.Decisions = append(page.Decisions, decision)
				}
			}

			return page, nil
		},
	)
}

func (r *repository) assignmentEvents(
	ctx context.Context,
	after dispatchlearningports.DecisionCursor,
	limit int,
) ([]*shipmentevent.Event, error) {
	events, err := DecisionEventsQuery(after, limit).List(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, fmt.Errorf("list driver assignment events: %w", err)
	}

	return events, nil
}

func DecisionEventsQuery(
	after dispatchlearningports.DecisionCursor,
	limit int,
) keel.Query[*shipmentevent.Event] {
	e := shipmenteventdb.Events
	q := e.Pick(e.ID, e.OccurredAt, e.MoveID, e.AssignmentID).
		Where(
			e.Type.Eq(shipmentevent.TypeDriverAssigned),
			e.ActorType.Eq(shipmentevent.ActorUser),
		).
		Order(e.OccurredAt.Asc(), e.ID.Asc()).
		Limit(limit)
	if after.At > 0 || !after.ID.IsNil() {
		q = q.Where(keel.Tuple(e.OccurredAt, e.ID).Gt(keel.Tuple(after.At, after.ID.String())))
	}

	return q
}

func (r *repository) pricesByMove(
	ctx context.Context,
	moveIDs []pulid.ID,
) (map[pulid.ID]*dispatchplanprice.DispatchPlanPrice, error) {
	p := dispatchplanpricedb.DispatchPlanPrices
	rows, err := p.Pick(
		p.ID,
		p.MoveID,
		p.WorkerID,
		p.CostCents,
		p.CostLines,
		p.Alternatives,
		p.GeneratedAt,
	).
		Where(p.MoveID.In(moveIDs...), p.WorkerID.IsNotNull()).
		Limit(len(moveIDs)).
		List(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, fmt.Errorf("list plan prices of assigned moves: %w", err)
	}

	out := make(map[pulid.ID]*dispatchplanprice.DispatchPlanPrice, len(rows))
	for _, row := range rows {
		out[row.MoveID] = row
	}

	return out, nil
}

func (r *repository) workersByAssignment(
	ctx context.Context,
	assignmentIDs []pulid.ID,
) (map[pulid.ID]pulid.ID, error) {
	a := shipmentdb.Assignments
	rows, err := a.Pick(a.ID, a.PrimaryWorkerID).
		Where(a.ID.In(assignmentIDs...)).
		Limit(len(assignmentIDs)).
		List(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, fmt.Errorf("list assigned workers: %w", err)
	}

	out := make(map[pulid.ID]pulid.ID, len(rows))
	for _, row := range rows {
		if worker := assignedWorker(row); !worker.IsNil() {
			out[row.ID] = worker
		}
	}

	return out, nil
}

func assignedWorker(row *shipment.Assignment) pulid.ID {
	if row.PrimaryWorkerID == nil {
		return pulid.Nil
	}

	return *row.PrimaryWorkerID
}

func decisionOf(
	event *shipmentevent.Event,
	price *dispatchplanprice.DispatchPlanPrice,
	chosen pulid.ID,
) (*dispatchlearning.Decision, bool) {
	if price == nil || chosen.IsNil() || price.WorkerID.IsNil() ||
		price.GeneratedAt > event.OccurredAt ||
		price.GeneratedAt < event.OccurredAt-planFreshnessSeconds {
		return nil, false
	}

	decision := &dispatchlearning.Decision{
		EventID:        event.ID,
		DecidedAt:      event.OccurredAt,
		MoveID:         event.MoveID,
		ChosenWorkerID: chosen,
		Planned: dispatchlearning.Pairing{
			WorkerID:  price.WorkerID,
			CostCents: price.CostCents,
			Lines:     price.CostLines,
		},
		Alternatives: make([]dispatchlearning.Pairing, 0, len(price.Alternatives)),
	}
	for _, alt := range price.Alternatives {
		decision.Alternatives = append(decision.Alternatives, dispatchlearning.Pairing{
			WorkerID:   alt.WorkerID,
			CostCents:  alt.CostCents,
			ExtraCents: alt.ExtraCents,
			Lines:      alt.Lines,
		})
	}

	return decision, true
}
