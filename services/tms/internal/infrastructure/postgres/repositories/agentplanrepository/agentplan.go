package agentplanrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
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

func New(p Params) repositories.AgentPlanRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentplan-repository"),
	}
}

func (r *repository) Create(ctx context.Context, entity *agent.AgentPlan) (*agent.AgentPlan, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		r.l.Error("failed to create agent plan", zap.Error(err))

		return nil, fmt.Errorf("create agent plan: %w", err)
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentPlanByIDRequest,
) (*agent.AgentPlan, error) {
	entity := new(agent.AgentPlan)
	cols := buncolgen.AgentPlanColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentPlanScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "AgentPlan")
	}

	return entity, nil
}

// ListByThread finds the plans of every run a conversation started, oldest
// first, so a reopened thread can show them where their turns were.
func (r *repository) ListByThread(
	ctx context.Context,
	req repositories.ListAgentPlansByThreadRequest,
) ([]*agent.AgentPlan, error) {
	cols := buncolgen.AgentPlanColumns
	runs := buncolgen.AgentRunColumns
	plans := make([]*agent.AgentPlan, 0)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&plans).
		Join("JOIN agent_runs AS ar ON "+runs.ID.Qualified()+" = "+cols.RunID.Qualified()+
			" AND "+runs.OrganizationID.Qualified()+" = "+cols.OrganizationID.Qualified()+
			" AND "+runs.BusinessUnitID.Qualified()+" = "+cols.BusinessUnitID.Qualified()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentPlanScopeTenant(sq, req.TenantInfo).
				Where(runs.SubjectType.Qualified()+" = ?", agent.SubjectAssistantThread).
				Where(runs.SubjectID.Qualified()+" = ?", req.ThreadID)
		}).
		OrderExpr(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list agent plans for thread", zap.Error(err))

		return nil, fmt.Errorf("list agent plans by thread: %w", err)
	}

	return plans, nil
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentPlanConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentPlan], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*agent.AgentPlan)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.AgentPlanTable.Alias,
					req.Filter,
					(*agent.AgentPlan)(nil),
				)
				sq = sq.Apply(buncolgen.AgentPlanApplyTenant(req.Filter.TenantInfo))
				if req.ExcludeShadowDefinitions {
					sq = excludeShadowDefinitions(sq)
				}

				return sq
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count agent plans", zap.Error(err))

			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*agent.AgentPlan]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(entities *[]*agent.AgentPlan) *bun.SelectQuery {
				q := dba.NewSelect().Model(entities)
				if len(req.Columns) > 0 {
					q = q.Column(req.Columns...)
				}

				return q
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.AgentPlanTable.Alias,
					req.Filter,
					req.Cursor,
					(*agent.AgentPlan)(nil),
				)
				if applyErr != nil {
					return nil, applyErr
				}
				if req.ExcludeShadowDefinitions {
					sq = excludeShadowDefinitions(sq)
				}

				return sq, nil
			},
		})
	if err != nil {
		log.Error("failed to list agent plans", zap.Error(err))

		return nil, err
	}

	return result, nil
}

// excludeShadowDefinitions hides plans whose agent is in shadow, with the
// same subquery the proposal list uses.
func excludeShadowDefinitions(q *bun.SelectQuery) *bun.SelectQuery {
	plans := buncolgen.AgentPlanColumns
	runs := buncolgen.AgentRunColumns
	definitions := buncolgen.DefinitionColumns

	return q.Where(
		"NOT EXISTS (SELECT 1 FROM ? AS ? JOIN ? AS ? ON ? = ? AND ? = ? AND ? = ? "+
			"WHERE ? = ? AND ? = ? AND ? = ? AND ? = TRUE)",
		bun.Ident(buncolgen.AgentRunTable.Name),
		bun.Ident(buncolgen.AgentRunTable.Alias),
		bun.Ident(buncolgen.DefinitionTable.Name),
		bun.Ident(buncolgen.DefinitionTable.Alias),
		bun.Safe(definitions.ID.Qualified()),
		bun.Safe(runs.AgentDefinitionID.Qualified()),
		bun.Safe(definitions.OrganizationID.Qualified()),
		bun.Safe(runs.OrganizationID.Qualified()),
		bun.Safe(definitions.BusinessUnitID.Qualified()),
		bun.Safe(runs.BusinessUnitID.Qualified()),
		bun.Safe(runs.ID.Qualified()),
		bun.Safe(plans.RunID.Qualified()),
		bun.Safe(runs.OrganizationID.Qualified()),
		bun.Safe(plans.OrganizationID.Qualified()),
		bun.Safe(runs.BusinessUnitID.Qualified()),
		bun.Safe(plans.BusinessUnitID.Qualified()),
		bun.Safe(definitions.ShadowMode.Qualified()),
	)
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req repositories.UpdateAgentPlanStatusRequest,
) (*agent.AgentPlan, error) {
	entity := new(agent.AgentPlan)
	cols := buncolgen.AgentPlanColumns
	query := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			scoped := buncolgen.AgentPlanScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
			if req.FromStatus != "" {
				scoped = scoped.Where(cols.Status.Eq(), req.FromStatus)
			}

			return scoped
		}).
		Set(cols.Status.Set(), req.Status).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Inc(1))
	if req.DecidedByUserID.IsNotNil() {
		query = query.
			Set(cols.DecidedByUserID.Set(), req.DecidedByUserID).
			Set(cols.DecidedAt.Set(), req.DecidedAt)
	}

	results, err := query.Returning("*").Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update agent plan status: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "AgentPlan", req.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) RecordProgress(
	ctx context.Context,
	req repositories.RecordAgentPlanProgressRequest,
) (*agent.AgentPlan, error) {
	entity := new(agent.AgentPlan)
	cols := buncolgen.AgentPlanColumns
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.AgentPlanScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Set(cols.Status.Set(), req.Status).
		Set(cols.CompletedSteps.Set(), req.CompletedSteps).
		Set(cols.FailedStep.Set(), req.FailedStep).
		Set(cols.FailureError.Set(), req.FailureError).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Inc(1)).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("record agent plan progress: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "AgentPlan", req.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ExpirePending(
	ctx context.Context,
	req repositories.ExpireAgentPlansRequest,
) (int, error) {
	cols := buncolgen.AgentPlanColumns
	results, err := r.db.DB().
		NewUpdate().
		Model((*agent.AgentPlan)(nil)).
		Where(cols.Status.Eq(), agent.PlanStatusPending).
		Where(cols.ExpiresAt.IsNotNull()).
		Where(cols.ExpiresAt.Lte(), req.Before).
		Set(cols.Status.Set(), agent.PlanStatusExpired).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to expire pending agent plans", zap.Error(err))

		return 0, err
	}

	affected, err := results.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(affected), nil
}
