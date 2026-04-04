package documentaiextraction

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type Status string

const (
	StatusPending   Status = "Pending"
	StatusCompleted Status = "Completed"
	StatusFailed    Status = "Failed"
	StatusApplied   Status = "Applied"
	StatusSkipped   Status = "Skipped"
)

type Extraction struct {
	bun.BaseModel `bun:"table:document_ai_extractions,alias:dae" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	DocumentID     pulid.ID `json:"documentId"     bun:"document_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	ExtractedAt    int64    `json:"extractedAt"    bun:"extracted_at,type:BIGINT,notnull"`
	RequestHash    string   `json:"requestHash"    bun:"request_hash,type:VARCHAR(64),notnull"`
	WorkflowID     string   `json:"workflowId"     bun:"workflow_id,type:VARCHAR(255),notnull"`
	WorkflowRunID  string   `json:"workflowRunId"  bun:"workflow_run_id,type:VARCHAR(255),notnull"`
	ActivityID     string   `json:"activityId"     bun:"activity_id,type:VARCHAR(255),notnull"`
	TaskToken      []byte   `json:"taskToken"      bun:"task_token,type:BYTEA,notnull"`
	ResponseID     string   `json:"responseId"     bun:"response_id,type:VARCHAR(255),nullzero"`
	Model          string   `json:"model"          bun:"model,type:VARCHAR(100),nullzero"`
	Status         Status   `json:"status"         bun:"status,type:VARCHAR(32),notnull,default:'Pending'"`
	FailureCode    string   `json:"failureCode"    bun:"failure_code,type:VARCHAR(100),nullzero"`
	FailureMessage string   `json:"failureMessage" bun:"failure_message,type:TEXT,nullzero"`
	SubmittedAt    *int64   `json:"submittedAt"    bun:"submitted_at,type:BIGINT,nullzero"`
	LastPolledAt   *int64   `json:"lastPolledAt"   bun:"last_polled_at,type:BIGINT,nullzero"`
	CompletedAt    *int64   `json:"completedAt"    bun:"completed_at,type:BIGINT,nullzero"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (e *Extraction) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("dax_")
		}
		if e.Status == "" {
			e.Status = StatusPending
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
}
