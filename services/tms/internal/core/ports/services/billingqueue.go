package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type TransferToBillingRequest struct {
	ShipmentID pulid.ID
	BillType   billingqueue.BillType
	TenantInfo pagination.TenantInfo
	// AutoApprove clears every new item straight through the queue, for freight
	// that passed every billing requirement and whose payers all asked for clean
	// loads to pass without review. The caller has already made that decision —
	// see ShipmentBillingReadiness.ShouldAutoApproveBilling.
	AutoApprove bool
	// AutoApprovePayerIDs narrows auto-approval to these payers' items when the
	// shipment is split-billed and only some payers opted in.
	AutoApprovePayerIDs []pulid.ID
	DetailedShipment    *shipment.Shipment
}

// TransferToBillingResult is every queue item a transfer created: one per payer
// of the shipment. Primary is the item for the shipment's own payer.
type TransferToBillingResult struct {
	Items   []*billingqueue.BillingQueueItem `json:"items"`
	Primary *billingqueue.BillingQueueItem   `json:"primary"`
}

type AssignBillerRequest struct {
	ItemID     pulid.ID
	BillerID   pulid.ID
	TenantInfo pagination.TenantInfo
}

type UpdateBillingQueueStatusRequest struct {
	ItemID              pulid.ID
	NewStatus           billingqueue.Status
	ExceptionReasonCode *billingqueue.ExceptionReasonCode
	ExceptionNotes      string
	ReviewNotes         string
	CancelReason        string
	TenantInfo          pagination.TenantInfo
}

type BillingQueueStats struct {
	ReadyForReview int `json:"readyForReview"`
	InReview       int `json:"inReview"`
	Approved       int `json:"approved"`
	Posted         int `json:"posted"`
	OnHold         int `json:"onHold"`
	Exception      int `json:"exception"`
	SentBackToOps  int `json:"sentBackToOps"`
	Canceled       int `json:"canceled"`
	Total          int `json:"total"`
}

type UpdateChargesRequest struct {
	ItemID            pulid.ID
	FormulaTemplateID *pulid.ID
	BaseRate          *decimal.Decimal
	AdditionalCharges []*shipment.AdditionalCharge
	TenantInfo        pagination.TenantInfo
	// ConvertAmountSplitsToPercent answers the refusal an edit gets when it
	// changes a charge that is split by amount: the split is rewritten as
	// percentages that keep each payer's proportion.
	ConvertAmountSplitsToPercent bool
}

// ReassignChargeRequest changes who pays for one charge on the item's shipment.
// An empty Allocations list gives the charge back whole to the shipment's payer.
type ReassignChargeRequest struct {
	ItemID             pulid.ID
	TenantInfo         pagination.TenantInfo
	ChargeKind         shipment.ChargeAllocationKind
	AdditionalChargeID pulid.ID
	Allocations        []*shipment.ChargeAllocation
}

// ReassignChargeResult is the queue after a reassignment: the item the request
// came from (canceled if its payer no longer pays anything), every active item
// for the shipment, and which items the reassignment created or canceled.
type ReassignChargeResult struct {
	Item            *billingqueue.BillingQueueItem   `json:"item"`
	Items           []*billingqueue.BillingQueueItem `json:"items"`
	CreatedItemIDs  []pulid.ID                       `json:"createdItemIds"`
	CanceledItemIDs []pulid.ID                       `json:"canceledItemIds"`
}

type BillingQueueService interface {
	List(
		ctx context.Context,
		req *repositories.ListBillingQueueItemsRequest,
	) (*pagination.ListResult[*billingqueue.BillingQueueItem], error)
	GetByID(
		ctx context.Context,
		req *repositories.GetBillingQueueItemByIDRequest,
	) (*billingqueue.BillingQueueItem, error)
	GetStats(
		ctx context.Context,
		req *repositories.GetBillingQueueStatsRequest,
	) (*BillingQueueStats, error)
	TransferToBilling(
		ctx context.Context,
		req *TransferToBillingRequest,
		actor *RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	TransferToBillingItems(
		ctx context.Context,
		req *TransferToBillingRequest,
		actor *RequestActor,
	) (*TransferToBillingResult, error)
	AssignBiller(
		ctx context.Context,
		req *AssignBillerRequest,
		actor *RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	UpdateStatus(
		ctx context.Context,
		req *UpdateBillingQueueStatusRequest,
		actor *RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	UpdateCharges(
		ctx context.Context,
		req *UpdateChargesRequest,
		actor *RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	ReassignCharge(
		ctx context.Context,
		req *ReassignChargeRequest,
		actor *RequestActor,
	) (*ReassignChargeResult, error)
}
