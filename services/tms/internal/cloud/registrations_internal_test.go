package cloud

import (
	"net/http"
	"testing"

	gqlgen "github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/routegroup"
	"github.com/emoss08/trenova/internal/api/routelint"
	"github.com/emoss08/trenova/internal/cloud/catalog"
	"github.com/emoss08/trenova/internal/cloud/catalog/platformcataloghandler"
	"github.com/emoss08/trenova/internal/cloud/controlplane/accessmiddleware"
	"github.com/emoss08/trenova/internal/cloud/controlplane/controlplaneprovisioninghandler"
	"github.com/emoss08/trenova/internal/cloud/controlplane/featureaccess"
	"github.com/emoss08/trenova/internal/cloud/networkpulse/networkpulsehandler"
	"github.com/emoss08/trenova/internal/cloud/signup/cloudsignuphandler"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

type registered struct {
	fx.In

	Routes       routegroup.Registrars
	Contributors []services.PublicConfigContributor `group:"public_config_contributors"`
	Extensions   []gqlgen.HandlerExtension          `group:"edition_graphql_extensions"`
}

func populateRegistrations(t *testing.T) registered {
	t.Helper()

	var got registered
	app := fx.New(
		fx.NopLogger,
		fx.Supply(
			&cloudsignuphandler.Handler{},
			&networkpulsehandler.Handler{},
			&controlplaneprovisioninghandler.Handler{},
			&platformcataloghandler.Handler{},
			&accessmiddleware.ControlPlaneAccessMiddleware{},
			&featureaccess.FeatureAccessExtension{},
		),
		apiRegistrations(),
		fx.Populate(&got),
	)
	require.NoError(t, app.Err())

	return got
}

func TestAPIRegistrationsFillTheEditionGroups(t *testing.T) {
	t.Parallel()

	got := populateRegistrations(t)
	assert.Len(t, got.Routes.Public, 3)
	assert.Len(t, got.Routes.Protected, 1)
	assert.Len(t, got.Routes.Middleware, 1)
	assert.Len(t, got.Contributors, 1)
	assert.Len(t, got.Extensions, 1)
}

type writeExemption struct {
	category string
	reason   string
}

var cloudWriteExemptions = map[string]writeExemption{
	"POST /api/v1/cloud/signups": {
		category: "security",
		reason: "Signing up creates an account and proves a person owns an email address; an agent " +
			"holds no identity of its own and acts only inside an organization that already exists.",
	},
	"POST /api/v1/cloud/signups/resend": {
		category: "security",
		reason: "Resending a signup verification link is part of a stranger proving they own an " +
			"email address before any organization exists.",
	},
	"POST /api/v1/cloud/signups/verify": {
		category: "security",
		reason: "Redeeming a signup link signs a person in and provisions their organization; it is " +
			"identity proof, not an operation an agent performs.",
	},
	"POST /api/v1/control-plane/tenants/provision": {
		category: "infrastructure",
		reason:   "Called by the platform control plane with a service credential to provision a tenant.",
	},
}

func TestEveryCloudWriteRouteIsAccountedFor(t *testing.T) {
	gin.SetMode(gin.TestMode)

	public, err := api.RouteTable()
	require.NoError(t, err)
	withCloud, err := api.RouteTableWith(populateRegistrations(t).Routes)
	require.NoError(t, err)

	publicRoutes := make(map[string]struct{}, len(public))
	for _, info := range public {
		publicRoutes[info.Method+" "+info.Path] = struct{}{}
	}

	seen := make(map[string]struct{})
	for _, info := range withCloud {
		key := info.Method + " " + info.Path
		if _, ok := publicRoutes[key]; ok || info.Method == http.MethodGet {
			continue
		}
		seen[key] = struct{}{}

		exemption, ok := cloudWriteExemptions[key]
		if assert.Truef(t, ok, "%s is a cloud write with no agent tool or exemption", key) {
			assert.NotEmpty(t, exemption.category, key)
			assert.NotEmpty(t, exemption.reason, key)
		}
	}

	for key := range cloudWriteExemptions {
		assert.Containsf(t, seen, key, "%s is exempted but no cloud route registers it", key)
	}
}

func TestEveryCloudProtectedRouteIsClassified(t *testing.T) {
	gin.SetMode(gin.TestMode)

	registry, err := catalog.NewRegistry(catalog.RegistryParams{
		Providers: []platformcatalog.CatalogProvider{catalog.NewStaticProvider()},
	})
	require.NoError(t, err)

	engine := gin.New()
	populateRegistrations(t).Routes.RegisterProtected(engine.Group(routelint.APIV1BasePath))

	routes := engine.Routes()
	require.NotEmpty(t, routes)
	for _, route := range routes {
		policy := registry.PolicyForRoute(route.Method, route.Path)
		assert.NotEqualf(t, platformcatalog.RouteAccessClassUnclassified, policy.AccessClass,
			"%s %s must be owned by a platform feature or listed as an account shell route",
			route.Method, route.Path)
	}
}

func TestCloudRoutesJoinThePublicRouteTable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	public, err := api.RouteTable()
	require.NoError(t, err)

	withCloud, err := api.RouteTableWith(populateRegistrations(t).Routes)
	require.NoError(t, err)

	routes := func(table gin.RoutesInfo) map[string]struct{} {
		keys := make(map[string]struct{}, len(table))
		for _, info := range table {
			keys[info.Method+" "+info.Path] = struct{}{}
		}
		return keys
	}
	publicRoutes, cloudRoutes := routes(public), routes(withCloud)

	cloudOnly := []string{
		http.MethodPost + " /api/v1/cloud/signups",
		http.MethodPost + " /api/v1/cloud/signups/resend",
		http.MethodPost + " /api/v1/cloud/signups/verify",
		http.MethodGet + " /api/v1/system/network-pulse",
		http.MethodPost + " /api/v1/control-plane/tenants/provision",
		http.MethodGet + " /api/v1/me/billing",
		http.MethodGet + " /api/v1/me/entitlements",
		http.MethodGet + " /api/v1/me/platform-catalog",
	}
	for _, route := range cloudOnly {
		assert.Contains(t, cloudRoutes, route)
		assert.NotContains(t, publicRoutes, route)
	}

	publicConfig := http.MethodGet + " /api/v1/system/public-config"
	assert.Contains(t, publicRoutes, publicConfig)
	assert.Contains(t, cloudRoutes, publicConfig)
	assert.Len(t, cloudRoutes, len(publicRoutes)+len(cloudOnly)+4)
}
