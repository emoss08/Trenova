package documenttemplate

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*DocumentTemplateAssignment)(nil)
	_ domaintypes.PostgresSearchable     = (*DocumentTemplateAssignment)(nil)
)

// DocumentTemplateAssignment points one customer at one template for one kind.
//
// Assignment replaces the whole template rather than merging pieces of it, which is
// simple to reason about and to preview: what the customer receives is exactly the
// assigned template's active version. The cost is that an assigned customer stops
// inheriting later changes to the organization default, which is why publishing
// shows how many customers are unaffected.
type DocumentTemplateAssignment struct {
	keel.Table `keel:"document_template_assignments"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:dta_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	// Kind is denormalized from the template so resolution can filter on it without
	// joining, and so a unique index can enforce one assignment per customer per
	// kind. BeforeAppendModel keeps it in step with the template.
	Kind Kind `json:"kind" keel:"kind"`

	TemplateID pulid.ID `json:"templateId" keel:"template_id"`
	CustomerID pulid.ID `json:"customerId" keel:"customer_id"`

	AssignedByID *pulid.ID `json:"assignedById" keel:"assigned_by_id"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
	Template     *DocumentTemplate    `json:"template,omitempty" keel:"belongs:template_id"`
	Customer     *customer.Customer   `json:"customer,omitempty" keel:"belongs:customer_id"`
}

func (a *DocumentTemplateAssignment) GetID() pulid.ID { return a.ID }

func (a *DocumentTemplateAssignment) GetCreatedAt() int64 { return a.CreatedAt }

func (a *DocumentTemplateAssignment) GetOrganizationID() pulid.ID { return a.OrganizationID }

func (a *DocumentTemplateAssignment) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }

func (a *DocumentTemplateAssignment) GetTableName() string {
	return "document_template_assignments"
}

// GetPostgresSearchConfig searches the only text the row owns. Customer and
// template names live on the joined rows, so the assignment list searches by kind
// and filters by customer rather than pretending to full-text those names.
func (a *DocumentTemplateAssignment) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dta",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "kind", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
	}
}

func (a *DocumentTemplateAssignment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&a.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&a.Kind,
			validation.Required.Error("Template kind is required"),
			validation.Length(1, MaxKindLength).
				Error("Template kind cannot be longer than 100 characters"),
		),
		validation.Field(&a.TemplateID, validation.Required.Error("Template is required")),
		validation.Field(&a.CustomerID, validation.Required.Error("Customer is required")),
	))
}

// TemplateSource says which tier of the resolution chain produced a template.
//
// It travels with a resolved template so a preview can tell an administrator why
// they are looking at what they are looking at, and so a generated document records
// whether it came from a customer override, the organization default, or the
// built-in.
type TemplateSource string

const (
	// SourceAssigned means a per-customer assignment matched.
	SourceAssigned = TemplateSource("Assigned")
	// SourceOrgDefault means the organization's default for the kind was used.
	SourceOrgDefault = TemplateSource("OrgDefault")
	// SourceBuiltIn means the organization has customized nothing, so the embedded
	// starter rendered.
	SourceBuiltIn = TemplateSource("BuiltIn")
)

func (s TemplateSource) String() string { return string(s) }

func (s TemplateSource) IsValid() bool {
	switch s {
	case SourceAssigned, SourceOrgDefault, SourceBuiltIn:
		return true
	default:
		return false
	}
}
