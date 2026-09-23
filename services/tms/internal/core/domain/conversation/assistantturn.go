package conversation

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*AssistantTurn)(nil)
	_ validationframework.TenantedEntity = (*AssistantTurn)(nil)
)

// AssistantTurnStatus is how far a turn got.
type AssistantTurnStatus string

const (
	AssistantTurnStatusPending   = AssistantTurnStatus("Pending")
	AssistantTurnStatusRunning   = AssistantTurnStatus("Running")
	AssistantTurnStatusCompleted = AssistantTurnStatus("Completed")
	// AssistantTurnStatusRefused is a turn the scope guard declined. It is a finished
	// turn, not a failed one: the question was read, judged and answered.
	AssistantTurnStatusRefused = AssistantTurnStatus("Refused")
	// AssistantTurnStatusStopped is a turn the person ended themselves.
	AssistantTurnStatusStopped = AssistantTurnStatus("Stopped")
	AssistantTurnStatusFailed  = AssistantTurnStatus("Failed")
)

func (s AssistantTurnStatus) IsValid() bool {
	switch s {
	case AssistantTurnStatusPending, AssistantTurnStatusRunning, AssistantTurnStatusCompleted,
		AssistantTurnStatusRefused, AssistantTurnStatusStopped, AssistantTurnStatusFailed:
		return true
	default:
		return false
	}
}

// AssistantTurnOrigin is what started a turn: a person asking, or the
// application following up a decision on one of the conversation's proposals.
type AssistantTurnOrigin string

const (
	AssistantTurnOriginPerson = AssistantTurnOrigin("Person")
	// AssistantTurnOriginDecisionFollowUp is the turn in which the agent says
	// what came of a proposal somebody decided, wherever they decided it.
	AssistantTurnOriginDecisionFollowUp = AssistantTurnOrigin("DecisionFollowUp")
)

func (o AssistantTurnOrigin) IsValid() bool {
	switch o {
	case AssistantTurnOriginPerson, AssistantTurnOriginDecisionFollowUp:
		return true
	default:
		return false
	}
}

// Terminal reports a turn that will produce nothing more.
func (s AssistantTurnStatus) Terminal() bool {
	switch s {
	case AssistantTurnStatusCompleted, AssistantTurnStatusRefused, AssistantTurnStatusStopped, AssistantTurnStatusFailed:
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
	bun.BaseModel `bun:"table:assistant_turns,alias:atrn" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	ThreadID pulid.ID `json:"threadId" bun:"thread_id,type:VARCHAR(100),notnull"`
	UserID   pulid.ID `json:"userId"   bun:"user_id,type:VARCHAR(100),notnull"`
	// RunID points at the agent run opened for this turn, which happens only
	// when it proposed a change somebody has to decide on.
	RunID pulid.ID `json:"runId" bun:"run_id,type:VARCHAR(100),nullzero"`
	// WorkflowID is the durable execution carrying the turn, empty while it
	// still runs in the request that asked for it.
	WorkflowID string `json:"workflowId" bun:"workflow_id,type:VARCHAR(255),nullzero"`

	// Origin and Input say what the turn answers, so a reader who rejoins a
	// reply already being written can show it under the right heading: the
	// person's question, or the decision it follows up.
	Origin AssistantTurnOrigin `json:"origin" bun:"origin,type:VARCHAR(30),notnull,default:'Person'"`
	Input  string              `json:"input"  bun:"input,type:TEXT,nullzero"`

	Status       AssistantTurnStatus `json:"status"       bun:"status,type:VARCHAR(20),notnull,default:'Pending'"`
	ErrorMessage string              `json:"errorMessage" bun:"error_message,type:TEXT,nullzero"`
	StartedAt    int64               `json:"startedAt"    bun:"started_at,type:BIGINT,nullzero"`
	CompletedAt  *int64              `json:"completedAt"  bun:"completed_at,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (t *AssistantTurn) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		t,
		validation.Field(&t.ThreadID, validation.Required.Error("Thread is required")),
		validation.Field(&t.UserID, validation.Required.Error("User is required")),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[AssistantTurnStatus]("Invalid status"),
		),
		validation.Field(&t.Origin,
			validation.Required.Error("Origin is required"),
			domainvalidation.ValidEnum[AssistantTurnOrigin]("Invalid origin"),
		),
	))
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

func (t *AssistantTurn) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("atrn_")
		}
		if t.Status == "" {
			t.Status = AssistantTurnStatusPending
		}
		if t.Origin == "" {
			t.Origin = AssistantTurnOriginPerson
		}
		if t.StartedAt == 0 {
			t.StartedAt = now
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}
