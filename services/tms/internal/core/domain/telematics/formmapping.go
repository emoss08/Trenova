package telematics

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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
