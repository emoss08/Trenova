package agentdefinitionrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultDueLimit = 100

var definitionErrors = dberror.Mapping{Entity: "AgentDefinition"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentdefinitionports.AgentDefinitionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentdefinition-repository"),
	}
}

type usability struct {
	enabledOnly bool
	chatOnly    bool
	audience    *kernelports.AgentAudience
}

func (u usability) where() []keel.Pred {
	d := agentdefinitiondb.Definitions
	preds := []keel.Pred{keel.When(u.enabledOnly, d.Enabled.IsTrue())}
	if u.chatOnly {
		preds = append(preds,
			d.TriggerMode.Eq(agentdefinition.TriggerChat),
			keel.Or(d.SystemKey.IsNull(), d.SystemKey.NotIn(agentdefinition.PageAgentKeys()...)),
		)
	}
	if u.audience != nil {
		preds = append(preds, keel.Or(
			d.AccessMode.Eq(agentdefinition.AccessEveryone),
			d.SystemKey.IsNotNull(),
			keel.When(
				len(u.audience.GrantedAgentIDs) > 0,
				d.ID.In(u.audience.GrantedAgentIDs...),
			),
		))
	}

	return preds
}

func (r *repository) List(
	ctx context.Context,
	req *agentdefinitionports.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*agentdefinition.Definition], error) {
			d := agentdefinitiondb.Definitions
			u := usability{
				enabledOnly: req.EnabledOnly,
				chatOnly:    req.ChatOnly,
				audience:    req.Audience,
			}

			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				d.Where(u.where()...),
				&listquery.Request[agentdefinition.Definition]{
					Filter: req.Filter,
					Then:   []keel.Order{d.Name.Asc(), d.ID.Asc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list agent definitions", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *agentdefinitionports.ListAgentDefinitionConnectionRequest,
) (*pagination.CursorListResult[*agentdefinition.Definition], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agentdefinition.Definition], error) {
			u := usability{
				enabledOnly: req.EnabledOnly,
				chatOnly:    req.ChatOnly,
				audience:    req.Audience,
			}

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				agentdefinitiondb.Definitions.Where(u.where()...),
				&listquery.Request[agentdefinition.Definition]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to list agent definitions", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListByIDs(
	ctx context.Context,
	req agentdefinitionports.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentdefinition.Definition, error) {
		if len(req.IDs) == 0 {
			return []*agentdefinition.Definition{}, nil
		}

		d := agentdefinitiondb.Definitions
		definitions, err := d.Where(d.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list agent definitions by ids: %w", err)
		}

		return definitions, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req agentdefinitionports.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentdefinition.Definition, error) {
		entity, err := agentdefinitiondb.Definitions.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, definitionErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetBySystemKey(
	ctx context.Context,
	req agentdefinitionports.GetAgentDefinitionBySystemKeyRequest,
) (*agentdefinition.Definition, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentdefinition.Definition, error) {
		d := agentdefinitiondb.Definitions
		entity, err := d.Where(d.SystemKey.Eq(req.SystemKey)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, definitionErrors.Map(err, req.SystemKey)
		}

		return entity, nil
	})
}

func (r *repository) ListEnabledByTrigger(
	ctx context.Context,
	req agentdefinitionports.ListAgentDefinitionsByTriggerRequest,
) ([]*agentdefinition.Definition, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentdefinition.Definition, error) {
		d := agentdefinitiondb.Definitions
		entities, err := d.Where(
			d.Enabled.IsTrue(),
			d.TriggerMode.Eq(req.Mode),
			keel.When(req.EventKind != "", d.EventKinds.Has(string(req.EventKind))),
		).
			Order(d.Name.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list agent definitions by trigger: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) ListEnabledForEvents(
	ctx context.Context,
	req agentdefinitionports.ListAgentDefinitionsForEventsRequest,
) ([]*agentdefinition.Definition, error) {
	if len(req.EventKinds) == 0 {
		return []*agentdefinition.Definition{}, nil
	}
	kinds := make([]string, 0, len(req.EventKinds))
	for _, kind := range req.EventKinds {
		kinds = append(kinds, string(kind))
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentdefinition.Definition, error) {
		d := agentdefinitiondb.Definitions
		entities, err := d.Where(
			d.Enabled.IsTrue(),
			d.TriggerMode.Eq(agentdefinition.TriggerEvent),
			d.EventKinds.Overlaps(kinds...),
		).
			Order(d.Name.Asc(), d.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list agent definitions for events: %w", err)
		}

		return entities, nil
	})
}

func scheduledModes() []agentdefinition.TriggerMode {
	return []agentdefinition.TriggerMode{
		agentdefinition.TriggerScheduled,
		agentdefinition.TriggerContinuous,
	}
}

func (r *repository) ListScheduledAcrossTenants(
	ctx context.Context,
	req agentdefinitionports.ListScheduledAcrossTenantsRequest,
) ([]*agentdefinition.Definition, error) {
	ctx = dbscope.WithSystem(ctx, "list scheduled agent definitions across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentdefinition.Definition, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultDueLimit
		}

		d := agentdefinitiondb.Definitions
		entities, err := d.AllTenants().
			Where(
				d.TriggerMode.In(scheduledModes()...),
				keel.When(req.AfterID.IsNotNil(), d.ID.Gt(req.AfterID)),
			).
			Order(d.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list scheduled agent definitions across tenants: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) ListDue(
	ctx context.Context,
	req agentdefinitionports.ListDueAgentDefinitionsRequest,
) ([]*agentdefinition.Definition, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentdefinition.Definition, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultDueLimit
		}

		d := agentdefinitiondb.Definitions
		entities, err := d.Where(
			d.Enabled.IsTrue(),
			d.TriggerMode.In(scheduledModes()...),
			d.NextRunAt.IsNotNull(),
			d.NextRunAt.Lte(req.Now),
			keel.Or(d.EndsAt.IsNull(), d.EndsAt.Gt(req.Now)),
		).
			Order(d.NextRunAt.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list due agent definitions: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *agentdefinition.Definition,
) (*agentdefinition.Definition, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentdefinition.Definition, error) {
		if err := agentdefinitiondb.Definitions.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, duplicateName(entity.Name)
			}
			r.l.Error("failed to create agent definition", zap.Error(err))

			return nil, definitionErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateSystem(
	ctx context.Context,
	entity *agentdefinition.Definition,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		if !entity.IsSystem() {
			return false, errors.New("only a system agent is created on first use")
		}

		inserted, err := agentdefinitiondb.Definitions.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*agentdefinition.Definition{entity},
		)
		if err != nil {
			r.l.Error("failed to create system agent definition",
				zap.String("systemKey", entity.SystemKey),
				zap.Error(err),
			)

			return false, fmt.Errorf("create system agent %s: %w", entity.SystemKey, err)
		}

		return inserted > 0, nil
	})
}

// Update saves the definition a person edited. The columns other writers own
// are left alone: who may use the agent, its unattended role, its system key,
// who made it and when it last ran.
func (r *repository) Update(
	ctx context.Context,
	entity *agentdefinition.Definition,
) (*agentdefinition.Definition, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentdefinition.Definition, error) {
		d := agentdefinitiondb.Definitions
		err := d.SaveExcept(
			ctx,
			r.db.Keel(ctx),
			entity,
			d.AccessMode,
			d.UnattendedRoleID,
			d.SystemKey,
			d.CreatedByID,
			d.LastRunAt,
		)
		if err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, duplicateName(entity.Name)
			}

			return nil, definitionErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) MarkRun(
	ctx context.Context,
	req agentdefinitionports.MarkAgentDefinitionRunRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		d := agentdefinitiondb.Definitions
		expected := d.NextRunAt.IsNull()
		if req.ExpectedNextRunAt != nil {
			expected = d.NextRunAt.Eq(*req.ExpectedNextRunAt)
		}
		next := d.NextRunAt.ToNull()
		if req.NextRunAt != nil {
			next = d.NextRunAt.To(*req.NextRunAt)
		}

		n, err := d.Where(d.ID.Eq(req.ID), expected).Update(
			ctx,
			r.db.Keel(ctx),
			d.LastRunAt.To(req.LastRunAt),
			next,
			d.Version.ToExpr(d.Version),
		)
		if err != nil {
			return false, fmt.Errorf("mark agent definition run: %w", err)
		}

		return n > 0, nil
	})
}

// SetToolTier merges one tool's tier into tool_tiers in place. Update writes
// the whole row from a definition a person loaded, so a tier the ledger set
// between their load and their save would be silently undone; a JSONB merge
// touches only the one key. An agent saved with no tiers holds JSON null, and
// null || {...} is an array that no definition can be read from, so anything
// but an object is merged into as empty. The tool must be one the agent holds, since a
// tier for a tool the agent cannot call is a validation error on the next
// save. A core tool is held by every agent without being listed.
func (r *repository) SetToolTier(
	ctx context.Context,
	req agentdefinitionports.SetAgentDefinitionToolTierRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := agentdefinitiondb.Definitions
		n, err := d.Where(
			d.ID.Eq(req.ID),
			keel.When(!agentdefinition.IsCoreTool(req.ToolName), d.ToolNames.Has(req.ToolName)),
		).Update(
			ctx,
			r.db.Keel(ctx),
			d.ToolTiers.ToExpr(keel.Raw(
				"(CASE WHEN jsonb_typeof(?) = 'object' THEN ? ELSE jsonb_build_object() END) "+
					"|| jsonb_build_object(?::text, ?::text)",
				d.ToolTiers,
				d.ToolTiers,
				req.ToolName,
				string(req.Tier),
			)),
		)
		if err != nil {
			return fmt.Errorf("set agent definition tool tier: %w", err)
		}

		return dberror.CheckAffected(n, "AgentDefinition", req.ID.String())
	})
}

// SetAccessMode changes who may use an agent without rewriting the rest of
// it. Update never writes the column, so a save of the agent's form cannot
// undo a change made here.
func (r *repository) SetAccessMode(
	ctx context.Context,
	req agentdefinitionports.SetAgentDefinitionAccessModeRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := agentdefinitiondb.Definitions
		n, err := d.Where(d.ID.Eq(req.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			d.AccessMode.To(req.Mode),
			d.Version.ToExpr(d.Version),
		)
		if err != nil {
			return fmt.Errorf("set agent definition access mode: %w", err)
		}

		return dberror.CheckAffected(n, "AgentDefinition", req.ID.String())
	})
}

func (r *repository) SetUnattendedRole(
	ctx context.Context,
	req agentdefinitionports.SetAgentDefinitionUnattendedRoleRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := agentdefinitiondb.Definitions
		n, err := d.Where(d.ID.Eq(req.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			d.UnattendedRoleID.To(req.RoleID),
			d.Version.ToExpr(d.Version),
		)
		if err != nil {
			return fmt.Errorf("set agent definition unattended role: %w", err)
		}

		return dberror.CheckAffected(n, "AgentDefinition", req.ID.String())
	})
}

func (r *repository) UnattendedRoleID(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (pulid.ID, error) {
		d := agentdefinitiondb.Definitions
		entity, err := d.Pick(d.UnattendedRoleID).Where(d.ID.Eq(id)).One(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return pulid.Nil, nil
		}
		if err != nil {
			return pulid.Nil, fmt.Errorf("read agent definition unattended role: %w", err)
		}

		return entity.UnattendedRoleID, nil
	})
}

// Delete removes an agent and, in the same transaction, takes it off every
// allowlist in its tenant that names it. An array column cannot carry a
// foreign key, so this is the cascade: no agent is left able to ask one that
// no longer exists.
func (r *repository) Delete(
	ctx context.Context,
	req agentdefinitionports.DeleteAgentDefinitionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := r.removeDelegate(ctx, req.ID); err != nil {
			return err
		}

		d := agentdefinitiondb.Definitions
		n, err := d.Where(d.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return fmt.Errorf("delete agent definition: %w", err)
		}

		return dberror.CheckAffected(n, "AgentDefinition", req.ID.String())
	})
}

// removeDelegate takes an agent off the allowlist of every agent in its
// tenant that may hand it work. Each one changed is a new version, so a save
// made from a copy loaded before the delete is refused rather than putting
// the deleted agent back.
func (r *repository) removeDelegate(ctx context.Context, id pulid.ID) error {
	d := agentdefinitiondb.Definitions
	_, err := d.Where(d.DelegateIDs.Has(id.String())).Update(
		ctx,
		r.db.Keel(ctx),
		d.DelegateIDs.ToExpr(keel.Raw(
			"NULLIF(array_remove(?, ?::text), ARRAY[]::text[])",
			d.DelegateIDs,
			id.String(),
		)),
	)
	if err != nil {
		return fmt.Errorf("remove a deleted agent from the agents that delegate to it: %w", err)
	}

	return nil
}

type proposalStat struct {
	DefinitionID     pulid.ID
	PendingProposals int
}

func openRunStatuses() []agent.RunStatus {
	return []agent.RunStatus{
		agent.RunStatusPending,
		agent.RunStatusGatheringContext,
		agent.RunStatusDiagnosing,
		agent.RunStatusAwaitingDecision,
	}
}

func (r *repository) StatsByIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]agentdefinitionports.AgentDefinitionStats, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]agentdefinitionports.AgentDefinitionStats, error) {
			stats := make(map[pulid.ID]agentdefinitionports.AgentDefinitionStats, len(ids))
			if len(ids) == 0 {
				return stats, nil
			}

			run := agentdb.AgentRuns
			p := agentdb.AgentProposals
			pRun := agentdb.AgentRuns.As("prun")

			var (
				runRows      []agentdefinitionports.AgentDefinitionStats
				proposalRows []proposalStat
			)
			err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.IntoSlice(run.
					Group(run.AgentDefinitionID).
					Pick(
						run.AgentDefinitionID,
						keel.DynLit(0).Cast(keel.TypeInt64),
						keel.Count().Filter(run.Status.In(openRunStatuses()...)),
						run.StartedAt.Max(),
					).
					Where(run.AgentDefinitionID.In(ids...)), &runRows),
				keel.IntoSlice(p.
					Join(pRun, pRun.ID.EqCol(p.RunID)).
					Group(pRun.AgentDefinitionID).
					Pick(pRun.AgentDefinitionID, keel.Count()).
					Where(
						p.Status.Eq(agent.ProposalStatusPending),
						pRun.AgentDefinitionID.In(ids...),
					), &proposalRows),
			)
			if err != nil {
				return nil, fmt.Errorf("agent definition stats: %w", err)
			}

			for _, row := range runRows {
				stats[row.DefinitionID] = row
			}
			for _, row := range proposalRows {
				entry := stats[row.DefinitionID]
				entry.DefinitionID = row.DefinitionID
				entry.PendingProposals = row.PendingProposals
				stats[row.DefinitionID] = entry
			}

			return stats, nil
		},
	)
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
