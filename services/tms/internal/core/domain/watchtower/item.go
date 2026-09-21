package watchtower

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
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

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	SourceKind SourceKind `json:"sourceKind" bun:"source_kind,type:VARCHAR(40),notnull"`
	SourceID   string     `json:"sourceId"   bun:"source_id,type:VARCHAR(200),notnull"`
	Severity   Severity   `json:"severity"   bun:"severity,type:VARCHAR(20),notnull"`
	Title      string     `json:"title"      bun:"title,type:VARCHAR(200),notnull"`
	Summary    string     `json:"summary"    bun:"summary,type:TEXT,nullzero"`

	// SubjectType and SubjectID name the record an agent would work on if
	// the item is handed to one; EventKind is the event that hand-off
	// publishes. Empty when the source has no agent subject.
	SubjectType agent.SubjectType `json:"subjectType" bun:"subject_type,type:VARCHAR(50),nullzero"`
	SubjectID   pulid.ID          `json:"subjectId"   bun:"subject_id,type:VARCHAR(100),nullzero"`
	EventKind   agent.EventKind   `json:"eventKind"   bun:"event_kind,type:VARCHAR(100),nullzero"`

	// Path opens the source record. It is built by the server and validated
	// like an insight's link, so a stored path can only be an application path.
	Path string `json:"path" bun:"path,type:VARCHAR(500),nullzero"`

	OccurredAt int64  `json:"occurredAt" bun:"occurred_at,type:BIGINT,notnull"`
	ResolvedAt *int64 `json:"resolvedAt" bun:"resolved_at,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Seen says whether this reader had already been shown the item, from
	// their cursor. Filled by the service, never stored.
	Seen bool `json:"seen" bun:"-"`
	// Inserted is set by an upsert that reports whether the row was created
	// rather than replaced; it is never selected on its own.
	Inserted bool `json:"-" bun:"inserted,scanonly"`
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
		multiErr.Add("subjectType", errortypes.ErrRequired, "Subject type is required with a subject")
	}
	if i.EventKind != "" && !i.EventKind.IsValid() {
		multiErr.Add("eventKind", errortypes.ErrInvalid, "Event kind is invalid")
	}
}

// Cursor is where one reader's eye last was: items that occurred after
// SeenAt are new to them. One row per reader per tenant.
type Cursor struct {
	bun.BaseModel `bun:"table:watchtower_cursors,alias:wtc" json:"-"`

	UserID         pulid.ID `json:"userId"         bun:"user_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	SeenAt         int64    `json:"seenAt"         bun:"seen_at,type:BIGINT,notnull"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (c *Cursor) BeforeAppendModel(_ context.Context, _ bun.Query) error {
	c.UpdatedAt = timeutils.NowUnix()

	return nil
}
