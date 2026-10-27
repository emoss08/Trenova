package customfield

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type CustomFieldValue struct {
	keel.Table `keel:"custom_field_values"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:cfv_"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	DefinitionID   pulid.ID `json:"definitionId"   keel:"definition_id"`
	ResourceType   string   `json:"resourceType"   keel:"resource_type"`
	ResourceID     string   `json:"resourceId"     keel:"resource_id"`
	Value          any      `json:"value"          keel:"value,jsonb"`
	Version        int64    `json:"version"        keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Definition *CustomFieldDefinition `json:"definition,omitempty" keel:"belongs:definition_id"`
}

func (v *CustomFieldValue) GetID() pulid.ID {
	return v.ID
}

func (v *CustomFieldValue) GetOrganizationID() pulid.ID {
	return v.OrganizationID
}

func (v *CustomFieldValue) GetBusinessUnitID() pulid.ID {
	return v.BusinessUnitID
}

func (v *CustomFieldValue) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(v,
		validation.Field(&v.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&v.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&v.DefinitionID,
			validation.Required.Error("Custom field definition is required"),
		),
		// A value with no owner is unreachable: nothing would ever load it and
		// nothing would ever delete it.
		validation.Field(&v.ResourceType,
			validation.Required.Error("Resource type is required"),
			validation.Length(1, maxResourceTypeLength).
				Error("Resource type cannot be longer than 100 characters"),
		),
		validation.Field(&v.ResourceID,
			validation.Required.Error("Resource is required"),
			validation.Length(1, maxResourceIDLength).
				Error("Resource cannot be longer than 100 characters"),
		),
		validation.Field(&v.Value, validation.NotNil.Error("Value is required")),
	))
}

const (
	maxResourceTypeLength = 100
	maxResourceIDLength   = 100
)
