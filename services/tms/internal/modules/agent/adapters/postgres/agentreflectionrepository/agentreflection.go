package agentreflectionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection/agentreflectiondb"
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
	entity *agentreflection.Reflection,
) (*agentreflection.Reflection, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentreflection.Reflection, error) {
		err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
			return agentreflectiondb.Reflections.Create(ctx, r.db.Keel(ctx), entity)
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
	entity *agentreflection.Reflection,
) (*agentreflection.Reflection, error) {
	rf := agentreflectiondb.Reflections
	subject := keel.And(
		idIs(rf.ThreadID, entity.ThreadID),
		rf.ThroughSequence.Eq(int64(entity.ThroughSequence)),
	)
	if entity.SubjectType == agentreflection.ReflectionSubjectRun {
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
) (*agentreflection.Reflection, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentreflection.Reflection, error) {
		entity, err := agentreflectiondb.Reflections.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, reflectionErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *agentreflection.Reflection,
) (*agentreflection.Reflection, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentreflection.Reflection, error) {
		rf := agentreflectiondb.Reflections
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
		rf := agentreflectiondb.Reflections
		latest, err := rf.Pick(rf.ThroughSequence).
			Where(
				rf.SubjectType.Eq(agentreflection.ReflectionSubjectThread),
				rf.ThreadID.Eq(req.ThreadID),
				rf.Status.Ne(agentreflection.ReflectionStatusFailed),
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
) (*pagination.CursorListResult[*agentreflection.Reflection], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agentreflection.Reflection], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				agentreflectiondb.Reflections.Query(),
				&listquery.Request[agentreflection.Reflection]{
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
