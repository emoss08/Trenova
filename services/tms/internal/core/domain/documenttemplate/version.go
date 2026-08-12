package documenttemplate

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

// VersionStatus is where a version sits in its lifecycle.
type VersionStatus string

const (
	// VersionStatusDraft is editable and renders nothing.
	VersionStatusDraft = VersionStatus("Draft")
	// VersionStatusActive is what customers receive. At most one per template.
	VersionStatusActive = VersionStatus("Active")
	// VersionStatusArchived was live once. It is kept rather than deleted because a
	// generated document points at the version that produced it.
	VersionStatusArchived = VersionStatus("Archived")
)

func (s VersionStatus) String() string { return string(s) }

func (s VersionStatus) IsValid() bool {
	switch s {
	case VersionStatusDraft, VersionStatusActive, VersionStatusArchived:
		return true
	default:
		return false
	}
}

func VersionStatusFromString(v string) (VersionStatus, error) {
	switch v {
	case "Draft":
		return VersionStatusDraft, nil
	case "Active":
		return VersionStatusActive, nil
	case "Archived":
		return VersionStatusArchived, nil
	default:
		return "", errors.New("invalid document template version status")
	}
}

var (
	_ bun.BeforeAppendModelHook          = (*DocumentTemplateVersion)(nil)
	_ validationframework.TenantedEntity = (*DocumentTemplateVersion)(nil)
)

// DocumentTemplateVersion is one revision of a template's content.
//
// A published version is treated as immutable: rollback creates a new version
// copied from an old one rather than editing history, so the bytes a customer
// received can always be reproduced from the version a generated document names.
type DocumentTemplateVersion struct {
	bun.BaseModel             `bun:"table:document_template_versions,alias:dtv" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                     json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	TemplateID     pulid.ID `json:"templateId"     bun:"template_id,type:VARCHAR(100),notnull"`

	// SourceVersionID records what this was copied from, which is what makes a
	// rollback auditable rather than a mystery edit.
	SourceVersionID *pulid.ID `json:"sourceVersionId" bun:"source_version_id,type:VARCHAR(100),nullzero"`

	VersionNumber int64         `json:"versionNumber" bun:"version_number,type:BIGINT,notnull"`
	Status        VersionStatus `json:"status"        bun:"status,type:document_template_version_status_enum,notnull,default:'Draft'"`

	Subject    string `json:"subject"    bun:"subject,type:TEXT,nullzero"`
	BodyHTML   string `json:"bodyHtml"   bun:"body_html,type:TEXT,nullzero"`
	BodyText   string `json:"bodyText"   bun:"body_text,type:TEXT,nullzero"`
	CSSContent string `json:"cssContent" bun:"css_content,type:TEXT,nullzero"`
	HeaderHTML string `json:"headerHtml" bun:"header_html,type:TEXT,nullzero"`
	FooterHTML string `json:"footerHtml" bun:"footer_html,type:TEXT,nullzero"`

	PageSize     PageSize    `json:"pageSize"     bun:"page_size,type:document_template_page_size_enum,nullzero"`
	Orientation  Orientation `json:"orientation"  bun:"orientation,type:document_template_orientation_enum,nullzero"`
	MarginTop    *float64    `json:"marginTop"    bun:"margin_top,type:NUMERIC(6,2),nullzero"`
	MarginBottom *float64    `json:"marginBottom" bun:"margin_bottom,type:NUMERIC(6,2),nullzero"`
	MarginLeft   *float64    `json:"marginLeft"   bun:"margin_left,type:NUMERIC(6,2),nullzero"`
	MarginRight  *float64    `json:"marginRight"  bun:"margin_right,type:NUMERIC(6,2),nullzero"`

	// ContentHash identifies these bytes. A generated document records it alongside
	// the version id so a mismatch is detectable even if a row were tampered with.
	ContentHash string `json:"contentHash" bun:"content_hash,type:VARCHAR(64),notnull"`

	// StarterHash is the built-in's hash at the time this version was seeded from
	// it, so the editor can say "the built-in has changed since you started".
	StarterHash string `json:"starterHash" bun:"starter_hash,type:VARCHAR(64),nullzero"`

	PublishNotes  string    `json:"publishNotes"  bun:"publish_notes,type:TEXT,nullzero"`
	PublishedByID *pulid.ID `json:"publishedById" bun:"published_by_id,type:VARCHAR(100),nullzero"`
	PublishedAt   *int64    `json:"publishedAt"   bun:"published_at,type:BIGINT,nullzero"`
	ArchivedByID  *pulid.ID `json:"archivedById"  bun:"archived_by_id,type:VARCHAR(100),nullzero"`
	ArchivedAt    *int64    `json:"archivedAt"    bun:"archived_at,type:BIGINT,nullzero"`

	Version     int64     `json:"version"     bun:"version,type:BIGINT,notnull"`
	CreatedAt   int64     `json:"createdAt"   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt   int64     `json:"updatedAt"   bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	CreatedByID *pulid.ID `json:"createdById" bun:"created_by_id,type:VARCHAR(100),nullzero"`
	UpdatedByID *pulid.ID `json:"updatedById" bun:"updated_by_id,type:VARCHAR(100),nullzero"`

	BusinessUnit *tenant.BusinessUnit `json:"-"                  bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-"                  bun:"rel:belongs-to,join:organization_id=id"`
	Template     *DocumentTemplate    `json:"template,omitempty" bun:"rel:belongs-to,join:template_id=id"`
}

func (v *DocumentTemplateVersion) GetID() pulid.ID { return v.ID }

func (v *DocumentTemplateVersion) GetCreatedAt() int64 { return v.CreatedAt }

func (v *DocumentTemplateVersion) GetOrganizationID() pulid.ID { return v.OrganizationID }

func (v *DocumentTemplateVersion) GetBusinessUnitID() pulid.ID { return v.BusinessUnitID }

func (v *DocumentTemplateVersion) GetTableName() string { return "document_template_versions" }

// IsEditable reports whether this version may still be changed. Publishing freezes
// it, because a generated document names it as the origin of bytes already sent.
func (v *DocumentTemplateVersion) IsEditable() bool {
	return v.Status == VersionStatusDraft
}

// Margins resolves the page box, falling back to the default when the version does
// not state one.
func (v *DocumentTemplateVersion) Margins() Margins {
	fallback := DefaultMargins()

	return Margins{
		Top:    floatOr(v.MarginTop, fallback.Top),
		Bottom: floatOr(v.MarginBottom, fallback.Bottom),
		Left:   floatOr(v.MarginLeft, fallback.Left),
		Right:  floatOr(v.MarginRight, fallback.Right),
	}
}

// ResolvedPageSize falls back to Letter rather than a zero value, which would ask
// the renderer for a page with no area.
func (v *DocumentTemplateVersion) ResolvedPageSize() PageSize {
	if v.PageSize.IsValid() {
		return v.PageSize
	}
	return PageSizeLetter
}

// ResolvedOrientation falls back to portrait.
func (v *DocumentTemplateVersion) ResolvedOrientation() Orientation {
	if v.Orientation.IsValid() {
		return v.Orientation
	}
	return OrientationPortrait
}

// Channel returns the stored source for one channel, so callers do not have to know
// which column a channel lives in.
func (v *DocumentTemplateVersion) Channel(channel Channel) string {
	switch channel {
	case ChannelPDF, ChannelEmailHTML:
		return v.BodyHTML
	case ChannelEmailText:
		return v.BodyText
	case ChannelSubject, ChannelNotificationTitle:
		return v.Subject
	case ChannelNotificationBody:
		return v.BodyText
	default:
		return ""
	}
}

// SetChannel writes one channel's source.
func (v *DocumentTemplateVersion) SetChannel(channel Channel, source string) {
	switch channel {
	case ChannelPDF, ChannelEmailHTML:
		v.BodyHTML = source
	case ChannelEmailText, ChannelNotificationBody:
		v.BodyText = source
	case ChannelSubject, ChannelNotificationTitle:
		v.Subject = source
	default:
	}
}

// HasContent reports whether anything renderable is stored. A version with nothing
// in it would print a blank page.
func (v *DocumentTemplateVersion) HasContent() bool {
	return strings.TrimSpace(v.BodyHTML) != "" ||
		strings.TrimSpace(v.BodyText) != "" ||
		strings.TrimSpace(v.Subject) != ""
}

func (v *DocumentTemplateVersion) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(v,
		validation.Field(&v.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&v.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&v.TemplateID, validation.Required.Error("Template is required")),
		validation.Field(&v.VersionNumber,
			validation.Min(int64(0)).Error("Version number cannot be negative"),
		),
		validation.Field(&v.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[VersionStatus]("Status is not a valid version status"),
		),
		validation.Field(&v.BodyHTML,
			validation.When(!v.HasContent(), validation.Required.Error(
				"A version must have content in at least one of its channels",
			)),
		),
		validation.Field(&v.ContentHash,
			validation.Required.Error("Content hash is required"),
			validation.Length(1, MaxHashLength).
				Error("Content hash cannot be longer than 64 characters"),
		),
		validation.Field(&v.StarterHash,
			validation.Length(0, MaxHashLength).
				Error("Starter hash cannot be longer than 64 characters"),
		),
		validation.Field(&v.PageSize,
			domainvalidation.ValidEnum[PageSize]("Page size is not supported"),
		),
		validation.Field(&v.Orientation,
			domainvalidation.ValidEnum[Orientation]("Orientation is not supported"),
		),
		validation.Field(&v.MarginTop, marginRules()...),
		validation.Field(&v.MarginBottom, marginRules()...),
		validation.Field(&v.MarginLeft, marginRules()...),
		validation.Field(&v.MarginRight, marginRules()...),
		// These mirror the table's CHECK constraints, so a lifecycle field left
		// half-set is a field error rather than an opaque constraint violation.
		validation.Field(&v.PublishedByID,
			validation.When(v.PublishedAt != nil, validation.Required.Error(
				"A published version must record who published it",
			)),
		),
		validation.Field(&v.PublishedAt,
			validation.When(
				v.PublishedByID != nil || v.Status == VersionStatusActive,
				validation.Required.Error("A published version must record when it went live"),
			),
		),
		validation.Field(&v.ArchivedAt,
			validation.When(v.Status != VersionStatusArchived, validation.Nil.Error(
				"Only an archived version can record when it was archived",
			)),
		),
	))
}

func marginRules() []validation.Rule {
	return []validation.Rule{
		validation.Min(0.0).Error("Margin must be between 0 and 100 millimeters"),
		validation.Max(float64(maxMarginMillimeters)).
			Error("Margin must be between 0 and 100 millimeters"),
	}
}

func (v *DocumentTemplateVersion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if v.ID.IsNil() {
			v.ID = pulid.MustNew("dtv_")
		}
		v.CreatedAt = now
		v.UpdatedAt = now
	case *bun.UpdateQuery:
		v.UpdatedAt = now
	}

	return nil
}

func floatOr(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}
