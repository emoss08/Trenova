package billingqueue

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*ItemEvent)(nil)

// EventKind is what happened to an item.
type EventKind string

const (
	EventTransferred       = EventKind("Transferred")
	EventAssigned          = EventKind("Assigned")
	EventIssueRaised       = EventKind("IssueRaised")
	EventIssueResolved     = EventKind("IssueResolved")
	EventIssueUndone       = EventKind("IssueUndone")
	EventIssueCleared      = EventKind("IssueCleared")
	EventDocumentRequested = EventKind("DocumentRequested")
	EventHeld              = EventKind("Held")
	EventReleased          = EventKind("Released")
	EventApproved          = EventKind("Approved")
	EventPosted            = EventKind("Posted")
	EventStatusChanged     = EventKind("StatusChanged")
)

// EventActorType is who an entry is about: the system, an agent or a person.
type EventActorType string

const (
	EventActorSystem = EventActorType("System")
	EventActorAgent  = EventActorType("Agent")
	EventActorUser   = EventActorType("User")
)

// ItemEvent is one line of an item's activity, written in plain words when it
// happens so the timeline never has to reconstruct it from audit diffs.
type ItemEvent struct {
	bun.BaseModel `bun:"table:billing_queue_events,alias:bqe" json:"-"`
	keel.Table    `keel:"billing_queue_events"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:bqe_"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	ItemID         pulid.ID       `json:"itemId"         bun:"item_id,type:VARCHAR(100),notnull" keel:"item_id"`
	Kind           EventKind      `json:"kind"           bun:"kind,type:VARCHAR(30),notnull" keel:"kind"`
	Text           string         `json:"text"           bun:"text,type:TEXT,notnull" keel:"text"`
	ActorType      EventActorType `json:"actorType"      bun:"actor_type,type:VARCHAR(10),notnull" keel:"actor_type"`
	ActorID        *pulid.ID      `json:"actorId"        bun:"actor_id,type:VARCHAR(100),nullzero" keel:"actor_id"`
	ActorName      string         `json:"actorName"      bun:"actor_name,type:VARCHAR(255),nullzero" keel:"actor_name,nullzero"`
	Payload        map[string]any `json:"payload"        bun:"payload,type:JSONB,notnull,default:'{}'" keel:"payload,jsonb"`
	At             int64          `json:"at"             bun:"at,type:BIGINT,notnull" keel:"at"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,type:BIGINT,notnull" keel:"created_at,autocreate"`
}

func (e *ItemEvent) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		now := timeutils.NowUnix()
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("bqe_")
		}
		if e.At == 0 {
			e.At = now
		}
		e.CreatedAt = now
		if e.Payload == nil {
			e.Payload = map[string]any{}
		}
	}

	return nil
}

func (e *ItemEvent) GetID() pulid.ID             { return e.ID }
func (e *ItemEvent) GetOrganizationID() pulid.ID { return e.OrganizationID }
func (e *ItemEvent) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }
func (e *ItemEvent) GetTableName() string        { return "billing_queue_events" }
func (e *ItemEvent) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{TableAlias: "bqe"}
}
