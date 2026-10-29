package agenttooltrustrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const trustEntity = "AgentToolTrust"

var trustErrors = dberror.Mapping{Entity: trustEntity}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agenttoolports.AgentToolTrustRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-tool-trust-repository"),
	}
}

// recordConflict folds an outcome into the stored row: the counters add the
// inserted deltas and the streak grows on a clean approval or resets.
func recordConflict() keel.Conflict {
	t := agentdb.ToolTrusts
	sum := func(col keel.Int64Col) keel.Assignment {
		return col.ToExpr(col.Add(keel.Int64Of(col.Excluded())))
	}

	return keel.OnConflict(t.OrganizationID, t.BusinessUnitID, t.AgentDefinitionID, t.ToolName).
		Set(
			t.Streak.ToExpr(keel.CaseInt64().
				When(keel.Int64Of(t.Streak.Excluded()).Gt(0), t.Streak.Add(keel.Int64(1))).
				Else(keel.Int64(0))),
			sum(t.Approvals),
			sum(t.Modifications),
			sum(t.Rejections),
			sum(t.ExecutionFailures),
			t.LastDecisionAt.ToExpr(t.LastDecisionAt.Excluded()),
		)
}

// Record folds one outcome into the row for the agent and tool, in a single
// upsert so two decisions landing at once both count. The inserted row
// carries the outcome as deltas: a one in the counter it belongs to, and a
// streak of one for a clean approval or zero for anything else. On conflict
// the counters add and the streak either grows or resets, which is the
// arithmetic the ledger needs and nothing a read-then-write would do better.
func (r *repository) Record(
	ctx context.Context,
	req agenttoolports.RecordToolTrustRequest,
) (*agenttooltrust.ToolTrust, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agenttooltrust.ToolTrust, error) {
		if !req.Outcome.IsValid() {
			return nil, errortypes.NewValidationError(
				"outcome",
				errortypes.ErrInvalid,
				fmt.Sprintf("%q is not an outcome the trust ledger records", req.Outcome),
			)
		}

		at := req.At
		entity := &agenttooltrust.ToolTrust{
			OrganizationID:    req.TenantInfo.OrgID,
			BusinessUnitID:    req.TenantInfo.BuID,
			AgentDefinitionID: req.AgentDefinitionID,
			ToolName:          req.ToolName,
			LastDecisionAt:    &at,
		}
		switch req.Outcome {
		case agenttooltrust.TrustOutcomeApproved:
			entity.Streak = 1
			entity.Approvals = 1
		case agenttooltrust.TrustOutcomeModified:
			entity.Modifications = 1
		case agenttooltrust.TrustOutcomeRejected:
			entity.Rejections = 1
		case agenttooltrust.TrustOutcomeExecutionFailed:
			entity.ExecutionFailures = 1
		}

		me := errortypes.NewMultiError()
		entity.Validate(me)
		if me.HasErrors() {
			return nil, me
		}

		t := agentdb.ToolTrusts
		inserted, err := t.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*agenttooltrust.ToolTrust{entity},
			recordConflict(),
		)
		if err != nil {
			r.l.Error("failed to record tool trust outcome",
				zap.String("agentDefinitionId", req.AgentDefinitionID.String()),
				zap.String("tool", req.ToolName),
				zap.Error(err),
			)

			return nil, fmt.Errorf("record tool trust: %w", err)
		}
		if len(inserted) > 0 && inserted[0] {
			return entity, nil
		}

		stored, err := t.Where(
			t.AgentDefinitionID.Eq(req.AgentDefinitionID),
			t.ToolName.Eq(req.ToolName),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, trustErrors.Map(err, req.ToolName)
		}

		return stored, nil
	})
}

func (r *repository) ListByDefinition(
	ctx context.Context,
	req agenttoolports.ListToolTrustRequest,
) ([]*agenttooltrust.ToolTrust, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agenttooltrust.ToolTrust, error) {
		t := agentdb.ToolTrusts
		rows, err := t.Where(t.AgentDefinitionID.Eq(req.AgentDefinitionID)).
			Order(t.ToolName.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list tool trust",
				zap.String("agentDefinitionId", req.AgentDefinitionID.String()),
				zap.Error(err),
			)

			return nil, fmt.Errorf("list tool trust: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) ListByDefinitionIDs(
	ctx context.Context,
	req agenttoolports.ListToolTrustByDefinitionsRequest,
) (map[pulid.ID][]*agenttooltrust.ToolTrust, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*agenttooltrust.ToolTrust, error) {
			grouped := make(map[pulid.ID][]*agenttooltrust.ToolTrust, len(req.AgentDefinitionIDs))
			if len(req.AgentDefinitionIDs) == 0 {
				return grouped, nil
			}

			t := agentdb.ToolTrusts
			rows, err := t.Where(t.AgentDefinitionID.In(req.AgentDefinitionIDs...)).
				Order(t.ToolName.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list tool trust for agents",
					zap.Int("agents", len(req.AgentDefinitionIDs)),
					zap.Error(err),
				)

				return nil, fmt.Errorf("list tool trust for agents: %w", err)
			}

			for _, row := range rows {
				grouped[row.AgentDefinitionID] = append(grouped[row.AgentDefinitionID], row)
			}

			return grouped, nil
		},
	)
}

// MarkTierChange moves the row's earned tier, conditioned on its version, so
// a decision recorded after the one that asked for the change wins the race
// and the stale one is told the row moved on.
func (r *repository) MarkTierChange(
	ctx context.Context,
	req agenttoolports.MarkToolTierChangeRequest,
) (*agenttooltrust.ToolTrust, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agenttooltrust.ToolTrust, error) {
		t := agentdb.ToolTrusts
		sets := []keel.Assignment{t.Streak.To(0), t.UpdatedAt.To(req.At)}
		if req.Promoted {
			sets = append(sets, t.EarnedTier.To(req.EarnedTier), t.PromotedAt.To(req.At))
		} else {
			sets = append(sets, t.EarnedTier.ToNull(), t.DemotedAt.To(req.At))
		}

		rows, err := t.Where(t.ID.Eq(req.ID), t.Version.Eq(req.Version)).
			UpdateReturning(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return nil, fmt.Errorf("mark tool trust tier change: %w", err)
		}
		if err = dberror.CheckAffected(int64(len(rows)), trustEntity, req.ID.String()); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}

func (r *repository) ListReady(
	ctx context.Context,
	req agenttoolports.ListReadyToolTrustRequest,
) ([]*agenttooltrust.ToolTrust, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agenttooltrust.ToolTrust, error) {
		t := agentdb.ToolTrusts
		rows, err := t.Where(t.Streak.Gte(int64(max(req.MinStreak, 1)))).
			Order(t.Streak.Desc()).
			Limit(req.Limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list tools ready for promotion: %w", err)
		}
		return rows, nil
	})
}
