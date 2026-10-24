package glaccount

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var (
	_ validationframework.TenantedEntity = (*GLAccount)(nil)
	_ domaintypes.PostgresSearchable     = (*GLAccount)(nil)
)

// The chart of accounts and its running balances
type GLAccount struct {
	keel.Table `                                           keel:"gl_accounts,report=Accounting"`

	ID             pulid.ID           `json:"id" keel:"id,pk,autoid:gla_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	Status         domaintypes.Status `json:"status" keel:"status"`
	AccountTypeID  pulid.ID           `json:"accountTypeId" keel:"account_type_id"`
	ParentID       pulid.ID           `json:"parentId" keel:"parent_id,nullzero"`
	AccountCode    string             `json:"accountCode" keel:"account_code"`
	Name           string             `json:"name" keel:"name,label=Account Name"`
	Description    string             `json:"description" keel:"description,nullzero"`
	IsSystem       bool               `json:"isSystem" keel:"is_system,label=System Account"`
	AllowManualJE  bool               `json:"allowManualJe" keel:"allow_manual_je,label=Allows Manual Journals"`
	RequireProject bool               `json:"requireProject" keel:"require_project,label=Requires Project"`
	CurrentBalance int64              `json:"currentBalance" keel:"current_balance,label=Current Balance (Minor)"`
	DebitBalance   int64              `json:"debitBalance" keel:"debit_balance,label=Debit Balance (Minor)"`
	CreditBalance  int64              `json:"creditBalance" keel:"credit_balance,label=Credit Balance (Minor)"`
	Version        int64              `json:"version" keel:"version,version"`
	CreatedAt      int64              `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector   string             `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string             `json:"-" keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit     `json:"businessUnit,omitempty"`
	Organization *tenant.Organization     `json:"organization,omitempty"`
	AccountType  *accounttype.AccountType `json:"accountType,omitempty" keel:"belongs:account_type_id"`
	Parent       *GLAccount               `json:"parent,omitempty" keel:"belongs:parent_id,label=Parent Account"`
	Children     []*GLAccount             `json:"children,omitempty" keel:"many:parent_id,label=Child Accounts"`
}

func (g *GLAccount) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(g,
		validation.Field(&g.AccountCode,
			validation.Required.Error("Account code is required"),
			validation.Length(1, 20).Error("Account code must be between 1 and 20 characters"),
		),
		validation.Field(&g.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 200).Error("Name must be between 1 and 200 characters"),
			is.PrintableASCII.Error("Name must contain only printable ASCII characters"),
		),
		validation.Field(&g.AccountTypeID,
			validation.Required.Error("Account type is required"),
		),
		validation.Field(&g.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				domaintypes.StatusActive,
				domaintypes.StatusInactive,
			).Error("Status must be either Active or Inactive"),
		),
	))
}

func (g *GLAccount) GetID() pulid.ID {
	return g.ID
}

func (g *GLAccount) GetTableName() string {
	return "gl_accounts"
}

func (g *GLAccount) GetOrganizationID() pulid.ID {
	return g.OrganizationID
}

func (g *GLAccount) GetBusinessUnitID() pulid.ID {
	return g.BusinessUnitID
}

func (g *GLAccount) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "gla",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "account_code",
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

func (g *GLAccount) BeforeCreate(ctx context.Context) error {
	if g.Status == "" {
		g.Status = domaintypes.StatusActive
	}

	return nil
}
