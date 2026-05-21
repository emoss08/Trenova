package platformcatalog

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
)

var currentUserShellRoutes = mergeRouteRefs(
	routeRefsFor("GET",
		"/api/v1/users/me",
		"/api/v1/users/me/",
		"/api/v1/users/me/organizations/",
	),
	routeRefsFor("POST",
		"/api/v1/users/me/switch-organization/",
		"/api/v1/users/me/profile-picture/",
		"/api/v1/users/me/change-password/",
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
	"/api/v1/realtime/token-request/",
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

func accountShellRouteRefs() []RouteRef {
	return accountShellRoutes
}
