package agentrunsteprepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.AgentRunStepRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentrunstep-repository"),
	}
}

// Claim takes the step key or reports who already holds it.
//
// The unique index does the work. Two attempts of the same run reaching the
// same tool call race to insert one row; the loser is told the operation was
// already begun and reads back the winner's row to find out how far it got.
// This is the whole mechanism that keeps a retried run from tendering a load
// twice.
func (r *repository) Claim(
	ctx context.Context,
	step *agent.AgentRunStep,
) (*agent.AgentRunStep, error) {
	log := r.l.With(
		zap.String("operation", "Claim"),
		zap.String("owner", step.OwnerID.String()),
		zap.String("tool", step.ToolName),
	)

	_, err := r.db.DBForContext(ctx).NewInsert().Model(step).Exec(ctx)
	if err == nil {
		return step, nil
	}

	if !dberror.IsUniqueConstraintViolation(err) {
		log.Error("failed to claim an agent run step", zap.Error(err))

		return nil, fmt.Errorf("claim agent run step: %w", err)
	}

	held, findErr := r.byKey(ctx, step)
	if findErr != nil {
		// The insert says somebody holds the key and the read cannot say who.
		// Reporting the claim as taken without the row is still the safe
		// answer: the caller treats an unreadable holder as an outcome it
		// cannot account for, which is what it is.
		log.Error("a claimed step could not be read back", zap.Error(findErr))

		return nil, repositories.ErrRunStepClaimed
	}

	return held, repositories.ErrRunStepClaimed
}

func (r *repository) byKey(
	ctx context.Context,
	step *agent.AgentRunStep,
) (*agent.AgentRunStep, error) {
	cols := buncolgen.AgentRunStepColumns
	held := new(agent.AgentRunStep)

	err := r.db.DBForContext(ctx).NewSelect().
		Model(held).
		Where(cols.OrganizationID.Eq(), step.OrganizationID).
		Where(cols.OwnerID.Eq(), step.OwnerID).
		Where(cols.StepKey.Eq(), step.StepKey).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "AgentRunStep")
	}

	return held, nil
}

func (r *repository) Settle(
	ctx context.Context,
	req repositories.SettleAgentRunStepRequest,
) error {
	cols := buncolgen.AgentRunStepColumns

	outcome := req.Outcome
	if outcome == nil {
		outcome = map[string]any{}
	}

	_, err := r.db.DBForContext(ctx).NewUpdate().
		Model((*agent.AgentRunStep)(nil)).
		Set(cols.Status.Set(), req.Status).
		Set(cols.Outcome.Set(), outcome).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.OwnerID.Eq(), req.OwnerID).
		Where(cols.StepKey.Eq(), req.StepKey).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to settle an agent run step",
			zap.String("owner", req.OwnerID.String()),
			zap.String("status", req.Status),
			zap.Error(err),
		)

		return fmt.Errorf("settle agent run step: %w", err)
	}

	return nil
}

func (r *repository) Record(ctx context.Context, step *agent.AgentRunStep) error {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(step).Exec(ctx); err != nil {
		r.l.Error("failed to record an agent run step",
			zap.String("owner", step.OwnerID.String()),
			zap.String("kind", step.Kind),
			zap.Error(err),
		)

		return fmt.Errorf("record agent run step: %w", err)
	}

	return nil
}

func (r *repository) List(
	ctx context.Context,
	req repositories.ListAgentRunStepsRequest,
) ([]*agent.AgentRunStep, error) {
	cols := buncolgen.AgentRunStepColumns
	steps := make([]*agent.AgentRunStep, 0, 8)

	err := r.db.DBForContext(ctx).NewSelect().
		Model(&steps).
		Apply(buncolgen.AgentRunStepApplyTenant(req.TenantInfo)).
		Where(cols.OwnerKind.Eq(), req.OwnerKind).
		Where(cols.OwnerID.Eq(), req.OwnerID).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list agent run steps",
			zap.String("owner", req.OwnerID.String()),
			zap.Error(err),
		)

		return nil, fmt.Errorf("list agent run steps: %w", err)
	}

	return steps, nil
}

// Prune drops a batch of old steps. It is deliberately not tenant-scoped: the
// sweep runs for the whole installation, and a step's only reason to exist is
// a retry that can no longer happen.
func (r *repository) Prune(
	ctx context.Context,
	req repositories.PruneAgentRunStepsRequest,
) (int, error) {
	cols := buncolgen.AgentRunStepColumns

	dba := r.db.DBForContext(ctx)
	doomed := dba.NewSelect().
		Model((*agent.AgentRunStep)(nil)).
		Column(cols.ID.String()).
		Where(cols.CreatedAt.Lt(), req.Before).
		Limit(req.Limit)

	result, err := dba.NewDelete().
		Model((*agent.AgentRunStep)(nil)).
		Where(cols.ID.In(), doomed).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to prune agent run steps", zap.Error(err))

		return 0, fmt.Errorf("prune agent run steps: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("prune agent run steps: %w", err)
	}

	return int(affected), nil
}
