package tableconfiguration

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*TableConfiguration)(nil)
	_ domaintypes.PostgresSearchable     = (*TableConfiguration)(nil)
)

type ColumnPinning struct {
	Left  []string `json:"left"`
	Right []string `json:"right"`
}

type FormatRule struct {
	ID       string `json:"id"`
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
	Color    string `json:"color"`
}

type TableConfig struct {
	FilterGroups     []domaintypes.FilterGroup `json:"filterGroups"`
	FieldFilters     []domaintypes.FieldFilter `json:"fieldFilters"`
	JoinOperator     string                    `json:"joinOperator"`
	Sort             []domaintypes.SortField   `json:"sort"`
	PageSize         int                       `json:"pageSize"`
	ColumnVisibility map[string]bool           `json:"columnVisibility"`
	ColumnOrder      []string                  `json:"columnOrder"`
	ColumnSizing     map[string]float64        `json:"columnSizing"`
	ColumnPinning    *ColumnPinning            `json:"columnPinning"`
	Density          string                    `json:"density"`
	FormatRules      []FormatRule              `json:"formatRules"`
}

type TableConfiguration struct {
	keel.Table `keel:"table_configurations"`

	ID             pulid.ID     `json:"id"             keel:"id,pk,autoid:tc_"`
	OrganizationID pulid.ID     `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID     `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	UserID         pulid.ID     `json:"userId"         keel:"user_id"`
	Name           string       `json:"name"           keel:"name"`
	Description    string       `json:"description"    keel:"description,nullzero"`
	Resource       string       `json:"resource"       keel:"resource"`
	TableConfig    *TableConfig `json:"tableConfig"    keel:"table_config,jsonb"`
	Visibility     Visibility   `json:"visibility"     keel:"visibility"`
	SearchVector   string       `json:"-"              keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string       `json:"-"              keel:"rank,computed"`
	IsDefault      bool         `json:"isDefault"      keel:"is_default"`
	IsOrgDefault   bool         `json:"isOrgDefault"   keel:"is_org_default"`
	Version        int64        `json:"version"        keel:"version,version"`
	CreatedAt      int64        `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64        `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	User         *tenant.User         `json:"user,omitempty"         keel:"belongs:user_id"`
}

func (tc *TableConfiguration) BeforeCreate(_ context.Context) error {
	if tc.Visibility == "" {
		tc.Visibility = VisibilityPrivate
	}

	return nil
}

func (tc *TableConfiguration) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(tc,
		validation.Field(&tc.Name, validation.Required, validation.Length(1, 255)),
		validation.Field(&tc.Resource, validation.Required, validation.Length(1, 100)),
		validation.Field(&tc.TableConfig, validation.Required),
		validation.Field(&tc.Visibility, validation.Required, validation.In(
			VisibilityPrivate,
			VisibilityPublic,
			VisibilityShared,
		)),
	))
}

func (tc *TableConfiguration) GetID() pulid.ID {
	return tc.ID
}

func (tc *TableConfiguration) GetCreatedAt() int64 {
	return tc.CreatedAt
}

func (tc *TableConfiguration) GetOrganizationID() pulid.ID {
	return tc.OrganizationID
}

func (tc *TableConfiguration) GetBusinessUnitID() pulid.ID {
	return tc.BusinessUnitID
}

func (tc *TableConfiguration) GetTableName() string {
	return "table_configurations"
}

func (tc *TableConfiguration) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "tc",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
		},
	}
}
