package deskbench

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// unlearned stands in for the reflection scheduler on the bench's worker. A
// conversation goes quiet after ten minutes, so in a run longer than that the
// lessons an early case taught reached the cases after it, and a case's result
// came to depend on what the bench had run before.
type unlearned struct{}

func (unlearned) AfterTurn(context.Context, *serviceports.ReflectOnThreadRequest) {}

func (unlearned) AfterRun(context.Context, *serviceports.ReflectOnRunRequest) {}

// untrusted keeps the bench's decisions out of the earned-autonomy ledger.
// The bench approves the same proposals run after run and rejects whatever a
// case leaves waiting, so its streaks promoted transfer_to_billing to
// AutoExecute on Billing exceptions and the case that approves its proposal
// found nothing to approve; the rejections would have taken tiers back the
// same way.
type untrusted struct {
	serviceports.AgentTrustService
}

func (untrusted) RecordDecision(context.Context, *agent.AgentProposal, *agent.AgentDecision) error {
	return nil
}

func (untrusted) RecordExecutionFailure(context.Context, *agent.AgentProposal) error {
	return nil
}
