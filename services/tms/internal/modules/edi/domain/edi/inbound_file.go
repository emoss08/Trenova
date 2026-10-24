package edi

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type EDIInboundFile struct {
	keel.Table `keel:"edi_inbound_files"`

	ID                       pulid.ID          `json:"id" keel:"id,pk,autoid:ediinf_"`
	BusinessUnitID           pulid.ID          `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID           pulid.ID          `json:"organizationId" keel:"organization_id,pk,tenant"`
	CommunicationProfileID   pulid.ID          `json:"communicationProfileId" keel:"communication_profile_id"`
	EDIPartnerID             pulid.ID          `json:"ediPartnerId" keel:"edi_partner_id,nullzero"`
	Method                   ConnectionMethod  `json:"method" keel:"method"`
	RemotePath               string            `json:"remotePath" keel:"remote_path"`
	FileName                 string            `json:"fileName" keel:"file_name"`
	Checksum                 string            `json:"checksum" keel:"checksum"`
	SizeBytes                int64             `json:"sizeBytes" keel:"size_bytes"`
	RawContent               string            `json:"rawContent" keel:"raw_content"`
	InterchangeControlNumber string            `json:"interchangeControlNumber" keel:"interchange_control_number,nullzero"`
	ISASenderQualifier       string            `json:"isaSenderQualifier" keel:"isa_sender_qualifier,nullzero"`
	ISASenderID              string            `json:"isaSenderId" keel:"isa_sender_id,nullzero"`
	ISAReceiverQualifier     string            `json:"isaReceiverQualifier" keel:"isa_receiver_qualifier,nullzero"`
	ISAReceiverID            string            `json:"isaReceiverId" keel:"isa_receiver_id,nullzero"`
	Status                   InboundFileStatus `json:"status" keel:"status"`
	FailureReason            string            `json:"failureReason" keel:"failure_reason,nullzero"`
	TransactionCount         int               `json:"transactionCount" keel:"transaction_count"`
	ReceivedAt               int64             `json:"receivedAt" keel:"received_at"`
	ProcessedAt              *int64            `json:"processedAt" keel:"processed_at"`
	RawPurgedAt              *int64            `json:"rawPurgedAt" keel:"raw_purged_at"`
	Version                  int64             `json:"version" keel:"version,version"`
	CreatedAt                int64             `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt                int64             `json:"updatedAt" keel:"updated_at,autoupdate"`

	Partner              *EDIPartner              `json:"partner,omitempty" keel:"belongs:edi_partner_id"`
	CommunicationProfile *EDICommunicationProfile `json:"communicationProfile,omitempty" keel:"belongs:communication_profile_id"`
	Messages             []*EDIMessage            `json:"messages,omitempty" keel:"many:inbound_file_id"`
}

func (f *EDIInboundFile) BeforeUpdate(ctx context.Context) error {
	if err := f.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (f *EDIInboundFile) BeforeCreate(ctx context.Context) error {
	if err := f.BeforeSave(ctx); err != nil {
		return err
	}
	if f.Status == "" {
		f.Status = InboundFileStatusReceived
	}

	return nil
}

func (f *EDIInboundFile) GetID() pulid.ID {
	return f.ID
}

func (f *EDIInboundFile) GetTableName() string {
	return "edi_inbound_files"
}

func (f *EDIInboundFile) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "eif",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "file_name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
			{Name: "method", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (f *EDIInboundFile) GetOrganizationID() pulid.ID {
	return f.OrganizationID
}

func (f *EDIInboundFile) GetBusinessUnitID() pulid.ID {
	return f.BusinessUnitID
}

func (f *EDIInboundFile) BeforeSave(_ context.Context) error {
	if f.ReceivedAt == 0 {
		f.ReceivedAt = timeutils.NowUnix()
	}
	return nil
}

func (f *EDIInboundFile) GetCreatedAt() int64 {
	return f.CreatedAt
}

func (f *EDIInboundFile) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(f,
		validation.Field(&f.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&f.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&f.CommunicationProfileID,
			validation.Required.Error("Communication profile is required"),
		),
		validation.Field(&f.Method,
			validation.Required.Error("Method is required"),
			domainvalidation.ValidEnum[ConnectionMethod]("Method is invalid"),
		),
		validation.Field(&f.RemotePath, validation.Required.Error("Remote path is required")),
		validation.Field(&f.FileName,
			validation.Required.Error("File name is required"),
			validation.Length(1, maxInboundFileNameLength).
				Error("File name cannot be longer than 512 characters"),
		),
		// The checksum is what makes a redelivery recognizable as the same file
		// rather than a second interchange.
		validation.Field(&f.Checksum,
			validation.Required.Error("Checksum is required"),
			validation.Length(inboundChecksumLength, inboundChecksumLength).
				Error("Checksum must be a 64 character digest"),
		),
		validation.Field(&f.SizeBytes,
			validation.Min(int64(0)).Error("Size cannot be negative"),
		),
		validation.Field(&f.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[InboundFileStatus]("Status is invalid"),
		),
		validation.Field(&f.FailureReason,
			validation.When(
				f.Status == InboundFileStatusQuarantined,
				validation.Required.Error("A quarantined file must record why"),
			),
		),
		validation.Field(&f.TransactionCount,
			validation.Min(0).Error("Transaction count cannot be negative"),
		),
		validation.Field(&f.InterchangeControlNumber,
			validation.Length(0, maxEDICodeLength).
				Error("Interchange control number cannot be longer than 20 characters"),
		),
		validation.Field(&f.ISASenderQualifier,
			validation.Length(0, maxISAQualifierLength).
				Error("Sender qualifier cannot be longer than 4 characters"),
		),
		validation.Field(&f.ISASenderID,
			validation.Length(0, maxISAIDLength).
				Error("Sender id cannot be longer than 20 characters"),
		),
		validation.Field(&f.ISAReceiverQualifier,
			validation.Length(0, maxISAQualifierLength).
				Error("Receiver qualifier cannot be longer than 4 characters"),
		),
		validation.Field(&f.ISAReceiverID,
			validation.Length(0, maxISAIDLength).
				Error("Receiver id cannot be longer than 20 characters"),
		),
		validation.Field(&f.ReceivedAt,
			validation.Required.Error("Received at is required"),
			validation.Min(int64(1)).Error("Received at must be a valid timestamp"),
		),
	))
}

const (
	maxInboundFileNameLength = 512
	inboundChecksumLength    = 64
	maxISAQualifierLength    = 4
	maxISAIDLength           = 20
)
