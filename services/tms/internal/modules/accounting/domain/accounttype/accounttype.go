package accounttype

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*AccountType)(nil)
	_ validationframework.TenantedEntity = (*AccountType)(nil)
	_ domaintypes.PostgresSearchable     = (*AccountType)(nil)
)

// Asset, liability, equity, revenue and expense classification
type AccountType struct {
	bun.BaseModel `bun:"table:account_types,alias:at" json:"-"`
	keel.Table    `                                            keel:"account_types,report=Accounting"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID           `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"                                          keel:"id,pk,autoid:at_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"                            keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"                             keel:"organization_id,pk,tenant"`
	Status         domaintypes.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"                         keel:"status"`
	Code           string             `json:"code"           bun:"code,type:VARCHAR(10),notnull"                                            keel:"code"`
	Name           string             `json:"name"           bun:"name,type:VARCHAR(100),notnull"                                           keel:"name"`
	Description    string             `json:"description"    bun:"description,type:TEXT,nullzero"                                           keel:"description,nullzero"`
	Category       Category           `json:"category"       bun:"category,type:account_category_enum,notnull"                              keel:"category,label=Account Category"`
	Color          string             `json:"color"          bun:"color,type:VARCHAR(10),nullzero"                                          keel:"color,nullzero"`
	IsSystem       bool               `json:"isSystem"       bun:"is_system,type:BOOLEAN"                                                   keel:"is_system,label=System Type"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT"                                                      keel:"version,version"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
	SearchVector   string             `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"                                     keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string             `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"                                          keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
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

func (a *AccountType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("at_")
		}

		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}
