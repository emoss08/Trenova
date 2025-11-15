package workflow

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*WorkflowExecutionStep)(nil)
	_ domain.Validatable        = (*WorkflowExecutionStep)(nil)
	_ framework.TenantedEntity  = (*WorkflowExecutionStep)(nil)
)

// WorkflowExecutionStep represents a single step in a workflow execution
type WorkflowExecutionStep struct {
	bun.BaseModel `bun:"table:workflow_execution_steps,alias:wfxs" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	ExecutionID    pulid.ID `json:"executionId"    bun:"execution_id,notnull,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,notnull,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,notnull,pk,type:VARCHAR(100)"`

	// Node Reference
	NodeID     pulid.ID    `json:"nodeId"     bun:"node_id,notnull,type:VARCHAR(100)"`
	NodeKey    string      `json:"nodeKey"    bun:"node_key,notnull,type:VARCHAR(100)"`
	NodeType   NodeType    `json:"nodeType"   bun:"node_type,type:workflow_node_type_enum,notnull"`
	ActionType *ActionType `json:"actionType" bun:"action_type,type:workflow_action_type_enum,nullzero"`

	// Step Info
	StepNumber int        `json:"stepNumber" bun:"step_number,notnull"`
	Status     StepStatus `json:"status"     bun:"status,type:workflow_execution_step_status_enum,default:'pending'"`

	// Execution Data
	InputData    *utils.JSONB `json:"inputData"    bun:"input_data,type:jsonb,nullzero"`
	OutputData   *utils.JSONB `json:"outputData"   bun:"output_data,type:jsonb,nullzero"`
	ErrorMessage *string      `json:"errorMessage" bun:"error_message,type:TEXT,nullzero"`
	ErrorStack   *string      `json:"errorStack"   bun:"error_stack,type:TEXT,nullzero"`

	// Timing
	StartedAt   *int64 `json:"startedAt"   bun:"started_at,type:BIGINT,nullzero"`
	CompletedAt *int64 `json:"completedAt" bun:"completed_at,type:BIGINT,nullzero"`
	DurationMs  *int64 `json:"durationMs"  bun:"duration_ms,type:BIGINT,nullzero"`

	// Retry Info
	RetryCount int `json:"retryCount" bun:"retry_count,default:0"`

	// Metadata
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
	Execution    *WorkflowExecution   `bun:"rel:belongs-to,join:execution_id=id" json:"-"`
}

func (wes *WorkflowExecutionStep) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(wes,
		validation.Field(&wes.ExecutionID,
			validation.Required.Error("Execution ID is required"),
		),
		validation.Field(&wes.NodeID,
			validation.Required.Error("Node ID is required"),
		),
		validation.Field(&wes.StepNumber,
			validation.Required.Error("Step number is required"),
			validation.Min(1).Error("Step number must be at least 1"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (wes *WorkflowExecutionStep) GetID() string {
	return wes.ID.String()
}

func (wes *WorkflowExecutionStep) GetTableName() string {
	return "workflow_execution_steps"
}

func (wes *WorkflowExecutionStep) GetOrganizationID() pulid.ID {
	return wes.OrganizationID
}

func (wes *WorkflowExecutionStep) GetBusinessUnitID() pulid.ID {
	return wes.BusinessUnitID
}

func (wes *WorkflowExecutionStep) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		if wes.ID.IsNil() {
			wes.ID = pulid.MustNew("wfxs_")
		}
	}
	return nil
}
