package watchtower

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	bun.BaseModel `bun:"table:watchtower_items,alias:wti" json:"-"`
	keel.Table    `keel:"watchtower_items"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:wt_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`

	SourceKind SourceKind `json:"sourceKind" bun:"source_kind,type:VARCHAR(40),notnull" keel:"source_kind"`
	SourceID   string     `json:"sourceId"   bun:"source_id,type:VARCHAR(200),notnull" keel:"source_id"`
	Severity   Severity   `json:"severity"   bun:"severity,type:VARCHAR(20),notnull" keel:"severity"`
	Title      string     `json:"title"      bun:"title,type:VARCHAR(200),notnull" keel:"title"`
	Summary    string     `json:"summary"    bun:"summary,type:TEXT,nullzero" keel:"summary,nullzero"`

	// SubjectType and SubjectID name the record an agent would work on if
	// the item is handed to one; EventKind is the event that hand-off
	// publishes. Empty when the source has no agent subject.
	SubjectType agent.SubjectType `json:"subjectType" bun:"subject_type,type:VARCHAR(50),nullzero" keel:"subject_type,nullzero"`
	SubjectID   pulid.ID          `json:"subjectId"   bun:"subject_id,type:VARCHAR(100),nullzero" keel:"subject_id,nullzero"`
	EventKind   agent.EventKind   `json:"eventKind"   bun:"event_kind,type:VARCHAR(100),nullzero" keel:"event_kind,nullzero"`

	// Path opens the source record. It is built by the server and validated
	// like an insight's link, so a stored path can only be an application path.
	Path string `json:"path" bun:"path,type:VARCHAR(500),nullzero" keel:"path,nullzero"`

	OccurredAt int64  `json:"occurredAt" bun:"occurred_at,type:BIGINT,notnull" keel:"occurred_at"`
	ResolvedAt *int64 `json:"resolvedAt" bun:"resolved_at,type:BIGINT,nullzero" keel:"resolved_at"`

	Status          Status         `json:"status"          bun:"status,type:VARCHAR(20),notnull,default:'Open'" keel:"status"`
	AssigneeID      pulid.ID       `json:"assigneeId"      bun:"assignee_id,type:VARCHAR(100),nullzero" keel:"assignee_id,nullzero"`
	DueAt           *int64         `json:"dueAt"           bun:"due_at,type:BIGINT,nullzero" keel:"due_at"`
	SnoozedUntil    *int64         `json:"snoozedUntil"    bun:"snoozed_until,type:BIGINT,nullzero" keel:"snoozed_until"`
	AcknowledgedAt  *int64         `json:"acknowledgedAt"  bun:"acknowledged_at,type:BIGINT,nullzero" keel:"acknowledged_at"`
	ResolutionCode  ResolutionCode `json:"resolutionCode"  bun:"resolution_code,type:VARCHAR(30),nullzero" keel:"resolution_code,nullzero"`
	ResolutionNote  string         `json:"resolutionNote"  bun:"resolution_note,type:TEXT,nullzero" keel:"resolution_note,nullzero"`
	ResolvedByID    pulid.ID       `json:"resolvedById"    bun:"resolved_by_id,type:VARCHAR(100),nullzero" keel:"resolved_by_id,nullzero"`
	SourceClearedAt *int64         `json:"sourceClearedAt" bun:"source_cleared_at,type:BIGINT,nullzero" keel:"source_cleared_at"`
	EscalationLevel int16          `json:"escalationLevel" bun:"escalation_level,type:SMALLINT,notnull,default:0" keel:"escalation_level"`
	LastEscalatedAt *int64         `json:"lastEscalatedAt" bun:"last_escalated_at,type:BIGINT,nullzero" keel:"last_escalated_at"`

	DueLabel  string    `json:"dueLabel"  bun:"due_label,type:VARCHAR(120),nullzero" keel:"due_label,nullzero"`
	DueSource DueSource `json:"dueSource" bun:"due_source,type:VARCHAR(20),nullzero" keel:"due_source,nullzero"`

	RecordKind  permission.RecordKind `json:"recordKind"  bun:"record_kind,type:VARCHAR(60),nullzero" keel:"record_kind,nullzero"`
	RecordLabel string                `json:"recordLabel" bun:"record_label,type:VARCHAR(200),nullzero" keel:"record_label,nullzero"`
	Facts       []Fact                `json:"facts"       bun:"facts,type:JSONB,notnull,default:'[]'::jsonb" keel:"facts,jsonb"`
	Impact      []string              `json:"impact"      bun:"impact,type:JSONB,notnull,default:'[]'::jsonb" keel:"impact,jsonb"`
	Trail       []TrailEvent          `json:"trail"       bun:"trail,type:JSONB,notnull,default:'[]'::jsonb" keel:"trail,jsonb"`

	AgentDefinitionID pulid.ID `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),nullzero" keel:"agent_definition_id,nullzero"`
	AgentRunID        pulid.ID `json:"agentRunId"        bun:"agent_run_id,type:VARCHAR(100),nullzero" keel:"agent_run_id,nullzero"`
	AgentHandedAt     *int64   `json:"agentHandedAt"     bun:"agent_handed_at,type:BIGINT,nullzero" keel:"agent_handed_at"`
	AgentHandedByID   pulid.ID `json:"agentHandedById"   bun:"agent_handed_by_id,type:VARCHAR(100),nullzero" keel:"agent_handed_by_id,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	// Seen says whether this reader had already been shown the item, from
	// their cursor. Filled by the service, never stored.
	Seen bool `json:"seen" bun:"-"`
	// Inserted is set by an upsert that reports whether the row was created
	// rather than replaced; it is never selected on its own.
	Inserted bool `json:"-"    bun:"inserted,scanonly" keel:"inserted,computed"`
	// Reopened is set by an upsert that brought a resolved item back.
	Reopened  bool `json:"-"    bun:"reopened,scanonly" keel:"reopened,computed"`
	Unchanged bool `json:"-"    bun:"-"`
}

func (i *Item) GetID() pulid.ID { return i.ID }

func (i *Item) GetTableName() string { return "watchtower_items" }

func (i *Item) IsResolved() bool { return i.ResolvedAt != nil }

func (i *Item) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("wt_")
		}
		if i.CreatedAt == 0 {
			i.CreatedAt = now
		}
		i.UpdatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
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

func (i *Item) ResolvedRecordKind() permission.RecordKind {
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
	bun.BaseModel `bun:"table:watchtower_cursors,alias:wtc" json:"-"`
	keel.Table    `keel:"watchtower_cursors"`

	UserID         pulid.ID `json:"userId"         bun:"user_id,pk,type:VARCHAR(100),notnull" keel:"user_id,pk"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	SeenAt         int64    `json:"seenAt"         bun:"seen_at,type:BIGINT,notnull" keel:"seen_at"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (c *Cursor) BeforeAppendModel(_ context.Context, _ bun.Query) error {
	c.UpdatedAt = timeutils.NowUnix()

	return nil
}

type Read struct {
	bun.BaseModel `bun:"table:watchtower_item_reads,alias:wtr" json:"-"`
	keel.Table    `keel:"watchtower_item_reads"`

	ItemID         pulid.ID `json:"itemId"         bun:"item_id,pk,type:VARCHAR(100),notnull" keel:"item_id,pk"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,pk,type:VARCHAR(100),notnull" keel:"user_id,pk"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	SeenAt         int64    `json:"seenAt"         bun:"seen_at,type:BIGINT,notnull" keel:"seen_at"`
}
