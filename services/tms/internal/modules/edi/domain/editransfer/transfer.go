package editransfer

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

type LoadTenderPayload struct {
	PurposeCode              LoadTenderPurposeCode                `json:"purposeCode,omitempty"`
	TenderOfferID            pulid.ID                             `json:"tenderOfferId,omitempty"`
	ShipmentID               pulid.ID                             `json:"shipmentId"`
	BusinessUnitID           pulid.ID                             `json:"businessUnitId"`
	OrganizationID           pulid.ID                             `json:"organizationId"`
	ServiceTypeID            pulid.ID                             `json:"serviceTypeId"`
	ServiceTypeLabel         string                               `json:"serviceTypeLabel,omitempty"`
	ShipmentTypeID           pulid.ID                             `json:"shipmentTypeId,omitempty"`
	ShipmentTypeLabel        string                               `json:"shipmentTypeLabel,omitempty"`
	CustomerID               pulid.ID                             `json:"customerId"`
	CustomerLabel            string                               `json:"customerLabel,omitempty"`
	FormulaTemplateID        pulid.ID                             `json:"formulaTemplateId"`
	FormulaTemplateLabel     string                               `json:"formulaTemplateLabel,omitempty"`
	BOL                      string                               `json:"bol,omitempty"`
	Pieces                   *int64                               `json:"pieces,omitempty"`
	Weight                   *int64                               `json:"weight,omitempty"`
	TemperatureMin           *int16                               `json:"temperatureMin,omitempty"`
	TemperatureMax           *int16                               `json:"temperatureMax,omitempty"`
	FreightChargeAmount      decimal.NullDecimal                  `json:"freightChargeAmount"`
	OtherChargeAmount        decimal.NullDecimal                  `json:"otherChargeAmount"`
	TotalChargeAmount        decimal.NullDecimal                  `json:"totalChargeAmount"`
	RatingDetail             map[string]any                       `json:"ratingDetail,omitempty"`
	Moves                    []LoadTenderMove                     `json:"moves"`
	Commodities              []LoadTenderCommodity                `json:"commodities"`
	AdditionalCharges        []LoadTenderCharge                   `json:"additionalCharges"`
	RequiredMappingEntityIDs map[edi.MappingEntityType][]pulid.ID `json:"requiredMappingEntityIds"`
	CustomFields             map[string]any                       `json:"customFields,omitempty"`
}

type LoadTenderMove struct {
	Loaded   bool             `json:"loaded"`
	Sequence int64            `json:"sequence"`
	Distance *float64         `json:"distance,omitempty"`
	Stops    []LoadTenderStop `json:"stops"`
}

type LoadTenderStop struct {
	LocationID           pulid.ID `json:"locationId"`
	LocationLabel        string   `json:"locationLabel,omitempty"`
	LocationName         string   `json:"locationName,omitempty"`
	LocationCode         string   `json:"locationCode,omitempty"`
	LocationAddressLine1 string   `json:"locationAddressLine1,omitempty"`
	LocationAddressLine2 string   `json:"locationAddressLine2,omitempty"`
	LocationCity         string   `json:"locationCity,omitempty"`
	LocationStateCode    string   `json:"locationStateCode,omitempty"`
	LocationPostalCode   string   `json:"locationPostalCode,omitempty"`
	Type                 string   `json:"type"`
	ScheduleType         string   `json:"scheduleType"`
	Sequence             int64    `json:"sequence"`
	Pieces               *int64   `json:"pieces,omitempty"`
	Weight               *int64   `json:"weight,omitempty"`
	ScheduledWindowStart int64    `json:"scheduledWindowStart"`
	ScheduledWindowEnd   *int64   `json:"scheduledWindowEnd,omitempty"`
	AddressLine          string   `json:"addressLine,omitempty"`
}

type LoadTenderCommodity struct {
	CommodityID          pulid.ID `json:"commodityId"`
	CommodityLabel       string   `json:"commodityLabel,omitempty"`
	CommodityName        string   `json:"commodityName,omitempty"`
	CommodityDescription string   `json:"commodityDescription,omitempty"`
	Weight               int64    `json:"weight"`
	Pieces               int64    `json:"pieces"`
}

type LoadTenderCharge struct {
	AccessorialChargeID    pulid.ID        `json:"accessorialChargeId"`
	AccessorialLabel       string          `json:"accessorialLabel,omitempty"`
	AccessorialCode        string          `json:"accessorialCode,omitempty"`
	AccessorialDescription string          `json:"accessorialDescription,omitempty"`
	Method                 string          `json:"method"`
	Amount                 decimal.Decimal `json:"amount"`
	Unit                   int16           `json:"unit"`
}

type MappingResolution struct {
	EntityType  edi.MappingEntityType `json:"entityType"`
	SourceID    pulid.ID              `json:"sourceId"`
	SourceLabel string                `json:"sourceLabel,omitempty"`
	TargetID    pulid.ID              `json:"targetId,omitempty"`
	TargetLabel string                `json:"targetLabel,omitempty"`
	Resolved    bool                  `json:"resolved"`
}

type EDITransfer struct {
	keel.Table `keel:"edi_load_tender_transfers"`

	ID                    pulid.ID            `json:"id"                    keel:"id,pk,autoid:edilt_"`
	SourceOrganizationID  pulid.ID            `json:"sourceOrganizationId"  keel:"source_organization_id"`
	SourceBusinessUnitID  pulid.ID            `json:"sourceBusinessUnitId"  keel:"source_business_unit_id"`
	TargetOrganizationID  pulid.ID            `json:"targetOrganizationId"  keel:"target_organization_id"`
	TargetBusinessUnitID  pulid.ID            `json:"targetBusinessUnitId"  keel:"target_business_unit_id"`
	SourcePartnerID       pulid.ID            `json:"sourcePartnerId"       keel:"source_partner_id"`
	TargetPartnerID       pulid.ID            `json:"targetPartnerId"       keel:"target_partner_id"`
	SourceShipmentID      pulid.ID            `json:"sourceShipmentId"      keel:"source_shipment_id,nullzero"`
	TargetShipmentID      pulid.ID            `json:"targetShipmentId"      keel:"target_shipment_id,nullzero"`
	InboundMessageID      pulid.ID            `json:"inboundMessageId"      keel:"inbound_message_id,nullzero"`
	Status                TransferStatus      `json:"status"                keel:"status"`
	TenderPayload         LoadTenderPayload   `json:"tenderPayload"         keel:"tender_payload,jsonb"`
	MappingSnapshot       []MappingResolution `json:"mappingSnapshot"       keel:"mapping_snapshot,jsonb"`
	RejectionReason       string              `json:"rejectionReason"       keel:"rejection_reason,nullzero"`
	FailureReason         string              `json:"failureReason"         keel:"failure_reason,nullzero"`
	ApprovalWorkflowID    string              `json:"approvalWorkflowId"    keel:"approval_workflow_id,nullzero"`
	ApprovalWorkflowRunID string              `json:"approvalWorkflowRunId" keel:"approval_workflow_run_id,nullzero"`
	SubmittedByID         pulid.ID            `json:"submittedById"         keel:"submitted_by_id,nullzero"`
	SubmittedAt           int64               `json:"submittedAt"           keel:"submitted_at"`
	ApprovedByID          pulid.ID            `json:"approvedById"          keel:"approved_by_id,nullzero"`
	ApprovedAt            *int64              `json:"approvedAt"            keel:"approved_at"`
	ProcessingStartedAt   *int64              `json:"processingStartedAt"   keel:"processing_started_at"`
	ProcessedAt           *int64              `json:"processedAt"           keel:"processed_at"`
	RejectedByID          pulid.ID            `json:"rejectedById"          keel:"rejected_by_id,nullzero"`
	RejectedAt            *int64              `json:"rejectedAt"            keel:"rejected_at"`
	CanceledByID          pulid.ID            `json:"canceledById"          keel:"canceled_by_id,nullzero"`
	CanceledAt            *int64              `json:"canceledAt"            keel:"canceled_at"`
	Version               int64               `json:"version"               keel:"version,version"`
	CreatedAt             int64               `json:"createdAt"             keel:"created_at,autocreate"`
	UpdatedAt             int64               `json:"updatedAt"             keel:"updated_at,autoupdate"`

	SourcePartner *edi.EDIPartner `json:"sourcePartner,omitempty" keel:"belongs:source_partner_id"`
	TargetPartner *edi.EDIPartner `json:"targetPartner,omitempty" keel:"belongs:target_partner_id"`
}

func (t *EDITransfer) BeforeUpdate(ctx context.Context) error {
	if err := t.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (t *EDITransfer) BeforeCreate(ctx context.Context) error {
	if err := t.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (t *EDITransfer) BeforeSave(_ context.Context) error {
	if t.MappingSnapshot == nil {
		t.MappingSnapshot = []MappingResolution{}
	}
	if t.SubmittedAt == 0 {
		t.SubmittedAt = timeutils.NowUnix()
	}
	return nil
}

func (t *EDITransfer) ReceivedBy(orgID pulid.ID) bool {
	return t.TargetOrganizationID == orgID
}

func (t *EDITransfer) Counterparty(orgID pulid.ID) *edi.EDIPartner {
	if t.ReceivedBy(orgID) {
		return t.TargetPartner
	}
	return t.SourcePartner
}

func (t *EDITransfer) GetID() pulid.ID {
	return t.ID
}

func (t *EDITransfer) GetOrganizationID() pulid.ID {
	return t.TargetOrganizationID
}

func (t *EDITransfer) GetBusinessUnitID() pulid.ID {
	return t.TargetBusinessUnitID
}

func (t *EDITransfer) GetTableName() string {
	return "edi_load_tender_transfers"
}

func (t *EDITransfer) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "eltt",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
		},
	}
}

func (t *EDITransfer) GetCreatedAt() int64 {
	return t.CreatedAt
}

func (t *EDITransfer) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.SourceOrganizationID,
			validation.Required.Error("Source organization is required"),
		),
		validation.Field(&t.SourceBusinessUnitID,
			validation.Required.Error("Source business unit is required"),
		),
		validation.Field(&t.TargetOrganizationID,
			validation.Required.Error("Target organization is required"),
		),
		validation.Field(&t.TargetBusinessUnitID,
			validation.Required.Error("Target business unit is required"),
		),
		validation.Field(&t.SourcePartnerID,
			validation.Required.Error("Source partner is required"),
		),
		validation.Field(&t.TargetPartnerID,
			validation.Required.Error("Target partner is required"),
		),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[TransferStatus]("Status is invalid"),
		),
		// A rejection or a failure that records no reason leaves the other side
		// of the tender with nothing to act on.
		validation.Field(&t.RejectionReason,
			validation.When(
				t.Status == TransferStatusRejected,
				validation.Required.Error("A rejected transfer must record why"),
			),
		),
		validation.Field(&t.FailureReason,
			validation.When(
				t.Status == TransferStatusFailed,
				validation.Required.Error("A failed transfer must record why"),
			),
		),
		validation.Field(&t.ApprovalWorkflowID,
			validation.Length(0, maxWorkflowIDLength).
				Error("Approval workflow cannot be longer than 255 characters"),
		),
		validation.Field(&t.ApprovalWorkflowRunID,
			validation.Length(0, maxWorkflowIDLength).
				Error("Approval workflow run cannot be longer than 255 characters"),
		),
		validation.Field(&t.SubmittedAt,
			validation.Required.Error("Submitted at is required"),
			validation.Min(int64(1)).Error("Submitted at must be a valid timestamp"),
		),
		validation.Field(&t.ApprovedAt,
			validation.When(t.ApprovedByID.IsNotNil(), validation.Required.Error(
				"An approved transfer must record when it was approved",
			)),
		),
		validation.Field(&t.RejectedAt,
			validation.When(t.RejectedByID.IsNotNil(), validation.Required.Error(
				"A rejected transfer must record when it was rejected",
			)),
		),
		validation.Field(&t.CanceledAt,
			validation.When(t.CanceledByID.IsNotNil(), validation.Required.Error(
				"A canceled transfer must record when it was canceled",
			)),
		),
	))

	t.TenderPayload.Validate(multiErr.WithPrefix("tenderPayload"))
}

// Validate checks the tender itself. The payload is the shipment the other
// organization will build from, so a missing stop or an unpriced move becomes
// their problem rather than ours once it is accepted.
func (p *LoadTenderPayload) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.PurposeCode,
			domainvalidation.ValidEnum[LoadTenderPurposeCode]("Purpose code is invalid"),
		),
		validation.Field(&p.ShipmentID, validation.Required.Error("Shipment is required")),
		validation.Field(&p.CustomerID, validation.Required.Error("Customer is required")),
		validation.Field(&p.ServiceTypeID, validation.Required.Error("Service type is required")),
		validation.Field(&p.Moves, validation.Required.Error("At least one move is required")),
	))

	for i := range p.Moves {
		p.Moves[i].Validate(multiErr.WithIndex("moves", i))
	}
}

func (m *LoadTenderMove) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.Sequence,
			validation.Min(int64(0)).Error("Sequence cannot be negative"),
		),
		validation.Field(&m.Distance,
			validation.Min(0.0).Error("Distance cannot be negative"),
		),
		// A move with one stop has an origin or a destination but not both, so
		// there is no leg for the receiving system to plan.
		validation.Field(&m.Stops,
			validation.Required.Error("A move must have stops"),
			validation.Length(minStopsPerMove, 0).
				Error("A move must have at least two stops"),
		),
	))

	for i := range m.Stops {
		m.Stops[i].Validate(multiErr.WithIndex("stops", i))
	}
}

func (s *LoadTenderStop) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.LocationID, validation.Required.Error("Location is required")),
		validation.Field(&s.Type, validation.Required.Error("Stop type is required")),
		validation.Field(&s.ScheduleType,
			validation.Required.Error("Schedule type is required"),
		),
		validation.Field(&s.Sequence,
			validation.Min(int64(0)).Error("Sequence cannot be negative"),
		),
		validation.Field(&s.ScheduledWindowStart,
			validation.Required.Error("Scheduled window start is required"),
			validation.Min(int64(1)).Error("Scheduled window start must be a valid timestamp"),
		),
		validation.Field(&s.ScheduledWindowEnd,
			validation.Min(s.ScheduledWindowStart).
				Error("Scheduled window end cannot precede its start"),
		),
	))
}

const (
	minStopsPerMove     = 2
	maxWorkflowIDLength = 255
)
