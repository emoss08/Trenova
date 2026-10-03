package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ResolveBillingQueueIssueRequest settles an issue with one of its options.
type ResolveBillingQueueIssueRequest struct {
	ItemID     pulid.ID
	IssueID    pulid.ID
	OptionKey  string
	TenantInfo pagination.TenantInfo
}

// UndoBillingQueueIssueRequest takes a settlement back and reopens the issue.
type UndoBillingQueueIssueRequest struct {
	ItemID     pulid.ID
	IssueID    pulid.ID
	TenantInfo pagination.TenantInfo
}

type BillingQueueItemRequest struct {
	ItemID     pulid.ID
	TenantInfo pagination.TenantInfo
}

// PostBillingQueueItemResult is the invoice an item posted as, and where it
// went.
type PostBillingQueueItemResult struct {
	Item          *billingqueue.BillingQueueItem `json:"item"`
	InvoiceID     pulid.ID                       `json:"invoiceId"`
	InvoiceNumber string                         `json:"invoiceNumber"`
	// SentTo is the address the invoice is emailed to when the bill-to has
	// emailing on; empty when posting does not send it.
	SentTo     string   `json:"sentTo"`
	Recipients []string `json:"recipients"`
}

type ListBillingQueueActivityRequest struct {
	ItemID     pulid.ID
	TenantInfo pagination.TenantInfo
	Limit      int
	BeforeAt   int64
	BeforeID   pulid.ID
}

// ApproveIfReadyResult is what one item's approval inside a bulk run came to.
type ApproveIfReadyResult struct {
	Item          *billingqueue.BillingQueueItem
	Approved      bool
	FailureCode   billingqueue.ApprovalFailureCode
	Reason        string
	InvoiceID     *pulid.ID
	InvoiceNumber string
}

// BillingQueueReviewService is a biller's review of one item from the item
// itself: settling what its checks raised, holding and releasing it, posting
// its invoice, its activity, and its place in the queue.
type BillingQueueReviewService interface {
	ResolveIssue(
		ctx context.Context,
		req *ResolveBillingQueueIssueRequest,
		actor *RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	UndoIssue(
		ctx context.Context,
		req *UndoBillingQueueIssueRequest,
		actor *RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	Release(
		ctx context.Context,
		req *BillingQueueItemRequest,
		actor *RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	Post(
		ctx context.Context,
		req *BillingQueueItemRequest,
		actor *RequestActor,
	) (*PostBillingQueueItemResult, error)
	ListActivity(
		ctx context.Context,
		req *ListBillingQueueActivityRequest,
	) (*repositories.BillingQueueEventPage, error)
	Neighbors(
		ctx context.Context,
		req *repositories.GetBillingQueueNeighborsRequest,
	) (*repositories.BillingQueueNeighbors, error)
	Summaries(
		ctx context.Context,
		req *repositories.ListBillingQueueSummariesRequest,
	) ([]*repositories.BillingQueueItemSummary, error)
	// ApproveIfReady re-reads the item, re-runs its checks and approves it
	// only if every one passes. It is what a bulk approval does per item.
	ApproveIfReady(
		ctx context.Context,
		req *BillingQueueItemRequest,
		actor *RequestActor,
	) (*ApproveIfReadyResult, error)
}

type StartBillingQueueApprovalRequest struct {
	TenantInfo     pagination.TenantInfo
	ItemIDs        []pulid.ID
	IdempotencyKey string
}

type BillingQueueApprovalRunRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
}

// BillingQueueApprovalService approves several queue items as one job that
// waits out an undo window before it writes anything.
type BillingQueueApprovalService interface {
	Start(
		ctx context.Context,
		req *StartBillingQueueApprovalRequest,
	) (*billingqueue.ApprovalRun, error)
	Get(
		ctx context.Context,
		req *BillingQueueApprovalRunRequest,
	) (*billingqueue.ApprovalRun, error)
	Undo(
		ctx context.Context,
		req *BillingQueueApprovalRunRequest,
	) (*billingqueue.ApprovalRun, error)
}
