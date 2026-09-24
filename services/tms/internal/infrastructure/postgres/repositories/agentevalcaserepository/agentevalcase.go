package agentevalcaserepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultListLimit  = 500
	maxListLimit      = 2000
	defaultBatchLimit = 200
	maxBatchLimit     = 1000
	entityName        = "AgentEvalCase"
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

func New(p Params) repositories.AgentEvalCaseRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentevalcase-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *agentquality.EvalCase,
) (*agentquality.EvalCase, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("create agent eval case: %w", err)
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *agentquality.EvalCase,
) (*agentquality.EvalCase, error) {
	cols := buncolgen.EvalCaseColumns
	ov := entity.Version
	entity.Version++
	entity.UpdatedAt = timeutils.NowUnix()

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.EvalCaseScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).
				Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), ov)
		}).
		Set(cols.Title.Set(), bun.NullZero(entity.Title)).
		Set(cols.Status.Set(), entity.Status).
		Set(cols.Input.Set(), entity.Input).
		Set(cols.History.Set(), entity.History).
		Set(cols.PageContext.Set(), entity.PageContext).
		Set(cols.Mentions.Set(), entity.Mentions).
		Set(cols.SubjectType.Set(), bun.NullZero(entity.SubjectType)).
		Set(cols.SubjectID.Set(), entity.SubjectID).
		Set(cols.HeldTools.Set(), pgArray(entity.HeldTools)).
		Set(cols.ToolFixtures.Set(), entity.ToolFixtures).
		Set(cols.Expected.Set(), entity.Expected).
		Set(cols.Rubric.Set(), bun.NullZero(entity.Rubric)).
		Set(cols.Redaction.Set(), entity.Redaction).
		Set(cols.ContentHash.Set(), entity.ContentHash).
		Set(cols.ExpiresAt.Set(), entity.ExpiresAt).
		Set(cols.UpdatedAt.Set(), entity.UpdatedAt).
		Set(cols.Version.Set(), entity.Version).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update agent eval case: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update agent eval case rows: %w", err)
	}
	if rows == 0 {
		return nil, dberror.CreateVersionMismatchError(entityName, entity.ID.String())
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentEvalCaseByIDRequest,
) (*agentquality.EvalCase, error) {
	entity := new(agentquality.EvalCase)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.EvalCaseScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.EvalCaseColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, entityName)
	}

	return entity, nil
}

func (r *repository) GetByContent(
	ctx context.Context,
	req repositories.GetAgentEvalCaseByContentRequest,
) (*agentquality.EvalCase, error) {
	cols := buncolgen.EvalCaseColumns
	entity := new(agentquality.EvalCase)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.EvalCaseScopeTenant(sq, req.TenantInfo).
				Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID).
				Where(cols.ContentHash.Eq(), req.ContentHash)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, entityName)
	}

	return entity, nil
}

func (r *repository) GetByProposal(
	ctx context.Context,
	req repositories.GetAgentEvalCaseByProposalRequest,
) (*agentquality.EvalCase, error) {
	entity := new(agentquality.EvalCase)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.EvalCaseScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.EvalCaseColumns.SourceProposalID.Eq(), req.ProposalID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, entityName)
	}

	return entity, nil
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentEvalCaseConnectionRequest,
) (*pagination.CursorListResult[*agentquality.EvalCase], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))
	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*agentquality.EvalCase)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.EvalCaseTable.Alias,
					req.Filter,
					(*agentquality.EvalCase)(nil),
				)

				scoped := sq.Apply(buncolgen.EvalCaseApplyTenant(req.Filter.TenantInfo))

				return applyScope(scoped, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count agent eval cases", zap.Error(err))

			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*agentquality.EvalCase]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(entities *[]*agentquality.EvalCase) *bun.SelectQuery {
				q := dba.NewSelect().Model(entities)
				if len(req.Columns) > 0 {
					q = q.Column(req.Columns...)
				}

				return q
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				filtered, err := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.EvalCaseTable.Alias,
					req.Filter,
					req.Cursor,
					(*agentquality.EvalCase)(nil),
				)
				if err != nil {
					return nil, err
				}

				return applyScope(filtered, req), nil
			},
		})
	if err != nil {
		log.Error("failed to list agent eval cases", zap.Error(err))

		return nil, err
	}

	return result, nil
}

func applyScope(
	sq *bun.SelectQuery,
	req *repositories.ListAgentEvalCaseConnectionRequest,
) *bun.SelectQuery {
	cols := buncolgen.EvalCaseColumns
	if req.AgentDefinitionID.IsNotNil() {
		sq = sq.Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID)
	}
	if len(req.Statuses) > 0 {
		sq = sq.Where(cols.Status.In(), bun.List(req.Statuses))
	}
	if len(req.Sources) > 0 {
		sq = sq.Where(cols.Source.In(), bun.List(req.Sources))
	}

	return sq
}

func (r *repository) ListByAgent(
	ctx context.Context,
	req repositories.ListAgentEvalCasesRequest,
) ([]*agentquality.EvalCase, error) {
	cols := buncolgen.EvalCaseColumns
	limit := clamp(req.Limit, defaultListLimit, maxListLimit)

	entities := make([]*agentquality.EvalCase, 0, min(limit, defaultBatchLimit))
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.EvalCaseScopeTenant(sq, req.TenantInfo).
				Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID)
			if len(req.Statuses) > 0 {
				sq = sq.Where(cols.Status.In(), bun.List(req.Statuses))
			}

			return sq
		}).
		Order(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list agent eval cases: %w", err)
	}

	return entities, nil
}

func (r *repository) ListCaptureCandidates(
	ctx context.Context,
	req repositories.ListEvalCaseCaptureCandidatesRequest,
) ([]repositories.EvalCaseCaptureCandidate, error) {
	decision := buncolgen.AgentDecisionColumns
	proposal := buncolgen.AgentProposalColumns
	run := buncolgen.AgentRunColumns
	evalCase := buncolgen.EvalCaseColumns
	limit := clamp(req.Limit, defaultBatchLimit, maxBatchLimit)

	db := r.db.DBForContext(ctx)
	captured := db.NewSelect().
		Model((*agentquality.EvalCase)(nil)).
		ColumnExpr("1").
		Where(evalCase.OrganizationID.EqColumn(decision.OrganizationID)).
		Where(evalCase.BusinessUnitID.EqColumn(decision.BusinessUnitID)).
		Where(evalCase.SourceProposalID.EqColumn(decision.ProposalID))

	rows := make([]repositories.EvalCaseCaptureCandidate, 0, limit)
	if err := db.NewSelect().
		Model((*agent.AgentDecision)(nil)).
		ColumnExpr(decision.OrganizationID.As("organization_id")).
		ColumnExpr(decision.BusinessUnitID.As("business_unit_id")).
		ColumnExpr(decision.ProposalID.As("proposal_id")).
		ColumnExpr(decision.ID.As("decision_id")).
		ColumnExpr(decision.DecidedByUserID.As("decided_by_user_id")).
		ColumnExpr(run.AgentDefinitionID.As("agent_definition_id")).
		ColumnExpr(decision.CreatedAt.As("decided_at")).
		Join("JOIN "+buncolgen.AgentProposalTable.As(buncolgen.AgentProposalTable.Alias)+
			" ON "+proposal.ID.EqColumn(decision.ProposalID)+
			" AND "+proposal.OrganizationID.EqColumn(decision.OrganizationID)+
			" AND "+proposal.BusinessUnitID.EqColumn(decision.BusinessUnitID)).
		Join("JOIN "+buncolgen.AgentRunTable.As(buncolgen.AgentRunTable.Alias)+
			" ON "+run.ID.EqColumn(proposal.RunID)+
			" AND "+run.OrganizationID.EqColumn(proposal.OrganizationID)+
			" AND "+run.BusinessUnitID.EqColumn(proposal.BusinessUnitID)).
		Where(decision.ProposalID.IsNotNull()).
		Where(decision.Decision.In(), bun.List([]agent.DecisionType{
			agent.DecisionAccepted,
			agent.DecisionModified,
			agent.DecisionRejected,
		})).
		Where(decision.CreatedAt.Gte(), req.Since).
		Where(run.AgentDefinitionID.IsNotNull()).
		Where("NOT EXISTS (?)", captured).
		Order(decision.CreatedAt.OrderAsc(), decision.ID.OrderAsc()).
		Limit(limit).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("list eval case capture candidates: %w", err)
	}

	return rows, nil
}

func (r *repository) PurgeExpired(
	ctx context.Context,
	req repositories.PurgeExpiredEvalCasesRequest,
) (int, error) {
	cols := buncolgen.EvalCaseColumns
	limit := clamp(req.Limit, defaultBatchLimit, maxBatchLimit)
	db := r.db.DBForContext(ctx)

	due := db.NewSelect().
		Model((*agentquality.EvalCase)(nil)).
		Column(cols.ID.Bare(), cols.BusinessUnitID.Bare(), cols.OrganizationID.Bare()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			if !req.AllTenants {
				sq = buncolgen.EvalCaseScopeTenant(sq, req.TenantInfo)
			}

			return sq.WhereGroup(" AND ", func(inner *bun.SelectQuery) *bun.SelectQuery {
				inner = inner.WhereOr(
					cols.ExpiresAt.Expr("({} IS NOT NULL AND {} <= ?)"),
					req.Now,
				)
				if req.CreatedBefore > 0 {
					inner = inner.WhereOr(cols.CreatedAt.Lt(), req.CreatedBefore)
				}

				return inner
			})
		}).
		Order(cols.CreatedAt.OrderAsc()).
		Limit(limit)

	return r.deleteKeys(ctx, due)
}

func (r *repository) PurgeOrphaned(
	ctx context.Context,
	req repositories.PurgeOrphanedEvalCasesRequest,
) (int, error) {
	cols := buncolgen.EvalCaseColumns
	thread := buncolgen.ThreadColumns
	limit := clamp(req.Limit, defaultBatchLimit, maxBatchLimit)
	db := r.db.DBForContext(ctx)

	live := db.NewSelect().
		Model((*conversation.Thread)(nil)).
		ColumnExpr("1").
		Where(thread.ID.EqColumn(cols.SourceThreadID)).
		Where(thread.OrganizationID.EqColumn(cols.OrganizationID)).
		Where(thread.BusinessUnitID.EqColumn(cols.BusinessUnitID))

	orphaned := db.NewSelect().
		Model((*agentquality.EvalCase)(nil)).
		Column(cols.ID.Bare(), cols.BusinessUnitID.Bare(), cols.OrganizationID.Bare()).
		Where(cols.SourceThreadID.IsNotNull()).
		Where("NOT EXISTS (?)", live).
		Order(cols.CreatedAt.OrderAsc()).
		Limit(limit)

	return r.deleteKeys(ctx, orphaned)
}

func (r *repository) deleteKeys(ctx context.Context, keys *bun.SelectQuery) (int, error) {
	cols := buncolgen.EvalCaseColumns
	res, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*agentquality.EvalCase)(nil)).
		Where(
			buncolgen.Expr(
				"({0}, {1}, {2}) IN (?)",
				cols.ID,
				cols.BusinessUnitID,
				cols.OrganizationID,
			),
			keys,
		).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("purge agent eval cases: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge agent eval cases rows: %w", err)
	}

	return int(affected), nil
}

func clamp(value, fallback, ceiling int) int {
	return intutils.Clamp(intutils.WithDefault(value, fallback), 1, ceiling)
}

func pgArray(values []string) any {
	if values == nil {
		values = []string{}
	}

	return pgdialect.Array(values)
}
