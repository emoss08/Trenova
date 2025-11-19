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
	_ bun.BeforeAppendModelHook = (*Node)(nil)
	_ domain.Validatable        = (*Node)(nil)
	_ framework.TenantedEntity  = (*Node)(nil)
)

type Node struct {
	bun.BaseModel `bun:"table:workflow_nodes,alias:wfn" json:"-"`

	ID                pulid.ID       `json:"id"                bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID    pulid.ID       `json:"businessUnitId"    bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID    pulid.ID       `json:"organizationId"    bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	WorkflowVersionID pulid.ID       `json:"workflowVersionId" bun:"workflow_version_id,type:VARCHAR(100),notnull"`
	Name              string         `json:"name"              bun:"name,type:VARCHAR(255),notnull"`
	NodeType          NodeType       `json:"nodeType"          bun:"node_type,type:workflow_node_type_enum,notnull"`
	PositionX         int            `json:"positionX"         bun:"position_x,type:INTEGER,notnull,default:0"`
	PositionY         int            `json:"positionY"         bun:"position_y,type:INTEGER,notnull,default:0"`
	Config            map[string]any `json:"config"            bun:"config,type:JSONB,notnull,default:'{}'"`
	Version           int64          `json:"version"           bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt         int64          `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64          `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relations
	BusinessUnit    *tenant.BusinessUnit `json:"businessUnit,omitempty"    bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization    *tenant.Organization `json:"organization,omitempty"    bun:"rel:belongs-to,join:organization_id=id"`
	WorkflowVersion *Version             `json:"workflowVersion,omitempty" bun:"rel:belongs-to,join:workflow_version_id=id"`
}

func (n *Node) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(n,
		validation.Field(&n.WorkflowVersionID,
			validation.Required.Error("Workflow Version ID is required"),
		),
		validation.Field(&n.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&n.NodeType,
			validation.Required.Error("Node Type is required"),
			validation.In(NodeTypeTrigger, NodeTypeEntityUpdate, NodeTypeCondition).
				Error("Node Type must be a valid node type"),
		),
		validation.Field(&n.Config,
			validation.Required.Error("Config is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (n *Node) GetID() string {
	return n.ID.String()
}

func (n *Node) GetOrganizationID() pulid.ID {
	return n.OrganizationID
}

func (n *Node) GetBusinessUnitID() pulid.ID {
	return n.BusinessUnitID
}

func (n *Node) GetTableName() string {
	return "workflow_nodes"
}

func (n *Node) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if n.ID.IsNil() {
			n.ID = pulid.MustNew("wfnd_")
		}
		n.CreatedAt = now
		n.UpdatedAt = now
	case *bun.UpdateQuery:
		n.UpdatedAt = now
	}

	return nil
}

func (n *Node) IsTriggerNode() bool {
	return n.NodeType == NodeTypeTrigger
}

func (n *Node) IsConditionNode() bool {
	return n.NodeType == NodeTypeCondition
}

func (n *Node) IsEntityUpdateNode() bool {
	return n.NodeType == NodeTypeEntityUpdate
}
