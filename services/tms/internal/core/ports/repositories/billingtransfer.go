package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetBillingTransferRunRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
}

// GetActiveBillingTransferRunRequest finds the one run a user still has going,
// which is what lets a reopened dialog reattach instead of starting a second run.
type GetActiveBillingTransferRunRequest struct {
	TenantInfo    pagination.TenantInfo
	RequestedByID pulid.ID
}

type SeedBillingTransferRunItemsRequest struct {
	TenantInfo  pagination.TenantInfo
	RunID       pulid.ID
	ShipmentIDs []pulid.ID
}

// SeedBillingTransferRetryItemsRequest copies every item of the source run
// forward: retryable outcomes are reset to Pending, the rest keep the answer they
// already have, so the retry run reads as a complete report on its own.
type SeedBillingTransferRetryItemsRequest struct {
	TenantInfo  pagination.TenantInfo
	RunID       pulid.ID
	SourceRunID pulid.ID
}

type SeedBillingTransferRetryItemsResult struct {
	TotalCount                int
	PendingCount              int
	TransferredCount          int
	NotTransferredCount       int
	MarkedReadyToInvoiceCount int
}

type MarkBillingTransferRunRunningRequest struct {
	TenantInfo         pagination.TenantInfo
	RunID              pulid.ID
	TemporalWorkflowID string
	TemporalRunID      string
	TotalCount         int
	UnmatchedCount     int
}

type NextPendingBillingTransferItemsRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
	Limit      int
}

// BillingTransferItemOutcome is one shipment's answer, written back to the item
// row the run was seeded with.
type BillingTransferItemOutcome struct {
	ShipmentID           pulid.ID
	ProNumber            string
	Status               billingtransfer.ItemStatus
	FailureCode          billingtransfer.FailureCode
	ErrorMessage         string
	MarkedReadyToInvoice bool
	BillingQueueItemID   pulid.ID
	BillingQueueNumber   string
	BillingQueueStatus   billingqueue.Status
	MissingRequirements  []billingtransfer.MissingRequirement
	ValidationFailures   []billingtransfer.ValidationFailure
	ProcessedAt          int64
}

type RecordBillingTransferOutcomesRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
	Outcomes   []BillingTransferItemOutcome
}

// BillingTransferRunProgress is what the workflow needs back after recording a
// flush: the counters to report and whether somebody asked it to stop.
type BillingTransferRunProgress struct {
	Status                    billingtransfer.RunStatus
	TotalCount                int
	ProcessedCount            int
	TransferredCount          int
	NotTransferredCount       int
	SkippedCount              int
	MarkedReadyToInvoiceCount int
	RetryableCount            int
	CancelRequested           bool
}

type RequestBillingTransferRunCancelRequest struct {
	TenantInfo    pagination.TenantInfo
	RunID         pulid.ID
	RequestedByID pulid.ID
}

type FinalizeBillingTransferRunRequest struct {
	TenantInfo     pagination.TenantInfo
	RunID          pulid.ID
	Status         billingtransfer.RunStatus
	FailureMessage string
}

type ListBillingTransferRunItemsRequest struct {
	Filter   *pagination.QueryOptions
	Cursor   pagination.CursorInfo
	RunID    pulid.ID
	Statuses []billingtransfer.ItemStatus
}

type ListStaleBillingTransferRunsRequest struct {
	Statuses          []billingtransfer.RunStatus
	UpdatedBeforeUnix int64
	Limit             int
}

type BillingTransferRunRepository interface {
	Create(
		ctx context.Context,
		entity *billingtransfer.BillingTransferRun,
	) (*billingtransfer.BillingTransferRun, error)
	Update(
		ctx context.Context,
		entity *billingtransfer.BillingTransferRun,
	) (*billingtransfer.BillingTransferRun, error)
	GetByID(
		ctx context.Context,
		req *GetBillingTransferRunRequest,
	) (*billingtransfer.BillingTransferRun, error)
	GetActive(
		ctx context.Context,
		req *GetActiveBillingTransferRunRequest,
	) (*billingtransfer.BillingTransferRun, error)

	SeedItems(ctx context.Context, req *SeedBillingTransferRunItemsRequest) (int, error)
	SeedRetryItems(
		ctx context.Context,
		req *SeedBillingTransferRetryItemsRequest,
	) (*SeedBillingTransferRetryItemsResult, error)

	MarkRunning(
		ctx context.Context,
		req *MarkBillingTransferRunRunningRequest,
	) (*billingtransfer.BillingTransferRun, error)
	NextPendingShipmentIDs(
		ctx context.Context,
		req *NextPendingBillingTransferItemsRequest,
	) ([]pulid.ID, error)
	RecordItemOutcomes(
		ctx context.Context,
		req *RecordBillingTransferOutcomesRequest,
	) (*BillingTransferRunProgress, error)
	RequestCancel(
		ctx context.Context,
		req *RequestBillingTransferRunCancelRequest,
	) (*billingtransfer.BillingTransferRun, error)
	Finalize(
		ctx context.Context,
		req *FinalizeBillingTransferRunRequest,
	) (*billingtransfer.BillingTransferRun, error)

	ListItems(
		ctx context.Context,
		req *ListBillingTransferRunItemsRequest,
	) (*pagination.CursorListResult[*billingtransfer.BillingTransferRunItem], error)
	ListStale(
		ctx context.Context,
		req *ListStaleBillingTransferRunsRequest,
	) ([]*billingtransfer.BillingTransferRun, error)
}
