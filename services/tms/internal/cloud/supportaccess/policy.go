package supportaccess

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
)

type Decision int

const (
	DecisionAllow Decision = iota
	DecisionReadOnly
	DecisionDenied
)

const graphQLRoute = "/graphql"

var deniedResources = []permission.Resource{
	permission.ResourceUser,
	permission.ResourceRole,
	permission.ResourceAPIKey,
	permission.ResourceIdentityProvider,
	permission.ResourceSCIMDirectory,
	permission.ResourceAccessPolicy,
	permission.ResourceMFAAuthenticator,
	permission.ResourceExternalIdentity,
	permission.ResourceDatabaseSession,
	permission.ResourcePlatformCatalog,
}

var deniedRoutePrefixes = []string{
	"/api/v1/users/",
	"/api/v1/me/",
	"/api/v1/api-keys",
	"/api/v1/roles",
	"/api/v1/role-assignments",
	"/api/v1/organizations/:id/iam/",
	"/api/v1/organizations/:id/microsoft-sso",
	"/api/v1/organizations/:id/okta-sso",
	"/api/v1/support-access/",
	"/api/v1/support/",
	"/api/v1/push/",
	"/api/v1/assistant/",
	"/api/v1/admin/database-sessions/",
	"/api/v1/onboarding/",
	"/api/v1/auth/",
}

var deniedRouteSuffixes = []string{
	"/comments/typing",
	"/comments/presence",
}

var readOnlyAllowedRoutes = map[string]struct{}{
	"/api/v1/me/permissions/check": {},
}

var deniedGraphQLSources = []string{
	"api_key.graphqls",
	"role.graphqls",
	"scim_group_role_mapping.graphqls",
	"user.graphqls",
	"self_service.graphqls",
	"agent.graphqls",
	"agentdefinition.graphqls",
	"agentpreview.graphqls",
	"agentquality.graphqls",
	"agentrunevent.graphqls",
	"agentsafety.graphqls",
	"agentscorecard.graphqls",
	"aiprovider.graphqls",
	"aitraining.graphqls",
	"decisions.graphqls",
	"desk_memory.graphqls",
}

func DeniedResources() []permission.Resource {
	return slices.Clone(deniedResources)
}

func IsDeniedResource(resource string) bool {
	return slices.ContainsFunc(deniedResources, func(r permission.Resource) bool {
		return r.String() == resource
	})
}

type RouteRequest struct {
	Method      string
	Route       string
	WriteActive bool
}

func DecideRoute(req RouteRequest) Decision {
	route := strings.TrimSuffix(req.Route, "/")
	if !middleware.IsUnsafeMethod(req.Method) {
		return DecisionAllow
	}

	if _, ok := readOnlyAllowedRoutes[route]; ok {
		return DecisionAllow
	}

	if isDeniedRoute(route) {
		return DecisionDenied
	}

	if route == graphQLRoute {
		return DecisionAllow
	}

	if req.WriteActive {
		return DecisionAllow
	}

	if middleware.ReadOnlyWriteAllowed(route) {
		return DecisionAllow
	}

	return DecisionReadOnly
}

func isDeniedRoute(route string) bool {
	for _, prefix := range deniedRoutePrefixes {
		if route == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(route, prefix) {
			return true
		}
	}
	for _, suffix := range deniedRouteSuffixes {
		if strings.HasSuffix(route, suffix) {
			return true
		}
	}

	return false
}

type MutationRequest struct {
	FieldName   string
	Source      string
	WriteActive bool
}

func DecideMutation(req MutationRequest) Decision {
	if slices.Contains(deniedGraphQLSources, req.Source) ||
		strings.HasPrefix(req.FieldName, "my") ||
		strings.HasPrefix(req.FieldName, "My") {
		return DecisionDenied
	}

	if !req.WriteActive {
		return DecisionReadOnly
	}

	return DecisionAllow
}
