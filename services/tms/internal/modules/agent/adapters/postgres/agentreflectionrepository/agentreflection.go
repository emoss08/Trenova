package agentreflectionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const reflectionEntity = "AgentReflection"

var reflectionErrors = dberror.Mapping{Entity: reflectionEntity}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentports.AgentReflectionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentreflection-repository"),
	}
}

func (r *repository) Claim(
	ctx context.Context,
	entity *agent.Reflection,
) (*agent.Reflection, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Reflection, error) {
		err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
			return agentdb.Reflections.Create(ctx, r.db.Keel(ctx), entity)
		})
		if err == nil {
			return entity, nil
		}
		if !dberror.IsUniqueConstraintViolation(err) {
			r.l.Error("failed to claim an agent reflection", zap.Error(err))

			return nil, fmt.Errorf("claim agent reflection: %w", err)
		}

		return r.claimed(ctx, entity)
	})
}

func idIs(col keelkinds.PulidIDCol, id *pulid.ID) keel.Pred {
	if id == nil {
		return col.IsNull()
	}

	return col.Eq(*id)
}

func (r *repository) claimed(
	ctx context.Context,
	entity *agent.Reflection,
) (*agent.Reflection, error) {
	rf := agentdb.Reflections
	subject := keel.And(
		idIs(rf.ThreadID, entity.ThreadID),
		rf.ThroughSequence.Eq(int64(entity.ThroughSequence)),
	)
	if entity.SubjectType == agent.ReflectionSubjectRun {
		subject = idIs(rf.RunID, entity.RunID)
	}

	held, err := rf.Where(rf.SubjectType.Eq(entity.SubjectType), subject).First(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, reflectionErrors.Map(err, entity.ID.String())
	}

	return held, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req agentports.GetAgentReflectionRequest,
) (*agent.Reflection, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.Reflection, error) {
		entity, err := agentdb.Reflections.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, reflectionErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *agent.Reflection,
) (*agent.Reflection, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Reflection, error) {
		rf := agentdb.Reflections
		err := rf.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			rf.Status,
			rf.SkipReason,
			rf.Signals,
			rf.Changes,
			rf.Notes,
			rf.Tainted,
			rf.Model,
			rf.ProviderID,
			rf.InputTokens,
			rf.OutputTokens,
			rf.ErrorMessage,
			rf.FinishedAt,
		)
		if err != nil {
			if dberror.IsVersionConflict(err) {
				return nil, reflectionErrors.Map(err, entity.ID.String())
			}

			return nil, fmt.Errorf("update agent reflection: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) ThreadReflectedThrough(
	ctx context.Context,
	req agentports.ThreadReflectedThroughRequest,
) (int, bool, error) {
	return dbtx.Read2(ctx, r.db, func(ctx context.Context) (int, bool, error) {
		rf := agentdb.Reflections
		latest, err := rf.Pick(rf.ThroughSequence).
			Where(
				rf.SubjectType.Eq(agent.ReflectionSubjectThread),
				rf.ThreadID.Eq(req.ThreadID),
				rf.Status.Ne(agent.ReflectionStatusFailed),
			).
			Order(rf.ThroughSequence.Desc()).
			First(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return 0, false, nil
		}
		if err != nil {
			return 0, false, fmt.Errorf("read how far a conversation was looked back over: %w", err)
		}

		return latest.ThroughSequence, true, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *agentports.ListAgentReflectionConnectionRequest,
) (*pagination.CursorListResult[*agent.Reflection], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agent.Reflection], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				agentdb.Reflections.Query(),
				&listquery.Request[agent.Reflection]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("list agent reflections: %w", err)
			}

			return result, nil
		},
	)
}
