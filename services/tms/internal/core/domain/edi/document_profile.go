package edi

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type EDIPartnerDocumentProfile struct {
	bun.BaseModel `json:"-" bun:"table:edi_partner_document_profiles,alias:epdp"`

	ID                           pulid.ID             `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID               pulid.ID             `json:"businessUnitId"         bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID               pulid.ID             `json:"organizationId"         bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	EDIPartnerID                 pulid.ID             `json:"ediPartnerId"           bun:"edi_partner_id,type:VARCHAR(100),notnull"`
	DocumentTypeID               pulid.ID             `json:"documentTypeId"         bun:"document_type_id,type:VARCHAR(100),notnull"`
	TemplateID                   pulid.ID             `json:"templateId"             bun:"template_id,type:VARCHAR(100),notnull"`
	TemplateVersionID            pulid.ID             `json:"templateVersionId"      bun:"template_version_id,type:VARCHAR(100),nullzero"`
	Name                         string               `json:"name"                   bun:"name,type:VARCHAR(200),notnull"`
	Status                       DocumentStatus       `json:"status"                 bun:"status,type:edi_document_status_enum,notnull"`
	Direction                    DocumentDirection    `json:"direction"              bun:"direction,type:edi_document_direction_enum,notnull"`
	Standard                     EDIStandard          `json:"standard"               bun:"standard,type:edi_standard_enum,notnull"`
	TransactionSet               TransactionSet       `json:"transactionSet"         bun:"transaction_set,type:edi_transaction_set_enum,notnull"`
	X12VersionOverride           string               `json:"x12VersionOverride"     bun:"x12_version_override,type:VARCHAR(20),nullzero"`
	FunctionalGroupID            string               `json:"functionalGroupId"      bun:"functional_group_id,type:VARCHAR(2),notnull"`
	Envelope                     X12EnvelopeSettings  `json:"envelope"               bun:"envelope,type:JSONB,notnull,default:'{}'::jsonb"`
	Acknowledgment               AcknowledgmentConfig `json:"acknowledgment"         bun:"acknowledgment,type:JSONB,notnull,default:'{}'::jsonb"`
	ValidationMode               ValidationMode       `json:"validationMode"         bun:"validation_mode,type:edi_validation_mode_enum,notnull"`
	PartnerSettings              map[string]any       `json:"partnerSettings"        bun:"partner_settings,type:JSONB,notnull,default:'{}'::jsonb"`
	PartnerSettingsSchemaID      pulid.ID             `json:"partnerSettingsSchemaId"      bun:"partner_settings_schema_id,type:VARCHAR(100),nullzero"`
	PartnerSettingsSchemaVersion int64                `json:"partnerSettingsSchemaVersion" bun:"partner_settings_schema_version,type:BIGINT,nullzero"`
	Version                      int64                `json:"version"                bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                    int64                `json:"createdAt"              bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                    int64                `json:"updatedAt"              bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Partner               *EDIPartner              `json:"partner,omitempty"         bun:"rel:belongs-to,join:edi_partner_id=id"`
	DocumentType          *EDIDocumentType         `json:"documentType,omitempty"    bun:"rel:belongs-to,join:document_type_id=id"`
	PartnerSettingsSchema *EDIPartnerSettingSchema `json:"partnerSettingsSchema,omitempty" bun:"rel:belongs-to,join:partner_settings_schema_id=id"`
	Template              *EDITemplate             `json:"template,omitempty"        bun:"rel:belongs-to,join:template_id=id"`
	TemplateVersion       *EDITemplateVersion      `json:"templateVersion,omitempty" bun:"rel:belongs-to,join:template_version_id=id"`
}

func (p *EDIPartnerDocumentProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if p.Envelope.ElementSeparator == "" {
		p.Envelope = DefaultX12EnvelopeSettings()
	}
	if p.PartnerSettings == nil {
		p.PartnerSettings = map[string]any{}
	}
	if p.FunctionalGroupID == "" {
		p.FunctionalGroupID = "SM"
	}
	if p.ValidationMode == "" {
		p.ValidationMode = ValidationModeStrict
	}
	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("edidp_")
		}
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}
	return nil
}

func (p *EDIPartnerDocumentProfile) GetID() pulid.ID {
	return p.ID
}

func (p *EDIPartnerDocumentProfile) GetOrganizationID() pulid.ID {
	return p.OrganizationID
}

func (p *EDIPartnerDocumentProfile) GetBusinessUnitID() pulid.ID {
	return p.BusinessUnitID
}

type EDIControlNumberSequence struct {
	bun.BaseModel `json:"-" bun:"table:edi_control_number_sequences,alias:ecns"`

	ID             pulid.ID          `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID          `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	EDIPartnerID   pulid.ID          `json:"ediPartnerId"   bun:"edi_partner_id,type:VARCHAR(100),notnull"`
	DocumentTypeID pulid.ID          `json:"documentTypeId" bun:"document_type_id,type:VARCHAR(100),notnull"`
	Kind           ControlNumberKind `json:"kind"           bun:"kind,type:edi_control_number_kind_enum,notnull"`
	NextValue      int64             `json:"nextValue"      bun:"next_value,type:BIGINT,notnull"`
	MinValue       int64             `json:"minValue"       bun:"min_value,type:BIGINT,notnull"`
	MaxValue       int64             `json:"maxValue"       bun:"max_value,type:BIGINT,notnull"`
	Version        int64             `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64             `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64             `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (s *EDIControlNumberSequence) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if s.MinValue == 0 {
		s.MinValue = 1
	}
	if s.NextValue == 0 {
		s.NextValue = 1
	}
	if s.MaxValue == 0 {
		s.MaxValue = 999999999
	}
	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("edicn_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}
	return nil
}
