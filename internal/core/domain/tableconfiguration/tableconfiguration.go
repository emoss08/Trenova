package tableconfiguration

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

type Filter struct {
	ID       string   `json:"id"`
	Value    []string `json:"value"`    // Assuming "value" is always an array of strings
	Operator string   `json:"operator"` // e.g., "in"
	Type     string   `json:"type"`     // e.g., "multi-select"
	RowID    string   `json:"rowId"`    // Additional field
}

// TableConfig is a JSONB blob that stores all user-specific table preferences.
// Over time we can safely extend it with new optional fields without having to
// run DB migrations.
type TableConfig struct {
	Filters          []Filter        `json:"filters"`          // User-defined filters
	JoinOperator     string          `json:"joinOperator"`     // "and" | "or"
	Sorting          []any           `json:"sorting"`          // Sorting preference
	PageSize         int             `json:"pageSize"`         // Default page size
	ColumnVisibility map[string]bool `json:"columnVisibility"` // NEW – column -> visible?
}

type Configuration struct {
	bun.BaseModel `bun:"table:table_configurations,alias:tc" json:"-"`

	// Primary identifiers
	ID             pulid.ID `json:"id" bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	UserID         pulid.ID `json:"userId" bun:"user_id,type:VARCHAR(100),notnull"`

	// Core fields
	Name            string      `json:"name" bun:"name,type:VARCHAR(255),notnull"`
	Description     string      `json:"description" bun:"description,type:TEXT"`
	TableIdentifier string      `json:"tableIdentifier" bun:"table_identifier,type:VARCHAR(100),notnull"`
	TableConfig     TableConfig `json:"tableConfig" bun:"table_config,type:JSONB,notnull"`
	Visibility      Visibility  `json:"visibility" bun:"visibility,type:configuration_visibility_enum,notnull,default:'Private'"`
	IsDefault       bool        `json:"isDefault" bun:"is_default,type:BOOLEAN,notnull,default:false"`

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Creator      *user.User                 `json:"creator,omitempty" bun:"rel:belongs-to,join:user_id=id"`
	Shares       []*ConfigurationShare      `json:"shares,omitempty" bun:"rel:has-many,join:id=configuration_id"`
}

func (c *Configuration) validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, c,
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&c.TableIdentifier,
			validation.Required.Error("Table identifier is required"),
		),
		validation.Field(&c.TableConfig,
			validation.Required.Error("Table configuration is required"),
		),
		validation.Field(&c.Visibility,
			validation.Required.Error("Visibility is required"),
			validation.In(VisibilityPrivate, VisibilityPublic, VisibilityShared).Error("Visibility must be Private, Public, or Shared"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (c *Configuration) DBValidate(ctx context.Context, _ bun.IDB) *errors.MultiError {
	multiErr := errors.NewMultiError()
	c.validate(ctx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// TODO(Wolfred): Write uniqueness checks for the name and table identifier

// Pagination Configuration
func (c *Configuration) GetTableName() string {
	return "table_configurations"
}

func (c *Configuration) GetTableAlias() string {
	return "tc"
}

func (c *Configuration) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID == "" {
			c.ID = pulid.MustNew("tcf_")
		}
		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
		c.Version++
	}

	return nil
}
