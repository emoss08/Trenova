package tablechangealert

import (
	"regexp"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type TCAAllowlistedTable struct {
	keel.Table `                                                       keel:"tca_allowlisted_tables"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:tcaw_"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	TableName      string   `json:"tableName" keel:"table_name"`
	DisplayName    string   `json:"displayName" keel:"display_name"`
	Enabled        bool     `json:"enabled" keel:"enabled"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
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
