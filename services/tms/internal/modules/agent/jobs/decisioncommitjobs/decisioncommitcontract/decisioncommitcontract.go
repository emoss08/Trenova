package decisioncommitcontract

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentapprovalports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const ProposalCommitWorkflowName = "ProposalCommitWorkflow"

// UndoSignalName takes the approval back; the workflow ends without
// committing. CommitNowSignalName ends the wait early.
const UndoSignalName = "decision-undo"

const CommitNowSignalName = "decision-commit-now"

// TaskQueue is the agent's cheap periodic queue: a commit is one short
// activity, and the decisions it runs start no model call of their own.
var TaskQueue = temporaltype.TaskQueueAgent.String()

// WorkflowID names the commit of one approval: a proposal's, a batch's by its
// first proposal, or a plan's.
func WorkflowID(key pulid.ID) string {
	return "proposal-commit/" + key.String()
}

// CommitsAt is when an approval made at now goes through: the undo window,
// rounded up to a whole second. Rounding up, never down, means the countdown
// a client draws from it starts at the full window and the person always has
// at least that long.
func CommitsAt(now time.Time) int64 {
	due := now.Add(agentapprovalports.ApprovalUndoWindow)
	if due.Nanosecond() > 0 {
		return due.Unix() + 1
	}

	return due.Unix()
}
