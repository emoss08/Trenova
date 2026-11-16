package middleware

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const assistantRoutePrefix = "/api/v1/assistant/"

var supportPersonalWrites = map[string]struct{}{
	changePasswordPath:          {},
	switchOrganizationPath:      {},
	"/api/v1/users/me/settings": {},
}

var supportNeverWrites = map[string]struct{}{
	changePasswordPath:     {},
	switchOrganizationPath: {},
}

func SupportWriteRefusal(
	support *authctx.Support,
	method, route string,
) *errortypes.AuthorizationError {
	if support == nil || !IsUnsafeMethod(method) {
		return nil
	}
	route = strings.TrimSuffix(route, "/")
	if _, never := supportNeverWrites[route]; never {
		return errortypes.NewAuthorizationError(
			"A Trenova support session has no password and stays in the organization it was opened for",
		)
	}
	if strings.HasPrefix(route+"/", assistantRoutePrefix) && strings.Contains(route, "/schedules") {
		return errortypes.NewAuthorizationError(
			"Trenova support cannot schedule requests: a schedule would keep running after the access ends",
		)
	}
	if supportaccess.Scope(support.Scope) != supportaccess.ScopeReadOnly {
		return nil
	}
	if strings.HasPrefix(route+"/", assistantRoutePrefix) && !strings.Contains(route, "/case") {
		return nil
	}
	if _, personal := supportPersonalWrites[route]; !personal && ReadOnlyWriteAllowed(route) {
		return nil
	}
	return supportaccess.ErrReadOnlySession()
}
