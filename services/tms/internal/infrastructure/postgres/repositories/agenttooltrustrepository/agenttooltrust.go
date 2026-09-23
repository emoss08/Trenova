package agenttooltrustrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
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

func New(p Params) repositories.AgentToolTrustRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-tool-trust-repository"),
	}
}

const conflictTarget = "CONFLICT (organization_id, business_unit_id, agent_definition_id, tool_name) DO UPDATE"

// Record folds one outcome into the row for the agent and tool, in a single
// upsert so two decisions landing at once both count. The inserted row
// carries the outcome as deltas: a one in the counter it belongs to, and a
// streak of one for a clean approval or zero for anything else. On conflict
// the counters add and the streak either grows or resets, which is the
// arithmetic the ledger needs and nothing a read-then-write would do better.
func (r *repository) Record(
	ctx context.Context,
	req repositories.RecordToolTrustRequest,
) (*agent.ToolTrust, error) {
	if !req.Outcome.IsValid() {
		return nil, errortypes.NewValidationError(
			"outcome",
			errortypes.ErrInvalid,
			fmt.Sprintf("%q is not an outcome the trust ledger records", req.Outcome),
		)
	}

	at := req.At
	entity := &agent.ToolTrust{
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		AgentDefinitionID: req.AgentDefinitionID,
		ToolName:          req.ToolName,
		LastDecisionAt:    &at,
	}
	switch req.Outcome {
	case agent.TrustOutcomeApproved:
		entity.Streak = 1
		entity.Approvals = 1
	case agent.TrustOutcomeModified:
		entity.Modifications = 1
	case agent.TrustOutcomeRejected:
		entity.Rejections = 1
	case agent.TrustOutcomeExecutionFailed:
		entity.ExecutionFailures = 1
	}

	me := errortypes.NewMultiError()
	entity.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	cols := buncolgen.ToolTrustColumns
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On(conflictTarget).
		Set(cols.Streak.SetExpr("CASE WHEN EXCLUDED.{} > 0 THEN " + cols.Streak.Qualified() + " + 1 ELSE 0 END")).
		Set(cols.Approvals.SetExpr(cols.Approvals.Qualified() + " + EXCLUDED.{}")).
		Set(cols.Modifications.SetExpr(cols.Modifications.Qualified() + " + EXCLUDED.{}")).
		Set(cols.Rejections.SetExpr(cols.Rejections.Qualified() + " + EXCLUDED.{}")).
		Set(cols.ExecutionFailures.SetExpr(cols.ExecutionFailures.Qualified() + " + EXCLUDED.{}")).
		Set(cols.LastDecisionAt.SetExpr("EXCLUDED.{}")).
		Set(cols.UpdatedAt.SetExpr("EXCLUDED.{}")).
		Set(cols.Version.IncConflict(1)).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to record tool trust outcome",
			zap.String("agentDefinitionId", req.AgentDefinitionID.String()),
			zap.String("tool", req.ToolName),
			zap.Error(err),
		)

		return nil, fmt.Errorf("record tool trust: %w", err)
	}

	return entity, nil
}

func (r *repository) ListByDefinition(
	ctx context.Context,
	req repositories.ListToolTrustRequest,
) ([]*agent.ToolTrust, error) {
	cols := buncolgen.ToolTrustColumns
	rows := make([]*agent.ToolTrust, 0)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&rows).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ToolTrustScopeTenant(sq, req.TenantInfo).
				Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID)
		}).
		OrderExpr(cols.ToolName.OrderAsc()).
		Scan(ctx); err != nil {
		r.l.Error("failed to list tool trust",
			zap.String("agentDefinitionId", req.AgentDefinitionID.String()),
			zap.Error(err),
		)

		return nil, fmt.Errorf("list tool trust: %w", err)
	}

	return rows, nil
}

// MarkTierChange moves the row's earned tier, conditioned on its version, so
// a decision recorded after the one that asked for the change wins the race
// and the stale one is told the row moved on.
func (r *repository) MarkTierChange(
	ctx context.Context,
	req repositories.MarkToolTierChangeRequest,
) (*agent.ToolTrust, error) {
	cols := buncolgen.ToolTrustColumns
	query := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*agent.ToolTrust)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ToolTrustScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID).
				Where(cols.Version.Eq(), req.Version)
		}).
		Set(cols.Streak.Set(), 0).
		Set(cols.UpdatedAt.Set(), req.At).
		Set(cols.Version.Inc(1))

	if req.Promoted {
		query = query.
			Set(cols.EarnedTier.Set(), req.EarnedTier).
			Set(cols.PromotedAt.Set(), req.At)
	} else {
		query = query.
			Set(cols.EarnedTier.Set(), nil).
			Set(cols.DemotedAt.Set(), req.At)
	}

	res, err := query.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("mark tool trust tier change: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "AgentToolTrust", req.ID.String()); err != nil {
		return nil, err
	}

	entity := new(agent.ToolTrust)
	if err = r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ToolTrustScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "AgentToolTrust")
	}

	return entity, nil
}
