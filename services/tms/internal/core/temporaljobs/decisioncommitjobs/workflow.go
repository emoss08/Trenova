// Package decisioncommitjobs holds an approval for its undo window and then
// carries it out.
//
// An approval made from a person's own conversation is recorded at once and
// committed a few seconds later, unless they undo it first. The wait lives in
// a workflow rather than in the request or the browser: it survives a closed
// tab and a restarted server, and the commit happens whether or not anybody is
// still watching.
package decisioncommitjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ProposalCommitWorkflowName = "ProposalCommitWorkflow"
	// UndoSignalName takes the approval back; the workflow ends without
	// committing. CommitNowSignalName ends the wait early.
	UndoSignalName      = "decision-undo"
	CommitNowSignalName = "decision-commit-now"
)

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
	due := now.Add(services.ApprovalUndoWindow)
	if due.Nanosecond() > 0 {
		return due.Unix() + 1
	}

	return due.Unix()
}

// CommitResult says how the window closed.
type CommitResult struct {
	Committed bool `json:"committed"`
	Undone    bool `json:"undone"`
}

var commitOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    5,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        ProposalCommitWorkflowName,
			Fn:          ProposalCommitWorkflow,
			TaskQueue:   TaskQueue,
			Description: "Hold an approval for its undo window, then carry it out",
		},
	}
}

// ProposalCommitWorkflow waits until the approval's window closes, or until
// the person asks for it to go through now, and then commits it. An undo
// signal ends it without committing.
//
// The undo itself is written by the request that asked for it, before the
// signal is sent, and the commit claims only what is still in its window. So
// the database decides a race between the two, and a lost signal costs a
// commit that finds nothing left to do, never a commit of something undone.
func ProposalCommitWorkflow(
	ctx workflow.Context,
	req *services.CommitApprovalRequest,
) (*CommitResult, error) {
	wait := time.Unix(req.CommitsAt, 0).Sub(workflow.Now(ctx))
	if wait < 0 {
		wait = 0
	}

	timerCtx, cancelTimer := workflow.WithCancel(ctx)
	timer := workflow.NewTimer(timerCtx, wait)
	undo := workflow.GetSignalChannel(ctx, UndoSignalName)
	now := workflow.GetSignalChannel(ctx, CommitNowSignalName)

	undone := false
	var signal struct{}
	sel := workflow.NewSelector(ctx)
	sel.AddFuture(timer, func(workflow.Future) {})
	sel.AddReceive(undo, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, &signal)
		undone = true
	})
	sel.AddReceive(now, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, &signal)
	})
	sel.Select(ctx)
	cancelTimer()

	if undone {
		return &CommitResult{Undone: true}, nil
	}

	var a *Activities
	commitCtx := workflow.WithActivityOptions(ctx, commitOptions)
	if err := workflow.ExecuteActivity(commitCtx, a.CommitApprovalActivity, req).
		Get(commitCtx, nil); err != nil {
		return nil, err
	}

	return &CommitResult{Committed: true}, nil
}
