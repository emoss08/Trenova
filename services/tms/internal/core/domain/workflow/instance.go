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
	_ bun.BeforeAppendModelHook = (*Instance)(nil)
	_ domain.Validatable        = (*Instance)(nil)
	_ framework.TenantedEntity  = (*Instance)(nil)
)

type Instance struct {
	bun.BaseModel `bun:"table:workflow_instances,alias:wfi" json:"-"`

	ID                 pulid.ID       `json:"id"                 bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID     pulid.ID       `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID     pulid.ID       `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	WorkflowTemplateID pulid.ID       `json:"workflowTemplateId" bun:"workflow_template_id,type:VARCHAR(100),notnull"`
	WorkflowVersionID  pulid.ID       `json:"workflowVersionId"  bun:"workflow_version_id,type:VARCHAR(100),notnull"`
	Status             InstanceStatus `json:"status"             bun:"status,type:workflow_instance_status_enum,notnull,default:'Running'"`
	ExecutionMode      ExecutionMode  `json:"executionMode"      bun:"execution_mode,type:workflow_execution_mode_enum,notnull,default:'Manual'"`
	TriggerPayload     map[string]any `json:"triggerPayload"     bun:"trigger_payload,type:JSONB,default:'{}'"`
	WorkflowVariables  map[string]any `json:"workflowVariables"  bun:"workflow_variables,type:JSONB,default:'{}'"`
	ExecutionContext   map[string]any `json:"executionContext"   bun:"execution_context,type:JSONB,default:'{}'"`
	ErrorMessage       string         `json:"errorMessage"       bun:"error_message,type:TEXT,nullzero"`
	StartedByID        *pulid.ID      `json:"startedById"        bun:"started_by_id,type:VARCHAR(100),nullzero"`
	StartedAt          int64          `json:"startedAt"          bun:"started_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	CompletedAt        *int64         `json:"completedAt"        bun:"completed_at,type:BIGINT,nullzero"`
	Version            int64          `json:"version"            bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt          int64          `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64          `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relations
	BusinessUnit     *tenant.BusinessUnit `json:"businessUnit,omitempty"     bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization     *tenant.Organization `json:"organization,omitempty"     bun:"rel:belongs-to,join:organization_id=id"`
	WorkflowTemplate *Template            `json:"workflowTemplate,omitempty" bun:"rel:belongs-to,join:workflow_template_id=id"`
	WorkflowVersion  *Version             `json:"workflowVersion,omitempty"  bun:"rel:belongs-to,join:workflow_version_id=id"`
	StartedBy        *tenant.User         `json:"startedBy,omitempty"        bun:"rel:belongs-to,join:started_by_id=id"`
	NodeExecutions   []*NodeExecution     `json:"nodeExecutions,omitempty"   bun:"rel:has-many,join:id=workflow_instance_id"`
}

func (i *Instance) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(i,
		validation.Field(&i.WorkflowTemplateID,
			validation.Required.Error("Workflow Template ID is required"),
		),
		validation.Field(&i.WorkflowVersionID,
			validation.Required.Error("Workflow Version ID is required"),
		),
		validation.Field(&i.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				InstanceStatusRunning,
				InstanceStatusCompleted,
				InstanceStatusFailed,
				InstanceStatusCancelled,
				InstanceStatusPaused,
			).Error("Status must be a valid instance status"),
		),
		validation.Field(&i.ExecutionMode,
			validation.Required.Error("Execution Mode is required"),
			validation.In(ExecutionModeManual, ExecutionModeScheduled, ExecutionModeEvent).
				Error("Execution Mode must be a valid execution mode"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (i *Instance) GetID() string {
	return i.ID.String()
}

func (i *Instance) GetOrganizationID() pulid.ID {
	return i.OrganizationID
}

func (i *Instance) GetBusinessUnitID() pulid.ID {
	return i.BusinessUnitID
}

func (i *Instance) GetTableName() string {
	return "workflow_instances"
}

func (i *Instance) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("wfin_")
		}
		i.CreatedAt = now
		i.UpdatedAt = now
		i.StartedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}

func (i *Instance) IsRunning() bool {
	return i.Status == InstanceStatusRunning
}

func (i *Instance) IsCompleted() bool {
	return i.Status == InstanceStatusCompleted
}

func (i *Instance) IsFailed() bool {
	return i.Status == InstanceStatusFailed
}

func (i *Instance) IsCancelled() bool {
	return i.Status == InstanceStatusCancelled
}

func (i *Instance) IsPaused() bool {
	return i.Status == InstanceStatusPaused
}

func (i *Instance) IsTerminal() bool {
	return i.Status.IsTerminal()
}

func (i *Instance) MarkCompleted() {
	i.Status = InstanceStatusCompleted
	now := utils.NowUnix()
	i.CompletedAt = &now
}

func (i *Instance) MarkFailed(errorMsg string) {
	i.Status = InstanceStatusFailed
	i.ErrorMessage = errorMsg
	now := utils.NowUnix()
	i.CompletedAt = &now
}

func (i *Instance) MarkCancelled() {
	i.Status = InstanceStatusCancelled
	now := utils.NowUnix()
	i.CompletedAt = &now
}

func (i *Instance) MarkPaused() {
	i.Status = InstanceStatusPaused
}

func (i *Instance) Resume() error {
	if !i.IsPaused() {
		return errors.New("cannot resume instance that is not paused")
	}
	i.Status = InstanceStatusRunning
	return nil
}
