package agentdefinitionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports"
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
			sq = applyUsability(
				sq.Apply(buncolgen.DefinitionApplyTenant(req.Filter.TenantInfo)),
				usability{
					enabledOnly: req.EnabledOnly,
					chatOnly:    req.ChatOnly,
					audience:    req.Audience,
				},
			)

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

				return applyUsability(
					sq.Apply(buncolgen.DefinitionApplyTenant(req.Filter.TenantInfo)),
					connectionUsability(req),
				)
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
					applyUsability(sq, connectionUsability(req)),
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

type usability struct {
	enabledOnly bool
	chatOnly    bool
	audience    *repositories.AgentAudience
}

func connectionUsability(req *repositories.ListAgentDefinitionConnectionRequest) usability {
	return usability{
		enabledOnly: req.EnabledOnly,
		chatOnly:    req.ChatOnly,
		audience:    req.Audience,
	}
}

func applyUsability(sq *bun.SelectQuery, u usability) *bun.SelectQuery {
	cols := buncolgen.DefinitionColumns
	if u.enabledOnly {
		sq = sq.Where(cols.Enabled.IsTrue())
	}
	if u.chatOnly {
		sq = sq.Where(cols.TriggerMode.Eq(), agentdefinition.TriggerChat)
	}
	if u.audience == nil {
		return sq
	}

	return sq.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		q = q.Where(cols.AccessMode.Eq(), agentdefinition.AccessEveryone).
			WhereOr(cols.SystemKey.IsNotNull())
		if len(u.audience.GrantedAgentIDs) > 0 {
			q = q.WhereOr(cols.ID.In(), bun.List(u.audience.GrantedAgentIDs))
		}

		return q
	})
}

func (r *repository) ListByIDs(
	ctx context.Context,
	req repositories.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	if len(req.IDs) == 0 {
		return []*agentdefinition.Definition{}, nil
	}

	definitions := make([]*agentdefinition.Definition, 0, len(req.IDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&definitions).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DefinitionScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.DefinitionColumns.ID.In(), bun.In(req.IDs))
		}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agent definitions by ids: %w", err)
	}

	return definitions, nil
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

func (r *repository) ListScheduledAcrossTenants(
	ctx context.Context,
	req repositories.ListScheduledAcrossTenantsRequest,
) ([]*agentdefinition.Definition, error) {
	cols := buncolgen.DefinitionColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultDueLimit
	}
	entities := make([]*agentdefinition.Definition, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.TriggerMode.In(), bun.In([]agentdefinition.TriggerMode{
			agentdefinition.TriggerScheduled,
			agentdefinition.TriggerContinuous,
		}))
	if req.AfterID.IsNotNil() {
		query = query.Where(cols.ID.Gt(), req.AfterID)
	}

	if err := query.Order(cols.ID.OrderAsc()).Limit(limit).Scan(ctx); err != nil {
		return nil, fmt.Errorf("list scheduled agent definitions across tenants: %w", err)
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
		Set(cols.Guardrails.Set(), dbhelper.TextArray(entity.Guardrails)).
		Set(cols.ToolNames.Set(), dbhelper.TextArray(entity.ToolNames)).
		Set(cols.ToolTiers.Set(), entity.ToolTiers).
		Set(cols.AutonomyCeiling.Set(), entity.AutonomyCeiling).
		Set(cols.Enabled.Set(), entity.Enabled).
		Set(cols.ShadowMode.Set(), entity.ShadowMode).
		Set(cols.DecisionTimeoutSeconds.Set(), entity.DecisionTimeoutSeconds).
		Set(cols.TriggerMode.Set(), entity.TriggerMode).
		Set(cols.CronExpression.Set(), entity.CronExpression).
		Set(cols.CronTimezone.Set(), entity.CronTimezone).
		Set(cols.EventKinds.Set(), dbhelper.TextArray(entity.EventKinds)).
		Set(cols.IntervalSeconds.Set(), entity.IntervalSeconds).
		Set(cols.EndsAt.Set(), entity.EndsAt).
		Set(cols.MaxConcurrentRuns.Set(), entity.MaxConcurrentRuns).
		Set(cols.RunTimeoutSeconds.Set(), entity.RunTimeoutSeconds).
		Set(cols.MaxToolCalls.Set(), entity.MaxToolCalls).
		Set(cols.Icon.Set(), entity.Icon).
		Set(cols.Accent.Set(), entity.Accent).
		Set(cols.ContextProviders.Set(), dbhelper.TextArray(entity.ContextProviders)).
		Set(cols.OutputMode.Set(), entity.OutputMode).
		Set(cols.PreferredProviderID.Set(), entity.PreferredProviderID).
		Set(cols.DelegateIDs.Set(), dbhelper.TextArray(entity.DelegateIDs)).
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

// SetToolTier merges one tool's tier into tool_tiers in place. Update writes
// the whole row from a definition a person loaded, so a tier the ledger set
// between their load and their save would be silently undone; a JSONB merge
// touches only the one key. The tool must be one the agent holds, since a
// tier for a tool the agent cannot call is a validation error on the next
// save. A core tool is held by every agent without being listed.
func (r *repository) SetToolTier(
	ctx context.Context,
	req repositories.SetAgentDefinitionToolTierRequest,
) error {
	cols := buncolgen.DefinitionColumns

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*agentdefinition.Definition)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			uq = buncolgen.DefinitionScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
			if agentdefinition.IsCoreTool(req.ToolName) {
				return uq
			}

			return uq.Where("? = ANY("+cols.ToolNames.Qualified()+")", req.ToolName)
		}).
		Set(
			cols.ToolTiers.SetExpr("COALESCE({}, jsonb_build_object()) || jsonb_build_object(?::text, ?::text)"),
			req.ToolName,
			string(req.Tier),
		).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("set agent definition tool tier: %w", err)
	}

	return dberror.CheckRowsAffected(res, "AgentDefinition", req.ID.String())
}

// SetAccessMode changes who may use an agent without rewriting the rest of
// it. Update never writes the column, so a save of the agent's form cannot
// undo a change made here.
func (r *repository) SetAccessMode(
	ctx context.Context,
	req repositories.SetAgentDefinitionAccessModeRequest,
) error {
	cols := buncolgen.DefinitionColumns

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*agentdefinition.Definition)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.DefinitionScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Set(cols.AccessMode.Set(), req.Mode).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("set agent definition access mode: %w", err)
	}

	return dberror.CheckRowsAffected(res, "AgentDefinition", req.ID.String())
}

// Delete removes an agent and, in the same transaction, takes it off every
// allowlist in its tenant that names it. An array column cannot carry a
// foreign key, so this is the cascade: no agent is left able to ask one that
// no longer exists.
func (r *repository) Delete(
	ctx context.Context,
	req repositories.DeleteAgentDefinitionRequest,
) error {
	return r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if err := r.removeDelegate(txCtx, req); err != nil {
			return err
		}

		res, err := r.db.DBForContext(txCtx).
			NewDelete().
			Model((*agentdefinition.Definition)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.DefinitionScopeTenantDelete(dq, req.TenantInfo).
					Where(buncolgen.DefinitionColumns.ID.Eq(), req.ID)
			}).
			Exec(txCtx)
		if err != nil {
			return fmt.Errorf("delete agent definition: %w", err)
		}

		return dberror.CheckRowsAffected(res, "AgentDefinition", req.ID.String())
	})
}

// removeDelegate takes an agent off the allowlist of every agent in its
// tenant that may hand it work. Each one changed is a new version, so a save
// made from a copy loaded before the delete is refused rather than putting
// the deleted agent back.
func (r *repository) removeDelegate(
	ctx context.Context,
	req repositories.DeleteAgentDefinitionRequest,
) error {
	cols := buncolgen.DefinitionColumns

	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*agentdefinition.Definition)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.DefinitionScopeTenantUpdate(uq, req.TenantInfo).
				Where("?::text = ANY("+cols.DelegateIDs.Qualified()+")", req.ID.String())
		}).
		Set(
			cols.DelegateIDs.SetExpr("NULLIF(array_remove({}, ?::text), '{}'::text[])"),
			req.ID.String(),
		).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("remove a deleted agent from the agents that delegate to it: %w", err)
	}

	return nil
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
