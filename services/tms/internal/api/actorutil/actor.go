package actorutil

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
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

// TenantInfoFrom reads the tenant a request is scoped to.
//
// The organization and business unit come from the authenticated session and
// never from a payload: a handler that took either from the body would let a
// caller read or write across tenants by editing the request.
func TenantInfoFrom(auth *authctx.AuthContext) pagination.TenantInfo {
	if auth == nil {
		return pagination.TenantInfo{}
	}

	return pagination.TenantInfo{
		OrgID: auth.OrganizationID,
		BuID:  auth.BusinessUnitID,
	}
}
