package pagination

import "github.com/emoss08/trenova/pkg/authctx"

// FromAuth is the organization and business unit a request is scoped to: what a
// repository needs to know about which rows belong to the caller. The acting
// user is left unset; a handler that reads it should say so with FromAuthAsUser
// rather than have it filled in by default.
func FromAuth(authCtx *authctx.AuthContext) TenantInfo {
	return TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}
}

// FromAuthAsUser is FromAuth plus the person the request came from, for the
// reads whose result records who asked.
func FromAuthAsUser(authCtx *authctx.AuthContext) TenantInfo {
	return TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
