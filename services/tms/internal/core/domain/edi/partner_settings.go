package edi

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type EDIPartnerSettingSchema struct {
	bun.BaseModel `json:"-" bun:"table:edi_partner_setting_schemas,alias:epss"`

	ID             pulid.ID             `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID             `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),nullzero"`
	OrganizationID pulid.ID             `json:"organizationId" bun:"organization_id,type:VARCHAR(100),nullzero"`
	DocumentTypeID pulid.ID             `json:"documentTypeId" bun:"document_type_id,type:VARCHAR(100),nullzero"`
	Standard       EDIStandard          `json:"standard"       bun:"standard,type:edi_standard_enum,notnull"`
	TransactionSet TransactionSet       `json:"transactionSet" bun:"transaction_set,type:edi_transaction_set_enum,notnull"`
	Direction      DocumentDirection    `json:"direction"      bun:"direction,type:edi_document_direction_enum,notnull"`
	X12Version     string               `json:"x12Version"     bun:"x12_version,type:VARCHAR(20),notnull"`
	SchemaVersion  int64                `json:"schemaVersion"  bun:"schema_version,type:BIGINT,notnull"`
	Name           string               `json:"name"           bun:"name,type:VARCHAR(200),notnull"`
	Description    string               `json:"description"    bun:"description,type:TEXT,nullzero"`
	Status         PartnerSettingStatus `json:"status"         bun:"status,type:edi_partner_setting_status_enum,notnull"`
	CreatedAt      int64                `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64                `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Fields []*EDIPartnerSettingField `json:"fields,omitempty" bun:"rel:has-many,join:id=schema_id"`
}

func (s *EDIPartnerSettingSchema) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("edips_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}
	return nil
}

type EDIPartnerSettingField struct {
	bun.BaseModel `json:"-" bun:"table:edi_partner_setting_fields,alias:epsf"`

	ID                pulid.ID               `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	SchemaID          pulid.ID               `json:"schemaId"          bun:"schema_id,type:VARCHAR(100),notnull"`
	Path              string                 `json:"path"              bun:"path,type:TEXT,notnull"`
	Label             string                 `json:"label"             bun:"label,type:VARCHAR(200),notnull"`
	Description       string                 `json:"description"       bun:"description,type:TEXT,nullzero"`
	DataType          PartnerSettingDataType `json:"dataType"          bun:"data_type,type:edi_partner_setting_data_type_enum,notnull"`
	Required          bool                   `json:"required"          bun:"required,type:BOOLEAN,notnull,default:false"`
	Nullable          bool                   `json:"nullable"          bun:"nullable,type:BOOLEAN,notnull,default:false"`
	DefaultValue      any                    `json:"defaultValue"      bun:"default_value,type:JSONB,nullzero"`
	AllowedValues     []string               `json:"allowedValues"     bun:"allowed_values,type:JSONB,notnull,default:'[]'::jsonb"`
	Secret            bool                   `json:"secret"            bun:"secret,type:BOOLEAN,notnull,default:false"`
	GroupKey          string                 `json:"groupKey"          bun:"group_key,type:VARCHAR(100),nullzero"`
	DisplayOrder      int                    `json:"displayOrder"      bun:"display_order,type:INTEGER,notnull,default:0"`
	ValidationPattern string                 `json:"validationPattern" bun:"validation_pattern,type:TEXT,nullzero"`
	MinLength         int                    `json:"minLength"         bun:"min_length,type:INTEGER,notnull,default:0"`
	MaxLength         int                    `json:"maxLength"         bun:"max_length,type:INTEGER,notnull,default:0"`
	UsageNotes        string                 `json:"usageNotes"        bun:"usage_notes,type:TEXT,nullzero"`
	Status            PartnerSettingStatus   `json:"status"            bun:"status,type:edi_partner_setting_status_enum,notnull"`
	CreatedAt         int64                  `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64                  `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Schema *EDIPartnerSettingSchema `json:"schema,omitempty" bun:"rel:belongs-to,join:schema_id=id"`
}

func (f *EDIPartnerSettingField) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if f.ID.IsNil() {
			f.ID = pulid.MustNew("edipsf_")
		}
		f.CreatedAt = now
	case *bun.UpdateQuery:
		f.UpdatedAt = now
	}
	return nil
}
