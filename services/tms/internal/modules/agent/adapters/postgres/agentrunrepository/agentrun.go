package agentrunrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const runEntity = "AgentRun"

var runErrors = dberror.Mapping{Entity: runEntity}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentports.AgentRunRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentrun-repository"),
	}
}

// ReadableBy keeps the runs a reader may see: for a person, every run but
// those of conversations someone else owns. run is the runs table as the
// statement references it.
func ReadableBy(run *agentdb.AgentRunsTable, reader agentports.RunReader) keel.Pred {
	if !reader.ForPerson() {
		return nil
	}

	notThread := run.SubjectType.Ne(agent.SubjectAssistantThread)
	if reader.UserID().IsNil() {
		return notThread
	}

	thread := conversationdb.Threads.As("readable_thread")

	return keel.Or(notThread, keel.Exists(thread.Where(
		thread.ID.EqCol(run.SubjectID),
		thread.UserID.Eq(reader.UserID()),
	)))
}

func readableRuns(reader agentports.RunReader) *keel.Query[agent.AgentRun] {
	run := agentdb.AgentRuns

	return run.Omit(run.Transcript).Where(ReadableBy(run, reader))
}

func transcripts(ids []pulid.ID, reader agentports.RunReader) *keel.Query[agent.AgentRun] {
	run := agentdb.AgentRuns

	return run.Pick(run.ID, run.BusinessUnitID, run.OrganizationID, run.Transcript).
		Where(run.ID.In(ids...), ReadableBy(run, reader)).
		Limit(len(ids))
}

func (r *repository) List(
	ctx context.Context,
	req *agentports.ListAgentRunRequest,
) (*pagination.ListResult[*agent.AgentRun], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*agent.AgentRun], error) {
			run := agentdb.AgentRuns
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				run.Where(ReadableBy(run, req.Reader)),
				&listquery.Request[agent.AgentRun]{
					Filter: req.Filter,
					Shape: func(q *keel.Query[agent.AgentRun]) *keel.Query[agent.AgentRun] {
						return q.Omit(run.Transcript)
					},
					Then: []keel.Order{run.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to scan and count agent runs", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *agentports.ListAgentRunConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentRun], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agent.AgentRun], error) {
			run := agentdb.AgentRuns
			request := &listquery.Request[agent.AgentRun]{
				Filter:  req.Filter,
				Cursor:  req.Cursor,
				Columns: req.Columns,
			}
			if len(req.Columns) == 0 {
				request.Shape = func(q *keel.Query[agent.AgentRun]) *keel.Query[agent.AgentRun] {
					return q.Omit(run.Transcript)
				}
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				run.Where(ReadableBy(run, req.Reader)),
				request,
			)
			if err != nil {
				r.l.Error("failed to scan agent runs", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListByIDs(
	ctx context.Context,
	req agentports.ListAgentRunsByIDsRequest,
) ([]*agent.AgentRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentRun, error) {
		if len(req.IDs) == 0 {
			return []*agent.AgentRun{}, nil
		}

		run := agentdb.AgentRuns
		runs, err := readableRuns(req.Reader).
			Where(run.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list agent runs by ids", zap.Error(err))

			return nil, fmt.Errorf("list agent runs by ids: %w", err)
		}

		return runs, nil
	})
}

func (r *repository) ListTranscriptsByIDs(
	ctx context.Context,
	req agentports.ListAgentRunsByIDsRequest,
) ([]*agent.AgentRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentRun, error) {
		if len(req.IDs) == 0 {
			return []*agent.AgentRun{}, nil
		}

		runs, err := transcripts(req.IDs, req.Reader).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list agent run transcripts by ids", zap.Error(err))

			return nil, fmt.Errorf("list agent run transcripts by ids: %w", err)
		}

		return runs, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req agentports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.AgentRun, error) {
		run := agentdb.AgentRuns
		entity, err := readableRuns(req.Reader).Where(run.ID.Eq(req.ID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get agent run",
				zap.String("id", req.ID.String()), zap.Error(err))
			return nil, runErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *agent.AgentRun,
) (*agent.AgentRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentRun, error) {
		if err := agentdb.AgentRuns.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create agent run", zap.Error(err))
			return nil, runErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

// Update saves the run under its version. A run read without its transcript
// keeps the stored one.
func (r *repository) Update(
	ctx context.Context,
	entity *agent.AgentRun,
) (*agent.AgentRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentRun, error) {
		run := agentdb.AgentRuns
		var err error
		if entity.Transcript == nil {
			err = run.SaveExcept(ctx, r.db.Keel(ctx), entity, run.Transcript)
		} else {
			err = run.Save(ctx, r.db.Keel(ctx), entity)
		}
		if err != nil {
			r.l.Error("failed to update agent run",
				zap.String("id", entity.ID.String()), zap.Error(err))
			return nil, runErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func closedRunStatuses() []agent.RunStatus {
	return []agent.RunStatus{
		agent.RunStatusCompleted,
		agent.RunStatusShadowCompleted,
		agent.RunStatusFailed,
	}
}

func (r *repository) CountOpen(
	ctx context.Context,
	req agentports.CountOpenAgentRunsRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		run := agentdb.AgentRuns
		count, err := run.Where(
			run.AgentDefinitionID.Eq(req.DefinitionID),
			run.Status.NotIn(closedRunStatuses()...),
			keel.When(req.SubjectID.IsNotNil(), run.SubjectID.Eq(req.SubjectID)),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count open agent runs", zap.Error(err))
			return 0, err
		}

		return int(count), nil
	})
}

func (r *repository) CountSince(
	ctx context.Context,
	req agentports.CountAgentRunsSinceRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		run := agentdb.AgentRuns
		count, err := run.Where(
			run.AgentDefinitionID.Eq(req.DefinitionID),
			run.CreatedAt.Gte(req.Since),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count agent runs since", zap.Error(err))

			return 0, fmt.Errorf("count agent runs since: %w", err)
		}

		return int(count), nil
	})
}
