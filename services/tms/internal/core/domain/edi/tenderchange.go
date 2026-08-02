package edi

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
			{
				Name:   "change_type",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
			{
				Name:   "failure_reason",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}

func (r *TenderRecipient) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&r.SourceOrganizationID,
			validation.Required.Error("Source organization is required"),
		),
		validation.Field(&r.SourceBusinessUnitID,
			validation.Required.Error("Source business unit is required"),
		),
		validation.Field(&r.SourceShipmentID,
			validation.Required.Error("Source shipment is required"),
		),
		validation.Field(&r.RecipientKind,
			validation.Required.Error("Recipient kind is required"),
			domainvalidation.ValidEnum[TenderRecipientKind]("Recipient kind is invalid"),
		),
		// An internal recipient is another organization on this system; an
		// external one is reached through an EDI partner. A recipient naming
		// neither has no address to tender to.
		validation.Field(&r.RecipientOrganizationID,
			validation.When(
				r.RecipientKind == TenderRecipientKindInternal,
				validation.Required.Error("An internal recipient must name its organization"),
			),
		),
		validation.Field(&r.EDIPartnerID,
			validation.When(
				r.RecipientKind == TenderRecipientKindExternal,
				validation.Required.Error("An external recipient must name its partner"),
			),
		),
		// The baseline hash is what a later revision is diffed against, so
		// without it every change would read as a whole new tender.
		validation.Field(&r.LatestBaselineHash,
			validation.Required.Error("Baseline hash is required"),
			validation.Length(1, maxTenderHashLength).
				Error("Baseline hash cannot be longer than 128 characters"),
		),
		validation.Field(&r.BaselineRecordedAt,
			validation.Required.Error("Baseline recorded at is required"),
			validation.Min(int64(1)).Error("Baseline recorded at must be a valid timestamp"),
		),
		validation.Field(&r.BaselineStatus,
			validation.Required.Error("Baseline status is required"),
			domainvalidation.ValidEnum[TenderRecipientBaselineStatus](
				"Baseline status is invalid",
			),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[TenderRecipientStatus]("Status is invalid"),
		),
	))

	r.LatestBaselinePayload.Validate(multiErr.WithPrefix("latestBaselinePayload"))
}

func (c *TenderChange) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&c.SourceOrganizationID,
			validation.Required.Error("Source organization is required"),
		),
		validation.Field(&c.SourceBusinessUnitID,
			validation.Required.Error("Source business unit is required"),
		),
		validation.Field(&c.SourceShipmentID,
			validation.Required.Error("Source shipment is required"),
		),
		validation.Field(&c.RecipientID, validation.Required.Error("Recipient is required")),
		validation.Field(&c.RecipientKind,
			validation.Required.Error("Recipient kind is required"),
			domainvalidation.ValidEnum[TenderRecipientKind]("Recipient kind is invalid"),
		),
		validation.Field(&c.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[TenderChangeStatus]("Status is invalid"),
		),
		validation.Field(&c.ChangeType,
			validation.Required.Error("Change type is required"),
			validation.Length(1, maxChangeTypeLength).
				Error("Change type cannot be longer than 100 characters"),
		),
		validation.Field(&c.IdempotencyKey,
			validation.Required.Error("Idempotency key is required"),
			validation.Length(1, maxIdempotencyKeyLength).
				Error("Idempotency key cannot be longer than 255 characters"),
		),
		validation.Field(&c.SourceShipmentVersion,
			validation.Min(int64(0)).Error("Source shipment version cannot be negative"),
		),
		// A revision that hashes the same as the baseline is not a change, and
		// sending it would look to the partner like a duplicate tender.
		validation.Field(&c.PreviousBaselineHash,
			validation.Required.Error("Previous baseline hash is required"),
			validation.Length(1, maxTenderHashLength).
				Error("Previous baseline hash cannot be longer than 128 characters"),
		),
		validation.Field(&c.NewPayloadHash,
			validation.Required.Error("New payload hash is required"),
			validation.Length(1, maxTenderHashLength).
				Error("New payload hash cannot be longer than 128 characters"),
			validation.NotIn(c.PreviousBaselineHash).
				Error("The revised tender is identical to the one already sent"),
		),
		validation.Field(&c.FailureReason,
			validation.When(
				c.Status == TenderChangeStatusFailed,
				validation.Required.Error("A failed change must record why"),
			),
		),
		validation.Field(&c.ReviewedAt,
			validation.When(c.ReviewedByID.IsNotNil(), validation.Required.Error(
				"A reviewed change must record when it was reviewed",
			)),
		),
		validation.Field(&c.AppliedAt,
			validation.When(c.AppliedByID.IsNotNil(), validation.Required.Error(
				"An applied change must record when it was applied",
			)),
		),
	))

	c.NewTenderPayload.Validate(multiErr.WithPrefix("newTenderPayload"))
}

const maxTenderHashLength = 128
