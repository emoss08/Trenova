package agentproposalrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.AgentProposalRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentproposal-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListAgentProposalRequest,
) *bun.SelectQuery {
	cols := buncolgen.AgentProposalColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.AgentProposalTable.Alias,
		req.Filter,
		(*agent.AgentProposal)(nil),
	)

	return q.Apply(buncolgen.AgentProposalApplyTenant(req.Filter.TenantInfo)).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.CreatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAgentProposalRequest,
) (*pagination.ListResult[*agent.AgentProposal], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*agent.AgentProposal, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count agent proposals", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*agent.AgentProposal]{Items: entities, Total: total}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	log := r.l.With(zap.String("operation", "GetByID"), zap.String("id", req.ID.String()))

	entity := new(agent.AgentProposal)
	cols := buncolgen.AgentProposalColumns
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentProposalScopeTenant(sq, *req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get agent proposal", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "AgentProposal")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *agent.AgentProposal,
) (*agent.AgentProposal, error) {
	log := r.l.With(zap.String("operation", "Create"))

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create agent proposal", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req repositories.UpdateAgentProposalStatusRequest,
) (*agent.AgentProposal, error) {
	log := r.l.With(zap.String("operation", "UpdateStatus"), zap.String("id", req.ID.String()))

	entity := new(agent.AgentProposal)
	cols := buncolgen.AgentProposalColumns
	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.AgentProposalScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Set(cols.Status.Set(), req.Status).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update agent proposal status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "AgentProposal", req.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ExpirePendingByRun(
	ctx context.Context,
	req repositories.ExpireAgentProposalsByRunRequest,
) (int, error) {
	log := r.l.With(zap.String("operation", "ExpirePendingByRun"), zap.String("runId", req.RunID.String()))

	cols := buncolgen.AgentProposalColumns
	results, err := r.db.DB().
		NewUpdate().
		Model((*agent.AgentProposal)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.AgentProposalScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.RunID.Eq(), req.RunID).
				Where(cols.Status.Eq(), agent.ProposalStatusPending)
		}).
		Set(cols.Status.Set(), agent.ProposalStatusExpired).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Exec(ctx)
	if err != nil {
		log.Error("failed to expire pending agent proposals", zap.Error(err))
		return 0, err
	}

	affected, err := results.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(affected), nil
}
