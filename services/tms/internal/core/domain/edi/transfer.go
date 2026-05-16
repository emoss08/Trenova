package edi

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type LoadTenderPayload struct {
	ShipmentID               pulid.ID                         `json:"shipmentId"`
	BusinessUnitID           pulid.ID                         `json:"businessUnitId"`
	OrganizationID           pulid.ID                         `json:"organizationId"`
	ServiceTypeID            pulid.ID                         `json:"serviceTypeId"`
	ServiceTypeLabel         string                           `json:"serviceTypeLabel,omitempty"`
	ShipmentTypeID           pulid.ID                         `json:"shipmentTypeId,omitempty"`
	ShipmentTypeLabel        string                           `json:"shipmentTypeLabel,omitempty"`
	CustomerID               pulid.ID                         `json:"customerId"`
	CustomerLabel            string                           `json:"customerLabel,omitempty"`
	FormulaTemplateID        pulid.ID                         `json:"formulaTemplateId"`
	FormulaTemplateLabel     string                           `json:"formulaTemplateLabel,omitempty"`
	BOL                      string                           `json:"bol,omitempty"`
	Pieces                   *int64                           `json:"pieces,omitempty"`
	Weight                   *int64                           `json:"weight,omitempty"`
	TemperatureMin           *int16                           `json:"temperatureMin,omitempty"`
	TemperatureMax           *int16                           `json:"temperatureMax,omitempty"`
	FreightChargeAmount      decimal.NullDecimal              `json:"freightChargeAmount"`
	OtherChargeAmount        decimal.NullDecimal              `json:"otherChargeAmount"`
	BaseRate                 decimal.NullDecimal              `json:"baseRate"`
	TotalChargeAmount        decimal.NullDecimal              `json:"totalChargeAmount"`
	RatingUnit               int64                            `json:"ratingUnit"`
	RatingDetail             map[string]any                   `json:"ratingDetail,omitempty"`
	Moves                    []LoadTenderMove                 `json:"moves"`
	Commodities              []LoadTenderCommodity            `json:"commodities"`
	AdditionalCharges        []LoadTenderCharge               `json:"additionalCharges"`
	RequiredMappingEntityIDs map[MappingEntityType][]pulid.ID `json:"requiredMappingEntityIds"`
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
	EntityType  MappingEntityType `json:"entityType"`
	SourceID    pulid.ID          `json:"sourceId"`
	SourceLabel string            `json:"sourceLabel,omitempty"`
	TargetID    pulid.ID          `json:"targetId,omitempty"`
	TargetLabel string            `json:"targetLabel,omitempty"`
	Resolved    bool              `json:"resolved"`
}

type EDITransfer struct {
	bun.BaseModel `json:"-" bun:"table:edi_load_tender_transfers,alias:eltt"`

	ID                    pulid.ID            `json:"id"                    bun:"id,pk,type:VARCHAR(100),notnull"`
	SourceOrganizationID  pulid.ID            `json:"sourceOrganizationId"  bun:"source_organization_id,type:VARCHAR(100),notnull"`
	SourceBusinessUnitID  pulid.ID            `json:"sourceBusinessUnitId"  bun:"source_business_unit_id,type:VARCHAR(100),notnull"`
	TargetOrganizationID  pulid.ID            `json:"targetOrganizationId"  bun:"target_organization_id,type:VARCHAR(100),notnull"`
	TargetBusinessUnitID  pulid.ID            `json:"targetBusinessUnitId"  bun:"target_business_unit_id,type:VARCHAR(100),notnull"`
	SourcePartnerID       pulid.ID            `json:"sourcePartnerId"       bun:"source_partner_id,type:VARCHAR(100),notnull"`
	TargetPartnerID       pulid.ID            `json:"targetPartnerId"       bun:"target_partner_id,type:VARCHAR(100),notnull"`
	SourceShipmentID      pulid.ID            `json:"sourceShipmentId"      bun:"source_shipment_id,type:VARCHAR(100),notnull"`
	TargetShipmentID      pulid.ID            `json:"targetShipmentId"      bun:"target_shipment_id,type:VARCHAR(100),nullzero"`
	Status                TransferStatus      `json:"status"                bun:"status,type:edi_load_tender_transfer_status_enum,notnull"`
	TenderPayload         LoadTenderPayload   `json:"tenderPayload"         bun:"tender_payload,type:JSONB,notnull"`
	MappingSnapshot       []MappingResolution `json:"mappingSnapshot"       bun:"mapping_snapshot,type:JSONB,notnull,default:'[]'::jsonb"`
	RejectionReason       string              `json:"rejectionReason"       bun:"rejection_reason,type:TEXT,nullzero"`
	FailureReason         string              `json:"failureReason"         bun:"failure_reason,type:TEXT,nullzero"`
	ApprovalWorkflowID    string              `json:"approvalWorkflowId"    bun:"approval_workflow_id,type:VARCHAR(255),nullzero"`
	ApprovalWorkflowRunID string              `json:"approvalWorkflowRunId" bun:"approval_workflow_run_id,type:VARCHAR(255),nullzero"`
	SubmittedByID         pulid.ID            `json:"submittedById"         bun:"submitted_by_id,type:VARCHAR(100),nullzero"`
	SubmittedAt           int64               `json:"submittedAt"           bun:"submitted_at,type:BIGINT,notnull"`
	ApprovedByID          pulid.ID            `json:"approvedById"          bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	ApprovedAt            *int64              `json:"approvedAt"            bun:"approved_at,type:BIGINT,nullzero"`
	ProcessingStartedAt   *int64              `json:"processingStartedAt"   bun:"processing_started_at,type:BIGINT,nullzero"`
	ProcessedAt           *int64              `json:"processedAt"           bun:"processed_at,type:BIGINT,nullzero"`
	RejectedByID          pulid.ID            `json:"rejectedById"          bun:"rejected_by_id,type:VARCHAR(100),nullzero"`
	RejectedAt            *int64              `json:"rejectedAt"            bun:"rejected_at,type:BIGINT,nullzero"`
	CanceledByID          pulid.ID            `json:"canceledById"          bun:"canceled_by_id,type:VARCHAR(100),nullzero"`
	CanceledAt            *int64              `json:"canceledAt"            bun:"canceled_at,type:BIGINT,nullzero"`
	Version               int64               `json:"version"               bun:"version,type:BIGINT"`
	CreatedAt             int64               `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64               `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	SourcePartner *EDIPartner `json:"sourcePartner,omitempty" bun:"rel:belongs-to,join:source_partner_id=id"`
	TargetPartner *EDIPartner `json:"targetPartner,omitempty" bun:"rel:belongs-to,join:target_partner_id=id"`
}

func (t *EDITransfer) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if t.MappingSnapshot == nil {
		t.MappingSnapshot = []MappingResolution{}
	}
	if t.SubmittedAt == 0 {
		t.SubmittedAt = now
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("edilt_")
		}
		t.CreatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
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
