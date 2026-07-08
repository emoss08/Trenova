package edi

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type EDIInboundFile struct {
	bun.BaseModel             `json:"-" bun:"table:edi_inbound_files,alias:eif"`
	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                       pulid.ID          `json:"id"                       bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID           pulid.ID          `json:"businessUnitId"           bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID           pulid.ID          `json:"organizationId"           bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	CommunicationProfileID   pulid.ID          `json:"communicationProfileId"   bun:"communication_profile_id,type:VARCHAR(100),notnull"`
	EDIPartnerID             pulid.ID          `json:"ediPartnerId"             bun:"edi_partner_id,type:VARCHAR(100),nullzero"`
	Method                   ConnectionMethod  `json:"method"                   bun:"method,type:edi_connection_method_enum,notnull"`
	RemotePath               string            `json:"remotePath"               bun:"remote_path,type:TEXT,notnull"`
	FileName                 string            `json:"fileName"                 bun:"file_name,type:VARCHAR(512),notnull"`
	Checksum                 string            `json:"checksum"                 bun:"checksum,type:VARCHAR(64),notnull"`
	SizeBytes                int64             `json:"sizeBytes"                bun:"size_bytes,type:BIGINT,notnull,default:0"`
	RawContent               string            `json:"rawContent"               bun:"raw_content,type:TEXT,notnull"`
	InterchangeControlNumber string            `json:"interchangeControlNumber" bun:"interchange_control_number,type:VARCHAR(20),nullzero"`
	ISASenderQualifier       string            `json:"isaSenderQualifier"       bun:"isa_sender_qualifier,type:VARCHAR(4),nullzero"`
	ISASenderID              string            `json:"isaSenderId"              bun:"isa_sender_id,type:VARCHAR(20),nullzero"`
	ISAReceiverQualifier     string            `json:"isaReceiverQualifier"     bun:"isa_receiver_qualifier,type:VARCHAR(4),nullzero"`
	ISAReceiverID            string            `json:"isaReceiverId"            bun:"isa_receiver_id,type:VARCHAR(20),nullzero"`
	Status                   InboundFileStatus `json:"status"                   bun:"status,type:edi_inbound_file_status_enum,notnull,default:'Received'"`
	FailureReason            string            `json:"failureReason"            bun:"failure_reason,type:TEXT,nullzero"`
	TransactionCount         int               `json:"transactionCount"         bun:"transaction_count,type:INTEGER,notnull,default:0"`
	ReceivedAt               int64             `json:"receivedAt"               bun:"received_at,type:BIGINT,notnull"`
	ProcessedAt              *int64            `json:"processedAt"              bun:"processed_at,type:BIGINT,nullzero"`
	RawPurgedAt              *int64            `json:"rawPurgedAt"              bun:"raw_purged_at,type:BIGINT,nullzero"`
	Version                  int64             `json:"version"                  bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                int64             `json:"createdAt"                bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                int64             `json:"updatedAt"                bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Partner              *EDIPartner              `json:"partner,omitempty"              bun:"rel:belongs-to,join:edi_partner_id=id"`
	CommunicationProfile *EDICommunicationProfile `json:"communicationProfile,omitempty" bun:"rel:belongs-to,join:communication_profile_id=id"`
	Messages             []*EDIMessage            `json:"messages,omitempty"             bun:"rel:has-many,join:id=inbound_file_id"`
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

func (f *EDIInboundFile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if f.ReceivedAt == 0 {
		f.ReceivedAt = now
	}
	switch query.(type) {
	case *bun.InsertQuery:
		if f.ID.IsNil() {
			f.ID = pulid.MustNew("ediinf_")
		}
		f.CreatedAt = now
	case *bun.UpdateQuery:
		f.UpdatedAt = now
	}
	return nil
}

func (f *EDIInboundFile) GetCreatedAt() int64 {
	return f.CreatedAt
}
