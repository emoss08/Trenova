package agentdecisionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.AgentDecisionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentdecision-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentDecisionByIDRequest,
) (*agent.AgentDecision, error) {
	log := r.l.With(zap.String("operation", "GetByID"), zap.String("id", req.ID.String()))

	entity := new(agent.AgentDecision)
	cols := buncolgen.AgentDecisionColumns
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentDecisionScopeTenant(sq, *req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get agent decision", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "AgentDecision")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *agent.AgentDecision,
) (*agent.AgentDecision, error) {
	log := r.l.With(zap.String("operation", "Create"))

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create agent decision", zap.Error(err))
		return nil, err
	}

	return entity, nil
}
