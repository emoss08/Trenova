package agent

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*AgentRunEvent)(nil)
	_ pagination.CursorEntity            = (*AgentRunEvent)(nil)
	_ pagination.CursorValueCarrier      = (*AgentRunEvent)(nil)
)

// AgentRunEvent is one thing that happened, in the order it happened.
//
// The runtime has always said all of this aloud — which tool it reached for,
// what came back, what it refused, when it gave up — but nobody was writing it
// down. A conversation's events went to a stream that is dropped a quarter of an
// hour after the reply ends, and a background run's went nowhere at all: they
// arrived at a function that used them to say "still working" and discarded
// them. What survived was the final reply, cut to two thousand characters.
//
// So these rows are not a new account of a run. They are the account the run was
// already giving, kept. An organization that lets an agent write to its records
// is owed the ability to ask what it did and be answered from something better
// than a summary.
//
// The rows are append-only. A step in the ledger beside this one is claimed and
// then settled, because its question is "has this already run"; an event is
// never revised, because its question is "what happened". The two join on
// StepKey where a tool is involved.
type AgentRunEvent struct {
	keel.Table `keel:"agent_run_events"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:are_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	// OwnerKind and OwnerID name the run or conversation turn this belongs to,
	// exactly as the step ledger does. There is no foreign key for the same
	// reason: one column cannot point at two tables.
	OwnerKind string   `json:"ownerKind" keel:"owner_kind"`
	OwnerID   pulid.ID `json:"ownerId" keel:"owner_id"`

	// Sequence is the event's place in its owner's account of itself.
	//
	// Ordering by time would not do. Several events can share a second, and a
	// trajectory read back in the wrong order is worse than no trajectory: it
	// reports the agent doing things in an order it never did.
	Sequence int `json:"sequence" keel:"sequence"`

	// Kind is the runtime's own event name, unchanged.
	//
	// Translating it into a second vocabulary would mean two lists to keep in
	// step and a reader having to learn both.
	Kind string `json:"kind" keel:"kind"`

	// StepKey joins a tool event to the ledger row that claimed it. Empty for
	// everything that is not a tool call.
	StepKey string `json:"stepKey" keel:"step_key"`
	CallID  string `json:"callId" keel:"call_id"`

	Payload map[string]any `json:"payload" keel:"payload,jsonb"`

	// Truncated marks a payload too large to keep whole. A reader that cannot
	// tell a short payload from a shortened one will eventually mistake one for
	// the other, so the row says which it is.
	Truncated bool `json:"truncated" keel:"truncated"`

	OccurredAt int64 `json:"occurredAt" keel:"occurred_at"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (e *AgentRunEvent) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if e.OccurredAt == 0 {
		e.OccurredAt = now
	}

	return nil
}

func (e *AgentRunEvent) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		e,
		validation.Field(&e.OwnerKind,
			validation.Required.Error("Owner kind is required"),
			validation.In("AgentRun", "AssistantTurn").Error("Invalid owner kind"),
		),
		validation.Field(&e.OwnerID, validation.Required.Error("Owner id is required")),
		validation.Field(&e.Kind, validation.Required.Error("Kind is required")),
		validation.Field(&e.Sequence,
			validation.Min(1).Error("Sequence starts at one"),
		),
		validation.Field(&e.OccurredAt,
			validation.Required.Error("Occurred at is required"),
		),
	))
}

func (e *AgentRunEvent) GetID() pulid.ID {
	return e.ID
}

func (e *AgentRunEvent) GetOrganizationID() pulid.ID {
	return e.OrganizationID
}

func (e *AgentRunEvent) GetBusinessUnitID() pulid.ID {
	return e.BusinessUnitID
}

func (e *AgentRunEvent) GetTableName() string {
	return "agent_run_events"
}

// GetPostgresSearchConfig makes a trajectory searchable.
//
// The issue this table comes from asks for events that support discovery and
// search, not only an ordered read. Kind and tool name are what anybody
// actually searches by — "show me the refusals", "what did it do with
// assign_worker" — so those are the fields offered rather than the payload,
// which is jsonb and would make every search a sequential scan.
// GetCreatedAt satisfies the cursor contract. Paging keys on creation rather
// than on sequence because a cursor has to be comparable across owners, and a
// sequence is only meaningful within one.
func (e *AgentRunEvent) GetCreatedAt() int64 {
	return e.CreatedAt
}

func (e *AgentRunEvent) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "are",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "kind", Type: domaintypes.FieldTypeEnum},
			{Name: "owner_kind", Type: domaintypes.FieldTypeEnum},
			{Name: "call_id", Type: domaintypes.FieldTypeText},
		},
	}
}
