package accounting

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*AccountType)(nil)
	_ domaintypes.PostgresSearchable = (*AccountType)(nil)
	_ domain.Validatable             = (*AccountType)(nil)
	_ framework.TenantedEntity       = (*AccountType)(nil)
)

type AccountType struct {
	bun.BaseModel `bun:"table:account_types,alias:at" json:"-"`

	ID             pulid.ID      `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID      `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID      `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Status         domain.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	Code           string        `json:"code"           bun:"code,type:VARCHAR(10),notnull"`
	Name           string        `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Description    string        `json:"description"    bun:"description,type:TEXT,nullzero"`
	Category       Category      `json:"category"       bun:"category,type:account_category_enum,notnull"`
	Color          string        `json:"color"          bun:"color,type:VARCHAR(10),nullzero"`
	IsSystem       bool          `json:"isSystem"       bun:"is_system,type:BOOLEAN,notnull,default:false"`
	Version        int64         `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64         `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64         `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector   string        `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string        `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (at *AccountType) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		at,
		validation.Field(
			&at.Code,
			validation.Required.Error("Code is required"),
			validation.Length(3, 10).Error("Code must be between 3 and 10 characters"),
		),
		validation.Field(
			&at.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(
			&at.Category,
			validation.Required.Error("Category is required"),
			validation.In(
				CategoryAsset,
				CategoryLiability,
				CategoryEquity,
				CategoryRevenue,
				CategoryCostOfRevenue,
				CategoryExpense,
			).Error("Category must be a valid category"),
		),
		validation.Field(
			&at.Color,
			is.HexColor.Error("Color must be a valid hex color. Please try again."),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (at *AccountType) GetID() string {
	return at.ID.String()
}

func (at *AccountType) GetTableName() string {
	return "account_types"
}

func (at *AccountType) GetOrganizationID() pulid.ID {
	return at.OrganizationID
}

func (at *AccountType) GetBusinessUnitID() pulid.ID {
	return at.BusinessUnitID
}

func (at *AccountType) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "at",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "category", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (at *AccountType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if at.ID.IsNil() {
			at.ID = pulid.MustNew("at_")
		}

		at.CreatedAt = now
	case *bun.UpdateQuery:
		at.UpdatedAt = now
	}
	return nil
}
