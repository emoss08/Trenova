package telematics

import (
	"context"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type FormMappingTargetKind string

const (
	FormMappingTargetShipmentField       = FormMappingTargetKind("ShipmentField")
	FormMappingTargetShipmentCustomField = FormMappingTargetKind("ShipmentCustomField")
	FormMappingTargetStopField           = FormMappingTargetKind("StopField")
)

func (k FormMappingTargetKind) IsValid() bool {
	switch k {
	case FormMappingTargetShipmentField,
		FormMappingTargetShipmentCustomField,
		FormMappingTargetStopField:
		return true
	}
	return false
}

var ShipmentFieldTargets = map[string]struct{}{
	"bol":            {},
	"temperatureMin": {},
	"temperatureMax": {},
	"pieces":         {},
	"weight":         {},
}

var StopFieldTargets = map[string]struct{}{
	"pieces": {},
	"weight": {},
}

type FormMapping struct {
	keel.Table `keel:"telematics_form_mappings"`

	ID             pulid.ID `json:"id" keel:"id,pk"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	Provider       string   `json:"provider" keel:"provider"`
	TemplateID     string   `json:"templateId" keel:"template_id"`
	TemplateName   string   `json:"templateName" keel:"template_name,nullzero"`
	Name           string   `json:"name" keel:"name"`
	Description    string   `json:"description" keel:"description,nullzero"`
	Enabled        bool     `json:"enabled" keel:"enabled"`
	Version        int64    `json:"version" keel:"version,version"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at"`

	Items []*FormMappingItem `json:"items,omitempty" keel:"many:mapping_id"`
}

func (m *FormMapping) BeforeCreate(_ context.Context) error {
	if m.Provider == "" {
		m.Provider = "Samsara"
	}
	if m.UpdatedAt == 0 {
		m.UpdatedAt = timeutils.NowUnix()
	}

	return nil
}

type FormMappingItem struct {
	keel.Table `keel:"telematics_form_mapping_items"`

	ID                   pulid.ID              `json:"id" keel:"id,pk"`
	OrganizationID       pulid.ID              `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID       pulid.ID              `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	MappingID            pulid.ID              `json:"mappingId" keel:"mapping_id"`
	SourceFieldLabel     string                `json:"sourceFieldLabel" keel:"source_field_label"`
	TargetKind           FormMappingTargetKind `json:"targetKind" keel:"target_kind"`
	TargetField          string                `json:"targetField" keel:"target_field,nullzero"`
	TargetCustomFieldKey string                `json:"targetCustomFieldKey" keel:"target_custom_field_key,nullzero"`
	CreatedAt            int64                 `json:"createdAt" keel:"created_at,autocreate"`
}

func NewFormMappingID() pulid.ID {
	return pulid.MustNew("tfmap_")
}

func NewFormMappingItemID() pulid.ID {
	return pulid.MustNew("tfmi_")
}

func (m *FormMapping) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&m.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&m.Provider,
			validation.Required.Error("Provider is required"),
			validation.Length(1, maxProviderLength).
				Error("Provider cannot be longer than 32 characters"),
		),
		// The template id is what an incoming submission is matched on, so a
		// mapping without one would never fire.
		validation.Field(&m.TemplateID, validation.Required.Error("Template is required")),
		validation.Field(&m.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxMappingNameLength).
				Error("Name cannot be longer than 200 characters"),
		),
	))

	for i, item := range m.Items {
		if item == nil {
			continue
		}
		item.Validate(multiErr.WithIndex("items", i))
	}
}

func (i *FormMappingItem) Validate(multiErr *errortypes.MultiError) {
	// Tenancy and the owning mapping are stamped when the parent is written, so
	// they are not the caller's to supply.
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.SourceFieldLabel,
			validation.Required.Error("Source field label is required"),
		),
		validation.Field(&i.TargetKind,
			validation.Required.Error("Target kind is required"),
			domainvalidation.ValidEnum[FormMappingTargetKind]("Target kind is invalid"),
		),
		// A mapping row that names no destination copies the answer nowhere,
		// which reads on the form as a field that quietly never applied.
		validation.Field(&i.TargetField,
			validation.When(
				i.TargetKind == FormMappingTargetShipmentField ||
					i.TargetKind == FormMappingTargetStopField,
				validation.Required.Error("A field mapping must name the field it writes"),
			),
			validation.Length(0, maxTargetFieldLength).
				Error("Target field cannot be longer than 64 characters"),
		),
		validation.Field(&i.TargetCustomFieldKey,
			validation.When(
				i.TargetKind == FormMappingTargetShipmentCustomField,
				validation.Required.Error("A custom field mapping must name its key"),
			),
			validation.Length(0, maxTargetKeyLength).
				Error("Target custom field key cannot be longer than 100 characters"),
		),
	))
}

const (
	maxMappingNameLength = 200
	maxTargetFieldLength = 64
	maxTargetKeyLength   = 100
)
