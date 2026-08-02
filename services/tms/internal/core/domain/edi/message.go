package edi

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ pagination.CursorEntity   = (*EDIMessage)(nil)
	_ bun.BeforeAppendModelHook = (*EDIMessage)(nil)
)

type EDIMessage struct {
	bun.BaseModel             `json:"-" bun:"table:edi_messages,alias:emsg"`
	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                       pulid.ID                    `json:"id"                       bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID           pulid.ID                    `json:"businessUnitId"           bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID           pulid.ID                    `json:"organizationId"           bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	EDIPartnerID             pulid.ID                    `json:"ediPartnerId"             bun:"edi_partner_id,type:VARCHAR(100),notnull"`
	DocumentTypeID           pulid.ID                    `json:"documentTypeId"           bun:"document_type_id,type:VARCHAR(100),notnull"`
	PartnerDocumentProfileID pulid.ID                    `json:"partnerDocumentProfileId" bun:"partner_document_profile_id,type:VARCHAR(100),nullzero"`
	TemplateID               pulid.ID                    `json:"templateId"               bun:"template_id,type:VARCHAR(100),nullzero"`
	TemplateVersionID        pulid.ID                    `json:"templateVersionId"        bun:"template_version_id,type:VARCHAR(100),nullzero"`
	ShipmentID               pulid.ID                    `json:"shipmentId"               bun:"shipment_id,type:VARCHAR(100),nullzero"`
	TransferID               pulid.ID                    `json:"transferId"               bun:"transfer_id,type:VARCHAR(100),nullzero"`
	InboundFileID            pulid.ID                    `json:"inboundFileId"            bun:"inbound_file_id,type:VARCHAR(100),nullzero"`
	Direction                DocumentDirection           `json:"direction"                bun:"direction,type:edi_document_direction_enum,notnull"`
	Standard                 EDIStandard                 `json:"standard"                 bun:"standard,type:edi_standard_enum,notnull"`
	TransactionSet           TransactionSet              `json:"transactionSet"           bun:"transaction_set,type:edi_transaction_set_enum,notnull"`
	X12Version               string                      `json:"x12Version"               bun:"x12_version,type:VARCHAR(20),notnull"`
	Status                   MessageStatus               `json:"status"                   bun:"status,type:edi_message_status_enum,notnull"`
	ValidationMode           ValidationMode              `json:"validationMode"           bun:"validation_mode,type:edi_validation_mode_enum,notnull"`
	InterchangeControlNumber string                      `json:"interchangeControlNumber" bun:"interchange_control_number,type:VARCHAR(20),notnull"`
	GroupControlNumber       string                      `json:"groupControlNumber"       bun:"group_control_number,type:VARCHAR(20),notnull"`
	TransactionControlNumber string                      `json:"transactionControlNumber" bun:"transaction_control_number,type:VARCHAR(20),notnull"`
	SegmentCount             int64                       `json:"segmentCount"             bun:"segment_count,type:BIGINT,notnull"`
	RawX12                   string                      `json:"rawX12"                   bun:"raw_x12,type:TEXT,notnull"`
	PayloadSnapshot          DocumentPayload             `json:"payloadSnapshot"          bun:"payload_snapshot,type:JSONB,notnull"`
	RawPurgedAt              *int64                      `json:"rawPurgedAt"              bun:"raw_purged_at,type:BIGINT,nullzero"`
	DeliveryStatus           MessageDeliveryStatus       `json:"deliveryStatus"           bun:"delivery_status,type:edi_message_delivery_status_enum,nullzero"`
	DeliveryRemotePath       string                      `json:"deliveryRemotePath"       bun:"delivery_remote_path,type:TEXT,nullzero"`
	DeliveryAttempts         int64                       `json:"deliveryAttempts"         bun:"delivery_attempts,type:BIGINT,notnull,default:0"`
	DeliveryLastAttemptAt    *int64                      `json:"deliveryLastAttemptAt"    bun:"delivery_last_attempt_at,type:BIGINT,nullzero"`
	DeliverySentAt           *int64                      `json:"deliverySentAt"           bun:"delivery_sent_at,type:BIGINT,nullzero"`
	DeliveryLastError        string                      `json:"deliveryLastError"        bun:"delivery_last_error,type:TEXT,nullzero"`
	AS2MessageID             string                      `json:"as2MessageId"             bun:"as2_message_id,type:VARCHAR(255),nullzero"`
	AS2MIC                   string                      `json:"as2Mic"                   bun:"as2_mic,type:VARCHAR(255),nullzero"`
	AckStatus                MessageAcknowledgmentStatus `json:"ackStatus"                bun:"ack_status,type:edi_message_ack_status_enum,nullzero"`
	AckMessageID             pulid.ID                    `json:"ackMessageId"             bun:"ack_message_id,type:VARCHAR(100),nullzero"`
	AckReceivedAt            *int64                      `json:"ackReceivedAt"            bun:"ack_received_at,type:BIGINT,nullzero"`
	AckLastError             string                      `json:"ackLastError"             bun:"ack_last_error,type:TEXT,nullzero"`
	GeneratedByID            pulid.ID                    `json:"generatedById"            bun:"generated_by_id,type:VARCHAR(100),nullzero"`
	GeneratedAt              int64                       `json:"generatedAt"              bun:"generated_at,type:BIGINT,notnull"`
	Version                  int64                       `json:"version"                  bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                int64                       `json:"createdAt"                bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                int64                       `json:"updatedAt"                bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

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

	ID              pulid.ID           `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID           `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID  pulid.ID           `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	MessageID       pulid.ID           `json:"messageId"       bun:"message_id,type:VARCHAR(100),notnull"`
	Severity        ValidationSeverity `json:"severity"        bun:"severity,type:edi_validation_severity_enum,notnull"`
	Code            string             `json:"code"            bun:"code,type:VARCHAR(100),notnull"`
	SegmentID       string             `json:"segmentId"       bun:"segment_id,type:VARCHAR(10),nullzero"`
	ElementPosition int                `json:"elementPosition" bun:"element_position,type:INTEGER,notnull,default:0"`
	Path            string             `json:"path"            bun:"path,type:TEXT,nullzero"`
	Message         string             `json:"message"         bun:"message,type:TEXT,notnull"`
	SuggestedFix    string             `json:"suggestedFix"    bun:"suggested_fix,type:TEXT,nullzero"`
	CreatedAt       int64              `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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
	bun.BaseModel             `json:"-" bun:"table:edi_test_cases,alias:etc"`
	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                       pulid.ID        `json:"id"                       bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID           pulid.ID        `json:"businessUnitId"           bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID           pulid.ID        `json:"organizationId"           bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	PartnerDocumentProfileID pulid.ID        `json:"partnerDocumentProfileId" bun:"partner_document_profile_id,type:VARCHAR(100),notnull"`
	Name                     string          `json:"name"                     bun:"name,type:VARCHAR(200),notnull"`
	Description              string          `json:"description"              bun:"description,type:TEXT,nullzero"`
	Payload                  DocumentPayload `json:"payload"                  bun:"payload,type:JSONB,notnull"`
	ExpectedWarnings         int             `json:"expectedWarnings"         bun:"expected_warnings,type:INTEGER,notnull,default:0"`
	ExpectedErrors           int             `json:"expectedErrors"           bun:"expected_errors,type:INTEGER,notnull,default:0"`
	ExpectedWarningCodes     []string        `json:"expectedWarningCodes"     bun:"expected_warning_codes,type:JSONB,notnull,default:'[]'::jsonb"`
	ExpectedErrorCodes       []string        `json:"expectedErrorCodes"       bun:"expected_error_codes,type:JSONB,notnull,default:'[]'::jsonb"`
	LastRunAt                *int64          `json:"lastRunAt"                bun:"last_run_at,type:BIGINT,nullzero"`
	LastRunPassed            *bool           `json:"lastRunPassed"            bun:"last_run_passed,type:BOOLEAN"`
	LastRunWarnings          int             `json:"lastRunWarnings"          bun:"last_run_warnings,type:INTEGER,notnull,default:0"`
	LastRunErrors            int             `json:"lastRunErrors"            bun:"last_run_errors,type:INTEGER,notnull,default:0"`
	Version                  int64           `json:"version"                  bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                int64           `json:"createdAt"                bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                int64           `json:"updatedAt"                bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	DocumentProfile *EDIPartnerDocumentProfile `json:"documentProfile,omitempty" bun:"rel:belongs-to,join:partner_document_profile_id=id"`
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

func (t *EDITestCase) GetID() pulid.ID {
	return t.ID
}

func (t *EDITestCase) GetTableName() string {
	return "edi_test_cases"
}

func (t *EDITestCase) GetOrganizationID() pulid.ID {
	return t.OrganizationID
}

func (t *EDITestCase) GetBusinessUnitID() pulid.ID {
	return t.BusinessUnitID
}

func (t *EDITestCase) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "etc",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (t *EDITestCase) GetCreatedAt() int64 {
	return t.CreatedAt
}

func (m *EDIMessage) GetID() pulid.ID {
	return m.ID
}

func (m *EDIMessage) GetTableName() string {
	return "edi_messages"
}

func (m *EDIMessage) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "emsg",
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "interchange_control_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "transaction_set",
				Type:   domaintypes.FieldTypeEnum,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (m *EDIMessage) GetCreatedAt() int64 {
	return m.CreatedAt
}

func (m *EDIMessage) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&m.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&m.EDIPartnerID, validation.Required.Error("Partner is required")),
		validation.Field(&m.DocumentTypeID,
			validation.Required.Error("Document type is required"),
		),
		validation.Field(&m.Direction,
			validation.Required.Error("Direction is required"),
			domainvalidation.ValidEnum[DocumentDirection]("Direction is invalid"),
		),
		validation.Field(&m.Standard,
			validation.Required.Error("Standard is required"),
			domainvalidation.ValidEnum[EDIStandard]("Standard is invalid"),
		),
		validation.Field(&m.TransactionSet,
			validation.Required.Error("Transaction set is required"),
			domainvalidation.ValidEnum[TransactionSet]("Transaction set is invalid"),
		),
		validation.Field(&m.X12Version,
			validation.Required.Error("X12 version is required"),
			validation.Length(1, maxEDICodeLength).
				Error("X12 version cannot be longer than 20 characters"),
		),
		validation.Field(&m.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[MessageStatus]("Status is invalid"),
		),
		validation.Field(&m.ValidationMode,
			validation.Required.Error("Validation mode is required"),
			domainvalidation.ValidEnum[ValidationMode]("Validation mode is invalid"),
		),
		// The three control numbers are how a partner's acknowledgment is
		// matched back to what was sent; a message missing one cannot be
		// reconciled against the 997 that answers it.
		validation.Field(&m.InterchangeControlNumber,
			validation.Required.Error("Interchange control number is required"),
			validation.Length(1, maxEDICodeLength).
				Error("Interchange control number cannot be longer than 20 characters"),
		),
		validation.Field(&m.GroupControlNumber,
			validation.Required.Error("Group control number is required"),
			validation.Length(1, maxEDICodeLength).
				Error("Group control number cannot be longer than 20 characters"),
		),
		validation.Field(&m.TransactionControlNumber,
			validation.Required.Error("Transaction control number is required"),
			validation.Length(1, maxEDICodeLength).
				Error("Transaction control number cannot be longer than 20 characters"),
		),
		validation.Field(&m.SegmentCount,
			validation.Min(int64(0)).Error("Segment count cannot be negative"),
		),
		validation.Field(&m.DeliveryStatus,
			domainvalidation.ValidEnum[MessageDeliveryStatus]("Delivery status is invalid"),
		),
		validation.Field(&m.DeliveryAttempts,
			validation.Min(int64(0)).Error("Delivery attempts cannot be negative"),
		),
		validation.Field(&m.DeliveryLastError,
			validation.When(
				m.DeliveryStatus == MessageDeliveryStatusFailed ||
					m.DeliveryStatus == MessageDeliveryStatusDeadLettered,
				validation.Required.Error("A failed delivery must record why it failed"),
			),
		),
		validation.Field(&m.AckStatus,
			domainvalidation.ValidEnum[MessageAcknowledgmentStatus](
				"Acknowledgment status is invalid",
			),
		),
		validation.Field(&m.AS2MessageID,
			validation.Length(0, maxAS2FieldLength).
				Error("AS2 message id cannot be longer than 255 characters"),
		),
		validation.Field(&m.AS2MIC,
			validation.Length(0, maxAS2FieldLength).
				Error("AS2 MIC cannot be longer than 255 characters"),
		),
		validation.Field(&m.GeneratedAt,
			validation.Required.Error("Generated at is required"),
			validation.Min(int64(1)).Error("Generated at must be a valid timestamp"),
		),
	))

	for i, validationErr := range m.ValidationErrors {
		if validationErr == nil {
			continue
		}
		validationErr.Validate(multiErr.WithIndex("validationErrors", i))
	}
}

func (e *EDIMessageValidationError) Validate(multiErr *errortypes.MultiError) {
	// Tenancy and the owning message are stamped when the parent is written, so
	// they are not the caller's to supply.
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.Severity,
			validation.Required.Error("Severity is required"),
			domainvalidation.ValidEnum[ValidationSeverity]("Severity is invalid"),
		),
		validation.Field(&e.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, maxDiagnosticCodeLength).
				Error("Code cannot be longer than 100 characters"),
		),
		validation.Field(&e.SegmentID,
			validation.Length(0, maxSegmentIDLength).
				Error("Segment cannot be longer than 10 characters"),
		),
		validation.Field(&e.ElementPosition,
			validation.Min(0).Error("Element position cannot be negative"),
		),
		validation.Field(&e.Message, validation.Required.Error("Message is required")),
	))
}

func (c *EDITestCase) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&c.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&c.PartnerDocumentProfileID,
			validation.Required.Error("Document profile is required"),
		),
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		// The expectations are what the run is judged against, so a negative one
		// could never be met and the case would fail forever.
		validation.Field(&c.ExpectedWarnings,
			validation.Min(0).Error("Expected warnings cannot be negative"),
		),
		validation.Field(&c.ExpectedErrors,
			validation.Min(0).Error("Expected errors cannot be negative"),
		),
		validation.Field(&c.LastRunWarnings,
			validation.Min(0).Error("Last run warnings cannot be negative"),
		),
		validation.Field(&c.LastRunErrors,
			validation.Min(0).Error("Last run errors cannot be negative"),
		),
	))
}

const (
	maxAS2FieldLength       = 255
	maxDiagnosticCodeLength = 100
)
