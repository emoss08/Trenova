package agentruntime

import (
	"context"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// loopRefusalKeyPrefix keys a loop refusal's step by the provider's call
// id, which is unique within the turn, so a retried finish finds the step
// already there and does not write it twice.
const loopRefusalKeyPrefix = "loop-refusal:"

// RecordLoopRefusalsRequest names the turn or run whose loop refusals to
// write and the agent that made the calls.
type RecordLoopRefusalsRequest struct {
	Ledger       serviceports.RunStepLedger
	Tenant       pagination.TenantInfo
	Owner        serviceports.RunStepOwner
	DefinitionID pulid.ID
	Attempt      int
	Refusals     []serviceports.LoopRefusal
}

// RecordLoopRefusals writes each call the loop refused without dispatching
// it to the step ledger as a failed tool step, with its verdict and reason,
// the shape dispatch writes for a call a tool refused. The scorecard's
// failure breakdown reads the ledger, and before this a model that kept
// sending arguments that did not parse, or calling a tool it did not hold,
// looked like an agent with no failures at all.
//
// Each is claimed before it is settled, so a finish that is retried writes
// nothing twice. It returns how many it could not write; the caller logs
// that and carries on, since the turn itself is already saved.
func RecordLoopRefusals(ctx context.Context, req *RecordLoopRefusalsRequest) int {
	if req.Ledger == nil || len(req.Refusals) == 0 {
		return 0
	}

	failed := 0
	for idx := range req.Refusals {
		refusal := &req.Refusals[idx]
		if refusal.CallID == "" {
			continue
		}
		step := serviceports.RunStep{
			OwnerKind:    req.Owner.Kind,
			OwnerID:      req.Owner.ID,
			Attempt:      req.Attempt,
			Kind:         serviceports.RunStepTool,
			Status:       serviceports.RunStepStarted,
			Key:          loopRefusalKeyPrefix + refusal.CallID,
			ToolName:     refusal.ToolName,
			CallID:       refusal.CallID,
			DefinitionID: req.DefinitionID,
		}
		verdict, err := req.Ledger.Claim(ctx, req.Tenant, step)
		if err != nil {
			failed++

			continue
		}
		if verdict.State != serviceports.StepFresh {
			continue
		}
		step.Status = serviceports.RunStepFailed
		step.Outcome = serviceports.RunStepOutcome{
			Content: refusal.Content,
			Failed:  true,
			Reason:  refusal.Reason,
			Verdict: refusal.Verdict,
		}
		if err = req.Ledger.Settle(ctx, req.Tenant, step); err != nil {
			failed++
		}
	}

	return failed
}
