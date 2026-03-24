package glaccount

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*GLAccount)(nil)
	_ validationframework.TenantedEntity = (*GLAccount)(nil)
	_ domaintypes.PostgresSearchable     = (*GLAccount)(nil)
)

type GLAccount struct {
	bun.BaseModel `bun:"table:gl_accounts,alias:gla" json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Status         domaintypes.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	AccountTypeID  pulid.ID           `json:"accountTypeId"  bun:"account_type_id,type:VARCHAR(100),notnull"`
	ParentID       pulid.ID           `json:"parentId"       bun:"parent_id,type:VARCHAR(100),nullzero"`
	AccountCode    string             `json:"accountCode"    bun:"account_code,type:VARCHAR(20),notnull"`
	Name           string             `json:"name"           bun:"name,type:VARCHAR(200),notnull"`
	Description    string             `json:"description"    bun:"description,type:TEXT,nullzero"`
	IsSystem       bool               `json:"isSystem"       bun:"is_system,type:BOOLEAN,default:false"`
	AllowManualJE  bool               `json:"allowManualJe"  bun:"allow_manual_je,type:BOOLEAN,default:true"`
	RequireProject bool               `json:"requireProject" bun:"require_project,type:BOOLEAN,default:false"`
	CurrentBalance int64              `json:"currentBalance" bun:"current_balance,type:BIGINT,default:0"`
	DebitBalance   int64              `json:"debitBalance"   bun:"debit_balance,type:BIGINT,default:0"`
	CreditBalance  int64              `json:"creditBalance"  bun:"credit_balance,type:BIGINT,default:0"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector   string             `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string             `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit     `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization     `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	AccountType  *accounttype.AccountType `json:"accountType,omitempty"  bun:"rel:belongs-to,join:account_type_id=id"`
	Parent       *GLAccount               `json:"parent,omitempty"       bun:"rel:belongs-to,join:parent_id=id"`
	Children     []*GLAccount             `json:"children,omitempty"     bun:"rel:has-many,join:id=parent_id"`
}

func (g *GLAccount) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(g,
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
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
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

func (g *GLAccount) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if g.ID.IsNil() {
			g.ID = pulid.MustNew("gla_")
		}

		g.CreatedAt = now
	case *bun.UpdateQuery:
		g.UpdatedAt = now
	}

	return nil
}
