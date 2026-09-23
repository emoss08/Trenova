package runstepledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/pagination"
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
	Repo    repositories.AgentRunStepRepository
	Metrics *metrics.Registry `optional:"true"`
}

type Service struct {
	l       *zap.Logger
	repo    repositories.AgentRunStepRepository
	metrics *metrics.Assistant
}

var _ serviceports.RunStepLedger = (*Service)(nil)

func New(p Params) serviceports.RunStepLedger {
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
	step serviceports.RunStep,
) (serviceports.StepVerdict, error) {
	held, err := s.repo.Claim(ctx, toEntity(tenant, step, serviceports.RunStepStarted))
	if err == nil {
		return serviceports.StepVerdict{State: serviceports.StepFresh}, nil
	}
	if !errors.Is(err, repositories.ErrRunStepClaimed) {
		return serviceports.StepVerdict{}, fmt.Errorf("claim run step: %w", err)
	}

	// The key is taken and the holder could not be read. Unknown is the only
	// safe answer: something ran, and this attempt cannot say what it did.
	if held == nil {
		s.countReplayed(step, serviceports.StepUnknown)

		return serviceports.StepVerdict{State: serviceports.StepUnknown}, nil
	}

	switch serviceports.RunStepStatus(held.Status) {
	case serviceports.RunStepCompleted:
		s.countReplayed(step, serviceports.StepCompleted)

		return serviceports.StepVerdict{
			State:   serviceports.StepCompleted,
			Outcome: decodeOutcome(s.l, held),
		}, nil
	case serviceports.RunStepFailed:
		s.countReplayed(step, serviceports.StepFailed)

		return serviceports.StepVerdict{
			State:   serviceports.StepFailed,
			Outcome: decodeOutcome(s.l, held),
		}, nil
	default:
		s.l.Warn("an earlier attempt began this step and never finished it",
			zap.String("owner", held.OwnerID.String()),
			zap.String("tool", held.ToolName),
			zap.Int("claimedOnAttempt", held.Attempt),
		)

		s.countReplayed(step, serviceports.StepUnknown)

		return serviceports.StepVerdict{State: serviceports.StepUnknown}, nil
	}
}

// countReplayed files an operation a later attempt declined to repeat.
//
// This is the reliability figure worth having: each one is a write that a
// retry would otherwise have made twice, so the count is a direct measure of
// what the ledger is preventing rather than a proxy for it.
func (s *Service) countReplayed(step serviceports.RunStep, state serviceports.StepState) {
	s.metrics.RecordStepReplayed(string(step.OwnerKind), string(state))
}

func (s *Service) Settle(
	ctx context.Context,
	tenant pagination.TenantInfo,
	step serviceports.RunStep,
) error {
	if len(step.Outcome.Content) > maxStoredOutcomeChars {
		s.l.Debug("a step's answer was too large to keep",
			zap.String("tool", step.ToolName),
			zap.Int("chars", len(step.Outcome.Content)),
		)
		step.Outcome.Content = ""
	}

	encoded, err := encodeOutcome(step.Outcome)
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

	return s.repo.Settle(ctx, repositories.SettleAgentRunStepRequest{
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
	step serviceports.RunStep,
) error {
	return s.repo.Record(ctx, toEntity(tenant, step, step.Status))
}

func (s *Service) Loaded(
	ctx context.Context,
	tenant pagination.TenantInfo,
	owner serviceports.RunStepOwner,
) ([]serviceports.RunStep, error) {
	rows, err := s.repo.List(ctx, repositories.ListAgentRunStepsRequest{
		TenantInfo: tenant,
		OwnerKind:  string(owner.Kind),
		OwnerID:    owner.ID,
	})
	if err != nil {
		return nil, err
	}

	steps := make([]serviceports.RunStep, 0, len(rows))
	for _, row := range rows {
		steps = append(steps, serviceports.RunStep{
			OwnerKind: serviceports.RunStepOwnerKind(row.OwnerKind),
			OwnerID:   row.OwnerID,
			Attempt:   row.Attempt,
			Kind:      serviceports.RunStepKind(row.Kind),
			Status:    serviceports.RunStepStatus(row.Status),
			Key:       row.StepKey,
			ToolName:  row.ToolName,
			CallID:    row.CallID,
			Args:      row.Arguments,
			Outcome:   decodeOutcome(s.l, row),
		})
	}

	return steps, nil
}

func toEntity(
	tenant pagination.TenantInfo,
	step serviceports.RunStep,
	status serviceports.RunStepStatus,
) *agent.AgentRunStep {
	args := step.Args
	if args == nil {
		args = map[string]any{}
	}

	attempt := step.Attempt
	if attempt < 1 {
		attempt = 1
	}

	return &agent.AgentRunStep{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		OwnerKind:      string(step.OwnerKind),
		OwnerID:        step.OwnerID,
		Attempt:        attempt,
		Kind:           string(step.Kind),
		Status:         string(status),
		StepKey:        step.Key,
		ToolName:       step.ToolName,
		CallID:         step.CallID,
		Arguments:      args,
		Outcome:        map[string]any{},
	}
}

// encodeOutcome and decodeOutcome move the outcome through the jsonb column.
// It goes in as a map rather than raw bytes so an operator reading the table
// can see what a tool answered without decoding anything.
func encodeOutcome(outcome serviceports.RunStepOutcome) (map[string]any, error) {
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

func decodeOutcome(logger *zap.Logger, row *agent.AgentRunStep) serviceports.RunStepOutcome {
	if len(row.Outcome) == 0 {
		return serviceports.RunStepOutcome{}
	}

	encoded, err := sonic.Marshal(row.Outcome)
	if err == nil {
		var outcome serviceports.RunStepOutcome
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

	return serviceports.RunStepOutcome{}
}
