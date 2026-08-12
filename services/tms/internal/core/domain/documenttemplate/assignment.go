package documenttemplate

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
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

var (
	_ bun.BeforeAppendModelHook          = (*DocumentTemplateAssignment)(nil)
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
	bun.BaseModel             `bun:"table:document_template_assignments,alias:dta" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                        json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	// Kind is denormalized from the template so resolution can filter on it without
	// joining, and so a unique index can enforce one assignment per customer per
	// kind. BeforeAppendModel keeps it in step with the template.
	Kind Kind `json:"kind" bun:"kind,type:VARCHAR(100),notnull"`

	TemplateID pulid.ID `json:"templateId" bun:"template_id,type:VARCHAR(100),notnull"`
	CustomerID pulid.ID `json:"customerId" bun:"customer_id,type:VARCHAR(100),notnull"`

	AssignedByID *pulid.ID `json:"assignedById" bun:"assigned_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"-"                  bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-"                  bun:"rel:belongs-to,join:organization_id=id"`
	Template     *DocumentTemplate    `json:"template,omitempty" bun:"rel:belongs-to,join:template_id=id"`
	Customer     *customer.Customer   `json:"customer,omitempty" bun:"rel:belongs-to,join:customer_id=id"`
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

func (a *DocumentTemplateAssignment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("dta_")
		}
		a.CreatedAt = now
		a.UpdatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
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
