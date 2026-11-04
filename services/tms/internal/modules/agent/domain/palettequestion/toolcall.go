package palettequestion

import (
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

// ToolCallIDPrefix names one read Nova made while answering a question.
const ToolCallIDPrefix = "patc_"

// ToolCallOutcome is how a read made for an answer turned out.
type ToolCallOutcome string

const (
	// ToolCallRan is a read that returned what it was asked for.
	ToolCallRan = ToolCallOutcome("Ran")
	// ToolCallRefused is a read refused before it ran: the person may not read
	// the kind, the arguments did not fit the read, or the id names no kind the
	// palette opens.
	ToolCallRefused = ToolCallOutcome("Refused")
	// ToolCallFailed is a read that ran and failed.
	ToolCallFailed = ToolCallOutcome("Failed")
)

func AllToolCallOutcomes() []ToolCallOutcome {
	return []ToolCallOutcome{ToolCallRan, ToolCallRefused, ToolCallFailed}
}

func (o ToolCallOutcome) IsValid() bool { return slices.Contains(AllToolCallOutcomes(), o) }

// ToolCall is one read Nova made while answering a palette question: the
// palette tool the model called, the Desk read it went to and what that read
// was asked, how it ended and why, and how long it took. It keeps what is
// needed to tell why an answer came out as it did, never what the read
// returned: the result stays in the answer's stream. The AI audit trail reads
// these rows (with the arguments redacted by the read's own policy) as the
// question's tool calls.
type ToolCall struct {
	keel.Table `keel:"assistant_palette_tool_calls"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:patc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	// QuestionID is the question the read was made for.
	QuestionID pulid.ID `json:"questionId"  keel:"question_id"`
	// UserID is the person who asked, whom every read is made as.
	UserID pulid.ID `json:"userId"      keel:"user_id"`
	// CallID is the model's id for the call, as the answer's stream names it.
	CallID string `json:"callId"      keel:"call_id"`
	// Sequence is the call's place among the question's reads, from 1.
	Sequence int `json:"sequence"    keel:"sequence"`
	// Model is the model that asked for the read.
	Model string `json:"model"       keel:"model,nullzero"`
	// Tool is the palette tool the model called: search_records, get_record
	// or list_records.
	Tool string `json:"tool"        keel:"tool"`
	// Arguments is what the model sent the palette tool.
	Arguments map[string]any `json:"arguments"   keel:"arguments,jsonb"`
	// Read is the Desk read the call went to (customer_get, invoice_list);
	// empty for a search, or for a call refused before a read was chosen.
	Read string `json:"read"        keel:"read,nullzero"`
	// Params is what the read was asked, after the palette fitted the
	// arguments to it.
	Params map[string]any `json:"params"      keel:"params,jsonb"`
	// Outcome is how the read ended.
	Outcome ToolCallOutcome `json:"outcome"     keel:"outcome"`
	// Reason is why a read was refused or failed, as the model was told.
	Reason string `json:"reason"      keel:"reason,nullzero"`
	// Summary is what a read that ran returned, in a few words ("16 records").
	Summary string `json:"summary"     keel:"summary,nullzero"`
	// Records is how many records the result named.
	Records int `json:"records"     keel:"records"`
	// StartedAtMs is when the read began, in Unix milliseconds.
	StartedAtMs int64 `json:"startedAtMs" keel:"started_at_ms"`
	// DurationMs is how long the read took.
	DurationMs int64 `json:"durationMs"  keel:"duration_ms"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
}
