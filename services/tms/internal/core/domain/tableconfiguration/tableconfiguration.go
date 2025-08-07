/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package tableconfiguration

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Configuration)(nil)
	_ domain.Validatable        = (*Configuration)(nil)
	_ infra.PostgresSearchable  = (*Configuration)(nil)
)

// TableConfig is a JSONB blob that stores all user-specific table preferences.
// Over time we can safely extend it with new optional fields without having to
// run DB migrations.
type TableConfig struct {
	Filters          []ports.FieldFilter `json:"filters"`
	JoinOperator     string              `json:"joinOperator"`
	Sort             []ports.SortField   `json:"sort"`
	PageSize         int                 `json:"pageSize"`
	ColumnVisibility map[string]bool     `json:"columnVisibility"`
	ColumnOrder      []string            `json:"columnOrder"`
}

type Configuration struct {
	bun.BaseModel `bun:"table:table_configurations,alias:tc" json:"-"`

	// Primary identifiers
	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`

	// Core fields
	Name        string      `json:"name"        bun:"name,type:VARCHAR(255),notnull"`
	Description string      `json:"description" bun:"description,type:TEXT"`
	Resource    string      `json:"resource"    bun:"resource,type:VARCHAR(100),notnull"`
	TableConfig TableConfig `json:"tableConfig" bun:"table_config,type:JSONB,notnull"`
	Visibility  Visibility  `json:"visibility"  bun:"visibility,type:configuration_visibility_enum,notnull,default:'Private'"`
	IsDefault   bool        `json:"isDefault"   bun:"is_default,type:BOOLEAN,notnull,default:false"`

	// Metadata
	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Creator      *user.User                 `json:"creator,omitempty"      bun:"rel:belongs-to,join:user_id=id"`
	Shares       []*ConfigurationShare      `json:"shares,omitempty"       bun:"rel:has-many,join:id=configuration_id"`
}

func (c *Configuration) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, c,
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
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// TODO(Wolfred): Move this to a validator
func (c *Configuration) DBValidate(ctx context.Context, _ bun.IDB) *errors.MultiError {
	multiErr := errors.NewMultiError()
	c.Validate(ctx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (c *Configuration) GetTableName() string {
	return "table_configurations"
}

func (c *Configuration) GetTableAlias() string {
	return "tc"
}

func (c *Configuration) GetID() string {
	return c.ID.String()
}

func (c *Configuration) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: c.GetTableAlias(),
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "name",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "description",
				Weight: "B",
				Type:   infra.PostgresSearchTypeText,
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
	}
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
