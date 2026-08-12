package documentaiextraction

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	Version        int64    `json:"version"        bun:"version,type:BIGINT,notnull"`
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

func (e *Extraction) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&e.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&e.DocumentID, validation.Required.Error("Document is required")),
		validation.Field(&e.UserID, validation.Required.Error("User is required")),
		validation.Field(&e.RequestHash,
			validation.Required.Error("Request hash is required"),
			validation.Length(1, maxRequestHashLength).
				Error("Request hash cannot be longer than 64 characters"),
		),
		// The workflow coordinates are how a completion is routed back to the
		// run that is waiting on it, so a record missing one strands the run.
		validation.Field(&e.WorkflowID,
			validation.Required.Error("Workflow is required"),
			validation.Length(1, maxWorkflowFieldLength).
				Error("Workflow cannot be longer than 255 characters"),
		),
		validation.Field(&e.WorkflowRunID,
			validation.Required.Error("Workflow run is required"),
			validation.Length(1, maxWorkflowFieldLength).
				Error("Workflow run cannot be longer than 255 characters"),
		),
		validation.Field(&e.ActivityID,
			validation.Required.Error("Activity is required"),
			validation.Length(1, maxWorkflowFieldLength).
				Error("Activity cannot be longer than 255 characters"),
		),
		validation.Field(&e.TaskToken, validation.Required.Error("Task token is required")),
		validation.Field(&e.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				StatusPending,
				StatusCompleted,
				StatusFailed,
				StatusApplied,
				StatusSkipped,
			).Error("Status is invalid"),
		),
		validation.Field(&e.FailureMessage,
			validation.When(e.Status == StatusFailed, validation.Required.Error(
				"A failed extraction must record why it failed",
			)),
		),
		validation.Field(&e.CompletedAt,
			validation.When(e.Status == StatusCompleted, validation.Required.Error(
				"A completed extraction must record when it completed",
			)),
		),
	))
}

const (
	maxRequestHashLength   = 64
	maxWorkflowFieldLength = 255
)
