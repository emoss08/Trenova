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
	_ bun.BeforeAppendModelHook = (*WorkflowEdge)(nil)
	_ domain.Validatable        = (*WorkflowEdge)(nil)
	_ framework.TenantedEntity  = (*WorkflowEdge)(nil)
)

// WorkflowEdge represents a connection between two nodes in a workflow
type WorkflowEdge struct {
	bun.BaseModel `bun:"table:workflow_edges,alias:wfe" json:"-"`

	ID                pulid.ID `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	WorkflowVersionID pulid.ID `json:"workflowVersionId" bun:"workflow_version_id,notnull,type:VARCHAR(100)"`
	BusinessUnitID    pulid.ID `json:"businessUnitId"    bun:"business_unit_id,notnull,pk,type:VARCHAR(100)"`
	OrganizationID    pulid.ID `json:"organizationId"    bun:"organization_id,notnull,pk,type:VARCHAR(100)"`

	// Edge Info
	SourceNodeID pulid.ID `json:"sourceNodeId" bun:"source_node_id,notnull,type:VARCHAR(100)"`
	TargetNodeID pulid.ID `json:"targetNodeId" bun:"target_node_id,notnull,type:VARCHAR(100)"`
	SourceHandle *string  `json:"sourceHandle" bun:"source_handle,type:VARCHAR(100),nullzero"` // For multiple outputs
	TargetHandle *string  `json:"targetHandle" bun:"target_handle,type:VARCHAR(100),nullzero"`

	// Condition (for conditional edges)
	Condition *utils.JSONB `json:"condition" bun:"condition,type:jsonb,nullzero"`

	// Display
	Label string `json:"label" bun:"label,type:VARCHAR(255)"`

	// Metadata
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit    *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization    *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
	WorkflowVersion *WorkflowVersion     `bun:"rel:belongs-to,join:workflow_version_id=id" json:"-"`
	SourceNode      *WorkflowNode        `bun:"rel:belongs-to,join:source_node_id=id" json:"sourceNode,omitempty"`
	TargetNode      *WorkflowNode        `bun:"rel:belongs-to,join:target_node_id=id" json:"targetNode,omitempty"`
}

func (we *WorkflowEdge) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(we,
		validation.Field(&we.WorkflowVersionID,
			validation.Required.Error("Workflow version ID is required"),
		),
		validation.Field(&we.SourceNodeID,
			validation.Required.Error("Source node ID is required"),
		),
		validation.Field(&we.TargetNodeID,
			validation.Required.Error("Target node ID is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (we *WorkflowEdge) GetID() string {
	return we.ID.String()
}

func (we *WorkflowEdge) GetTableName() string {
	return "workflow_edges"
}

func (we *WorkflowEdge) GetOrganizationID() pulid.ID {
	return we.OrganizationID
}

func (we *WorkflowEdge) GetBusinessUnitID() pulid.ID {
	return we.BusinessUnitID
}

func (we *WorkflowEdge) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		if we.ID.IsNil() {
			we.ID = pulid.MustNew("wfe_")
		}
	}
	return nil
}
