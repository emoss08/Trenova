package workflow

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*WorkflowExecution)(nil)
	_ domain.Validatable        = (*WorkflowExecution)(nil)
	_ framework.TenantedEntity  = (*WorkflowExecution)(nil)
)

// WorkflowExecution represents an execution instance of a workflow
type WorkflowExecution struct {
	bun.BaseModel `bun:"table:workflow_executions,alias:wfx" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,notnull,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,notnull,pk,type:VARCHAR(100)"`

	// Workflow Reference
	WorkflowID        pulid.ID `json:"workflowId"        bun:"workflow_id,notnull,type:VARCHAR(100)"`
	WorkflowVersionID pulid.ID `json:"workflowVersionId" bun:"workflow_version_id,notnull,type:VARCHAR(100)"`

	// Execution Info
	Status      ExecutionStatus `json:"status"      bun:"status,type:workflow_execution_status_enum,default:'pending'"`
	TriggerType TriggerType     `json:"triggerType" bun:"trigger_type,type:workflow_trigger_type_enum,notnull"`

	// Trigger Context
	TriggerData map[string]any `json:"triggerData" bun:"trigger_data,type:jsonb,default:'{}'"`
	TriggeredBy *pulid.ID      `json:"triggeredBy" bun:"triggered_by,type:VARCHAR(100),nullzero"` // User ID if manual

	// Temporal Workflow Info
	TemporalWorkflowID *string `json:"temporalWorkflowId" bun:"temporal_workflow_id,type:VARCHAR(255),nullzero"`
	TemporalRunID      *string `json:"temporalRunId"      bun:"temporal_run_id,type:VARCHAR(255),nullzero"`

	// Execution Results
	InputData    map[string]any `json:"inputData"    bun:"input_data,type:jsonb,nullzero"`
	OutputData   map[string]any `json:"outputData"   bun:"output_data,type:jsonb,nullzero"`
	ErrorMessage *string        `json:"errorMessage" bun:"error_message,type:TEXT,nullzero"`
	ErrorStack   *string        `json:"errorStack"   bun:"error_stack,type:TEXT,nullzero"`

	// Timing
	StartedAt   *int64 `json:"startedAt"   bun:"started_at,type:BIGINT,nullzero"`
	CompletedAt *int64 `json:"completedAt" bun:"completed_at,type:BIGINT,nullzero"`
	DurationMs  *int64 `json:"durationMs"  bun:"duration_ms,type:BIGINT,nullzero"`

	// Retry Info
	RetryCount int `json:"retryCount" bun:"retry_count,default:0"`
	MaxRetries int `json:"maxRetries" bun:"max_retries,default:3"`

	// Metadata
	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit    *tenant.BusinessUnit     `bun:"rel:belongs-to,join:business_unit_id=id"    json:"-"`
	Organization    *tenant.Organization     `bun:"rel:belongs-to,join:organization_id=id"     json:"-"`
	Workflow        *Workflow                `bun:"rel:belongs-to,join:workflow_id=id"         json:"workflow,omitempty"`
	WorkflowVersion *WorkflowVersion         `bun:"rel:belongs-to,join:workflow_version_id=id" json:"workflowVersion,omitempty"`
	Steps           []*WorkflowExecutionStep `bun:"rel:has-many,join:id=execution_id"          json:"steps,omitempty"`
}

func (wx *WorkflowExecution) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(wx,
		validation.Field(&wx.WorkflowID,
			validation.Required.Error("Workflow ID is required"),
		),
		validation.Field(&wx.WorkflowVersionID,
			validation.Required.Error("Workflow version ID is required"),
		),
		validation.Field(&wx.TriggerType,
			validation.Required.Error("Trigger type is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (wx *WorkflowExecution) GetID() string {
	return wx.ID.String()
}

func (wx *WorkflowExecution) GetTableName() string {
	return "workflow_executions"
}

func (wx *WorkflowExecution) GetOrganizationID() pulid.ID {
	return wx.OrganizationID
}

func (wx *WorkflowExecution) GetBusinessUnitID() pulid.ID {
	return wx.BusinessUnitID
}

func (wx *WorkflowExecution) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		if wx.ID.IsNil() {
			wx.ID = pulid.MustNew("wfx_")
		}
	case *bun.UpdateQuery:
		wx.Version++
	}
	return nil
}

// IsComplete checks if the execution has finished (successfully or not)
func (wx *WorkflowExecution) IsComplete() bool {
	return wx.Status == ExecutionStatusCompleted ||
		wx.Status == ExecutionStatusFailed ||
		wx.Status == ExecutionStatusCanceled ||
		wx.Status == ExecutionStatusTimeout
}

// CanRetry checks if the execution can be retried
func (wx *WorkflowExecution) CanRetry() bool {
	return wx.Status == ExecutionStatusFailed && wx.RetryCount < wx.MaxRetries
}
