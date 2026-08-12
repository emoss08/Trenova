package documenttemplate

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	// MaxCodeLength and friends mirror the column widths, so a value that would be
	// truncated by the database is refused with a field error the form can show
	// instead of failing on insert.
	MaxCodeLength        = 50
	MaxNameLength        = 200
	MaxDescriptionLength = 2000
	MaxKindLength        = 100
	MaxHashLength        = 64

	MaxReferenceTypeLength = 50
	MaxFileNameLength      = 255
	MaxFilePathLength      = 500
	MaxMimeTypeLength      = 100
	MaxDeliveredToLength   = 255
)

var (
	_ bun.BeforeAppendModelHook          = (*DocumentTemplate)(nil)
	_ validationframework.TenantedEntity = (*DocumentTemplate)(nil)
	_ domaintypes.PostgresSearchable     = (*DocumentTemplate)(nil)
)

// DocumentTemplate is a named, versioned template an organization owns.
//
// It carries identity only. Content lives on the versions, which is what lets an
// administrator edit without changing what customers are currently receiving.
type DocumentTemplate struct {
	bun.BaseModel             `bun:"table:document_templates,alias:dtpl" json:"-"`
	pagination.CursorValueSet `bun:",embed"                              json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	// Kind names an entry in the Go registry. It is a plain string in the database
	// so adding a kind never needs a migration.
	Kind Kind `json:"kind" bun:"kind,type:VARCHAR(100),notnull"`

	Code        string `json:"code"        bun:"code,type:VARCHAR(50),notnull"`
	Name        string `json:"name"        bun:"name,type:VARCHAR(200),notnull"`
	Description string `json:"description" bun:"description,type:TEXT,nullzero"`

	// IsOrgDefault marks the template every customer without an assignment gets.
	// A partial unique index enforces one per kind.
	IsOrgDefault bool `json:"isOrgDefault" bun:"is_org_default,type:BOOLEAN,notnull"`

	// ActiveVersionID is what renders. Nil means the template has only drafts and
	// resolution therefore falls through to the built-in.
	ActiveVersionID *pulid.ID `json:"activeVersionId" bun:"active_version_id,type:VARCHAR(100),nullzero"`

	Version     int64     `json:"version"     bun:"version,type:BIGINT,notnull"`
	CreatedAt   int64     `json:"createdAt"   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt   int64     `json:"updatedAt"   bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	CreatedByID *pulid.ID `json:"createdById" bun:"created_by_id,type:VARCHAR(100),nullzero"`
	UpdatedByID *pulid.ID `json:"updatedById" bun:"updated_by_id,type:VARCHAR(100),nullzero"`

	SearchVector string `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-" bun:"rank,type:VARCHAR(100),scanonly"`

	BusinessUnit  *tenant.BusinessUnit          `json:"-"                       bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization  *tenant.Organization          `json:"-"                       bun:"rel:belongs-to,join:organization_id=id"`
	ActiveVersion *DocumentTemplateVersion      `json:"activeVersion,omitempty" bun:"rel:belongs-to,join:active_version_id=id"`
	Versions      []*DocumentTemplateVersion    `json:"versions,omitempty"      bun:"rel:has-many,join:id=template_id"`
	Assignments   []*DocumentTemplateAssignment `json:"assignments,omitempty"   bun:"rel:has-many,join:id=template_id"`
}

func (t *DocumentTemplate) GetID() pulid.ID { return t.ID }

func (t *DocumentTemplate) GetCreatedAt() int64 { return t.CreatedAt }

func (t *DocumentTemplate) GetOrganizationID() pulid.ID { return t.OrganizationID }

func (t *DocumentTemplate) GetBusinessUnitID() pulid.ID { return t.BusinessUnitID }

func (t *DocumentTemplate) GetTableName() string { return "document_templates" }

func (t *DocumentTemplate) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dtpl",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

// HasActiveVersion reports whether this template currently renders anything.
func (t *DocumentTemplate) HasActiveVersion() bool {
	return t.ActiveVersionID != nil && !t.ActiveVersionID.IsNil()
}

// Validate checks the template's identity. Content validation belongs to the
// version, and the registry decides whether the kind exists — the domain cannot
// import the registry into a method without a cycle, so the service passes the
// verdict in via ValidateWithRegistry.
func (t *DocumentTemplate) Validate(multiErr *errortypes.MultiError) {
	t.Normalize()

	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&t.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&t.Kind,
			validation.Required.Error("Template kind is required"),
			validation.Length(1, MaxKindLength).
				Error("Template kind cannot be longer than 100 characters"),
		),
		validation.Field(&t.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, MaxCodeLength).
				Error("Code cannot be longer than 50 characters"),
		),
		validation.Field(&t.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, MaxNameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&t.Description,
			validation.Length(0, MaxDescriptionLength).
				Error("Description cannot be longer than 2000 characters"),
		),
	))
}

// Normalize puts the identity fields in the shape the unique index compares, so
// validation judges the same value the database will store.
func (t *DocumentTemplate) Normalize() {
	t.Code = strings.ToUpper(strings.TrimSpace(t.Code))
	t.Name = strings.TrimSpace(t.Name)
	t.Description = strings.TrimSpace(t.Description)
	t.Kind = Kind(strings.TrimSpace(string(t.Kind)))
}

func (t *DocumentTemplate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	t.Normalize()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("dtpl_")
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}

// ErrTemplateKindUnknown is returned when a template names a kind the registry does
// not publish, which would leave it unrenderable.
var ErrTemplateKindUnknown = errors.New("template kind is not registered")
