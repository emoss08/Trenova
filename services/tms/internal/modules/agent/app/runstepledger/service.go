package runstepledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aimetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunstep"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// maxStoredOutcomeChars bounds what one step keeps.
//
// A write's answer is a sentence and never comes near this. A query tool's is
// whatever it returned — a report's rows, a list of every unbilled shipment —
// and storing those verbatim on every step would put report-sized blobs in
// this table for no gain. Past the bound the answer is dropped rather than
// truncated, because half a fenced JSON document handed back to a model is
// worse than none: the replay then tells it the answer was not kept, and a
// read is free to make again.
const maxStoredOutcomeChars = 64 * 1024

type Params struct {
	fx.In

	Logger  *zap.Logger
	Repo    agentruntimeports.AgentRunStepRepository
	Metrics *metrics.Registry `optional:"true"`
}

type Service struct {
	l       *zap.Logger
	repo    agentruntimeports.AgentRunStepRepository
	metrics *aimetrics.Assistant
}

var _ agentruntimeports.RunStepLedger = (*Service)(nil)

func New(p Params) agentruntimeports.RunStepLedger {
	service := &Service{
		l:    p.Logger.Named("service.runstepledger"),
		repo: p.Repo,
	}
	if p.Metrics != nil {
		service.metrics = p.Metrics.Assistant
	}

	return service
}

// Claim reserves a step key before the work happens.
//
// A key nobody holds comes back Fresh and the caller goes ahead. A key an
// earlier attempt settled comes back with that attempt's answer, so the model
// is handed the same result it was handed the first time and the tool does not
// run twice. A key an earlier attempt claimed and never settled comes back
// Unknown, which is the honest reading: the tool was started and nothing
// recorded how it ended.
func (s *Service) Claim(
	ctx context.Context,
	tenant pagination.TenantInfo,
	step *agentruntimeports.RunStep,
) (agentruntimeports.StepVerdict, error) {
	held, err := s.repo.Claim(ctx, toEntity(tenant, step, agentruntimeports.RunStepStarted))
	if err == nil {
		return agentruntimeports.StepVerdict{State: agentruntimeports.StepFresh}, nil
	}
	if !errors.Is(err, agentruntimeports.ErrRunStepClaimed) {
		return agentruntimeports.StepVerdict{}, fmt.Errorf("claim run step: %w", err)
	}

	// The key is taken and the holder could not be read. Unknown is the only
	// safe answer: something ran, and this attempt cannot say what it did.
	if held == nil {
		s.countReplayed(step, agentruntimeports.StepUnknown)

		return agentruntimeports.StepVerdict{State: agentruntimeports.StepUnknown}, nil
	}

	switch agentruntimeports.RunStepStatus(held.Status) {
	case agentruntimeports.RunStepCompleted:
		s.countReplayed(step, agentruntimeports.StepCompleted)

		return agentruntimeports.StepVerdict{
			State:   agentruntimeports.StepCompleted,
			Outcome: decodeOutcome(s.l, held),
		}, nil
	case agentruntimeports.RunStepFailed:
		s.countReplayed(step, agentruntimeports.StepFailed)

		return agentruntimeports.StepVerdict{
			State:   agentruntimeports.StepFailed,
			Outcome: decodeOutcome(s.l, held),
		}, nil
	case agentruntimeports.RunStepStarted:
		return s.unfinished(step, held), nil
	default:
		return s.unfinished(step, held), nil
	}
}

func (s *Service) unfinished(
	step *agentruntimeports.RunStep,
	held *agentrunstep.AgentRunStep,
) agentruntimeports.StepVerdict {
	s.l.Warn("an earlier attempt began this step and never finished it",
		zap.String("owner", held.OwnerID.String()),
		zap.String("tool", held.ToolName),
		zap.Int("claimedOnAttempt", held.Attempt),
	)

	s.countReplayed(step, agentruntimeports.StepUnknown)

	return agentruntimeports.StepVerdict{State: agentruntimeports.StepUnknown}
}

// countReplayed files an operation a later attempt declined to repeat.
//
// This is the reliability figure worth having: each one is a write that a
// retry would otherwise have made twice, so the count is a direct measure of
// what the ledger is preventing rather than a proxy for it.
func (s *Service) countReplayed(
	step *agentruntimeports.RunStep,
	state agentruntimeports.StepState,
) {
	s.metrics.RecordStepReplayed(string(step.OwnerKind), string(state))
}

func (s *Service) Settle(
	ctx context.Context,
	tenant pagination.TenantInfo,
	step *agentruntimeports.RunStep,
) error {
	outcome := withCause(step.Kind, &step.Outcome)
	if len(outcome.Content) > maxStoredOutcomeChars {
		s.l.Debug("a step's answer was too large to keep",
			zap.String("tool", step.ToolName),
			zap.Int("chars", len(outcome.Content)),
		)
		outcome.Content = ""
	}

	encoded, err := encodeOutcome(&outcome)
	if err != nil {
		// A settled step whose outcome cannot be encoded would come back as
		// Unknown on the next attempt, which is wrong — it finished, and we
		// know how. Record the status without the body rather than leave it
		// Started.
		s.l.Error("could not encode a run step outcome; recording the status alone",
			zap.String("tool", step.ToolName),
			zap.Error(err),
		)
		encoded = map[string]any{}
	}

	return s.repo.Settle(ctx, &agentruntimeports.SettleAgentRunStepRequest{
		TenantInfo: tenant,
		OwnerID:    step.OwnerID,
		StepKey:    step.Key,
		Status:     string(step.Status),
		Outcome:    encoded,
	})
}

func (s *Service) Record(
	ctx context.Context,
	tenant pagination.TenantInfo,
	step *agentruntimeports.RunStep,
) error {
	return s.repo.Record(ctx, toEntity(tenant, step, step.Status))
}

func (s *Service) Loaded(
	ctx context.Context,
	tenant pagination.TenantInfo,
	owner agentruntimeports.RunStepOwner,
) ([]agentruntimeports.RunStep, error) {
	rows, err := s.repo.List(ctx, &agentruntimeports.ListAgentRunStepsRequest{
		TenantInfo: tenant,
		OwnerKind:  string(owner.Kind),
		OwnerID:    owner.ID,
	})
	if err != nil {
		return nil, err
	}

	steps := make([]agentruntimeports.RunStep, 0, len(rows))
	for _, row := range rows {
		steps = append(steps, agentruntimeports.RunStep{
			OwnerKind:         kernelports.RunStepOwnerKind(row.OwnerKind),
			OwnerID:           row.OwnerID,
			Attempt:           row.Attempt,
			Kind:              agentruntimeports.RunStepKind(row.Kind),
			Status:            agentruntimeports.RunStepStatus(row.Status),
			Key:               row.StepKey,
			ToolName:          row.ToolName,
			CallID:            row.CallID,
			Args:              row.Arguments,
			Outcome:           decodeOutcome(s.l, row),
			TraceID:           row.TraceID,
			SpanID:            row.SpanID,
			DefinitionID:      row.AgentDefinitionID,
			DefinitionVersion: intutils.ClonePointer(row.AgentDefinitionVersion),
			DelegateCallID:    row.DelegateCallID,
		})
	}

	return steps, nil
}

func toEntity(
	tenant pagination.TenantInfo,
	step *agentruntimeports.RunStep,
	status agentruntimeports.RunStepStatus,
) *agentrunstep.AgentRunStep {
	args := step.Args
	if args == nil {
		args = map[string]any{}
	}

	attempt := step.Attempt
	if attempt < 1 {
		attempt = 1
	}

	return &agentrunstep.AgentRunStep{
		OrganizationID:         tenant.OrgID,
		BusinessUnitID:         tenant.BuID,
		OwnerKind:              string(step.OwnerKind),
		OwnerID:                step.OwnerID,
		Attempt:                attempt,
		Kind:                   string(step.Kind),
		Status:                 string(status),
		StepKey:                step.Key,
		ToolName:               step.ToolName,
		CallID:                 step.CallID,
		Arguments:              args,
		Outcome:                map[string]any{},
		TraceID:                step.TraceID,
		SpanID:                 step.SpanID,
		AgentDefinitionID:      step.DefinitionID,
		AgentDefinitionVersion: intutils.ClonePointer(step.DefinitionVersion),
		DelegateCallID:         step.DelegateCallID,
	}
}

// encodeOutcome and decodeOutcome move the outcome through the jsonb column.
// It goes in as a map rather than raw bytes so an operator reading the table
// can see what a tool answered without decoding anything.
func withCause(
	kind agentruntimeports.RunStepKind,
	outcome *agentruntimeports.RunStepOutcome,
) agentruntimeports.RunStepOutcome {
	caused := *outcome
	if kind != agentruntimeports.RunStepTool || !caused.Failed || caused.Cause != "" {
		return caused
	}
	caused.Cause = aianalytics.ClassifyToolFailure(
		caused.Verdict,
		caused.Reason,
		caused.Content,
	)

	return caused
}

func encodeOutcome(outcome *agentruntimeports.RunStepOutcome) (map[string]any, error) {
	encoded, err := sonic.Marshal(outcome)
	if err != nil {
		return nil, fmt.Errorf("encode outcome: %w", err)
	}

	var decoded map[string]any
	if err = sonic.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("encode outcome: %w", err)
	}

	return decoded, nil
}

func decodeOutcome(
	logger *zap.Logger,
	row *agentrunstep.AgentRunStep,
) agentruntimeports.RunStepOutcome {
	if len(row.Outcome) == 0 {
		return agentruntimeports.RunStepOutcome{}
	}

	encoded, err := sonic.Marshal(row.Outcome)
	if err == nil {
		var outcome agentruntimeports.RunStepOutcome
		if err = sonic.Unmarshal(encoded, &outcome); err == nil {
			return outcome
		}
	}

	// A stored outcome that cannot be read back is reported as an empty one.
	// The caller sees a replayed step with no content and tells the model the
	// tool ran without a recorded answer, which beats running it again.
	logger.Error("a recorded run step outcome could not be read back",
		zap.String("owner", row.OwnerID.String()),
		zap.String("tool", row.ToolName),
		zap.Error(err),
	)

	return agentruntimeports.RunStepOutcome{}
}
