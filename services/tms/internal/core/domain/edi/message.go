package edi

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type EDIMessage struct {
	bun.BaseModel `json:"-" bun:"table:edi_messages,alias:emsg"`

	ID                       pulid.ID          `json:"id"                        bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID           pulid.ID          `json:"businessUnitId"            bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID           pulid.ID          `json:"organizationId"            bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	EDIPartnerID             pulid.ID          `json:"ediPartnerId"              bun:"edi_partner_id,type:VARCHAR(100),notnull"`
	DocumentTypeID           pulid.ID          `json:"documentTypeId"            bun:"document_type_id,type:VARCHAR(100),notnull"`
	PartnerDocumentProfileID pulid.ID          `json:"partnerDocumentProfileId"  bun:"partner_document_profile_id,type:VARCHAR(100),notnull"`
	TemplateID               pulid.ID          `json:"templateId"                bun:"template_id,type:VARCHAR(100),notnull"`
	TemplateVersionID        pulid.ID          `json:"templateVersionId"         bun:"template_version_id,type:VARCHAR(100),notnull"`
	ShipmentID               pulid.ID          `json:"shipmentId"                bun:"shipment_id,type:VARCHAR(100),nullzero"`
	TransferID               pulid.ID          `json:"transferId"                bun:"transfer_id,type:VARCHAR(100),nullzero"`
	Direction                DocumentDirection `json:"direction"             bun:"direction,type:edi_document_direction_enum,notnull"`
	Standard                 EDIStandard       `json:"standard"                  bun:"standard,type:edi_standard_enum,notnull"`
	TransactionSet           TransactionSet    `json:"transactionSet"            bun:"transaction_set,type:edi_transaction_set_enum,notnull"`
	X12Version               string            `json:"x12Version"                bun:"x12_version,type:VARCHAR(20),notnull"`
	Status                   MessageStatus     `json:"status"                    bun:"status,type:edi_message_status_enum,notnull"`
	ValidationMode           ValidationMode    `json:"validationMode"            bun:"validation_mode,type:edi_validation_mode_enum,notnull"`
	InterchangeControlNumber string            `json:"interchangeControlNumber"  bun:"interchange_control_number,type:VARCHAR(20),notnull"`
	GroupControlNumber       string            `json:"groupControlNumber"        bun:"group_control_number,type:VARCHAR(20),notnull"`
	TransactionControlNumber string            `json:"transactionControlNumber"  bun:"transaction_control_number,type:VARCHAR(20),notnull"`
	SegmentCount             int64             `json:"segmentCount"              bun:"segment_count,type:BIGINT,notnull"`
	RawX12                   string            `json:"rawX12"                    bun:"raw_x12,type:TEXT,notnull"`
	PayloadSnapshot          LoadTenderPayload `json:"payloadSnapshot"        bun:"payload_snapshot,type:JSONB,notnull"`
	GeneratedByID            pulid.ID          `json:"generatedById"             bun:"generated_by_id,type:VARCHAR(100),nullzero"`
	GeneratedAt              int64             `json:"generatedAt"               bun:"generated_at,type:BIGINT,notnull"`
	Version                  int64             `json:"version"                   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                int64             `json:"createdAt"                 bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                int64             `json:"updatedAt"                 bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	DiagnosticCount int64 `json:"diagnosticCount" bun:"diagnostic_count,scanonly"`

	Partner                *EDIPartner                  `json:"partner,omitempty"                bun:"rel:belongs-to,join:edi_partner_id=id"`
	DocumentType           *EDIDocumentType             `json:"documentType,omitempty"           bun:"rel:belongs-to,join:document_type_id=id"`
	PartnerDocumentProfile *EDIPartnerDocumentProfile   `json:"partnerDocumentProfile,omitempty" bun:"rel:belongs-to,join:partner_document_profile_id=id"`
	Template               *EDITemplate                 `json:"template,omitempty"               bun:"rel:belongs-to,join:template_id=id"`
	TemplateVersion        *EDITemplateVersion          `json:"templateVersion,omitempty"        bun:"rel:belongs-to,join:template_version_id=id"`
	ValidationErrors       []*EDIMessageValidationError `json:"validationErrors,omitempty"       bun:"rel:has-many,join:id=message_id"`
}

func (m *EDIMessage) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if m.GeneratedAt == 0 {
		m.GeneratedAt = now
	}
	switch query.(type) {
	case *bun.InsertQuery:
		if m.ID.IsNil() {
			m.ID = pulid.MustNew("edimsg_")
		}
		m.CreatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = now
	}
	return nil
}

type EDIMessageValidationError struct {
	bun.BaseModel `json:"-" bun:"table:edi_message_validation_errors,alias:emve"`

	ID              pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID           `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID  pulid.ID           `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	MessageID       pulid.ID           `json:"messageId"      bun:"message_id,type:VARCHAR(100),notnull"`
	Severity        ValidationSeverity `json:"severity"       bun:"severity,type:edi_validation_severity_enum,notnull"`
	Code            string             `json:"code"           bun:"code,type:VARCHAR(100),notnull"`
	SegmentID       string             `json:"segmentId"      bun:"segment_id,type:VARCHAR(10),nullzero"`
	ElementPosition int                `json:"elementPosition" bun:"element_position,type:INTEGER,notnull,default:0"`
	Path            string             `json:"path"           bun:"path,type:TEXT,nullzero"`
	Message         string             `json:"message"        bun:"message,type:TEXT,notnull"`
	SuggestedFix    string             `json:"suggestedFix"   bun:"suggested_fix,type:TEXT,nullzero"`
	CreatedAt       int64              `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (e *EDIMessageValidationError) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("edive_")
		}
		e.CreatedAt = timeutils.NowUnix()
	}
	return nil
}

type EDITestCase struct {
	bun.BaseModel `json:"-" bun:"table:edi_test_cases,alias:etc"`

	ID                       pulid.ID          `json:"id"                       bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID           pulid.ID          `json:"businessUnitId"           bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID           pulid.ID          `json:"organizationId"           bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	PartnerDocumentProfileID pulid.ID          `json:"partnerDocumentProfileId" bun:"partner_document_profile_id,type:VARCHAR(100),notnull"`
	Name                     string            `json:"name"                     bun:"name,type:VARCHAR(200),notnull"`
	Description              string            `json:"description"              bun:"description,type:TEXT,nullzero"`
	Payload                  LoadTenderPayload `json:"payload"                  bun:"payload,type:JSONB,notnull"`
	ExpectedWarnings         int               `json:"expectedWarnings"         bun:"expected_warnings,type:INTEGER,notnull,default:0"`
	ExpectedErrors           int               `json:"expectedErrors"           bun:"expected_errors,type:INTEGER,notnull,default:0"`
	Version                  int64             `json:"version"                  bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                int64             `json:"createdAt"                bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                int64             `json:"updatedAt"                bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (t *EDITestCase) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("editc_")
		}
		t.CreatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}
	return nil
}
