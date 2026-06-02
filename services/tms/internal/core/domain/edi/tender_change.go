package edi

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type TenderRecipient struct {
	bun.BaseModel `json:"-" bun:"table:edi_tender_recipients,alias:etr"`

	ID                       pulid.ID                      `json:"id" bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID           pulid.ID                      `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	SourceOrganizationID     pulid.ID                      `json:"sourceOrganizationId" bun:"source_organization_id,type:VARCHAR(100),notnull"`
	SourceBusinessUnitID     pulid.ID                      `json:"sourceBusinessUnitId" bun:"source_business_unit_id,type:VARCHAR(100),notnull"`
	SourceShipmentID         pulid.ID                      `json:"sourceShipmentId" bun:"source_shipment_id,type:VARCHAR(100),notnull"`
	RecipientKind            TenderRecipientKind           `json:"recipientKind" bun:"recipient_kind,type:edi_tender_recipient_kind_enum,notnull"`
	RecipientOrganizationID  pulid.ID                      `json:"recipientOrganizationId" bun:"recipient_organization_id,type:VARCHAR(100),nullzero"`
	RecipientBusinessUnitID  pulid.ID                      `json:"recipientBusinessUnitId" bun:"recipient_business_unit_id,type:VARCHAR(100),nullzero"`
	EDIPartnerID             pulid.ID                      `json:"ediPartnerId" bun:"edi_partner_id,type:VARCHAR(100),nullzero"`
	PartnerDocumentProfileID pulid.ID                      `json:"partnerDocumentProfileId" bun:"partner_document_profile_id,type:VARCHAR(100),nullzero"`
	CommunicationProfileID   pulid.ID                      `json:"communicationProfileId" bun:"communication_profile_id,type:VARCHAR(100),nullzero"`
	OriginalTransferID       pulid.ID                      `json:"originalTransferId" bun:"original_transfer_id,type:VARCHAR(100),nullzero"`
	ShipmentLinkID           pulid.ID                      `json:"shipmentLinkId" bun:"shipment_link_id,type:VARCHAR(100),nullzero"`
	OriginalMessageID        pulid.ID                      `json:"originalMessageId" bun:"original_message_id,type:VARCHAR(100),nullzero"`
	LatestBaselinePayload    LoadTenderPayload             `json:"latestBaselinePayload" bun:"latest_baseline_payload,type:JSONB,notnull"`
	LatestBaselineHash       string                        `json:"latestBaselineHash" bun:"latest_baseline_hash,type:VARCHAR(128),notnull"`
	BaselineRecordedAt       int64                         `json:"baselineRecordedAt" bun:"baseline_recorded_at,type:BIGINT,notnull"`
	BaselineStatus           TenderRecipientBaselineStatus `json:"baselineStatus" bun:"baseline_status,type:edi_tender_recipient_baseline_status_enum,notnull"`
	Status                   TenderRecipientStatus         `json:"status" bun:"status,type:edi_tender_recipient_status_enum,notnull,default:'Active'"`
	Version                  int64                         `json:"version" bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                int64                         `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                int64                         `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type TenderChange struct {
	bun.BaseModel `json:"-" bun:"table:edi_tender_changes,alias:etcg"`

	ID                      pulid.ID            `json:"id" bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID          pulid.ID            `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	SourceOrganizationID    pulid.ID            `json:"sourceOrganizationId" bun:"source_organization_id,type:VARCHAR(100),notnull"`
	SourceBusinessUnitID    pulid.ID            `json:"sourceBusinessUnitId" bun:"source_business_unit_id,type:VARCHAR(100),notnull"`
	SourceShipmentID        pulid.ID            `json:"sourceShipmentId" bun:"source_shipment_id,type:VARCHAR(100),notnull"`
	RecipientID             pulid.ID            `json:"recipientId" bun:"recipient_id,type:VARCHAR(100),notnull"`
	RecipientKind           TenderRecipientKind `json:"recipientKind" bun:"recipient_kind,type:edi_tender_recipient_kind_enum,notnull"`
	Status                  TenderChangeStatus  `json:"status" bun:"status,type:edi_tender_change_status_enum,notnull,default:'PendingReview'"`
	ChangeType              string              `json:"changeType" bun:"change_type,type:VARCHAR(100),notnull"`
	IdempotencyKey          string              `json:"idempotencyKey" bun:"idempotency_key,type:VARCHAR(255),notnull"`
	SourceShipmentVersion   int64               `json:"sourceShipmentVersion" bun:"source_shipment_version,type:BIGINT,notnull"`
	PreviousBaselinePayload LoadTenderPayload   `json:"previousBaselinePayload" bun:"previous_baseline_payload,type:JSONB,notnull"`
	NewTenderPayload        LoadTenderPayload   `json:"newTenderPayload" bun:"new_tender_payload,type:JSONB,notnull"`
	PreviousBaselineHash    string              `json:"previousBaselineHash" bun:"previous_baseline_hash,type:VARCHAR(128),notnull"`
	NewPayloadHash          string              `json:"newPayloadHash" bun:"new_payload_hash,type:VARCHAR(128),notnull"`
	DiffSummary             map[string]any      `json:"diffSummary" bun:"diff_summary,type:JSONB,notnull,default:'{}'::jsonb"`
	ConflictMetadata        map[string]any      `json:"conflictMetadata" bun:"conflict_metadata,type:JSONB,notnull,default:'{}'::jsonb"`
	InternalTransferID      pulid.ID            `json:"internalTransferId" bun:"internal_transfer_id,type:VARCHAR(100),nullzero"`
	ShipmentLinkID          pulid.ID            `json:"shipmentLinkId" bun:"shipment_link_id,type:VARCHAR(100),nullzero"`
	OutboundMessageID       pulid.ID            `json:"outboundMessageId" bun:"outbound_message_id,type:VARCHAR(100),nullzero"`
	ReviewedByID            pulid.ID            `json:"reviewedById" bun:"reviewed_by_id,type:VARCHAR(100),nullzero"`
	ReviewedAt              *int64              `json:"reviewedAt" bun:"reviewed_at,type:BIGINT,nullzero"`
	AppliedByID             pulid.ID            `json:"appliedById" bun:"applied_by_id,type:VARCHAR(100),nullzero"`
	AppliedAt               *int64              `json:"appliedAt" bun:"applied_at,type:BIGINT,nullzero"`
	FailureReason           string              `json:"failureReason" bun:"failure_reason,type:TEXT,nullzero"`
	SearchVector            string              `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                    string              `json:"-" bun:"rank,type:VARCHAR(100),scanonly"`
	Version                 int64               `json:"version" bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt               int64               `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64               `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Recipient *TenderRecipient `json:"recipient,omitempty" bun:"rel:belongs-to,join:recipient_id=id"`
}

func (r *TenderRecipient) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if r.Status == "" {
		r.Status = TenderRecipientStatusActive
	}
	if r.BaselineRecordedAt == 0 {
		r.BaselineRecordedAt = now
	}
	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("editr_")
		}
		r.CreatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}
	return nil
}

func (c *TenderChange) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if c.Status == "" {
		c.Status = TenderChangeStatusPendingReview
	}
	if c.ChangeType == "" {
		c.ChangeType = TenderChangeTypeLoadTender
	}
	if c.DiffSummary == nil {
		c.DiffSummary = map[string]any{}
	}
	if c.ConflictMetadata == nil {
		c.ConflictMetadata = map[string]any{}
	}
	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("editcg_")
		}
		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}
	return nil
}

func (r *TenderRecipient) GetID() pulid.ID { return r.ID }

func (r *TenderRecipient) GetOrganizationID() pulid.ID { return r.SourceOrganizationID }

func (r *TenderRecipient) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (c *TenderChange) GetID() pulid.ID { return c.ID }

func (c *TenderChange) GetOrganizationID() pulid.ID { return c.SourceOrganizationID }

func (c *TenderChange) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *TenderChange) GetTableName() string { return "edi_tender_changes" }

func (c *TenderChange) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "etcg",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "change_type", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
			{Name: "failure_reason", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
		},
	}
}
