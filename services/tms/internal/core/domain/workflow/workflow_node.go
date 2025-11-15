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
	_ bun.BeforeAppendModelHook = (*WorkflowNode)(nil)
	_ domain.Validatable        = (*WorkflowNode)(nil)
	_ framework.TenantedEntity  = (*WorkflowNode)(nil)
)

// WorkflowNode represents a node in a workflow
type WorkflowNode struct {
	bun.BaseModel `bun:"table:workflow_nodes,alias:wfn" json:"-"`

	ID                pulid.ID `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	WorkflowVersionID pulid.ID `json:"workflowVersionId" bun:"workflow_version_id,notnull,type:VARCHAR(100)"`
	BusinessUnitID    pulid.ID `json:"businessUnitId"    bun:"business_unit_id,notnull,pk,type:VARCHAR(100)"`
	OrganizationID    pulid.ID `json:"organizationId"    bun:"organization_id,notnull,pk,type:VARCHAR(100)"`

	// Node Info
	NodeKey    string      `json:"nodeKey"    bun:"node_key,notnull,type:VARCHAR(100)"`
	NodeType   NodeType    `json:"nodeType"   bun:"node_type,type:workflow_node_type_enum,notnull"`
	ActionType *ActionType `json:"actionType" bun:"action_type,type:workflow_action_type_enum,nullzero"`

	// Display
	Label       string `json:"label"       bun:"label,notnull,type:VARCHAR(255)"`
	Description string `json:"description" bun:"description,type:TEXT"`

	// Configuration
	Config map[string]any `json:"config" bun:"config,type:jsonb,default:'{}'"`

	// Position (for UI canvas)
	PositionX float64 `json:"positionX" bun:"position_x,default:0"`
	PositionY float64 `json:"positionY" bun:"position_y,default:0"`

	// Metadata
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit    *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id"    json:"-"`
	Organization    *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"     json:"-"`
	WorkflowVersion *WorkflowVersion     `bun:"rel:belongs-to,join:workflow_version_id=id" json:"-"`
}

func (wn *WorkflowNode) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(wn,
		validation.Field(&wn.WorkflowVersionID,
			validation.Required.Error("Workflow version ID is required"),
		),
		validation.Field(&wn.NodeKey,
			validation.Required.Error("Node key is required"),
			validation.Length(1, 100).Error("Node key must be between 1 and 100 characters"),
		),
		validation.Field(&wn.NodeType,
			validation.Required.Error("Node type is required"),
			validation.In(
				NodeTypeTrigger,
				NodeTypeAction,
				NodeTypeCondition,
				NodeTypeLoop,
				NodeTypeDelay,
				NodeTypeEnd,
			).Error("Invalid node type"),
		),
		validation.Field(&wn.Label,
			validation.Required.Error("Label is required"),
			validation.Length(1, 255).Error("Label must be between 1 and 255 characters"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (wn *WorkflowNode) GetID() string {
	return wn.ID.String()
}

func (wn *WorkflowNode) GetTableName() string {
	return "workflow_nodes"
}

func (wn *WorkflowNode) GetOrganizationID() pulid.ID {
	return wn.OrganizationID
}

func (wn *WorkflowNode) GetBusinessUnitID() pulid.ID {
	return wn.BusinessUnitID
}

func (wn *WorkflowNode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		if wn.ID.IsNil() {
			wn.ID = pulid.MustNew("wfn_")
		}
	}
	return nil
}
