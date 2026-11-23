package watchtower

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/caseowner"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerpanel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxTitleLength    = 200
	maxSummaryLength  = 1000
	maxSourceIDLength = 200
	maxPathLength     = 500
)

// Watchtower items: what the operation is working or has recently closed, with each item's kind,
// severity, status, owner, due time and escalations. Resolved items are removed after 30 days;
// Watchtower Resolutions keep the history. Says nothing about what an item reads.
type Item struct {
	keel.Table `keel:"watchtower_items,report=Desk,label=Watchtower Item"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:wt_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	// The kind of item
	SourceKind SourceKind `json:"sourceKind" keel:"source_kind,label=Kind"`
	SourceID   string     `json:"sourceId"   keel:"source_id,noreport"`
	// How severe the item is now, escalations included
	Severity Severity `json:"severity"   keel:"severity"`
	Title    string   `json:"title"      keel:"title,noreport"`
	Summary  string   `json:"summary"    keel:"summary,nullzero,noreport"`

	SubjectType agent.SubjectType      `json:"subjectType" keel:"subject_type,nullzero,noreport"`
	SubjectID   pulid.ID               `json:"subjectId"   keel:"subject_id,nullzero,noreport"`
	EventKind   agenttrigger.EventKind `json:"eventKind"   keel:"event_kind,nullzero,noreport"`

	Path string `json:"path" keel:"path,nullzero,noreport"`

	// When the problem occurred
	OccurredAt int64 `json:"occurredAt" keel:"occurred_at"`
	// When the item was resolved; empty while it is open
	ResolvedAt *int64 `json:"resolvedAt" keel:"resolved_at,label=Resolved"`

	// Where the item stands: open (nobody has said they are on it), acknowledged, in
	// progress, snoozed or resolved
	Status     Status   `json:"status"          keel:"status"`
	AssigneeID pulid.ID `json:"assigneeId"      keel:"assignee_id,nullzero"`
	// When the item is due to be dealt with
	DueAt *int64 `json:"dueAt"           keel:"due_at,label=Due"`
	// When a snoozed item comes back
	SnoozedUntil *int64 `json:"snoozedUntil"    keel:"snoozed_until"`
	// When someone first said they had seen it
	AcknowledgedAt *int64 `json:"acknowledgedAt"  keel:"acknowledged_at"`
	// Why a resolved item was closed
	ResolutionCode ResolutionCode `json:"resolutionCode"  keel:"resolution_code,nullzero,label=Reason"`
	ResolutionNote string         `json:"resolutionNote"  keel:"resolution_note,nullzero,noreport"`
	ResolvedByID   pulid.ID       `json:"resolvedById"    keel:"resolved_by_id,nullzero"`
	// When the source stopped reporting the problem after a person had resolved it
	SourceClearedAt *int64 `json:"sourceClearedAt" keel:"source_cleared_at"`
	// How many times the item has escalated
	EscalationLevel int16 `json:"escalationLevel" keel:"escalation_level,label=Escalations"`
	// When it last escalated
	LastEscalatedAt *int64 `json:"lastEscalatedAt" keel:"last_escalated_at"`

	DueLabel string `json:"dueLabel"  keel:"due_label,nullzero,noreport"`
	// Where the due time came from: the record's own deadline, the organization's policy or a
	// person
	DueSource DueSource `json:"dueSource" keel:"due_source,nullzero"`

	RecordKind  recordkind.Kind `json:"recordKind"  keel:"record_kind,nullzero,noreport"`
	RecordLabel string          `json:"recordLabel" keel:"record_label,nullzero,noreport"`

	AboutKind recordkind.Kind `json:"aboutKind" keel:"about_kind,nullzero,noreport"`
	// The record the item is about, which is its case; empty when it is about nothing
	// beyond itself. Count it to count cases.
	AboutID pulid.ID               `json:"aboutId"   keel:"about_id,nullzero,report,label=Case"`
	Facts   []Fact                 `json:"facts"     keel:"facts,jsonb,noreport"`
	Impact  []string               `json:"impact"    keel:"impact,jsonb,noreport"`
	Trail   []TrailEvent           `json:"trail"     keel:"trail,jsonb,noreport"`
	Panel   *watchtowerpanel.Panel `json:"panel"     keel:"panel,jsonb,noreport"`

	AgentDefinitionID pulid.ID `json:"agentDefinitionId" keel:"agent_definition_id,nullzero,noreport"`
	AgentRunID        pulid.ID `json:"agentRunId"        keel:"agent_run_id,nullzero,noreport"`
	// When the item was handed to an agent; empty when it never was
	AgentHandedAt   *int64   `json:"agentHandedAt"     keel:"agent_handed_at,label=Handed to an Agent"`
	AgentHandedByID pulid.ID `json:"agentHandedById"   keel:"agent_handed_by_id,nullzero,noreport"`

	LastRequest *Request `json:"lastRequest" keel:"last_request,jsonb,noreport"`

	DescribedAt int64 `json:"describedAt" keel:"described_at,noreport"`

	Version int64 `json:"version"   keel:"version,version"`
	// When the item was raised
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate,label=Raised"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate,noreport"`

	// The person the item is assigned to
	Assignee *tenant.Person `json:"-" keel:"belongs:assignee_id"`
	// The person who resolved it, or for whom an agent resolved it
	ResolvedBy *tenant.Person `json:"-" keel:"belongs:resolved_by_id,label=Resolved By"`
	// Who owns the case the item belongs to
	CaseOwner *caseowner.Owner `json:"-" keel:"belongs:about_id=record_id"`

	Seen      bool `json:"seen"`
	Inserted  bool `json:"-"    keel:"inserted,computed"`
	Reopened  bool `json:"-"    keel:"reopened,computed"`
	Unchanged bool `json:"-"`
}

func (i *Item) GetID() pulid.ID { return i.ID }

func (i *Item) GetTableName() string { return "watchtower_items" }

func (i *Item) IsResolved() bool { return i.ResolvedAt != nil }

func (i *Item) BeforeCreate(ctx context.Context) error {
	if i.Status == "" {
		i.Status = StatusOpen
	}

	return nil
}

// Normalize trims what a person reads and bounds it to what the columns
// hold, so a source whose summary runs long is cut rather than refused.
func (i *Item) Normalize() {
	i.Title = stringutils.Ellipsize(Plain(i.Title), maxTitleLength)
	i.Summary = stringutils.Ellipsize(Plain(i.Summary), maxSummaryLength)
	i.SourceID = strings.TrimSpace(i.SourceID)
	i.Path = strings.TrimSpace(i.Path)
	i.DueLabel = NormalizeDueLabel(i.DueLabel)
	i.RecordLabel = NormalizeRecordLabel(i.RecordLabel)
	i.Facts = NormalizeFacts(i.Facts)
	i.Impact = NormalizeImpact(i.Impact)
	i.Trail = NormalizeTrail(i.Trail)
	i.Panel = watchtowerpanel.Normalize(i.Panel)
}

func (i *Item) ApplyDeadline(deadline *Deadline) {
	if deadline == nil || deadline.At <= 0 {
		return
	}
	at := deadline.At
	i.DueAt = &at
	i.DueLabel = NormalizeDueLabel(deadline.Label)
	i.DueSource = DueSourceSource
}

func (i *Item) SameDescription(other *Item) bool {
	return i.Title == other.Title &&
		i.Summary == other.Summary &&
		i.SubjectType == other.SubjectType &&
		i.SubjectID == other.SubjectID &&
		i.EventKind == other.EventKind &&
		i.Path == other.Path &&
		i.RecordKind == other.RecordKind &&
		i.RecordLabel == other.RecordLabel &&
		i.AboutKind == other.AboutKind &&
		i.AboutID == other.AboutID &&
		slices.Equal(i.Facts, other.Facts) &&
		slices.Equal(i.Impact, other.Impact) &&
		slices.Equal(i.Trail, other.Trail) &&
		watchtowerpanel.Same(i.Panel, other.Panel)
}

func (i *Item) SetAbout(id pulid.ID) {
	i.AboutKind, i.AboutID = "", pulid.Nil
	if id.IsNil() {
		return
	}
	kind, ok := recordkind.OfIDPrefix(id.Prefix())
	if !ok {
		return
	}
	i.AboutKind, i.AboutID = kind, id
}

func (i *Item) GroupKey() pulid.ID {
	if i.AboutID.IsNotNil() {
		return i.AboutID
	}

	return i.ID
}

func (i *Item) SameDeadline(other *Item) bool {
	return i.DueLabel == other.DueLabel &&
		i.DueSource == other.DueSource &&
		((i.DueAt == nil && other.DueAt == nil) ||
			(i.DueAt != nil && other.DueAt != nil && *i.DueAt == *other.DueAt))
}

func (i *Item) ResolvedRecordKind() recordkind.Kind {
	if i.RecordKind != "" {
		return i.RecordKind
	}
	if kind, ok := i.SubjectType.RecordKind(); ok {
		return kind
	}
	return ""
}

func (i *Item) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&i.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&i.SourceKind,
			validation.Required.Error("Source kind is required"),
			domainvalidation.ValidEnum[SourceKind]("Source kind is invalid"),
		),
		validation.Field(&i.SourceID,
			validation.Required.Error("Source identifier is required"),
			validation.Length(1, maxSourceIDLength).Error("Source identifier is too long"),
		),
		validation.Field(&i.Severity,
			validation.Required.Error("Severity is required"),
			domainvalidation.ValidEnum[Severity]("Severity is invalid"),
		),
		validation.Field(&i.Title,
			validation.Required.Error("Title is required"),
			validation.Length(1, maxTitleLength).Error("Title is too long"),
		),
		validation.Field(&i.Summary,
			validation.Length(0, maxSummaryLength).Error("Summary is too long"),
		),
		validation.Field(&i.Path,
			validation.Length(0, maxPathLength).Error("Path is too long"),
		),
		validation.Field(&i.OccurredAt,
			validation.Required.Error("Occurred at is required"),
			validation.Min(int64(1)).Error("Occurred at must be a timestamp"),
		),
	))

	if i.Path != "" && !stringutils.IsSafeAppPath(i.Path) {
		multiErr.Add("path", errortypes.ErrInvalid, "Path must be an application path")
	}
	if i.SubjectID.IsNotNil() && i.SubjectType == "" {
		multiErr.Add(
			"subjectType",
			errortypes.ErrRequired,
			"Subject type is required with a subject",
		)
	}
	if i.EventKind != "" && !i.EventKind.IsValid() {
		multiErr.Add("eventKind", errortypes.ErrInvalid, "Event kind is invalid")
	}
	if i.Status != "" && !i.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Status is invalid")
	}
	if i.ResolutionCode != "" && !i.ResolutionCode.IsValid() {
		multiErr.Add("resolutionCode", errortypes.ErrInvalid, "Resolution code is invalid")
	}
	if i.DueSource != "" && !i.DueSource.IsValid() {
		multiErr.Add("dueSource", errortypes.ErrInvalid, "Due source is invalid")
	}
	if i.DueLabel != "" && i.DueSource != DueSourceSource {
		multiErr.Add(
			"dueLabel",
			errortypes.ErrInvalid,
			"Only a deadline the source reports carries a label",
		)
	}
	if i.RecordKind != "" && !i.RecordKind.Known() {
		multiErr.Add("recordKind", errortypes.ErrInvalid, "Record kind is invalid")
	}
}

// Cursor is where one reader's eye last was: items that occurred after
// SeenAt are new to them. One row per reader per tenant.
type Cursor struct {
	keel.Table `keel:"watchtower_cursors"`

	UserID         pulid.ID `json:"userId"         keel:"user_id,pk"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	SeenAt         int64    `json:"seenAt"         keel:"seen_at"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

type Read struct {
	keel.Table `keel:"watchtower_item_reads"`

	ItemID         pulid.ID `json:"itemId"         keel:"item_id,pk"`
	UserID         pulid.ID `json:"userId"         keel:"user_id,pk"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	SeenAt         int64    `json:"seenAt"         keel:"seen_at"`
}

// HandOffBrief is the item as an agent is handed it: what was raised and the
// sentence that says why.
func (i *Item) HandOffBrief() string {
	title, summary := strings.TrimSpace(i.Title), strings.TrimSpace(i.Summary)
	switch {
	case title == "":
		return summary
	case summary == "":
		return title
	default:
		return title + ". " + summary
	}
}
