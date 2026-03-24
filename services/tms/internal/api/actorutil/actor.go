package actorutil

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
)

func FromAuthContext(auth *authctx.AuthContext) *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalType(auth.PrincipalType),
		PrincipalID:    auth.PrincipalID,
		UserID:         auth.UserID,
		APIKeyID:       auth.APIKeyID,
		BusinessUnitID: auth.BusinessUnitID,
		OrganizationID: auth.OrganizationID,
	}
}
