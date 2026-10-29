package agentruneventrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/agentrunrepository"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunevent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunevent/agentruneventdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentruntimeports.AgentRunEventRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentrunevent-repository"),
	}
}

// Append writes a batch of events.
//
// There is no conflict clause. A duplicated sequence is a writer that lost
// count, and the unique index refusing it is the correct outcome: silently
// absorbing it would leave a trajectory that reads plausibly and is wrong.
func (r *repository) Append(
	ctx context.Context,
	req agentruntimeports.AppendAgentRunEventsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(req.Events) == 0 {
			return nil
		}

		if err := agentruneventdb.AgentRunEvents.CreateMany(
			ctx,
			r.db.Keel(ctx),
			req.Events,
		); err != nil {
			r.l.Error("failed to append agent run events",
				zap.Int("events", len(req.Events)),
				zap.Error(err),
			)

			return fmt.Errorf("append agent run events: %w", err)
		}

		return nil
	})
}

// NextSequence is where a writer resumes counting.
//
// A run that is retried starts a fresh writer with no memory of the numbers the
// last attempt used, so the count is read back from what is already stored
// rather than started again at one, which would collide.
func (r *repository) NextSequence(
	ctx context.Context,
	_ pagination.TenantInfo,
	ownerKind string,
	ownerID pulid.ID,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		e := agentruneventdb.AgentRunEvents
		highest, err := keel.IntoOne[int](ctx, r.db.Keel(ctx), e.
			Pick(keel.Int64Of(e.Sequence.Max()).Coalesce(0)).
			Where(e.OwnerKind.Eq(ownerKind), e.OwnerID.Eq(ownerID)))
		if err != nil {
			r.l.Error("failed to read the next event sequence",
				zap.String("owner", ownerID.String()),
				zap.Error(err),
			)

			return 0, fmt.Errorf("read next agent run event sequence: %w", err)
		}

		return highest + 1, nil
	})
}

func listQuery(
	req *agentruntimeports.ListAgentRunEventsRequest,
) *keel.Query[*agentrunevent.AgentRunEvent] {
	e := agentruneventdb.AgentRunEvents

	return e.Where(
		e.OwnerKind.Eq(req.OwnerKind),
		e.OwnerID.Eq(req.OwnerID),
		ofReadableRuns(req.Reader),
		keel.When(req.After > 0, e.Sequence.Gt(int64(req.After))),
	).Order(e.Sequence.Asc())
}

func (r *repository) List(
	ctx context.Context,
	req *agentruntimeports.ListAgentRunEventsRequest,
) ([]*agentrunevent.AgentRunEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentrunevent.AgentRunEvent, error) {
		q := listQuery(req)

		var (
			events []*agentrunevent.AgentRunEvent
			err    error
		)
		if req.Limit > 0 {
			events, err = q.Limit(req.Limit).List(ctx, r.db.Keel(ctx))
		} else {
			events, err = q.All(ctx, r.db.Keel(ctx))
		}
		if err != nil {
			r.l.Error("failed to list agent run events",
				zap.String("owner", req.OwnerID.String()),
				zap.Error(err),
			)

			return nil, fmt.Errorf("list agent run events: %w", err)
		}

		return events, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *agentruntimeports.ListAgentRunEventConnectionRequest,
) (*pagination.CursorListResult[*agentrunevent.AgentRunEvent], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agentrunevent.AgentRunEvent], error) {
			e := agentruneventdb.AgentRunEvents
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				e.Where(
					keel.When(req.OwnerKind != "", e.OwnerKind.Eq(req.OwnerKind)),
					ofReadableRuns(req.Reader),
				),
				&listquery.Request[agentrunevent.AgentRunEvent]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to list agent run events", zap.Error(err))

				return nil, fmt.Errorf("list agent run events: %w", err)
			}

			return result, nil
		},
	)
}

// ofReadableRuns keeps, for a person, the events of the runs they may read;
// an event an assistant turn owns is the turn's, and stays.
func ofReadableRuns(reader agentruntimeports.RunReader) keel.Pred {
	if !reader.ForPerson() {
		return nil
	}

	e := agentruneventdb.AgentRunEvents
	run := agentdb.AgentRuns.As("event_run")

	return keel.Or(
		e.OwnerKind.Ne(string(agentruntimeports.RunStepOwnerAgentRun)),
		keel.Exists(run.Where(
			run.ID.EqCol(e.OwnerID),
			agentrunrepository.ReadableBy(run, reader),
		)),
	)
}

// Prune drops a batch of old events of the tenant on the context.
func (r *repository) Prune(
	ctx context.Context,
	req agentruntimeports.PruneAgentRunEventsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		e := agentruneventdb.AgentRunEvents
		doomed := agentruneventdb.AgentRunEvents.As("doomed")
		n, err := e.Where(e.ID.InQuery(doomed.
			Pick(doomed.ID).
			Where(doomed.CreatedAt.Lt(req.Before)).
			Limit(req.Limit))).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to prune agent run events", zap.Error(err))

			return 0, fmt.Errorf("prune agent run events: %w", err)
		}

		return int(n), nil
	})
}
