package billingtransferjobs

import (
	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	BulkBillingTransferWorkflowName = "BulkBillingTransferWorkflow"
	ReconcileZombieRunsWorkflowName = "ReconcileZombieBillingTransferRunsWorkflow"

	// CancelSignalName shortens how long a cancel waits to take effect. It is
	// never the authority for one: the run row's cancel_requested_at is, because
	// that survives a worker restart and a Temporal outage and is visible to the
	// dialog the instant it is written.
	CancelSignalName = "billing-transfer-cancel"
)

const (
	// BatchSize is how many shipments one activity claims. It stays under the
	// service's own MaxBulkTransferToBillingShipments cap of 100, keeps an
	// activity comfortably inside its ten-minute budget, and bounds how long a
	// cancel waits: the batch in flight always finishes.
	BatchSize = 25

	// flushEvery is how often outcomes are written back mid-batch. This is what
	// the progress bar actually moves on, so it is deliberately much smaller
	// than the batch: 25 would make the bar jump exactly as coarsely as the old
	// client-driven loop did.
	flushEvery = 5

	ErrTypeTransferValidation = "BILLING_TRANSFER_VALIDATION"
	ErrTypeTransferCandidates = "BILLING_TRANSFER_CANDIDATES"
)

type TransferRunPayload struct {
	temporaltype.BasePayload

	RunID pulid.ID `json:"runId"`
}

type PreparedRun struct {
	RunID          pulid.ID `json:"runId"`
	RequestedByID  pulid.ID `json:"requestedById"`
	TotalCount     int      `json:"totalCount"`
	UnmatchedCount int      `json:"unmatchedCount"`
	BatchSize      int      `json:"batchSize"`
}

type ProcessBatchPayload struct {
	temporaltype.BasePayload

	RunID pulid.ID `json:"runId"`
	Limit int      `json:"limit"`
}

type ProcessBatchResult struct {
	// Processed is how many shipments this batch answered for. Zero means the
	// run has nothing Pending left, which is how the loop learns to stop.
	Processed       int  `json:"processed"`
	Transferred     int  `json:"transferred"`
	NotTransferred  int  `json:"notTransferred"`
	CancelRequested bool `json:"cancelRequested"`
}

type FinalizeRunPayload struct {
	temporaltype.BasePayload

	RunID          pulid.ID                  `json:"runId"`
	Status         billingtransfer.RunStatus `json:"status"`
	FailureMessage string                    `json:"failureMessage,omitempty"`
}

type TransferRunResult struct {
	RunID                     pulid.ID                  `json:"runId"`
	Status                    billingtransfer.RunStatus `json:"status"`
	TotalCount                int                       `json:"totalCount"`
	ProcessedCount            int                       `json:"processedCount"`
	TransferredCount          int                       `json:"transferredCount"`
	NotTransferredCount       int                       `json:"notTransferredCount"`
	SkippedCount              int                       `json:"skippedCount"`
	MarkedReadyToInvoiceCount int                       `json:"markedReadyToInvoiceCount"`
	RetryableCount            int                       `json:"retryableCount"`
	UnmatchedCount            int                       `json:"unmatchedCount"`
}

type ReconcileZombieRunsResult struct {
	Examined int `json:"examined"`
	Failed   int `json:"failed"`
}
