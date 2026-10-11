package billingqueue

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*ApprovalRun)(nil)
	_ bun.BeforeAppendModelHook = (*ApprovalRunItem)(nil)
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
	bun.BaseModel `bun:"table:billingqueue_approval_runs,alias:bqar" json:"-"`

	ID                  pulid.ID          `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID          `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID          `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RequestedByID       pulid.ID          `json:"requestedById"       bun:"requested_by_id,type:VARCHAR(100),notnull"`
	IdempotencyKey      string            `json:"idempotencyKey"      bun:"idempotency_key,type:VARCHAR(100),notnull"`
	Status              ApprovalRunStatus `json:"status"              bun:"status,type:VARCHAR(20),notnull"`
	TotalCount          int               `json:"totalCount"          bun:"total_count,type:INTEGER,notnull"`
	ApprovedCount       int               `json:"approvedCount"       bun:"approved_count,type:INTEGER,notnull"`
	FailedCount         int               `json:"failedCount"         bun:"failed_count,type:INTEGER,notnull"`
	SkippedCount        int               `json:"skippedCount"        bun:"skipped_count,type:INTEGER,notnull"`
	CommitAt            int64             `json:"commitAt"            bun:"commit_at,type:BIGINT,notnull"`
	FailureMessage      string            `json:"failureMessage"      bun:"failure_message,type:TEXT,nullzero"`
	CancelRequestedAt   *int64            `json:"cancelRequestedAt"   bun:"cancel_requested_at,type:BIGINT,nullzero"`
	CancelRequestedByID *pulid.ID         `json:"cancelRequestedById" bun:"cancel_requested_by_id,type:VARCHAR(100),nullzero"`
	TemporalWorkflowID  string            `json:"-"                   bun:"temporal_workflow_id,type:VARCHAR(255),nullzero"`
	TemporalRunID       string            `json:"-"                   bun:"temporal_run_id,type:VARCHAR(255),nullzero"`
	StartedAt           *int64            `json:"startedAt"           bun:"started_at,type:BIGINT,nullzero"`
	CompletedAt         *int64            `json:"completedAt"         bun:"completed_at,type:BIGINT,nullzero"`
	Version             int64             `json:"version"             bun:"version,type:BIGINT,notnull"`
	CreatedAt           int64             `json:"createdAt"           bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt           int64             `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull"`

	Items []*ApprovalRunItem `json:"items,omitempty" bun:"rel:has-many,join:id=run_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (r *ApprovalRun) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("bqar_")
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
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
	bun.BaseModel `bun:"table:billingqueue_approval_run_items,alias:bqari" json:"-"`

	ID             pulid.ID            `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID            `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RunID          pulid.ID            `json:"runId"          bun:"run_id,type:VARCHAR(100),notnull"`
	ItemID         pulid.ID            `json:"itemId"         bun:"item_id,type:VARCHAR(100),notnull"`
	Sequence       int                 `json:"sequence"       bun:"sequence,type:INTEGER,notnull"`
	Status         ApprovalItemStatus  `json:"status"         bun:"status,type:VARCHAR(20),notnull"`
	FailureCode    ApprovalFailureCode `json:"failureCode"    bun:"failure_code,type:VARCHAR(30),nullzero"`
	ErrorMessage   string              `json:"errorMessage"   bun:"error_message,type:TEXT,nullzero"`
	InvoiceID      *pulid.ID           `json:"invoiceId"      bun:"invoice_id,type:VARCHAR(100),nullzero"`
	InvoiceNumber  string              `json:"invoiceNumber"  bun:"invoice_number,type:VARCHAR(100),nullzero"`
	ProcessedAt    *int64              `json:"processedAt"    bun:"processed_at,type:BIGINT,nullzero"`
	CreatedAt      int64               `json:"createdAt"      bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt      int64               `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull"`
}

func (i *ApprovalRunItem) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("bqari_")
		}
		i.CreatedAt = now
		i.UpdatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}

func (i *ApprovalRunItem) GetID() pulid.ID             { return i.ID }
func (i *ApprovalRunItem) GetOrganizationID() pulid.ID { return i.OrganizationID }
func (i *ApprovalRunItem) GetBusinessUnitID() pulid.ID { return i.BusinessUnitID }
func (i *ApprovalRunItem) GetTableName() string        { return "billingqueue_approval_run_items" }
func (i *ApprovalRunItem) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{TableAlias: "bqari"}
}
