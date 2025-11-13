package accounting

import (
	"context"
	"errors"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*GLAccount)(nil)
	_ domaintypes.PostgresSearchable = (*GLAccount)(nil)
	_ domain.Validatable             = (*GLAccount)(nil)
	_ framework.TenantedEntity       = (*GLAccount)(nil)
)

type GLAccount struct {
	bun.BaseModel `bun:"table:gl_accounts,alias:gla" json:"-"`

	ID             pulid.ID      `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID      `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID      `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Status         domain.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	AccountTypeID  pulid.ID      `json:"accountTypeId"  bun:"account_type_id,type:VARCHAR(100),notnull"`
	ParentID       *pulid.ID     `json:"parentId"       bun:"parent_id,type:VARCHAR(100),nullzero"`

	AccountCode string `json:"accountCode" bun:"account_code,type:VARCHAR(20),notnull"`
	Name        string `json:"name"        bun:"name,type:VARCHAR(200),notnull"`
	Description string `json:"description" bun:"description,type:TEXT,nullzero"`

	// Account Properties
	IsSystem       bool `json:"isSystem"       bun:"is_system,type:BOOLEAN,notnull,default:false"`
	AllowManualJE  bool `json:"allowManualJE"  bun:"allow_manual_je,type:BOOLEAN,notnull,default:true"`
	RequireProject bool `json:"requireProject" bun:"require_project,type:BOOLEAN,notnull,default:false"`

	// Balance Tracking (denormalized for performance)
	CurrentBalance int64 `json:"currentBalance" bun:"current_balance,type:BIGINT,notnull,default:0"` // In cents
	DebitBalance   int64 `json:"debitBalance"   bun:"debit_balance,type:BIGINT,notnull,default:0"`   // In cents
	CreditBalance  int64 `json:"creditBalance"  bun:"credit_balance,type:BIGINT,notnull,default:0"`  // In cents

	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	AccountType  *AccountType         `json:"accountType,omitempty"  bun:"rel:belongs-to,join:account_type_id=id"`
	Parent       *GLAccount           `json:"parent,omitempty"       bun:"rel:belongs-to,join:parent_id=id"`
	Children     []*GLAccount         `json:"children,omitempty"     bun:"rel:has-many,join:id=parent_id"`
}

func (gla *GLAccount) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		gla,
		validation.Field(&gla.AccountCode,
			validation.Required.Error("Account code is required"),
			validation.Length(1, 20).Error("Account code must be between 1 and 20 characters"),
		),
		validation.Field(
			&gla.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 200).Error("Name must be between 1 and 200 characters"),
			validation.Match(regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9\-\.]*[A-Za-z0-9])?$`)).
				Error("Name must be alphanumeric and may contain hyphens or dots"),
		),
		validation.Field(&gla.AccountTypeID,
			validation.Required.Error("Account type is required"),
		),
		validation.Field(&gla.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				domain.StatusActive,
				domain.StatusInactive,
			).Error("Status must be a valid status"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (gla *GLAccount) GetID() string {
	return gla.ID.String()
}

func (gla *GLAccount) GetTableName() string {
	return "gl_accounts"
}

func (gla *GLAccount) GetOrganizationID() pulid.ID {
	return gla.OrganizationID
}

func (gla *GLAccount) GetBusinessUnitID() pulid.ID {
	return gla.BusinessUnitID
}

func (gla *GLAccount) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "gla",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "account_code",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (gla *GLAccount) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if gla.ID.IsNil() {
			gla.ID = pulid.MustNew("gla_")
		}

		gla.CreatedAt = now
	case *bun.UpdateQuery:
		gla.UpdatedAt = now
	}

	return nil
}
