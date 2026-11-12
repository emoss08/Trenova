package billingqueueports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type AssistAvailability struct {
	Review bool `json:"review"`
	Brief  bool `json:"brief"`
	Search bool `json:"search"`
}

type AgentReviewOutcome struct {
	Reviewed bool                      `json:"reviewed"`
	Verdict  billingqueue.AgentVerdict `json:"verdict,omitempty"`
}

type QueueBriefView struct {
	Available   bool   `json:"available"`
	Text        string `json:"text"`
	GeneratedAt int64  `json:"generatedAt"`
	Stale       bool   `json:"stale"`
}

// SearchIntentRequest is a sentence to turn into queue filters, with the
// person's own date (YYYY-MM-DD) so "last week" means their week.
type SearchIntentRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Text       string
	Today      string
}

// SearchDates is a window of days a sentence names: the last N days, or a range
// of calendar days (YYYY-MM-DD, both ends kept) in the person's own calendar,
// which the client turns into its timezone's moments.
type SearchDates struct {
	LastDays int    `json:"lastDays,omitempty"`
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
}

type SearchIntent struct {
	Lanes                []billingqueue.Lane `json:"lanes"`
	Search               string              `json:"search"`
	PayerID              pulid.ID            `json:"payerId,omitempty"`
	PayerName            string              `json:"payerName,omitempty"`
	ShipmentCustomerID   pulid.ID            `json:"shipmentCustomerId,omitempty"`
	ShipmentCustomerName string              `json:"shipmentCustomerName,omitempty"`
	MinAmount            decimal.NullDecimal `json:"minAmount"`
	MaxAmount            decimal.NullDecimal `json:"maxAmount"`
	CreatedBefore        *int64              `json:"createdBefore"`
	Delivered            *SearchDates        `json:"delivered"`
	PickedUp             *SearchDates        `json:"pickedUp"`
	Queued               *SearchDates        `json:"queued"`
	Unresolved           string              `json:"unresolved,omitempty"`
}

type AutoApproveOutcome struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}

type RecentInvoice struct {
	ID          pulid.ID        `json:"id"`
	Number      string          `json:"number"`
	InvoiceDate int64           `json:"invoiceDate"`
	Subtotal    decimal.Decimal `json:"subtotal"`
	Other       decimal.Decimal `json:"other"`
	Total       decimal.Decimal `json:"total"`
	Charges     []*RecentCharge `json:"charges"`
}

// RecentCharge is what one invoice billed for one charge, its lines summed.
type RecentCharge struct {
	Type        invoice.InvoiceLineType `json:"type"`
	Code        string                  `json:"code"`
	Description string                  `json:"description"`
	Amount      decimal.Decimal         `json:"amount"`
}

type BillingQueueAssistService interface {
	Availability(ctx context.Context, tenantInfo pagination.TenantInfo) (*AssistAvailability, error)
	ReviewItem(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		itemID pulid.ID,
	) (*AgentReviewOutcome, error)
	AutoApprove(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		itemID pulid.ID,
	) (*AutoApproveOutcome, error)
	RefreshBrief(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error)
	GetBrief(ctx context.Context, tenantInfo pagination.TenantInfo) (*QueueBriefView, error)
	ComposeSearch(ctx context.Context, req *SearchIntentRequest) (*SearchIntent, error)
	SubmitReviews(ctx context.Context, req *SubmitReviewsRequest) (*ReviewSubmission, error)
	ReviewBatchEnded(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		submission *ReviewSubmission,
	) (bool, error)
	CancelReviews(ctx context.Context, tenantInfo pagination.TenantInfo, submission *ReviewSubmission) error
	ApplyReviews(ctx context.Context, req *ApplyReviewsRequest) (int, error)
	// NovaGeneration is the bulk generation under way, starting one when the
	// queue is missing reviews or its brief; nil when nothing is generating.
	NovaGeneration(ctx context.Context, tenantInfo pagination.TenantInfo) (*NovaGeneration, error)
	// HasUnreviewedItems is whether the queue has items Nova can review and
	// has not.
	HasUnreviewedItems(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error)
}

type BriefRequester interface {
	RequestBrief(ctx context.Context, tenantInfo pagination.TenantInfo)
}
