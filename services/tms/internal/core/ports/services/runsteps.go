package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// RunStepOwnerKind says which ledger a step belongs to. One table serves both
// because the rule is the same for both; a column cannot carry two foreign
// keys, so the rows are swept on a retention schedule rather than cascaded.
type RunStepOwnerKind string

const (
	RunStepOwnerAgentRun      RunStepOwnerKind = "AgentRun"
	RunStepOwnerAssistantTurn RunStepOwnerKind = "AssistantTurn"
)

// RunStepKind distinguishes a model reply from a tool call.
type RunStepKind string

const (
	// RunStepCompletion is one model reply. It is recorded but never replayed:
	// it is what makes retry waste measurable, and the precondition for
	// replaying a reply later when the retry lands on the same provider.
	RunStepCompletion RunStepKind = "Completion"
	// RunStepTool is one tool call. This is the one that matters — it is what
	// stands between a retried run and a second tender.
	RunStepTool RunStepKind = "Tool"
)

// RunStepStatus is how far a step got.
type RunStepStatus string

const (
	// RunStepStarted is claimed but not yet settled. A step left in this state
	// is a tool that began and whose outcome was never written down.
	RunStepStarted   RunStepStatus = "Started"
	RunStepCompleted RunStepStatus = "Completed"
	RunStepFailed    RunStepStatus = "Failed"
)

// RunStep is one thing a run did, or began to do.
type RunStep struct {
	OwnerKind RunStepOwnerKind `json:"ownerKind"`
	OwnerID   pulid.ID         `json:"ownerId"`
	// Attempt is which try of the run recorded this, for reading the ledger
	// back when something went wrong.
	Attempt int           `json:"attempt"`
	Kind    RunStepKind   `json:"kind"`
	Status  RunStepStatus `json:"status"`
	// Key identifies the operation across attempts. See agentruntime.StepKey.
	Key      string         `json:"key"`
	ToolName string         `json:"toolName"`
	CallID   string         `json:"callId"`
	Args     map[string]any `json:"args"`
	Outcome  RunStepOutcome `json:"outcome"`
}

// RunStepOutcome is what a settled step produced, in the shape the runtime
// needs to hand the model the same answer twice.
type RunStepOutcome struct {
	// Content is what the model was told.
	Content string `json:"content"`
	Failed  bool   `json:"failed"`
	// Action is the write the call proposed or made. It is stored whole, and
	// replayed whole: a proposal's pinned Target must come back as it was
	// taken, because re-pinning it would record the record as it is now and
	// defeat the staleness check the pin exists for.
	Action *PendingAction `json:"action,omitempty"`
	// Data is what a query tool returned, in its JSON form, kept for a run
	// whose results are shown beside it. A replayed step has no raw result to
	// build its artifacts from, and without this the pane lost what the
	// original call had shown.
	Data map[string]any `json:"data,omitempty"`
}

// StepState is what a claim found.
type StepState string

const (
	// StepFresh means nobody has run this. Go ahead.
	StepFresh StepState = "Fresh"
	// StepCompleted and StepFailed mean an earlier attempt ran it and recorded
	// how it went. Hand back the answer; do not run it again.
	StepCompleted StepState = "Completed"
	StepFailed    StepState = "Failed"
	// StepUnknown means an earlier attempt began it and never recorded an
	// outcome. Something may have changed. This is the honest answer, and the
	// model is told it rather than the call being quietly repeated.
	StepUnknown StepState = "Unknown"
)

// StepVerdict is a claim's answer.
type StepVerdict struct {
	State   StepState
	Outcome RunStepOutcome
}

// Replayed reports a verdict that carries an earlier attempt's answer.
func (v StepVerdict) Replayed() bool {
	return v.State == StepCompleted || v.State == StepFailed
}

// RunStepLedger records what a run has done, so that a run started over does
// not do it again.
//
// An agent run is one activity the length of a whole conversation with a
// model, and an activity that fails is retried from its start. Without this,
// the retry asks the model again and runs every write again — including the
// ones that had already succeeded. Nothing else prevents that: a tool's
// RequiresIdempotencyKey is checked for presence and, with two exceptions that
// forward it to an email provider, never looked up.
//
// The claim comes before the work, not after. Recording afterwards would be
// simpler and would make a crash between the write and the record replay the
// write; claiming first turns that same crash into a step nobody can account
// for, which the run reports rather than repeats. For a system where the
// duplicate is a second shipment or a second tender, being told "this was
// begun and I cannot confirm it finished" is worth more than a silent second
// attempt.
type RunStepLedger interface {
	// Claim reserves a key before the work happens and reports what is already
	// known about it.
	Claim(ctx context.Context, tenant pagination.TenantInfo, step RunStep) (StepVerdict, error)
	// Settle records how a claimed step turned out.
	Settle(ctx context.Context, tenant pagination.TenantInfo, step RunStep) error
	// Record files a step that needs no claim, such as a model reply.
	Record(ctx context.Context, tenant pagination.TenantInfo, step RunStep) error
	// Loaded is every step already on record for this run, read once when the
	// run starts. The repeat guard is seeded from it, so a resumed attempt
	// does not re-learn a failure the first attempt already had.
	Loaded(ctx context.Context, tenant pagination.TenantInfo, owner RunStepOwner) ([]RunStep, error)
}

// RunStepOwner names whose steps to read.
type RunStepOwner struct {
	Kind RunStepOwnerKind
	ID   pulid.ID
}
