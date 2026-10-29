package watchtower

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	maxTitleLength    = 200
	maxSummaryLength  = 1000
	maxSourceIDLength = 200
	maxPathLength     = 500
)

// Item is one thing worth a person's attention, projected from the record
// that raised it. It paginates by time, carries a severity, and resolves
// when its source does; the source stays authoritative by kind and id.
type Item struct {
	keel.Table `keel:"watchtower_items"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:wt_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	SourceKind SourceKind `json:"sourceKind" keel:"source_kind"`
	SourceID   string     `json:"sourceId"   keel:"source_id"`
	Severity   Severity   `json:"severity"   keel:"severity"`
	Title      string     `json:"title"      keel:"title"`
	Summary    string     `json:"summary"    keel:"summary,nullzero"`

	// SubjectType and SubjectID name the record an agent would work on if
	// the item is handed to one; EventKind is the event that hand-off
	// publishes. Empty when the source has no agent subject.
	SubjectType agent.SubjectType      `json:"subjectType" keel:"subject_type,nullzero"`
	SubjectID   pulid.ID               `json:"subjectId"   keel:"subject_id,nullzero"`
	EventKind   agenttrigger.EventKind `json:"eventKind"   keel:"event_kind,nullzero"`

	// Path opens the source record. It is built by the server and validated
	// like an insight's link, so a stored path can only be an application path.
	Path string `json:"path" keel:"path,nullzero"`

	OccurredAt int64  `json:"occurredAt" keel:"occurred_at"`
	ResolvedAt *int64 `json:"resolvedAt" keel:"resolved_at"`

	Status          Status         `json:"status"          keel:"status"`
	AssigneeID      pulid.ID       `json:"assigneeId"      keel:"assignee_id,nullzero"`
	DueAt           *int64         `json:"dueAt"           keel:"due_at"`
	SnoozedUntil    *int64         `json:"snoozedUntil"    keel:"snoozed_until"`
	AcknowledgedAt  *int64         `json:"acknowledgedAt"  keel:"acknowledged_at"`
	ResolutionCode  ResolutionCode `json:"resolutionCode"  keel:"resolution_code,nullzero"`
	ResolutionNote  string         `json:"resolutionNote"  keel:"resolution_note,nullzero"`
	ResolvedByID    pulid.ID       `json:"resolvedById"    keel:"resolved_by_id,nullzero"`
	SourceClearedAt *int64         `json:"sourceClearedAt" keel:"source_cleared_at"`
	EscalationLevel int16          `json:"escalationLevel" keel:"escalation_level"`
	LastEscalatedAt *int64         `json:"lastEscalatedAt" keel:"last_escalated_at"`

	DueLabel  string    `json:"dueLabel"  keel:"due_label,nullzero"`
	DueSource DueSource `json:"dueSource" keel:"due_source,nullzero"`

	RecordKind  recordkind.Kind `json:"recordKind"  keel:"record_kind,nullzero"`
	RecordLabel string          `json:"recordLabel" keel:"record_label,nullzero"`
	Facts       []Fact          `json:"facts"       keel:"facts,jsonb"`
	Impact      []string        `json:"impact"      keel:"impact,jsonb"`
	Trail       []TrailEvent    `json:"trail"       keel:"trail,jsonb"`

	AgentDefinitionID pulid.ID `json:"agentDefinitionId" keel:"agent_definition_id,nullzero"`
	AgentRunID        pulid.ID `json:"agentRunId"        keel:"agent_run_id,nullzero"`
	AgentHandedAt     *int64   `json:"agentHandedAt"     keel:"agent_handed_at"`
	AgentHandedByID   pulid.ID `json:"agentHandedById"   keel:"agent_handed_by_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	// Seen says whether this reader had already been shown the item, from
	// their cursor. Filled by the service, never stored.
	Seen bool `json:"seen"`
	// Inserted is set by an upsert that reports whether the row was created
	// rather than replaced; it is never selected on its own.
	Inserted bool `json:"-" keel:"inserted,computed"`
	// Reopened is set by an upsert that brought a resolved item back.
	Reopened  bool `json:"-" keel:"reopened,computed"`
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
	i.Title = stringutils.Ellipsize(i.Title, maxTitleLength)
	i.Summary = stringutils.Ellipsize(i.Summary, maxSummaryLength)
	i.SourceID = strings.TrimSpace(i.SourceID)
	i.Path = strings.TrimSpace(i.Path)
	i.DueLabel = NormalizeDueLabel(i.DueLabel)
	i.RecordLabel = NormalizeRecordLabel(i.RecordLabel)
	i.Facts = NormalizeFacts(i.Facts)
	i.Impact = NormalizeImpact(i.Impact)
	i.Trail = NormalizeTrail(i.Trail)
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
		slices.Equal(i.Facts, other.Facts) &&
		slices.Equal(i.Impact, other.Impact) &&
		slices.Equal(i.Trail, other.Trail)
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
