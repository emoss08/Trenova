package testutil

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

func NewSessionActor(userID, orgID, buID pulid.ID) *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
}

func NewAPIKeyActor(apiKeyID, orgID, buID pulid.ID) *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeAPIKey,
		PrincipalID:    apiKeyID,
		APIKeyID:       apiKeyID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
}
