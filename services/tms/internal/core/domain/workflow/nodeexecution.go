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
	_ bun.BeforeAppendModelHook = (*NodeExecution)(nil)
	_ domain.Validatable        = (*NodeExecution)(nil)
	_ framework.TenantedEntity  = (*NodeExecution)(nil)
)

type NodeExecution struct {
	bun.BaseModel `bun:"table:workflow_node_executions,alias:wfne" json:"-"`

	ID                 pulid.ID            `json:"id"                 bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID     pulid.ID            `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID     pulid.ID            `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	WorkflowInstanceID pulid.ID            `json:"workflowInstanceId" bun:"workflow_instance_id,type:VARCHAR(100),notnull"`
	WorkflowNodeID     pulid.ID            `json:"workflowNodeId"     bun:"workflow_node_id,type:VARCHAR(100),notnull"`
	Status             NodeExecutionStatus `json:"status"             bun:"status,type:workflow_node_execution_status_enum,notnull,default:'Pending'"`
	AttemptCount       int16               `json:"attemptCount"       bun:"attempt_count,type:SMALLINT,notnull,default:0"`
	InputData          map[string]any      `json:"inputData"          bun:"input_data,type:JSONB,default:'{}'"`
	OutputData         map[string]any      `json:"outputData"         bun:"output_data,type:JSONB,default:'{}'"`
	ErrorDetails       map[string]any      `json:"errorDetails"       bun:"error_details,type:JSONB,nullzero"`
	StartedAt          *int64              `json:"startedAt"          bun:"started_at,type:BIGINT,nullzero"`
	CompletedAt        *int64              `json:"completedAt"        bun:"completed_at,type:BIGINT,nullzero"`
	Version            int64               `json:"version"            bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt          int64               `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64               `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relations
	BusinessUnit     *tenant.BusinessUnit `json:"businessUnit,omitempty"     bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization     *tenant.Organization `json:"organization,omitempty"     bun:"rel:belongs-to,join:organization_id=id"`
	WorkflowInstance *Instance            `json:"workflowInstance,omitempty" bun:"rel:belongs-to,join:workflow_instance_id=id"`
	WorkflowNode     *Node                `json:"workflowNode,omitempty"     bun:"rel:belongs-to,join:workflow_node_id=id"`
}

func (ne *NodeExecution) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(ne,
		validation.Field(&ne.WorkflowInstanceID,
			validation.Required.Error("Workflow Instance ID is required"),
		),
		validation.Field(&ne.WorkflowNodeID,
			validation.Required.Error("Workflow Node ID is required"),
		),
		validation.Field(&ne.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				NodeExecutionStatusPending,
				NodeExecutionStatusRunning,
				NodeExecutionStatusCompleted,
				NodeExecutionStatusFailed,
				NodeExecutionStatusSkipped,
			).Error("Status must be a valid node execution status"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (ne *NodeExecution) GetID() string {
	return ne.ID.String()
}

func (ne *NodeExecution) GetOrganizationID() pulid.ID {
	return ne.OrganizationID
}

func (ne *NodeExecution) GetBusinessUnitID() pulid.ID {
	return ne.BusinessUnitID
}

func (ne *NodeExecution) GetTableName() string {
	return "workflow_node_executions"
}

func (ne *NodeExecution) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if ne.ID.IsNil() {
			ne.ID = pulid.MustNew("wfnex_")
		}
		ne.CreatedAt = now
		ne.UpdatedAt = now
	case *bun.UpdateQuery:
		ne.UpdatedAt = now
	}

	return nil
}

func (ne *NodeExecution) IsPending() bool {
	return ne.Status == NodeExecutionStatusPending
}

func (ne *NodeExecution) IsRunning() bool {
	return ne.Status == NodeExecutionStatusRunning
}

func (ne *NodeExecution) IsCompleted() bool {
	return ne.Status == NodeExecutionStatusCompleted
}

func (ne *NodeExecution) IsFailed() bool {
	return ne.Status == NodeExecutionStatusFailed
}

func (ne *NodeExecution) IsSkipped() bool {
	return ne.Status == NodeExecutionStatusSkipped
}

func (ne *NodeExecution) IsTerminal() bool {
	return ne.Status.IsTerminal()
}

func (ne *NodeExecution) MarkRunning() {
	ne.Status = NodeExecutionStatusRunning
	now := utils.NowUnix()
	ne.StartedAt = &now
	ne.AttemptCount++
}

func (ne *NodeExecution) MarkCompleted(outputData map[string]any) {
	ne.Status = NodeExecutionStatusCompleted
	ne.OutputData = outputData
	now := utils.NowUnix()
	ne.CompletedAt = &now
}

func (ne *NodeExecution) MarkFailed(errorDetails map[string]any) {
	ne.Status = NodeExecutionStatusFailed
	ne.ErrorDetails = errorDetails
	now := utils.NowUnix()
	ne.CompletedAt = &now
}

func (ne *NodeExecution) MarkSkipped() {
	ne.Status = NodeExecutionStatusSkipped
	now := utils.NowUnix()
	ne.CompletedAt = &now
}

func (ne *NodeExecution) CanRetry(maxAttempts int16) bool {
	return ne.IsFailed() && ne.AttemptCount < maxAttempts
}
