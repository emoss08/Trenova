package billingqueueevent

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

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
	keel.Table `keel:"billing_queue_events"`

	ID             pulid.ID       `json:"id"             keel:"id,pk,autoid:bqe_"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	ItemID         pulid.ID       `json:"itemId"         keel:"item_id"`
	Kind           EventKind      `json:"kind"           keel:"kind"`
	Text           string         `json:"text"           keel:"text"`
	ActorType      EventActorType `json:"actorType"      keel:"actor_type"`
	ActorID        *pulid.ID      `json:"actorId"        keel:"actor_id"`
	ActorName      string         `json:"actorName"      keel:"actor_name,nullzero"`
	Payload        map[string]any `json:"payload"        keel:"payload,jsonb"`
	At             int64          `json:"at"             keel:"at"`
	CreatedAt      int64          `json:"createdAt"      keel:"created_at,autocreate"`
}

func (e *ItemEvent) BeforeCreate(ctx context.Context) error {
	now := timeutils.NowUnix()
	if e.At == 0 {
		e.At = now
	}
	if e.Payload == nil {
		e.Payload = map[string]any{}
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
