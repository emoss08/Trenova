package agentevaluationrepository

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

func New(p Params) repositories.AgentEvaluationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentevaluation-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *agent.Evaluation,
) (*agent.Evaluation, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		r.l.Error("failed to create agent evaluation", zap.Error(err))

		return nil, fmt.Errorf("create agent evaluation: %w", err)
	}

	return entity, nil
}

// Update writes the whole outcome under the version, since one workflow
// owns an evaluation from start to finish and nothing else edits it.
func (r *repository) Update(
	ctx context.Context,
	entity *agent.Evaluation,
) (*agent.Evaluation, error) {
	cols := buncolgen.EvaluationColumns
	ov := entity.Version
	entity.Version++

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.EvaluationScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), ov)
		}).
		Set(cols.Status.Set(), entity.Status).
		Set(cols.Input.Set(), entity.Input).
		Set(cols.DefinitionVersion.Set(), entity.DefinitionVersion).
		Set(cols.PromptVersion.Set(), entity.PromptVersion).
		Set(cols.Model.Set(), entity.Model).
		Set(cols.ProviderID.Set(), entity.ProviderID).
		Set(cols.Reply.Set(), entity.Reply).
		Set(cols.Actions.Set(), entity.Actions).
		Set(cols.Comparison.Set(), entity.Comparison).
		Set(cols.OriginalProposals.Set(), entity.OriginalProposals).
		Set(cols.ToolCallsUsed.Set(), entity.ToolCallsUsed).
		Set(cols.WorkflowID.Set(), entity.WorkflowID).
		Set(cols.ErrorMessage.Set(), entity.ErrorMessage).
		Set(cols.StartedAt.Set(), entity.StartedAt).
		Set(cols.CompletedAt.Set(), entity.CompletedAt).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Set(), entity.Version).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update agent evaluation: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update agent evaluation rows: %w", err)
	}
	if rows == 0 {
		return nil, dberror.CreateVersionMismatchError("AgentEvaluation", entity.ID.String())
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentEvaluationByIDRequest,
) (*agent.Evaluation, error) {
	entity := new(agent.Evaluation)
	cols := buncolgen.EvaluationColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.EvaluationScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "AgentEvaluation")
	}

	return entity, nil
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentEvaluationConnectionRequest,
) (*pagination.CursorListResult[*agent.Evaluation], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*agent.Evaluation)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.EvaluationTable.Alias,
					req.Filter,
					(*agent.Evaluation)(nil),
				)

				return sq.Apply(buncolgen.EvaluationApplyTenant(req.Filter.TenantInfo))
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count agent evaluations", zap.Error(err))

			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*agent.Evaluation]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(entities *[]*agent.Evaluation) *bun.SelectQuery {
				q := dba.NewSelect().Model(entities)
				if len(req.Columns) > 0 {
					q = q.Column(req.Columns...)
				}

				return q
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.EvaluationTable.Alias,
					req.Filter,
					req.Cursor,
					(*agent.Evaluation)(nil),
				)
			},
		})
	if err != nil {
		log.Error("failed to list agent evaluations", zap.Error(err))

		return nil, err
	}

	return result, nil
}
