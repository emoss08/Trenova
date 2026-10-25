package billingqueue

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// MaxApprovalRunItems bounds one bulk approval to what a queue table shows.
const MaxApprovalRunItems = 500

// ApprovalUndoWindowSeconds is how long a bulk approval waits for an undo
// before it touches anything.
const ApprovalUndoWindowSeconds = 6

// ApprovalRunStatus is where a bulk approval stands.
type ApprovalRunStatus string

const (
	// ApprovalRunScheduled is inside the undo window; nothing is approved yet.
	ApprovalRunScheduled = ApprovalRunStatus("Scheduled")
	ApprovalRunRunning   = ApprovalRunStatus("Running")
	ApprovalRunCompleted = ApprovalRunStatus("Completed")
	// ApprovalRunUndone was taken back inside the window and wrote nothing.
	ApprovalRunUndone = ApprovalRunStatus("Undone")
	ApprovalRunFailed = ApprovalRunStatus("Failed")
)

func (s ApprovalRunStatus) IsTerminal() bool {
	return s == ApprovalRunCompleted || s == ApprovalRunUndone || s == ApprovalRunFailed
}

// ApprovalItemStatus is what became of one item in a bulk approval.
type ApprovalItemStatus string

const (
	ApprovalItemPending  = ApprovalItemStatus("Pending")
	ApprovalItemApproved = ApprovalItemStatus("Approved")
	ApprovalItemFailed   = ApprovalItemStatus("Failed")
	ApprovalItemSkipped  = ApprovalItemStatus("Skipped")
)

// ApprovalFailureCode says why an item was not approved.
type ApprovalFailureCode string

const (
	ApprovalFailureNotReady    = ApprovalFailureCode("NotReady")
	ApprovalFailureAlreadyDone = ApprovalFailureCode("AlreadyDone")
	ApprovalFailureOnHold      = ApprovalFailureCode("OnHold")
	ApprovalFailureUndone      = ApprovalFailureCode("Undone")
	ApprovalFailureUnexpected  = ApprovalFailureCode("Unexpected")
)

// ApprovalRun is one press of Approve over several queue items.
type ApprovalRun struct {
	keel.Table `keel:"billingqueue_approval_runs"`

	ID                  pulid.ID          `json:"id"                  keel:"id,pk,autoid:bqar_"`
	BusinessUnitID      pulid.ID          `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	OrganizationID      pulid.ID          `json:"organizationId"      keel:"organization_id,pk,tenant"`
	RequestedByID       pulid.ID          `json:"requestedById"       keel:"requested_by_id"`
	IdempotencyKey      string            `json:"idempotencyKey"      keel:"idempotency_key"`
	Status              ApprovalRunStatus `json:"status"              keel:"status"`
	TotalCount          int               `json:"totalCount"          keel:"total_count"`
	ApprovedCount       int               `json:"approvedCount"       keel:"approved_count"`
	FailedCount         int               `json:"failedCount"         keel:"failed_count"`
	SkippedCount        int               `json:"skippedCount"        keel:"skipped_count"`
	CommitAt            int64             `json:"commitAt"            keel:"commit_at"`
	FailureMessage      string            `json:"failureMessage"      keel:"failure_message,nullzero"`
	CancelRequestedAt   *int64            `json:"cancelRequestedAt"   keel:"cancel_requested_at"`
	CancelRequestedByID *pulid.ID         `json:"cancelRequestedById" keel:"cancel_requested_by_id"`
	TemporalWorkflowID  string            `json:"-"                   keel:"temporal_workflow_id,nullzero"`
	TemporalRunID       string            `json:"-"                   keel:"temporal_run_id,nullzero"`
	StartedAt           *int64            `json:"startedAt"           keel:"started_at"`
	CompletedAt         *int64            `json:"completedAt"         keel:"completed_at"`
	Version             int64             `json:"version"             keel:"version,version"`
	CreatedAt           int64             `json:"createdAt"           keel:"created_at,autocreate"`
	UpdatedAt           int64             `json:"updatedAt"           keel:"updated_at,autoupdate"`

	Items []*ApprovalRunItem `json:"items,omitempty" keel:"many:run_id"`
}

func (r *ApprovalRun) GetID() pulid.ID             { return r.ID }
func (r *ApprovalRun) GetOrganizationID() pulid.ID { return r.OrganizationID }
func (r *ApprovalRun) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }
func (r *ApprovalRun) GetTableName() string        { return "billingqueue_approval_runs" }
func (r *ApprovalRun) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{TableAlias: "bqar"}
}

// ApprovalRunItem is one item inside a bulk approval.
type ApprovalRunItem struct {
	keel.Table `keel:"billingqueue_approval_run_items"`

	ID             pulid.ID            `json:"id"             keel:"id,pk,autoid:bqari_"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID            `json:"organizationId" keel:"organization_id,pk,tenant"`
	RunID          pulid.ID            `json:"runId"          keel:"run_id"`
	ItemID         pulid.ID            `json:"itemId"         keel:"item_id"`
	Sequence       int                 `json:"sequence"       keel:"sequence"`
	Status         ApprovalItemStatus  `json:"status"         keel:"status"`
	FailureCode    ApprovalFailureCode `json:"failureCode"    keel:"failure_code,nullzero"`
	ErrorMessage   string              `json:"errorMessage"   keel:"error_message,nullzero"`
	InvoiceID      *pulid.ID           `json:"invoiceId"      keel:"invoice_id"`
	InvoiceNumber  string              `json:"invoiceNumber"  keel:"invoice_number,nullzero"`
	ProcessedAt    *int64              `json:"processedAt"    keel:"processed_at"`
	CreatedAt      int64               `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64               `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (i *ApprovalRunItem) GetID() pulid.ID             { return i.ID }
func (i *ApprovalRunItem) GetOrganizationID() pulid.ID { return i.OrganizationID }
func (i *ApprovalRunItem) GetBusinessUnitID() pulid.ID { return i.BusinessUnitID }
func (i *ApprovalRunItem) GetTableName() string        { return "billingqueue_approval_run_items" }
func (i *ApprovalRunItem) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{TableAlias: "bqari"}
}
