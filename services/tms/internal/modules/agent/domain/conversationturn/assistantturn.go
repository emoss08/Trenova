package conversationturn

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentfingerprint"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*AssistantTurn)(nil)
)

// AssistantTurnOrigin is what started a turn: a person asking, or the
// application following up a decision on one of the conversation's proposals.
type AssistantTurnOrigin string

const (
	AssistantTurnOriginPerson = AssistantTurnOrigin("Person")
	// AssistantTurnOriginDecisionFollowUp is the turn in which the agent says
	// what came of a proposal somebody decided, wherever they decided it.
	AssistantTurnOriginDecisionFollowUp = AssistantTurnOrigin("DecisionFollowUp")
	// AssistantTurnOriginScheduled is a run of a request the person scheduled
	// in the conversation, asked on their behalf when its time came.
	AssistantTurnOriginScheduled = AssistantTurnOrigin("Scheduled")
	// AssistantTurnOriginCompaction is the conversation being summarized to
	// free its context. It holds the conversation's one live slot like any
	// reply, so nothing is asked while the history is being rewritten, and
	// it is stopped the way a reply is.
	AssistantTurnOriginCompaction = AssistantTurnOrigin("Compaction")
	// AssistantTurnOriginWaitResolved is the turn in which the agent picks up
	// work it parked on a wait, once what it waited for happened or the wait
	// ran out.
	AssistantTurnOriginWaitResolved = AssistantTurnOrigin("WaitResolved")
)

// AllAssistantTurnOrigins is every origin a turn may have, in the order they
// were added.
func AllAssistantTurnOrigins() []AssistantTurnOrigin {
	return []AssistantTurnOrigin{
		AssistantTurnOriginPerson,
		AssistantTurnOriginDecisionFollowUp,
		AssistantTurnOriginScheduled,
		AssistantTurnOriginCompaction,
		AssistantTurnOriginWaitResolved,
	}
}

func (o AssistantTurnOrigin) IsValid() bool {
	switch o {
	case AssistantTurnOriginPerson, AssistantTurnOriginDecisionFollowUp,
		AssistantTurnOriginScheduled, AssistantTurnOriginCompaction,
		AssistantTurnOriginWaitResolved:
		return true
	default:
		return false
	}
}

// AssistantTurn is one question and the reply it is producing.
//
// It exists so a reply has an identity of its own while it is still being
// written. Its events are published under this id, and a reader whose stream
// expired reads this row to learn how it ended — which is the difference
// between telling them the answer is in the conversation and leaving them
// watching a connection that will never say anything again.
type AssistantTurn struct {
	keel.Table `keel:"assistant_turns"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:atrn_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	ThreadID pulid.ID `json:"threadId" keel:"thread_id"`
	UserID   pulid.ID `json:"userId"   keel:"user_id"`
	// RunID points at the agent run opened for this turn, which happens only
	// when it proposed a change somebody has to decide on.
	RunID pulid.ID `json:"runId" keel:"run_id,nullzero"`
	// WorkflowID is the durable execution carrying the turn, empty until its
	// start is recorded. ExecutionID answers for a turn either way.
	WorkflowID string `json:"workflowId" keel:"workflow_id,nullzero"`

	// Origin and Input say what the turn answers, so a reader who rejoins a
	// reply already being written can show it under the right heading: the
	// person's question, or the decision it follows up.
	Origin AssistantTurnOrigin `json:"origin" keel:"origin"`
	Input  string              `json:"input"  keel:"input,nullzero"`

	Status       conversation.AssistantTurnStatus `json:"status"       keel:"status"`
	ErrorMessage string                           `json:"errorMessage" keel:"error_message,nullzero"`
	StartedAt    int64                            `json:"startedAt"    keel:"started_at,nullzero"`
	CompletedAt  *int64                           `json:"completedAt"  keel:"completed_at"`
	Fingerprint  *agentfingerprint.Fingerprint    `json:"fingerprint"  keel:"fingerprint,jsonb"`
	// TraceID is the turn's trace.
	TraceID string `json:"traceId" keel:"trace_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (t *AssistantTurn) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if t.Status == "" {
		t.Status = conversation.AssistantTurnStatusPending
	}
	if t.Origin == "" {
		t.Origin = AssistantTurnOriginPerson
	}
	if t.StartedAt == 0 {
		t.StartedAt = now
	}

	return nil
}

func (t *AssistantTurn) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		t,
		validation.Field(&t.ThreadID, validation.Required.Error("Thread is required")),
		validation.Field(&t.UserID, validation.Required.Error("User is required")),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[conversation.AssistantTurnStatus]("Invalid status"),
		),
		validation.Field(&t.Origin,
			validation.Required.Error("Origin is required"),
			domainvalidation.ValidEnum[AssistantTurnOrigin]("Invalid origin"),
		),
		validation.Field(&t.TraceID, domainvalidation.TraceID("Trace id is invalid")),
	))
}

// AssistantTurnWorkflowPrefix makes a turn's execution findable from its
// record, and its id unique: one turn, one execution, for ever.
const AssistantTurnWorkflowPrefix = "assistant-turn:"

// AssistantTurnWorkflowID names the execution carrying a turn.
func AssistantTurnWorkflowID(turnID pulid.ID) string {
	return AssistantTurnWorkflowPrefix + turnID.String()
}

// ExecutionID is the execution carrying the turn. The id is derived from the
// turn, so it is known before the start is recorded, and a turn whose start
// was never written down can still be followed and stopped.
func (t *AssistantTurn) ExecutionID() string {
	if t.WorkflowID != "" {
		return t.WorkflowID
	}

	return AssistantTurnWorkflowID(t.ID)
}

func (t *AssistantTurn) GetID() pulid.ID {
	return t.ID
}

func (t *AssistantTurn) GetOrganizationID() pulid.ID {
	return t.OrganizationID
}

func (t *AssistantTurn) GetBusinessUnitID() pulid.ID {
	return t.BusinessUnitID
}

func (t *AssistantTurn) GetTableName() string {
	return "assistant_turns"
}
