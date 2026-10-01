package resolvertest

import (
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

func AllocationAuthCtx() *authctx.AuthContext {
	return &authctx.AuthContext{
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
	}
}
