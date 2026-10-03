package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// SyncBillingQueueIssuesRequest stores what the checks found on an item.
// Findings are matched to stored issues by code and subject: a new one is
// raised, an open one whose finding is gone is cleared, and one the checks
// cleared before is reopened when its finding comes back. An issue a person
// settled stays settled.
type SyncBillingQueueIssuesRequest struct {
	TenantInfo pagination.TenantInfo
	ItemID     pulid.ID
	Findings   []*billingqueue.Issue
}

// SyncBillingQueueIssuesResult is the item's issues after a sync, and which
// changed so the activity can say so.
type SyncBillingQueueIssuesResult struct {
	Issues   []*billingqueue.Issue
	Raised   []*billingqueue.Issue
	Cleared  []*billingqueue.Issue
	Reopened []*billingqueue.Issue
}

type GetBillingQueueIssueRequest struct {
	TenantInfo pagination.TenantInfo
	ItemID     pulid.ID
	IssueID    pulid.ID
}

type ListBillingQueueEventsRequest struct {
	TenantInfo pagination.TenantInfo
	ItemID     pulid.ID
	Limit      int
	// BeforeAt and BeforeID continue a page: entries strictly older than this
	// one, in the timeline's order.
	BeforeAt int64
	BeforeID pulid.ID
}

type BillingQueueEventPage struct {
	Items   []*billingqueue.ItemEvent `json:"items"`
	HasMore bool                      `json:"hasMore"`
	Total   int                       `json:"total"`
}

// GetBillingQueueNeighborsRequest finds an item's place in the queue under the
// same filter the list was read with.
type GetBillingQueueNeighborsRequest struct {
	ItemID        pulid.ID
	Filter        *pagination.QueryOptions
	IncludePosted bool
}

type BillingQueueNeighbors struct {
	PrevID   *pulid.ID `json:"prevId"`
	NextID   *pulid.ID `json:"nextId"`
	Position int       `json:"position"`
	Total    int       `json:"total"`
}

type FindBillingQueueDuplicatesRequest struct {
	TenantInfo       pagination.TenantInfo
	ItemID           pulid.ID
	ShipmentID       pulid.ID
	BillToCustomerID pulid.ID
	BillType         billingqueue.BillType
	InvoiceID        pulid.ID
}

type ListBillingQueueSummariesRequest struct {
	TenantInfo pagination.TenantInfo
	ItemIDs    []pulid.ID
}

// BillingQueueItemSummary is an item's live state as a queue row shows it.
type BillingQueueItemSummary struct {
	ID                   pulid.ID                     `json:"id"                   bun:"id"`
	Number               string                       `json:"number"               bun:"number"`
	Status               billingqueue.Status          `json:"status"               bun:"status"`
	HoldReasonCode       *billingqueue.HoldReasonCode `json:"holdReasonCode"       bun:"hold_reason_code"`
	AssignedBillerID     *pulid.ID                    `json:"assignedBillerId"     bun:"assigned_biller_id"`
	AllocatedTotalAmount decimal.Decimal              `json:"allocatedTotalAmount" bun:"allocated_total_amount"`
	// OpenChecks is how many of the four issue-backed checks wait on an issue.
	OpenChecks int  `json:"-"          bun:"open_checks"`
	NeedsCount int  `json:"needsCount" bun:"-"`
	Ready      bool `json:"ready"      bun:"-"`
}

type CreateApprovalRunRequest struct {
	Run     *billingqueue.ApprovalRun
	ItemIDs []pulid.ID
}

type GetApprovalRunRequest struct {
	TenantInfo   pagination.TenantInfo
	RunID        pulid.ID
	IncludeItems bool
}

type MarkApprovalRunRunningRequest struct {
	TenantInfo         pagination.TenantInfo
	RunID              pulid.ID
	TemporalWorkflowID string
	TemporalRunID      string
}

type RequestApprovalRunUndoRequest struct {
	TenantInfo    pagination.TenantInfo
	RunID         pulid.ID
	RequestedByID pulid.ID
}

type RecordApprovalRunItemRequest struct {
	TenantInfo pagination.TenantInfo
	Item       *billingqueue.ApprovalRunItem
}

type FinalizeApprovalRunRequest struct {
	TenantInfo     pagination.TenantInfo
	RunID          pulid.ID
	Status         billingqueue.ApprovalRunStatus
	FailureMessage string
	// LeftoverCode is what the items nobody got to are marked with.
	LeftoverCode billingqueue.ApprovalFailureCode
}

// BillingQueueReviewRepository stores what a biller's review of an item reads
// and writes beyond the item itself: its issues and activity, its place in
// the queue, and bulk approvals.
type BillingQueueReviewRepository interface {
	ListIssues(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		itemID pulid.ID,
	) ([]*billingqueue.Issue, error)
	SyncIssues(
		ctx context.Context,
		req *SyncBillingQueueIssuesRequest,
	) (*SyncBillingQueueIssuesResult, error)
	GetIssue(ctx context.Context, req *GetBillingQueueIssueRequest) (*billingqueue.Issue, error)
	UpdateIssue(ctx context.Context, issue *billingqueue.Issue) (*billingqueue.Issue, error)
	CountOpenIssues(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		itemID pulid.ID,
	) (int, error)

	CreateEvents(ctx context.Context, events ...*billingqueue.ItemEvent) error
	ListEvents(
		ctx context.Context,
		req *ListBillingQueueEventsRequest,
	) (*BillingQueueEventPage, error)

	GetNeighbors(
		ctx context.Context,
		req *GetBillingQueueNeighborsRequest,
	) (*BillingQueueNeighbors, error)
	FindDuplicates(
		ctx context.Context,
		req *FindBillingQueueDuplicatesRequest,
	) ([]*billingqueue.DuplicateRef, error)
	ListSummaries(
		ctx context.Context,
		req *ListBillingQueueSummariesRequest,
	) ([]*BillingQueueItemSummary, error)

	CreateApprovalRun(
		ctx context.Context,
		req *CreateApprovalRunRequest,
	) (*billingqueue.ApprovalRun, error)
	GetApprovalRun(
		ctx context.Context,
		req *GetApprovalRunRequest,
	) (*billingqueue.ApprovalRun, error)
	GetApprovalRunByKey(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		key string,
	) (*billingqueue.ApprovalRun, error)
	// MarkApprovalRunRunning moves a run out of its undo window. It reports
	// false, and changes nothing, when the run was undone first.
	MarkApprovalRunRunning(
		ctx context.Context,
		req *MarkApprovalRunRunningRequest,
	) (bool, error)
	// RequestApprovalRunUndo stops a run inside its undo window. It reports
	// false when the run had already started.
	RequestApprovalRunUndo(
		ctx context.Context,
		req *RequestApprovalRunUndoRequest,
	) (bool, error)
	SetApprovalRunWorkflow(
		ctx context.Context,
		req *MarkApprovalRunRunningRequest,
	) error
	ListPendingApprovalRunItems(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		runID pulid.ID,
	) ([]*billingqueue.ApprovalRunItem, error)
	// RecordApprovalRunItem writes an item's outcome once; a retried
	// activity writing the same item again changes nothing.
	RecordApprovalRunItem(ctx context.Context, req *RecordApprovalRunItemRequest) error
	FinalizeApprovalRun(
		ctx context.Context,
		req *FinalizeApprovalRunRequest,
	) (*billingqueue.ApprovalRun, error)
}
