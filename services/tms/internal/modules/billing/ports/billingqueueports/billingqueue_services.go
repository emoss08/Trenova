package billingqueueports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
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
	// HoldReasonCode is why the item goes on hold; read only for OnHold.
	HoldReasonCode *billingqueue.HoldReasonCode
	TenantInfo     pagination.TenantInfo
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
	TariffInputs      tarifftypes.Inputs
	AdditionalCharges []*shipment.AdditionalCharge
	TenantInfo        pagination.TenantInfo
	// ConvertAmountSplitsToPercent answers the refusal an edit gets when it
	// changes a charge that is split by amount: the split is rewritten as
	// percentages that keep each payer's proportion.
	ConvertAmountSplitsToPercent bool
}

// ChargeUpdatePreview is a billing queue item's shipment before and after a
// charge edit. Before is the shipment as it was read; its charges are the
// ones the edit replaces.
type ChargeUpdatePreview struct {
	Item            *billingqueue.BillingQueueItem
	Before          *shipment.Shipment
	After           *shipment.Shipment
	ConvertedSplits int
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
		req *ListBillingQueueItemsRequest,
	) (*pagination.ListResult[*billingqueue.BillingQueueItem], error)
	GetByID(
		ctx context.Context,
		req *GetBillingQueueItemByIDRequest,
	) (*billingqueue.BillingQueueItem, error)
	GetStats(
		ctx context.Context,
		req *GetBillingQueueStatsRequest,
	) (*BillingQueueStats, error)
	TransferToBilling(
		ctx context.Context,
		req *TransferToBillingRequest,
		actor *permissionports.RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	TransferToBillingItems(
		ctx context.Context,
		req *TransferToBillingRequest,
		actor *permissionports.RequestActor,
	) (*TransferToBillingResult, error)
	AssignBiller(
		ctx context.Context,
		req *AssignBillerRequest,
		actor *permissionports.RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	CheckBiller(ctx context.Context, tenantInfo pagination.TenantInfo, billerID pulid.ID) error
	UpdateStatus(
		ctx context.Context,
		req *UpdateBillingQueueStatusRequest,
		actor *permissionports.RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	UpdateCharges(
		ctx context.Context,
		req *UpdateChargesRequest,
		actor *permissionports.RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	// PreviewUpdateCharges is UpdateCharges up to the save: the item's
	// shipment with the charges and totals the edit would leave. A rate or
	// formula change reprices through the rating engine, which writes, and is
	// refused rather than previewed.
	PreviewUpdateCharges(
		ctx context.Context,
		req *UpdateChargesRequest,
		actor *permissionports.RequestActor,
	) (*ChargeUpdatePreview, error)
	ReassignCharge(
		ctx context.Context,
		req *ReassignChargeRequest,
		actor *permissionports.RequestActor,
	) (*ReassignChargeResult, error)
	PreviewReassignCharge(
		ctx context.Context,
		req *ReassignChargeRequest,
		actor *permissionports.RequestActor,
	) (*ReassignChargePreview, error)
}

type ReassignChargePreview struct {
	Item        *billingqueue.BillingQueueItem
	Shipment    *shipment.Shipment
	Description string
	Shares      []*chargeallocation.PayerShare
	ToCreate    []*chargeallocation.PayerShare
	ToCancel    []*billingqueue.BillingQueueItem
}

func (r *ReassignChargeResult) WithoutFormulaContent() *ReassignChargeResult {
	if r == nil {
		return nil
	}

	redacted := *r
	redacted.Item = r.Item.WithoutFormulaContent()
	redacted.Items = billingqueue.WithoutFormulaContent(r.Items)

	return &redacted
}
