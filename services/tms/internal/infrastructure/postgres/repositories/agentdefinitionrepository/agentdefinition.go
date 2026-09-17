package agentdefinitionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultDueLimit = 100

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.AgentDefinitionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentdefinition-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	cols := buncolgen.DefinitionColumns

	entities := make([]*agentdefinition.Definition, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = querybuilder.ApplyFilters(
				sq,
				buncolgen.DefinitionTable.Alias,
				req.Filter,
				(*agentdefinition.Definition)(nil),
			)
			sq = sq.Apply(buncolgen.DefinitionApplyTenant(req.Filter.TenantInfo))
			if req.EnabledOnly {
				sq = sq.Where(cols.Enabled.IsTrue())
			}
			if req.ChatOnly {
				sq = sq.Where(cols.TriggerMode.Eq(), agentdefinition.TriggerChat)
			}

			return sq.
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset()).
				Order(cols.Name.OrderAsc())
		}).ScanAndCount(ctx)
	if err != nil {
		r.l.Error("failed to list agent definitions", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*agentdefinition.Definition]{Items: entities, Total: total}, nil
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentDefinitionConnectionRequest,
) (*pagination.CursorListResult[*agentdefinition.Definition], error) {
	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*agentdefinition.Definition)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.DefinitionTable.Alias,
					req.Filter,
					(*agentdefinition.Definition)(nil),
				)

				return sq.Apply(buncolgen.DefinitionApplyTenant(req.Filter.TenantInfo))
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count agent definitions", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*agentdefinition.Definition]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(entities *[]*agentdefinition.Definition) *bun.SelectQuery {
				sq := dba.NewSelect().Model(entities)
				if len(req.Columns) == 0 {
					return sq.ColumnExpr(buncolgen.DefinitionTable.All())
				}

				return sq.Column(req.Columns...)
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.DefinitionTable.Alias,
					req.Filter,
					req.Cursor,
					(*agentdefinition.Definition)(nil),
				)
			},
		})
	if err != nil {
		r.l.Error("failed to scan agent definitions", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	entity := new(agentdefinition.Definition)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DefinitionScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.DefinitionColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "AgentDefinition")
	}

	return entity, nil
}

func (r *repository) GetBySystemKey(
	ctx context.Context,
	req repositories.GetAgentDefinitionBySystemKeyRequest,
) (*agentdefinition.Definition, error) {
	entity := new(agentdefinition.Definition)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DefinitionScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.DefinitionColumns.SystemKey.Eq(), req.SystemKey)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "AgentDefinition")
	}

	return entity, nil
}

func (r *repository) ListEnabledByTrigger(
	ctx context.Context,
	req repositories.ListAgentDefinitionsByTriggerRequest,
) ([]*agentdefinition.Definition, error) {
	cols := buncolgen.DefinitionColumns
	entities := make([]*agentdefinition.Definition, 0, 8)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.DefinitionScopeTenant(sq, req.TenantInfo).
				Where(cols.Enabled.IsTrue()).
				Where(cols.TriggerMode.Eq(), req.Mode)
			if req.EventKind != "" {
				sq = sq.Where("? = ANY("+cols.EventKinds.Qualified()+")", string(req.EventKind))
			}

			return sq
		}).
		Order(cols.Name.OrderAsc()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agent definitions by trigger: %w", err)
	}

	return entities, nil
}

func (r *repository) ListDue(
	ctx context.Context,
	req repositories.ListDueAgentDefinitionsRequest,
) ([]*agentdefinition.Definition, error) {
	cols := buncolgen.DefinitionColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultDueLimit
	}
	entities := make([]*agentdefinition.Definition, 0, limit)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DefinitionScopeTenant(sq, req.TenantInfo).
				Where(cols.Enabled.IsTrue()).
				Where(cols.TriggerMode.In(), bun.In([]agentdefinition.TriggerMode{
					agentdefinition.TriggerScheduled,
					agentdefinition.TriggerContinuous,
				})).
				Where(cols.NextRunAt.IsNotNull()).
				Where(cols.NextRunAt.Lte(), req.Now).
				WhereGroup(" AND ", func(inner *bun.SelectQuery) *bun.SelectQuery {
					return inner.Where(cols.EndsAt.IsNull()).
						WhereOr(cols.EndsAt.Gt(), req.Now)
				})
		}).
		Order(cols.NextRunAt.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list due agent definitions: %w", err)
	}

	return entities, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *agentdefinition.Definition,
) (*agentdefinition.Definition, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateName(entity.Name)
		}
		r.l.Error("failed to create agent definition", zap.Error(err))

		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *agentdefinition.Definition,
) (*agentdefinition.Definition, error) {
	cols := buncolgen.DefinitionColumns
	ov := entity.Version
	entity.Version++

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.DefinitionScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), ov)
		}).
		Set(cols.Name.Set(), entity.Name).
		Set(cols.Description.Set(), entity.Description).
		Set(cols.Template.Set(), entity.Template).
		Set(cols.Instructions.Set(), entity.Instructions).
		Set(cols.Guardrails.Set(), entity.Guardrails).
		Set(cols.ToolNames.Set(), entity.ToolNames).
		Set(cols.ToolTiers.Set(), entity.ToolTiers).
		Set(cols.AutonomyCeiling.Set(), entity.AutonomyCeiling).
		Set(cols.Enabled.Set(), entity.Enabled).
		Set(cols.ShadowMode.Set(), entity.ShadowMode).
		Set(cols.DecisionTimeoutSeconds.Set(), entity.DecisionTimeoutSeconds).
		Set(cols.TriggerMode.Set(), entity.TriggerMode).
		Set(cols.CronExpression.Set(), entity.CronExpression).
		Set(cols.CronTimezone.Set(), entity.CronTimezone).
		Set(cols.EventKinds.Set(), entity.EventKinds).
		Set(cols.IntervalSeconds.Set(), entity.IntervalSeconds).
		Set(cols.EndsAt.Set(), entity.EndsAt).
		Set(cols.MaxConcurrentRuns.Set(), entity.MaxConcurrentRuns).
		Set(cols.RunTimeoutSeconds.Set(), entity.RunTimeoutSeconds).
		Set(cols.MaxToolCalls.Set(), entity.MaxToolCalls).
		Set(cols.ContextProviders.Set(), entity.ContextProviders).
		Set(cols.OutputMode.Set(), entity.OutputMode).
		Set(cols.PreferredProviderID.Set(), entity.PreferredProviderID).
		Set(cols.NextRunAt.Set(), entity.NextRunAt).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Set(), entity.Version).
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateName(entity.Name)
		}

		return nil, fmt.Errorf("update agent definition: %w", err)
	}

	if err = dberror.CheckRowsAffected(res, "AgentDefinition", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) MarkRun(
	ctx context.Context,
	req repositories.MarkAgentDefinitionRunRequest,
) (bool, error) {
	cols := buncolgen.DefinitionColumns

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*agentdefinition.Definition)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			uq = buncolgen.DefinitionScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
			if req.ExpectedNextRunAt == nil {
				return uq.Where(cols.NextRunAt.IsNull())
			}

			return uq.Where(cols.NextRunAt.Eq(), *req.ExpectedNextRunAt)
		}).
		Set(cols.LastRunAt.Set(), req.LastRunAt).
		Set(cols.NextRunAt.Set(), req.NextRunAt).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("mark agent definition run: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("mark agent definition run: %w", err)
	}

	return affected > 0, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.DeleteAgentDefinitionRequest,
) error {
	res, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*agentdefinition.Definition)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.DefinitionScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.DefinitionColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete agent definition: %w", err)
	}

	return dberror.CheckRowsAffected(res, "AgentDefinition", req.ID.String())
}

func (r *repository) StatsByIDs(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]repositories.AgentDefinitionStats, error) {
	stats := make(map[pulid.ID]repositories.AgentDefinitionStats, len(ids))
	if len(ids) == 0 {
		return stats, nil
	}

	runCols := buncolgen.AgentRunColumns
	var runRows []repositories.AgentDefinitionStats
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*agent.AgentRun)(nil)).
		ColumnExpr(runCols.AgentDefinitionID.Qualified()+" AS definition_id").
		ColumnExpr("SUM(CASE WHEN "+runCols.Status.Qualified()+" IN (?) THEN 1 ELSE 0 END) AS open_runs",
			bun.In([]agent.RunStatus{
				agent.RunStatusPending,
				agent.RunStatusGatheringContext,
				agent.RunStatusDiagnosing,
				agent.RunStatusAwaitingDecision,
			})).
		ColumnExpr("MAX("+runCols.StartedAt.Qualified()+") AS last_run_at").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentRunScopeTenant(sq, tenant).
				Where(runCols.AgentDefinitionID.In(), bun.In(ids))
		}).
		GroupExpr(runCols.AgentDefinitionID.Qualified()).
		Scan(ctx, &runRows)
	if err != nil {
		return nil, fmt.Errorf("agent definition run stats: %w", err)
	}
	for _, row := range runRows {
		stats[row.DefinitionID] = row
	}

	proposalCols := buncolgen.AgentProposalColumns
	var proposalRows []struct {
		DefinitionID     pulid.ID `bun:"definition_id"`
		PendingProposals int      `bun:"pending_proposals"`
	}
	err = r.db.DBForContext(ctx).
		NewSelect().
		Model((*agent.AgentProposal)(nil)).
		Join("JOIN agent_runs AS ar ON ar.id = "+proposalCols.RunID.Qualified()+
			" AND ar.organization_id = "+proposalCols.OrganizationID.Qualified()+
			" AND ar.business_unit_id = "+proposalCols.BusinessUnitID.Qualified()).
		ColumnExpr("ar.agent_definition_id AS definition_id").
		ColumnExpr("COUNT(*) AS pending_proposals").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AgentProposalScopeTenant(sq, tenant).
				Where(proposalCols.Status.Eq(), agent.ProposalStatusPending).
				Where("ar.agent_definition_id IN (?)", bun.In(ids))
		}).
		GroupExpr("ar.agent_definition_id").
		Scan(ctx, &proposalRows)
	if err != nil {
		return nil, fmt.Errorf("agent definition proposal stats: %w", err)
	}
	for _, row := range proposalRows {
		entry := stats[row.DefinitionID]
		entry.DefinitionID = row.DefinitionID
		entry.PendingProposals = row.PendingProposals
		stats[row.DefinitionID] = entry
	}

	return stats, nil
}

func duplicateName(name string) error {
	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"name",
		errortypes.ErrDuplicate,
		fmt.Sprintf("An agent named %q already exists", name),
	)

	return multiErr
}
