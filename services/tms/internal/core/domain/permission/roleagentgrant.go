package permission

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RoleAgentGrant)(nil)

// RoleAgentGrant lets everyone holding a role, directly or through a role
// that inherits it, use an agent whose access is restricted to roles.
type RoleAgentGrant struct {
	bun.BaseModel `bun:"table:role_agent_grants,alias:rag" json:"-"`

	ID                pulid.ID `json:"id"                bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID    pulid.ID `json:"organizationId"    bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID `json:"businessUnitId"    bun:"business_unit_id,type:VARCHAR(100),notnull"`
	RoleID            pulid.ID `json:"roleId"            bun:"role_id,type:VARCHAR(100),notnull"`
	AgentDefinitionID pulid.ID `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),notnull"`
	GrantedBy         pulid.ID `json:"grantedBy"         bun:"granted_by,type:VARCHAR(100),nullzero"`
	GrantedAt         int64    `json:"grantedAt"         bun:"granted_at,notnull"`
}

func (g *RoleAgentGrant) BeforeAppendModel(_ context.Context, q bun.Query) error {
	if _, ok := q.(*bun.InsertQuery); ok {
		if g.ID.IsNil() {
			g.ID = pulid.MustNew("rag_")
		}
		if g.GrantedAt == 0 {
			g.GrantedAt = timeutils.NowUnix()
		}
	}

	return nil
}

func (g *RoleAgentGrant) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(g,
		validation.Field(&g.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&g.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&g.RoleID, validation.Required.Error("Role is required")),
		validation.Field(&g.AgentDefinitionID, validation.Required.Error("Agent is required")),
	))
}
