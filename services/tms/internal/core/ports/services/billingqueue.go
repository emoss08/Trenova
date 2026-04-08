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
}
