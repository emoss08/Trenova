package permission

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// RoleAgentGrant lets everyone holding a role, directly or through a role
// that inherits it, use an agent whose access is restricted to roles.
type RoleAgentGrant struct {
	keel.Table `keel:"role_agent_grants"`

	ID                pulid.ID `json:"id"                keel:"id,pk,autoid:rag_"`
	OrganizationID    pulid.ID `json:"organizationId"    keel:"organization_id,tenant"`
	BusinessUnitID    pulid.ID `json:"businessUnitId"    keel:"business_unit_id,tenant"`
	RoleID            pulid.ID `json:"roleId"            keel:"role_id"`
	AgentDefinitionID pulid.ID `json:"agentDefinitionId" keel:"agent_definition_id"`
	GrantedBy         pulid.ID `json:"grantedBy"         keel:"granted_by,nullzero"`
	GrantedAt         int64    `json:"grantedAt"         keel:"granted_at"`
}

func (g *RoleAgentGrant) BeforeCreate(_ context.Context) error {
	if g.GrantedAt == 0 {
		g.GrantedAt = timeutils.NowUnix()
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
