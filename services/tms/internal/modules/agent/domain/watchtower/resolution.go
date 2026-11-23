package watchtower

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

const ResolutionRetentionDays = 25 * 31

type ResolvedVia string

const (
	// A person resolved it
	ResolvedViaPerson = ResolvedVia("Person")
	// An agent resolved it for the person it was acting for
	ResolvedViaAgent = ResolvedVia("Agent")
	// The source stopped reporting the problem, so it resolved on its own
	ResolvedViaSourceCleared = ResolvedVia("SourceCleared")
)

func (v ResolvedVia) String() string { return string(v) }

func AllResolvedVia() []ResolvedVia {
	return []ResolvedVia{ResolvedViaPerson, ResolvedViaAgent, ResolvedViaSourceCleared}
}

func (v ResolvedVia) IsValid() bool {
	switch v {
	case ResolvedViaPerson, ResolvedViaAgent, ResolvedViaSourceCleared:
		return true
	default:
		return false
	}
}

func (v ResolvedVia) ByPerson() bool {
	return v == ResolvedViaPerson || v == ResolvedViaAgent
}

// Watchtower resolutions: one row each time a Watchtower item is resolved, by whom, why and how
// long it took. Kept for 25 months, long after the item itself is removed, so handled work and
// resolution reasons can be compared across periods. Says nothing about what the item read.
type Resolution struct {
	keel.Table `keel:"watchtower_resolutions,report=Desk,label=Watchtower Resolution"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:wtr_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	ItemID      pulid.ID `json:"itemId"      keel:"item_id,noreport"`
	ItemVersion int64    `json:"itemVersion" keel:"item_version,noreport"`

	// The case the item belonged to: the record it was about, or the item itself when it was
	// about nothing else. Count it to count cases.
	CaseID pulid.ID `json:"caseId"         keel:"case_id,label=Case"`
	// The kind of item
	Kind SourceKind `json:"kind"           keel:"kind"`
	// How severe the item was when it was resolved, escalations included
	Severity Severity `json:"severity"       keel:"severity"`
	// How the item was resolved: by a person, by an agent for a person, or by its source
	// clearing
	ResolvedVia ResolvedVia `json:"resolvedVia"    keel:"resolved_via,label=Resolved Via"`
	// Why the item was closed
	ResolutionCode ResolutionCode `json:"resolutionCode" keel:"resolution_code,label=Reason"`

	AssigneeID   pulid.ID `json:"assigneeId"   keel:"assignee_id,nullzero"`
	ResolvedByID pulid.ID `json:"resolvedById" keel:"resolved_by_id,nullzero"`

	// When the problem occurred
	OccurredAt int64 `json:"occurredAt"           keel:"occurred_at"`
	// When someone first said they had seen it
	AcknowledgedAt *int64 `json:"acknowledgedAt"       keel:"acknowledged_at"`
	// When it was due to be dealt with
	DueAt *int64 `json:"dueAt"                keel:"due_at,label=Due"`
	// When it was resolved
	ResolvedAt int64 `json:"resolvedAt"           keel:"resolved_at,label=Resolved"`
	// Seconds from when the problem occurred to when someone acknowledged it; empty when nobody
	// did
	SecondsToAcknowledge *int64 `json:"secondsToAcknowledge" keel:"seconds_to_acknowledge,label=Time to Acknowledge"`
	// Seconds from when the problem occurred to when it was resolved
	SecondsToResolve int64 `json:"secondsToResolve"     keel:"seconds_to_resolve,label=Time to Resolve"`
	// Whether it was resolved after it was due
	PastDue bool `json:"pastDue"              keel:"past_due,label=Resolved After Due"`
	// How many times it escalated before it was resolved
	EscalationLevel int16 `json:"escalationLevel"      keel:"escalation_level,label=Escalations"`
	// Whether it had been handed to an agent
	AgentHanded bool `json:"agentHanded"          keel:"agent_handed,label=Handed to an Agent"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`

	// The person the item was assigned to when it was resolved
	Assignee *tenant.Person `json:"-" keel:"belongs:assignee_id"`
	// The person who resolved it, or for whom an agent resolved it
	ResolvedBy *tenant.Person `json:"-" keel:"belongs:resolved_by_id,label=Resolved By"`
}

func (r *Resolution) GetID() pulid.ID { return r.ID }

func (r *Resolution) GetTableName() string { return "watchtower_resolutions" }

func ResolutionOf(item *Item, via ResolvedVia) *Resolution {
	if item == nil || item.ResolvedAt == nil {
		return nil
	}

	resolvedAt := *item.ResolvedAt
	resolution := &Resolution{
		ID:               pulid.MustNew("wtr_"),
		BusinessUnitID:   item.BusinessUnitID,
		OrganizationID:   item.OrganizationID,
		ItemID:           item.ID,
		ItemVersion:      item.Version,
		CaseID:           item.GroupKey(),
		Kind:             item.SourceKind,
		Severity:         item.Severity,
		ResolvedVia:      via,
		ResolutionCode:   item.ResolutionCode,
		AssigneeID:       item.AssigneeID,
		OccurredAt:       item.OccurredAt,
		AcknowledgedAt:   item.AcknowledgedAt,
		DueAt:            item.DueAt,
		ResolvedAt:       resolvedAt,
		SecondsToResolve: max(resolvedAt-item.OccurredAt, 0),
		PastDue:          item.DueAt != nil && resolvedAt > *item.DueAt,
		EscalationLevel:  item.EscalationLevel,
		AgentHanded:      item.AgentHandedAt != nil,
	}
	if via.ByPerson() {
		resolution.ResolvedByID = item.ResolvedByID
	}
	if item.AcknowledgedAt != nil {
		seconds := max(*item.AcknowledgedAt-item.OccurredAt, 0)
		resolution.SecondsToAcknowledge = &seconds
	}

	return resolution
}
