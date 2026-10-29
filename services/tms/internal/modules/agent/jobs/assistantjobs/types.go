// Package assistantjobs answers an assistant turn as a workflow.
//
// A chat turn used to occupy a Gin goroutine for its whole life, which made
// the reply a property of the connection carrying it: a deploy ended every
// conversation in flight. The turn is now a workflow of its own, answered in
// three steps:
//
//  1. Prepare reads what the question needs, checks it may be answered and
//     runs the scope guard.
//  2. The agent loop runs in workflow code (agentflow), one activity per
//     model call and per tool call, so a failure retries the step that
//     failed and a lost worker's turn resumes where it was.
//  3. Finish saves what the turn came to and closes its record.
//
// The reader follows the turn on a Workflow Stream the workflow hosts. The
// stream exists as soon as the workflow does, which is before the request
// that started it returns, so a reader can never arrive ahead of it.
package assistantjobs

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/assistantjobs/assistantcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall/modelcallcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantturnports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

// FinishTurnInput is everything the turn did, for saving.
type FinishTurnInput struct {
	Payload *assistantcontract.AssistantTurnPayload `json:"payload"`
	// Plan is nil when the question was turned away before it was planned:
	// then there is nothing to save, only the turn's record to close.
	Plan *assistantservice.TurnPlan `json:"plan,omitempty"`
	// Rejection is why the question was turned away, written for the person
	// who asked.
	Rejection string `json:"rejection,omitempty"`
	// RejectionParams are the figures the refusal carried, such as which
	// usage cap and when it lifts.
	RejectionParams map[string]string             `json:"rejectionParams,omitempty"`
	Run             *agentruntimeports.RunResult  `json:"run,omitempty"`
	Failure         *modelcallcontract.Failure    `json:"failure,omitempty"`
	Artifacts       []*assistantartifact.Artifact `json:"artifacts,omitempty"`
	Events          []temporaltype.StreamItem     `json:"events,omitempty"`
	// Steered are the queued messages the turn read while it worked, which
	// are cleared from the queue once the turn is saved.
	Steered []pulid.ID `json:"steered,omitempty"`
}

// TurnEnding is how the turn ended: its result, and the last event its reader
// is sent.
type TurnEnding struct {
	Result assistantcontract.AssistantTurnResult `json:"result"`
	Event  temporaltype.StreamItem               `json:"event"`
	// Compact is set when the turn left the conversation full enough to
	// compact itself, with the context use either side of doing so.
	Compact *CompactionCue `json:"compact,omitempty"`
	// Next is the turn the conversation's queue started once this one was
	// saved.
	Next *assistantturnports.QueuedTurn `json:"next,omitempty"`
}

// CompactionCue is a conversation a turn left full enough to compact itself.
type CompactionCue struct {
	Before int `json:"before"`
	After  int `json:"after"`
}

// NotifyUnseenTurnInput is a turn that ended with nobody reading it, for
// telling the person who asked.
type NotifyUnseenTurnInput struct {
	Payload *assistantcontract.AssistantTurnPayload `json:"payload"`
	// Status is how the turn ended.
	Status conversation.AssistantTurnStatus `json:"status"`
}
