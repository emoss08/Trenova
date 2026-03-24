package tableconfiguration

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*TableConfiguration)(nil)
	_ validationframework.TenantedEntity = (*TableConfiguration)(nil)
	_ domaintypes.PostgresSearchable     = (*TableConfiguration)(nil)
)

type TableConfig struct {
	FilterGroups     []domaintypes.FilterGroup `json:"filterGroups"`
	FieldFilters     []domaintypes.FieldFilter `json:"fieldFilters"`
	JoinOperator     string                    `json:"joinOperator"`
	Sort             []domaintypes.SortField   `json:"sort"`
	PageSize         int                       `json:"pageSize"`
	ColumnVisibility map[string]bool           `json:"columnVisibility"`
	ColumnOrder      []string                  `json:"columnOrder"`
}

type TableConfiguration struct {
	bun.BaseModel `bun:"table:table_configurations,alias:tc" json:"-"`

	ID             pulid.ID     `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID     `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID     `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	UserID         pulid.ID     `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	Name           string       `json:"name"           bun:"name,type:VARCHAR(255),notnull"`
	Description    string       `json:"description"    bun:"description,type:TEXT"`
	Resource       string       `json:"resource"       bun:"resource,type:VARCHAR(100),notnull"`
	TableConfig    *TableConfig `json:"tableConfig"    bun:"table_config,type:JSONB,notnull"`
	Visibility     Visibility   `json:"visibility"     bun:"visibility,type:configuration_visibility_enum,notnull,default:'Private'"`
	SearchVector   string       `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string       `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`
	IsDefault      bool         `json:"isDefault"      bun:"is_default,type:BOOLEAN,notnull,default:false"`
	Version        int64        `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64        `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64        `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	User         *tenant.User         `json:"user,omitempty"         bun:"rel:belongs-to,join:user_id=id"`
}

func (tc *TableConfiguration) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(tc,
		validation.Field(&tc.Name, validation.Required, validation.Length(1, 255)),
		validation.Field(&tc.Resource, validation.Required, validation.Length(1, 100)),
		validation.Field(&tc.TableConfig, validation.Required),
		validation.Field(&tc.Visibility, validation.Required, validation.In(
			VisibilityPrivate,
			VisibilityPublic,
			VisibilityShared,
		)),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (tc *TableConfiguration) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if tc.ID.IsNil() {
			tc.ID = pulid.MustNew("tc_")
		}
		tc.CreatedAt = now
	case *bun.UpdateQuery:
		tc.UpdatedAt = now
	}

	return nil
}

func (tc *TableConfiguration) GetID() pulid.ID {
	return tc.ID
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
