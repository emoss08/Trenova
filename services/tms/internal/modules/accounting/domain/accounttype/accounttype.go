package accounttype

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var (
	_ validationframework.TenantedEntity = (*AccountType)(nil)
	_ domaintypes.PostgresSearchable     = (*AccountType)(nil)
)

// Asset, liability, equity, revenue and expense classification
type AccountType struct {
	keel.Table `keel:"account_types,report=Accounting"`

	ID             pulid.ID           `json:"id"             keel:"id,pk,autoid:at_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	Status         domaintypes.Status `json:"status"         keel:"status"`
	Code           string             `json:"code"           keel:"code"`
	Name           string             `json:"name"           keel:"name"`
	Description    string             `json:"description"    keel:"description,nullzero"`
	Category       Category           `json:"category"       keel:"category,label=Account Category"`
	Color          string             `json:"color"          keel:"color,nullzero"`
	IsSystem       bool               `json:"isSystem"       keel:"is_system,label=System Type"`
	Version        int64              `json:"version"        keel:"version,version"`
	CreatedAt      int64              `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt"      keel:"updated_at,autoupdate"`
	SearchVector   string             `json:"-"              keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string             `json:"-"              keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
}

func (a *AccountType) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.Code,
			validation.Required.Error("Code is required"),
			validation.Length(3, 10).Error("Code must be between 3 and 10 characters"),
		),
		validation.Field(&a.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&a.Category,
			validation.Required.Error("Category is required"),
			validation.In(
				CategoryAsset,
				CategoryLiability,
				CategoryEquity,
				CategoryRevenue,
				CategoryCostOfRevenue,
				CategoryExpense,
			).Error("Category must be a valid account category"),
		),
		validation.Field(&a.Color,
			is.HexColor.Error("Color must be a valid hex color"),
		),
		validation.Field(&a.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				domaintypes.StatusActive,
				domaintypes.StatusInactive,
			).Error("Status must be either Active or Inactive"),
		),
	))
}

func (a *AccountType) GetID() pulid.ID {
	return a.ID
}

func (a *AccountType) GetCreatedAt() int64 {
	return a.CreatedAt
}

func (a *AccountType) GetTableName() string {
	return "account_types"
}

func (a *AccountType) GetOrganizationID() pulid.ID {
	return a.OrganizationID
}

func (a *AccountType) GetBusinessUnitID() pulid.ID {
	return a.BusinessUnitID
}

func (a *AccountType) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "at",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "code",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (a *AccountType) BeforeCreate(ctx context.Context) error {
	if a.Status == "" {
		a.Status = domaintypes.StatusActive
	}

	return nil
}

func (AccountType) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_account_types_unique_code",
			Message: "Account type with this code already exists in your organization",
		},
	}
}
