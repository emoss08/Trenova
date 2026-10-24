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

type TenderRecipient struct {
	keel.Table `keel:"edi_tender_recipients"`

	ID                       pulid.ID                      `json:"id" keel:"id,pk,autoid:editr_"`
	BusinessUnitID           pulid.ID                      `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	SourceOrganizationID     pulid.ID                      `json:"sourceOrganizationId" keel:"source_organization_id"`
	SourceBusinessUnitID     pulid.ID                      `json:"sourceBusinessUnitId" keel:"source_business_unit_id"`
	SourceShipmentID         pulid.ID                      `json:"sourceShipmentId" keel:"source_shipment_id"`
	RecipientKind            TenderRecipientKind           `json:"recipientKind" keel:"recipient_kind"`
	RecipientOrganizationID  pulid.ID                      `json:"recipientOrganizationId" keel:"recipient_organization_id,nullzero"`
	RecipientBusinessUnitID  pulid.ID                      `json:"recipientBusinessUnitId" keel:"recipient_business_unit_id,nullzero"`
	EDIPartnerID             pulid.ID                      `json:"ediPartnerId" keel:"edi_partner_id,nullzero"`
	PartnerDocumentProfileID pulid.ID                      `json:"partnerDocumentProfileId" keel:"partner_document_profile_id,nullzero"`
	CommunicationProfileID   pulid.ID                      `json:"communicationProfileId" keel:"communication_profile_id,nullzero"`
	OriginalTransferID       pulid.ID                      `json:"originalTransferId" keel:"original_transfer_id,nullzero"`
	ShipmentLinkID           pulid.ID                      `json:"shipmentLinkId" keel:"shipment_link_id,nullzero"`
	OriginalMessageID        pulid.ID                      `json:"originalMessageId" keel:"original_message_id,nullzero"`
	LatestBaselinePayload    LoadTenderPayload             `json:"latestBaselinePayload" keel:"latest_baseline_payload,jsonb"`
	LatestBaselineHash       string                        `json:"latestBaselineHash" keel:"latest_baseline_hash"`
	BaselineRecordedAt       int64                         `json:"baselineRecordedAt" keel:"baseline_recorded_at"`
	BaselineStatus           TenderRecipientBaselineStatus `json:"baselineStatus" keel:"baseline_status"`
	Status                   TenderRecipientStatus         `json:"status" keel:"status"`
	Version                  int64                         `json:"version" keel:"version,version"`
	CreatedAt                int64                         `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt                int64                         `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (r *TenderRecipient) BeforeUpdate(ctx context.Context) error {
	if err := r.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (r *TenderRecipient) BeforeCreate(ctx context.Context) error {
	if err := r.BeforeSave(ctx); err != nil {
		return err
	}
	if r.Status == "" {
		r.Status = TenderRecipientStatusActive
	}

	return nil
}

type TenderChange struct {
	keel.Table `keel:"edi_tender_changes"`

	ID                      pulid.ID            `json:"id" keel:"id,pk,autoid:editcg_"`
	BusinessUnitID          pulid.ID            `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	SourceOrganizationID    pulid.ID            `json:"sourceOrganizationId" keel:"source_organization_id"`
	SourceBusinessUnitID    pulid.ID            `json:"sourceBusinessUnitId" keel:"source_business_unit_id"`
	SourceShipmentID        pulid.ID            `json:"sourceShipmentId" keel:"source_shipment_id"`
	RecipientID             pulid.ID            `json:"recipientId" keel:"recipient_id"`
	RecipientKind           TenderRecipientKind `json:"recipientKind" keel:"recipient_kind"`
	Status                  TenderChangeStatus  `json:"status" keel:"status"`
	ChangeType              string              `json:"changeType" keel:"change_type"`
	IdempotencyKey          string              `json:"idempotencyKey" keel:"idempotency_key"`
	SourceShipmentVersion   int64               `json:"sourceShipmentVersion" keel:"source_shipment_version"`
	PreviousBaselinePayload LoadTenderPayload   `json:"previousBaselinePayload" keel:"previous_baseline_payload,jsonb"`
	NewTenderPayload        LoadTenderPayload   `json:"newTenderPayload" keel:"new_tender_payload,jsonb"`
	PreviousBaselineHash    string              `json:"previousBaselineHash" keel:"previous_baseline_hash"`
	NewPayloadHash          string              `json:"newPayloadHash" keel:"new_payload_hash"`
	DiffSummary             map[string]any      `json:"diffSummary" keel:"diff_summary,jsonb"`
	ConflictMetadata        map[string]any      `json:"conflictMetadata" keel:"conflict_metadata,jsonb"`
	InternalTransferID      pulid.ID            `json:"internalTransferId" keel:"internal_transfer_id,nullzero"`
	ShipmentLinkID          pulid.ID            `json:"shipmentLinkId" keel:"shipment_link_id,nullzero"`
	OutboundMessageID       pulid.ID            `json:"outboundMessageId" keel:"outbound_message_id,nullzero"`
	ReviewedByID            pulid.ID            `json:"reviewedById" keel:"reviewed_by_id,nullzero"`
	ReviewedAt              *int64              `json:"reviewedAt" keel:"reviewed_at"`
	AppliedByID             pulid.ID            `json:"appliedById" keel:"applied_by_id,nullzero"`
	AppliedAt               *int64              `json:"appliedAt" keel:"applied_at"`
	FailureReason           string              `json:"failureReason" keel:"failure_reason,nullzero"`
	SearchVector            string              `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank                    string              `json:"-" keel:"rank,computed"`
	Version                 int64               `json:"version" keel:"version,version"`
	CreatedAt               int64               `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt               int64               `json:"updatedAt" keel:"updated_at,autoupdate"`

	Recipient *TenderRecipient `json:"recipient,omitempty" keel:"belongs:recipient_id"`
}

func (c *TenderChange) BeforeUpdate(ctx context.Context) error {
	if err := c.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (c *TenderChange) BeforeCreate(ctx context.Context) error {
	if err := c.BeforeSave(ctx); err != nil {
		return err
	}
	if c.Status == "" {
		c.Status = TenderChangeStatusPendingReview
	}

	return nil
}

func (r *TenderRecipient) BeforeSave(_ context.Context) error {
	if r.Status == "" {
		r.Status = TenderRecipientStatusActive
	}
	if r.BaselineRecordedAt == 0 {
		r.BaselineRecordedAt = timeutils.NowUnix()
	}
	return nil
}

func (c *TenderChange) BeforeSave(_ context.Context) error {
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
