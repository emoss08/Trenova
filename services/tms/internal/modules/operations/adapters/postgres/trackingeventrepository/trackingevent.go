package trackingeventrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent/trackingeventdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var trackingEventErrors = dberror.Mapping{Entity: "Tracking event"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) trackingeventports.TrackingEventRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tracking-event-repository"),
	}
}

func (r *repository) Insert(
	ctx context.Context,
	entity *trackingevent.TrackingEvent,
) (*trackingevent.TrackingEvent, bool, error) {
	return dbtx.Write2(
		ctx,
		r.db,
		func(ctx context.Context) (*trackingevent.TrackingEvent, bool, error) {
			te := trackingeventdb.TrackingEvents
			db := r.db.Keel(ctx)
			inserted, err := te.CreateOrSkip(
				ctx,
				db,
				[]*trackingevent.TrackingEvent{entity},
				te.OrganizationID,
				te.BusinessUnitID,
				te.Source,
				te.SourceKey,
			)
			if err != nil {
				return nil, false, fmt.Errorf(
					"insert tracking event: %w",
					trackingEventErrors.Map(err, entity.ID.String()),
				)
			}
			if inserted > 0 {
				return entity, true, nil
			}

			existing, err := te.Where(
				te.Source.Eq(entity.Source),
				te.SourceKey.Eq(entity.SourceKey),
			).One(ctx, db)
			if err != nil {
				return nil, false, trackingEventErrors.Map(err, entity.SourceKey)
			}

			return existing, false, nil
		},
	)
}

func (r *repository) ListByMove(
	ctx context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
) ([]*trackingevent.TrackingEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*trackingevent.TrackingEvent, error) {
		te := trackingeventdb.TrackingEvents
		entities, err := te.Where(te.ShipmentMoveID.Eq(moveID)).
			Order(te.EventAt.Asc(), te.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list tracking events by move: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) ListByShipment(
	ctx context.Context,
	req *trackingeventports.ListTrackingEventsByShipmentRequest,
) ([]*trackingevent.TrackingEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*trackingevent.TrackingEvent, error) {
		te := trackingeventdb.TrackingEvents
		entities, err := te.Where(te.ShipmentID.Eq(req.ShipmentID)).
			Order(te.EventAt.Desc(), te.ID.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list tracking events by shipment: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) RecordVerdicts(
	ctx context.Context,
	_ pagination.TenantInfo,
	verdicts []trackingeventports.TrackingEventVerdict,
	at int64,
) error {
	if len(verdicts) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ids := make([]string, len(verdicts))
		outcomes := make([]string, len(verdicts))
		reasons := make([]string, len(verdicts))
		for i, verdict := range verdicts {
			ids[i] = verdict.ID.String()
			outcomes[i] = string(verdict.Outcome)
			reasons[i] = verdict.Reason
		}

		te := trackingeventdb.TrackingEvents
		v := keel.Unnest("verdicts", []string{"id", "outcome", "reason"}, ids, outcomes, reasons)
		_, err := te.With(v).
			Join(v, v.Col("id").EqCol(te.ID)).
			Where(keel.True()).
			Update(
				ctx,
				r.db.Keel(ctx),
				te.Outcome.ToExpr(v.Col("outcome")),
				te.OutcomeReason.ToExpr(keel.DynOf(v.Col("reason")).NullIf("")),
				te.UpdatedAt.To(at),
			)
		if err != nil {
			return fmt.Errorf("record tracking event verdict: %w", err)
		}

		return nil
	})
}
