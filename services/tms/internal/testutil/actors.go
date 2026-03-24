package testutil

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

func NewSessionActor(userID, orgID, buID pulid.ID) *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
}

func NewAPIKeyActor(apiKeyID, orgID, buID pulid.ID) *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeAPIKey,
		PrincipalID:    apiKeyID,
		APIKeyID:       apiKeyID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
}
