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
	_ bun.BeforeAppendModelHook = (*Connection)(nil)
	_ domain.Validatable        = (*Connection)(nil)
	_ framework.TenantedEntity  = (*Connection)(nil)
)

type Connection struct {
	bun.BaseModel `bun:"table:workflow_connections,alias:wfc" json:"-"`

	ID                pulid.ID       `json:"id"                bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID    pulid.ID       `json:"businessUnitId"    bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID    pulid.ID       `json:"organizationId"    bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	WorkflowVersionID pulid.ID       `json:"workflowVersionId" bun:"workflow_version_id,type:VARCHAR(100),notnull"`
	SourceNodeID      pulid.ID       `json:"sourceNodeId"      bun:"source_node_id,type:VARCHAR(100),notnull"`
	TargetNodeID      pulid.ID       `json:"targetNodeId"      bun:"target_node_id,type:VARCHAR(100),notnull"`
	Condition         map[string]any `json:"condition"         bun:"condition,type:JSONB,nullzero"`
	IsDefaultBranch   bool           `json:"isDefaultBranch"   bun:"is_default_branch,type:BOOLEAN,notnull,default:false"`
	Version           int64          `json:"version"           bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt         int64          `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64          `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relations
	BusinessUnit    *tenant.BusinessUnit `json:"businessUnit,omitempty"    bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization    *tenant.Organization `json:"organization,omitempty"    bun:"rel:belongs-to,join:organization_id=id"`
	WorkflowVersion *Version             `json:"workflowVersion,omitempty" bun:"rel:belongs-to,join:workflow_version_id=id"`
	SourceNode      *Node                `json:"sourceNode,omitempty"      bun:"rel:belongs-to,join:source_node_id=id"`
	TargetNode      *Node                `json:"targetNode,omitempty"      bun:"rel:belongs-to,join:target_node_id=id"`
}

func (c *Connection) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(c,
		validation.Field(&c.WorkflowVersionID,
			validation.Required.Error("Workflow Version ID is required"),
		),
		validation.Field(&c.SourceNodeID,
			validation.Required.Error("Source Node ID is required"),
		),
		validation.Field(&c.TargetNodeID,
			validation.Required.Error("Target Node ID is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if c.SourceNodeID == c.TargetNodeID {
		multiErr.Add(
			"sourceNodeId",
			errortypes.ErrInvalid,
			"Source node and target node cannot be the same",
		)
	}
}

func (c *Connection) GetID() string {
	return c.ID.String()
}

func (c *Connection) GetOrganizationID() pulid.ID {
	return c.OrganizationID
}

func (c *Connection) GetBusinessUnitID() pulid.ID {
	return c.BusinessUnitID
}

func (c *Connection) GetTableName() string {
	return "workflow_connections"
}

func (c *Connection) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("wfcn_")
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}

func (c *Connection) HasCondition() bool {
	return len(c.Condition) > 0
}
