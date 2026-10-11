package agenttoolstest

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

func AgentActorFor(orgID, buID pulid.ID) *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeAgent,
		PrincipalID:    permissionports.AgentPrincipalID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
}
