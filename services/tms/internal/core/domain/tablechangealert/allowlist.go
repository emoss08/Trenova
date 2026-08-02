package tablechangealert

import (
	"context"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*TCAAllowlistedTable)(nil)

type TCAAllowlistedTable struct {
	bun.BaseModel `bun:"table:tca_allowlisted_tables,alias:tcaw" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	TableName      string   `json:"tableName"      bun:"table_name,type:VARCHAR(100),notnull"`
	DisplayName    string   `json:"displayName"    bun:"display_name,type:VARCHAR(255),notnull"`
	Enabled        bool     `json:"enabled"        bun:"enabled,type:BOOLEAN,notnull,default:true"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
}

func (a *TCAAllowlistedTable) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("tcaw_")
		}
		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}

func (a *TCAAllowlistedTable) GetID() pulid.ID {
	return a.ID
}

func (t *TCAAllowlistedTable) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&t.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		// The name is interpolated into the trigger definition the alert
		// installs, so it is held to what Postgres accepts as an identifier
		// rather than to anything a caller cares to send.
		validation.Field(&t.TableName,
			validation.Required.Error("Table name is required"),
			validation.Length(1, maxTableNameLength).
				Error("Table name cannot be longer than 100 characters"),
			validation.Match(tableNamePattern).
				Error("Table name must be a lowercase identifier"),
		),
		validation.Field(&t.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(1, maxDisplayNameLength).
				Error("Display name cannot be longer than 255 characters"),
		),
	))
}

var tableNamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

const (
	maxTableNameLength   = 100
	maxDisplayNameLength = 255
)
