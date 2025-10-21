package tableconfiguration

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Configuration)(nil)
	_ domain.Validatable        = (*Configuration)(nil)
)

type TableConfig struct {
	Filters          []pagination.FieldFilter `json:"filters"`
	JoinOperator     string                   `json:"joinOperator"`
	Sort             []pagination.SortField   `json:"sort"`
	PageSize         int                      `json:"pageSize"`
	ColumnVisibility map[string]bool          `json:"columnVisibility"`
	ColumnOrder      []string                 `json:"columnOrder"`
}

type Configuration struct {
	bun.BaseModel `bun:"table:table_configurations,alias:tc" json:"-"`

	ID             pulid.ID              `json:"id"                     bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID              `json:"businessUnitId"         bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID              `json:"organizationId"         bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	UserID         pulid.ID              `json:"userId"                 bun:"user_id,type:VARCHAR(100),notnull"`
	Name           string                `json:"name"                   bun:"name,type:VARCHAR(255),notnull"`
	Description    string                `json:"description"            bun:"description,type:TEXT"`
	Resource       string                `json:"resource"               bun:"resource,type:VARCHAR(100),notnull"`
	SearchVector   string                `json:"-"                      bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string                `json:"-"                      bun:"rank,type:VARCHAR(100),scanonly"`
	TableConfig    TableConfig           `json:"tableConfig"            bun:"table_config,type:JSONB,notnull"`
	Visibility     Visibility            `json:"visibility"             bun:"visibility,type:configuration_visibility_enum,notnull,default:'Private'"`
	IsDefault      bool                  `json:"isDefault"              bun:"is_default,type:BOOLEAN,notnull,default:false"`
	Version        int64                 `json:"version"                bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64                 `json:"createdAt"              bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64                 `json:"updatedAt"              bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	BusinessUnit   *tenant.BusinessUnit  `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization   *tenant.Organization  `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Creator        *tenant.User          `json:"creator,omitempty"      bun:"rel:belongs-to,join:user_id=id"`
	Shares         []*ConfigurationShare `json:"shares,omitempty"       bun:"rel:has-many,join:id=configuration_id"`
}

func (c *Configuration) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(c,
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&c.Resource,
			validation.Required.Error("Resource is required"),
		),
		validation.Field(&c.TableConfig,
			validation.Required.Error("Table configuration is required"),
		),
		validation.Field(
			&c.Visibility,
			validation.Required.Error("Visibility is required"),
			validation.In(VisibilityPrivate, VisibilityPublic, VisibilityShared).
				Error("Visibility must be Private, Public, or Shared"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (c *Configuration) GetTableName() string {
	return "table_configurations"
}

func (c *Configuration) GetID() string {
	return c.ID.String()
}

func (c *Configuration) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("tcf_")
		}
		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
		c.Version++
	}

	return nil
}
