package agent

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*AgentRunEvent)(nil)
	_ validationframework.TenantedEntity = (*AgentRunEvent)(nil)
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
	bun.BaseModel `bun:"table:agent_run_events,alias:are" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	// OwnerKind and OwnerID name the run or conversation turn this belongs to,
	// exactly as the step ledger does. There is no foreign key for the same
	// reason: one column cannot point at two tables.
	OwnerKind string   `json:"ownerKind" bun:"owner_kind,type:VARCHAR(20),notnull"`
	OwnerID   pulid.ID `json:"ownerId"   bun:"owner_id,type:VARCHAR(100),notnull"`

	// Sequence is the event's place in its owner's account of itself.
	//
	// Ordering by time would not do. Several events can share a second, and a
	// trajectory read back in the wrong order is worse than no trajectory: it
	// reports the agent doing things in an order it never did.
	Sequence int `json:"sequence" bun:"sequence,type:INTEGER,notnull"`

	// Kind is the runtime's own event name, unchanged.
	//
	// Translating it into a second vocabulary would mean two lists to keep in
	// step and a reader having to learn both.
	Kind string `json:"kind" bun:"kind,type:VARCHAR(40),notnull"`

	// StepKey joins a tool event to the ledger row that claimed it. Empty for
	// everything that is not a tool call.
	StepKey string `json:"stepKey" bun:"step_key,type:VARCHAR(120),notnull,default:''"`
	CallID  string `json:"callId"  bun:"call_id,type:VARCHAR(200),notnull,default:''"`

	Payload map[string]any `json:"payload" bun:"payload,type:JSONB,notnull,default:'{}'::jsonb"`

	// Truncated marks a payload too large to keep whole. A reader that cannot
	// tell a short payload from a shortened one will eventually mistake one for
	// the other, so the row says which it is.
	Truncated bool `json:"truncated" bun:"truncated,type:BOOLEAN,notnull,default:false"`

	OccurredAt int64 `json:"occurredAt" bun:"occurred_at,type:BIGINT,notnull"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
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

func (e *AgentRunEvent) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("are_")
		}
		if e.OccurredAt == 0 {
			e.OccurredAt = now
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
}
