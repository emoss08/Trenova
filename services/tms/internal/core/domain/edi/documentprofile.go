package edi

import (
	"context"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type EDIPartnerDocumentProfile struct {
	bun.BaseModel `json:"-" bun:"table:edi_partner_document_profiles,alias:epdp"`

	ID                           pulid.ID             `json:"id"                           bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID               pulid.ID             `json:"businessUnitId"               bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID               pulid.ID             `json:"organizationId"               bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	EDIPartnerID                 pulid.ID             `json:"ediPartnerId"                 bun:"edi_partner_id,type:VARCHAR(100),notnull"`
	DocumentTypeID               pulid.ID             `json:"documentTypeId"               bun:"document_type_id,type:VARCHAR(100),notnull"`
	TemplateID                   pulid.ID             `json:"templateId"                   bun:"template_id,type:VARCHAR(100),notnull"`
	TemplateVersionID            pulid.ID             `json:"templateVersionId"            bun:"template_version_id,type:VARCHAR(100),nullzero"`
	Name                         string               `json:"name"                         bun:"name,type:VARCHAR(200),notnull"`
	Status                       DocumentStatus       `json:"status"                       bun:"status,type:edi_document_status_enum,notnull"`
	Direction                    DocumentDirection    `json:"direction"                    bun:"direction,type:edi_document_direction_enum,notnull"`
	Standard                     EDIStandard          `json:"standard"                     bun:"standard,type:edi_standard_enum,notnull"`
	TransactionSet               TransactionSet       `json:"transactionSet"               bun:"transaction_set,type:edi_transaction_set_enum,notnull"`
	X12VersionOverride           string               `json:"x12VersionOverride"           bun:"x12_version_override,type:VARCHAR(20),nullzero"`
	FunctionalGroupID            string               `json:"functionalGroupId"            bun:"functional_group_id,type:VARCHAR(2),notnull"`
	Envelope                     X12EnvelopeSettings  `json:"envelope"                     bun:"envelope,type:JSONB,notnull,default:'{}'"`
	Acknowledgment               AcknowledgmentConfig `json:"acknowledgment"               bun:"acknowledgment,type:JSONB,notnull,default:'{}'"`
	ValidationMode               ValidationMode       `json:"validationMode"               bun:"validation_mode,type:edi_validation_mode_enum,notnull"`
	PartnerSettings              map[string]any       `json:"partnerSettings"              bun:"partner_settings,type:JSONB,notnull,default:'{}'"`
	PartnerSettingsSchemaID      pulid.ID             `json:"partnerSettingsSchemaId"      bun:"partner_settings_schema_id,type:VARCHAR(100),nullzero"`
	PartnerSettingsSchemaVersion int64                `json:"partnerSettingsSchemaVersion" bun:"partner_settings_schema_version,type:BIGINT,nullzero"`
	Version                      int64                `json:"version"                      bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                    int64                `json:"createdAt"                    bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                    int64                `json:"updatedAt"                    bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Partner               *EDIPartner              `json:"partner,omitempty"               bun:"rel:belongs-to,join:edi_partner_id=id"`
	DocumentType          *EDIDocumentType         `json:"documentType,omitempty"          bun:"rel:belongs-to,join:document_type_id=id"`
	PartnerSettingsSchema *EDIPartnerSettingSchema `json:"partnerSettingsSchema,omitempty" bun:"rel:belongs-to,join:partner_settings_schema_id=id"`
	Template              *EDITemplate             `json:"template,omitempty"              bun:"rel:belongs-to,join:template_id=id"`
	TemplateVersion       *EDITemplateVersion      `json:"templateVersion,omitempty"       bun:"rel:belongs-to,join:template_version_id=id"`
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
		p.FunctionalGroupID = FunctionalGroupDefault(p.TransactionSet)
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

func FunctionalGroupDefault(transactionSet TransactionSet) string {
	switch transactionSet {
	case TransactionSet210:
		return "IM"
	case TransactionSet204:
		return "SM"
	case TransactionSet214:
		return "QM"
	case TransactionSet990:
		return "GF"
	case TransactionSet997, TransactionSet999:
		return "FA"
	default:
		return "SM"
	}
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

func (c *EDIControlNumberSequence) GetID() pulid.ID {
	return c.ID
}

func (c *EDIControlNumberSequence) GetOrganizationID() pulid.ID {
	return c.OrganizationID
}

func (c *EDIControlNumberSequence) GetBusinessUnitID() pulid.ID {
	return c.BusinessUnitID
}

func (p *EDIPartnerDocumentProfile) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&p.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&p.EDIPartnerID, validation.Required.Error("Partner is required")),
		validation.Field(&p.DocumentTypeID,
			validation.Required.Error("Document type is required"),
		),
		validation.Field(&p.TemplateID, validation.Required.Error("Template is required")),
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[DocumentStatus]("Status is invalid"),
		),
		validation.Field(&p.Direction,
			validation.Required.Error("Direction is required"),
			domainvalidation.ValidEnum[DocumentDirection]("Direction is invalid"),
		),
		validation.Field(&p.Standard,
			validation.Required.Error("Standard is required"),
			domainvalidation.ValidEnum[EDIStandard]("Standard is invalid"),
		),
		validation.Field(&p.TransactionSet,
			validation.Required.Error("Transaction set is required"),
			domainvalidation.ValidEnum[TransactionSet]("Transaction set is invalid"),
		),
		validation.Field(&p.X12VersionOverride,
			validation.Length(0, maxEDICodeLength).
				Error("X12 version cannot be longer than 20 characters"),
		),
		validation.Field(&p.FunctionalGroupID,
			validation.Required.Error("Functional group is required"),
			validation.Length(1, functionalGroupIDLength).
				Error("Functional group must be at most 2 characters"),
		),
		validation.Field(&p.ValidationMode,
			validation.Required.Error("Validation mode is required"),
			domainvalidation.ValidEnum[ValidationMode]("Validation mode is invalid"),
		),
	))

	p.Acknowledgment.Validate(multiErr.WithPrefix("acknowledgment"))
}

// Validate checks the acknowledgment expectations, which is where an unmet
// service level turns into an alert rather than a silence.
func (c *AcknowledgmentConfig) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.Type,
			validation.When(
				c.Expected,
				validation.Required.Error("An expected acknowledgment must name its type"),
				validation.NotIn(AcknowledgmentTypeNone).
					Error("An expected acknowledgment must name its type"),
			),
			domainvalidation.ValidEnum[AcknowledgmentType]("Acknowledgment type is invalid"),
		),
		validation.Field(&c.SLAInMinutes,
			validation.Min(int64(0)).Error("Acknowledgment SLA cannot be negative"),
		),
		validation.Field(&c.MissingAckSeverity,
			domainvalidation.ValidEnum[ValidationSeverity]("Severity is invalid"),
		),
	))
}

func (s *EDIControlNumberSequence) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&s.EDIPartnerID, validation.Required.Error("Partner is required")),
		validation.Field(&s.DocumentTypeID,
			validation.Required.Error("Document type is required"),
		),
		validation.Field(&s.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[ControlNumberKind]("Kind is invalid"),
		),
		// Control numbers roll over between the bounds, so an inverted or empty
		// range would either never issue a number or reissue one a partner has
		// already seen.
		validation.Field(&s.MinValue,
			validation.Min(int64(1)).Error("Minimum control number must be at least one"),
		),
		validation.Field(&s.MaxValue,
			validation.Min(s.MinValue+1).Error("Maximum control number must exceed the minimum"),
		),
		validation.Field(&s.NextValue,
			validation.Min(s.MinValue).Error("Next control number cannot precede the minimum"),
			validation.Max(s.MaxValue).Error("Next control number cannot exceed the maximum"),
		),
	))
}
