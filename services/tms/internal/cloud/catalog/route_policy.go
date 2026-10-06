package catalog

import "github.com/emoss08/trenova/internal/core/domain/platformcatalog"

var accountShellRoutes = mergeRouteRefs(
	currentUserShellRoutes,
	permissionShellRoutes,
	billingShellRoutes,
	organizationShellRoutes,
	notificationShellRoutes,
	pageFavoriteShellRoutes,
	realtimeShellRoutes,
	platformCatalogShellRoutes,
	usStateShellRoutes,
	graphQLTransportShellRoutes,
	pushSubscriptionShellRoutes,
	onboardingShellRoutes,
	supportAccessShellRoutes,
)

var supportAccessShellRoutes = mergeRouteRefs(
	routeRefsFor("GET",
		"/api/v1/support-access/",
		"/api/v1/support/profile/",
		"/api/v1/support/organizations/",
		"/api/v1/support/sessions/current/",
	),
	routeRefsFor("POST",
		"/api/v1/support-access/grant/",
		"/api/v1/support-access/grant/revoke/",
		"/api/v1/support/sessions/",
		"/api/v1/support/sessions/current/elevate/",
		"/api/v1/support/sessions/current/drop-elevation/",
		"/api/v1/support/sessions/current/end/",
	),
)

var onboardingShellRoutes = mergeRouteRefs(
	routeRefsFor("GET",
		"/api/v1/onboarding/",
	),
	routeRefsFor("POST",
		"/api/v1/onboarding/complete/",
	),
)

var pushSubscriptionShellRoutes = mergeRouteRefs(
	routeRefsFor("GET",
		"/api/v1/push/public-key/",
	),
	routeRefsFor("POST",
		"/api/v1/push/subscriptions/",
	),
	routeRefsFor("DELETE",
		"/api/v1/push/subscriptions/",
	),
)

var graphQLTransportShellRoutes = mergeRouteRefs(
	routeRefsFor("GET",
		"/graphql",
	),
	routeRefsFor("POST",
		"/graphql",
	),
)

var currentUserShellRoutes = mergeRouteRefs(
	routeRefsFor("GET",
		"/api/v1/users/me",
		"/api/v1/users/me/",
		"/api/v1/users/me/organizations/",
		"/api/v1/users/me/mfa/",
	),
	routeRefsFor("POST",
		"/api/v1/users/me/switch-organization/",
		"/api/v1/users/me/profile-picture/",
		"/api/v1/users/me/change-password/",
		"/api/v1/users/me/mfa/totp/enroll/",
		"/api/v1/users/me/mfa/totp/confirm/",
		"/api/v1/users/me/mfa/totp/disable/",
		"/api/v1/users/me/mfa/recovery-codes/",
	),
	routeRefsFor("PATCH",
		"/api/v1/users/me/settings/",
	),
	routeRefsFor("DELETE",
		"/api/v1/users/me/profile-picture/",
	),
)

var permissionShellRoutes = mergeRouteRefs(
	routeRefsFor("GET",
		"/api/v1/me/permissions",
		"/api/v1/me/permissions/",
		"/api/v1/me/permissions/version",
		"/api/v1/me/permissions/:resource",
	),
	routeRefsFor("POST",
		"/api/v1/me/permissions/check",
	),
)

var billingShellRoutes = routeRefsFor("GET",
	"/api/v1/me/billing",
	"/api/v1/me/billing/",
)

var organizationShellRoutes = mergeRouteRefs(
	routeRefsFor("GET",
		"/api/v1/organizations/:id",
		"/api/v1/organizations/:id/logo",
		"/api/v1/organizations/:id/microsoft-sso",
		"/api/v1/organizations/:id/okta-sso",
	),
	routeRefsFor("POST",
		"/api/v1/organizations/:id/logo",
	),
	routeRefsFor("PUT",
		"/api/v1/organizations/:id",
		"/api/v1/organizations/:id/microsoft-sso",
		"/api/v1/organizations/:id/okta-sso",
	),
	routeRefsFor("DELETE",
		"/api/v1/organizations/:id/logo",
	),
)

var usStateShellRoutes = mergeRouteRefs(
	routeRefsFor(
		"GET",
		"/api/v1/us-states/select-options/",
		"/api/v1/us-states/select-options/:usStateID",
	),
)

var notificationShellRoutes = mergeRouteRefs(
	routeRefsFor("GET",
		"/api/v1/notifications/",
		"/api/v1/notifications/unread-count",
	),
	routeRefsFor("PATCH",
		"/api/v1/notifications/mark-read",
		"/api/v1/notifications/mark-all-read",
	),
)

var pageFavoriteShellRoutes = mergeRouteRefs(
	routeRefsFor("GET",
		"/api/v1/page-favorites/",
		"/api/v1/page-favorites/check",
	),
	routeRefsFor("POST",
		"/api/v1/page-favorites/toggle",
	),
)

var realtimeShellRoutes = routeRefsFor("GET",
	"/api/v1/realtime/stream/",
)

var platformCatalogShellRoutes = routeRefsFor("GET",
	"/api/v1/me/platform-catalog",
	"/api/v1/me/platform-catalog/",
	"/api/v1/me/entitlements",
	"/api/v1/me/entitlements/",
	"/api/v1/platform-catalog/products",
	"/api/v1/platform-catalog/features",
	"/api/v1/platform-catalog/meters",
	"/api/v1/platform-catalog/validate",
)

func accountShellRouteRefs() []platformcatalog.RouteRef {
	return accountShellRoutes
}
