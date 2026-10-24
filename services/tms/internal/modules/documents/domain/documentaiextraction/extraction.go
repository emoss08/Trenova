package documentaiextraction

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	keel.Table `keel:"document_ai_extractions"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:dax_"`
	DocumentID     pulid.ID `json:"documentId" keel:"document_id"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	UserID         pulid.ID `json:"userId" keel:"user_id"`
	ExtractedAt    int64    `json:"extractedAt" keel:"extracted_at"`
	RequestHash    string   `json:"requestHash" keel:"request_hash"`
	WorkflowID     string   `json:"workflowId" keel:"workflow_id"`
	WorkflowRunID  string   `json:"workflowRunId" keel:"workflow_run_id"`
	ActivityID     string   `json:"activityId" keel:"activity_id"`
	ResponseID     string   `json:"responseId" keel:"response_id,nullzero"`
	ProviderID     pulid.ID `json:"providerId" keel:"provider_id,nullzero"`
	Model          string   `json:"model" keel:"model,nullzero"`
	Status         Status   `json:"status" keel:"status"`
	FailureCode    string   `json:"failureCode" keel:"failure_code,nullzero"`
	FailureMessage string   `json:"failureMessage" keel:"failure_message,nullzero"`
	SubmittedAt    *int64   `json:"submittedAt" keel:"submitted_at"`
	LastPolledAt   *int64   `json:"lastPolledAt" keel:"last_polled_at"`
	CompletedAt    *int64   `json:"completedAt" keel:"completed_at"`
	Version        int64    `json:"version" keel:"version,version"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (e *Extraction) BeforeCreate(ctx context.Context) error {
	if e.Status == "" {
		e.Status = StatusPending
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
