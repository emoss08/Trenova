package assistantservice

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// DryRunRequest is a draft agent and one message to try it with.
type DryRunRequest struct {
	// RunID names the dry run. It carries the evaluation prefix, so what the
	// run costs is counted as an evaluation and nothing it proposes is kept.
	RunID      pulid.ID
	Definition *agentdefinition.Definition
	Prompt     string
}

// DryRunPlan is a dry run made ready to drive: the draft as it runs, with
// simulation forced on, and the opened turn.
type DryRunPlan struct {
	RunID      pulid.ID                    `json:"runId"`
	Definition *agentdefinition.Definition `json:"definition"`
	Input      string                      `json:"input"`
	Decision   agentguard.Decision         `json:"decision"`
	Timezone   string                      `json:"timezone,omitempty"`
	// Shadow is the draft's own mode before simulation was forced on, so its
	// writes are named as a shadow agent's would be.
	Shadow bool                   `json:"shadow"`
	Turn   agentruntime.TurnState `json:"turn"`
}

func (p *DryRunPlan) Refused() bool { return !p.Decision.Allowed }

func (p *DryRunPlan) Opening() kernelports.StreamEvent {
	return openingEvent(p.Input, p.Decision)
}

// RunRequest is the plan as the runtime reads it. No thread, no step ledger
// and no publishing: a dry run leaves nothing behind but what it cost.
func (p *DryRunPlan) RunRequest(actor *permissionports.RequestActor) *agentports.RunRequest {
	return &agentports.RunRequest{
		Definition:   p.Definition,
		Actor:        actor,
		Context:      agentdefinition.RuntimeContext{Timezone: p.Timezone},
		Input:        p.Input,
		RunID:        p.RunID,
		UsagePurpose: aiusageports.AIUsagePurposeEvaluation,
	}
}

// PrepareDryRun checks the message, runs the scope guard over it and opens
// the turn a draft agent would answer it with, against live data and with
// every write simulated.
func (s *Service) PrepareDryRun(
	ctx context.Context,
	req *DryRunRequest,
	actor *permissionports.RequestActor,
) (*DryRunPlan, error) {
	prompt := strings.TrimSpace(req.Prompt)
	multiErr := errortypes.NewMultiError()
	switch {
	case prompt == "":
		multiErr.Add("prompt", errortypes.ErrRequired, "Enter a message to try the agent with")
	case utf8.RuneCountInString(prompt) > agentdefinition.MaxTestPromptRunes:
		multiErr.Add("prompt", errortypes.ErrInvalidLength, "A message cannot be longer than 2000 characters")
	}
	if req.Definition == nil {
		multiErr.Add("definition", errortypes.ErrRequired, "A dry run needs the agent it tries")
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	// The draft runs as a copy with simulation on, whatever mode it is in,
	// so nothing it would write is written.
	draft := *req.Definition
	draft.SimulationMode = true

	decision, runReq := s.admit(ctx, &TurnRequest{
		Definition: &draft,
		Actor:      actor,
		Input:      prompt,
	})
	plan := &DryRunPlan{
		RunID:      req.RunID,
		Definition: &draft,
		Input:      prompt,
		Decision:   decision,
		Shadow:     req.Definition.ShadowMode,
	}
	if runReq == nil {
		return plan, nil
	}

	runReq.RunID = req.RunID
	runReq.UsagePurpose = aiusageports.AIUsagePurposeEvaluation
	plan.Timezone = runReq.Context.Timezone
	plan.Turn = s.runtime.OpenTurn(ctx, runReq).State()

	return plan, nil
}
