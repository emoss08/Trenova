package deskbench

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
)

// unlearned stands in for the reflection scheduler on the bench's worker. A
// conversation goes quiet after ten minutes, so in a run longer than that the
// lessons an early case taught reached the cases after it, and a case's result
// came to depend on what the bench had run before.
type unlearned struct{}

func (unlearned) AfterTurn(context.Context, *agentports.ReflectOnThreadRequest) {}

func (unlearned) AfterRun(context.Context, *agentports.ReflectOnRunRequest) {}

// unchaptered keeps chapter titling out of the bench: it is a model call per
// question that the bench neither scores nor should pay for, and its calls
// would land in the recorder beside the turns being measured.
type unchaptered struct{}

func (unchaptered) Cue(context.Context, *assistantports.ChapterTurnRequest) {}

// untrusted keeps the bench's decisions out of the earned-autonomy ledger.
// The bench approves the same proposals run after run and rejects whatever a
// case leaves waiting, so its streaks promoted billing_transfer to
// AutoExecute on Billing exceptions and the case that approves its proposal
// found nothing to approve; the rejections would have taken tiers back the
// same way.
type untrusted struct {
	agentports.AgentTrustService
}

func (untrusted) RecordDecision(
	context.Context,
	*agent.AgentProposal,
	*agentdecision.AgentDecision,
) error {
	return nil
}

func (untrusted) RecordExecutionFailure(context.Context, *agent.AgentProposal) error {
	return nil
}
