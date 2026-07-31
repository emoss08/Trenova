package telematics

import (
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	bun.BaseModel `bun:"table:telematics_form_mappings,alias:tfmap" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	Provider       string   `json:"provider"       bun:"provider,type:VARCHAR(32),notnull,default:'Samsara'"`
	TemplateID     string   `json:"templateId"     bun:"template_id,type:TEXT,notnull"`
	TemplateName   string   `json:"templateName"   bun:"template_name,type:TEXT,nullzero"`
	Name           string   `json:"name"           bun:"name,type:VARCHAR(200),notnull"`
	Description    string   `json:"description"    bun:"description,type:TEXT,nullzero"`
	Enabled        bool     `json:"enabled"        bun:"enabled,type:BOOLEAN,notnull,default:true"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Items []*FormMappingItem `json:"items,omitempty" bun:"rel:has-many,join:id=mapping_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

type FormMappingItem struct {
	bun.BaseModel `bun:"table:telematics_form_mapping_items,alias:tfmi" json:"-"`

	ID                   pulid.ID              `json:"id"                    bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID              `json:"organizationId"        bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID              `json:"businessUnitId"        bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	MappingID            pulid.ID              `json:"mappingId"             bun:"mapping_id,type:VARCHAR(100),notnull"`
	SourceFieldLabel     string                `json:"sourceFieldLabel"      bun:"source_field_label,type:TEXT,notnull"`
	TargetKind           FormMappingTargetKind `json:"targetKind"            bun:"target_kind,type:VARCHAR(32),notnull"`
	TargetField          string                `json:"targetField"           bun:"target_field,type:VARCHAR(64),nullzero"`
	TargetCustomFieldKey string                `json:"targetCustomFieldKey"  bun:"target_custom_field_key,type:VARCHAR(100),nullzero"`
	CreatedAt            int64                 `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&i.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&i.MappingID, validation.Required.Error("Mapping is required")),
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
