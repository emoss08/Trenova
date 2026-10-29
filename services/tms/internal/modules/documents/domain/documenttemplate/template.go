package documenttemplate

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
)

var (
	_ validationframework.TenantedEntity = (*DocumentTemplate)(nil)
	_ domaintypes.PostgresSearchable     = (*DocumentTemplate)(nil)
)

// DocumentTemplate is a named, versioned template an organization owns.
//
// It carries identity only. Content lives on the versions, which is what lets an
// administrator edit without changing what customers are currently receiving.
type DocumentTemplate struct {
	keel.Table `keel:"document_templates"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:dtpl_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	// Kind names an entry in the Go registry. It is a plain string in the database
	// so adding a kind never needs a migration.
	Kind Kind `json:"kind" keel:"kind"`

	Code        string `json:"code"        keel:"code"`
	Name        string `json:"name"        keel:"name"`
	Description string `json:"description" keel:"description,nullzero"`

	// IsOrgDefault marks the template every customer without an assignment gets.
	// A partial unique index enforces one per kind.
	IsOrgDefault bool `json:"isOrgDefault" keel:"is_org_default"`

	// ActiveVersionID is what renders. Nil means the template has only drafts and
	// resolution therefore falls through to the built-in.
	ActiveVersionID *pulid.ID `json:"activeVersionId" keel:"active_version_id"`

	Version     int64     `json:"version"     keel:"version,version"`
	CreatedAt   int64     `json:"createdAt"   keel:"created_at,autocreate"`
	UpdatedAt   int64     `json:"updatedAt"   keel:"updated_at,autoupdate"`
	CreatedByID *pulid.ID `json:"createdById" keel:"created_by_id"`
	UpdatedByID *pulid.ID `json:"updatedById" keel:"updated_by_id"`

	SearchVector string `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-" keel:"rank,computed"`

	BusinessUnit  *tenant.BusinessUnit          `json:"-"`
	Organization  *tenant.Organization          `json:"-"`
	ActiveVersion *DocumentTemplateVersion      `json:"activeVersion,omitempty" keel:"belongs:active_version_id"`
	Versions      []*DocumentTemplateVersion    `json:"versions,omitempty"      keel:"many:template_id"`
	Assignments   []*DocumentTemplateAssignment `json:"assignments,omitempty"   keel:"many:template_id"`
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

func (t *DocumentTemplate) BeforeCreate(ctx context.Context) error {
	t.Normalize()

	return nil
}

func (t *DocumentTemplate) BeforeUpdate(_ context.Context) error {
	t.Normalize()
	t.Normalize()

	return nil
}

func (DocumentTemplate) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_document_templates_code",
			Message: "A document template with this code already exists",
		},
		{
			Name:    "uq_document_templates_org_default",
			Field:   "isOrgDefault",
			Message: "Another template is already the organization default for this kind",
		},
		{
			Name:    "fk_document_templates_active_version",
			Message: "Active version does not exist",
		},
	}
}

// ErrTemplateKindUnknown is returned when a template names a kind the registry does
// not publish, which would leave it unrenderable.
var ErrTemplateKindUnknown = errors.New("template kind is not registered")
