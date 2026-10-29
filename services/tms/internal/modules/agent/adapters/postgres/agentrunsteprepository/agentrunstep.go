package agentrunsteprepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunstep"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunstep/agentrunstepdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
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

func New(p Params) agentruntimeports.AgentRunStepRepository {
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
	step *agentrunstep.AgentRunStep,
) (*agentrunstep.AgentRunStep, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentrunstep.AgentRunStep, error) {
		log := r.l.With(
			zap.String("operation", "Claim"),
			zap.String("owner", step.OwnerID.String()),
			zap.String("tool", step.ToolName),
		)

		err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
			return agentrunstepdb.AgentRunSteps.Create(ctx, r.db.Keel(ctx), step)
		})
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

			return nil, agentruntimeports.ErrRunStepClaimed
		}

		return held, agentruntimeports.ErrRunStepClaimed
	})
}

func (r *repository) byKey(
	ctx context.Context,
	step *agentrunstep.AgentRunStep,
) (*agentrunstep.AgentRunStep, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentrunstep.AgentRunStep, error) {
		s := agentrunstepdb.AgentRunSteps
		held, err := s.Where(s.OwnerID.Eq(step.OwnerID), s.StepKey.Eq(step.StepKey)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "AgentRunStep")
		}

		return held, nil
	})
}

func (r *repository) Get(
	ctx context.Context,
	req *agentruntimeports.GetAgentRunStepRequest,
) (*agentrunstep.AgentRunStep, error) {
	return r.byKey(ctx, &agentrunstep.AgentRunStep{
		OrganizationID: req.TenantInfo.OrgID,
		OwnerID:        req.OwnerID,
		StepKey:        req.StepKey,
	})
}

func (r *repository) Settle(
	ctx context.Context,
	req agentruntimeports.SettleAgentRunStepRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		s := agentrunstepdb.AgentRunSteps
		outcome := req.Outcome
		if outcome == nil {
			outcome = map[string]any{}
		}

		_, err := s.Where(s.OwnerID.Eq(req.OwnerID), s.StepKey.Eq(req.StepKey)).Update(
			ctx,
			r.db.Keel(ctx),
			s.Status.To(req.Status),
			s.Outcome.To(outcome),
			s.SettledAtMs.To(timeutils.NowUnixMilli()),
			s.Version.ToExpr(s.Version),
		)
		if err != nil {
			r.l.Error("failed to settle an agent run step",
				zap.String("owner", req.OwnerID.String()),
				zap.String("status", req.Status),
				zap.Error(err),
			)

			return fmt.Errorf("settle agent run step: %w", err)
		}

		return nil
	})
}

func (r *repository) Record(ctx context.Context, step *agentrunstep.AgentRunStep) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := agentrunstepdb.AgentRunSteps.Create(ctx, r.db.Keel(ctx), step); err != nil {
			r.l.Error("failed to record an agent run step",
				zap.String("owner", step.OwnerID.String()),
				zap.String("kind", step.Kind),
				zap.Error(err),
			)

			return fmt.Errorf("record agent run step: %w", err)
		}

		return nil
	})
}

func (r *repository) List(
	ctx context.Context,
	req agentruntimeports.ListAgentRunStepsRequest,
) ([]*agentrunstep.AgentRunStep, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentrunstep.AgentRunStep, error) {
		s := agentrunstepdb.AgentRunSteps
		steps, err := s.Where(s.OwnerKind.Eq(req.OwnerKind), s.OwnerID.Eq(req.OwnerID)).
			Order(s.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list agent run steps",
				zap.String("owner", req.OwnerID.String()),
				zap.Error(err),
			)

			return nil, fmt.Errorf("list agent run steps: %w", err)
		}

		return steps, nil
	})
}

// Prune drops a batch of the old steps of the tenant on the context: a step's
// only reason to exist is a retry that can no longer happen.
func (r *repository) Prune(
	ctx context.Context,
	req agentruntimeports.PruneAgentRunStepsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		s := agentrunstepdb.AgentRunSteps
		doomed := agentrunstepdb.AgentRunSteps.As("doomed")
		n, err := s.Where(s.ID.InQuery(doomed.
			Pick(doomed.ID).
			Where(doomed.CreatedAt.Lt(req.Before)).
			Limit(req.Limit))).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to prune agent run steps", zap.Error(err))

			return 0, fmt.Errorf("prune agent run steps: %w", err)
		}

		return int(n), nil
	})
}
