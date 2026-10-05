package catalog_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/routelint"
	"github.com/emoss08/trenova/internal/cloud/catalog"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/stretchr/testify/require"
)

const (
	apiDir      = "../../api"
	handlersDir = "../../api/handlers"
)

func newCatalogRegistry(t *testing.T) *catalog.Registry {
	t.Helper()

	registry, err := catalog.NewRegistry(catalog.RegistryParams{
		Providers: []platformcatalog.CatalogProvider{catalog.NewStaticProvider()},
	})
	require.NoError(t, err)

	return registry
}

func TestEveryProtectedRouteIsClassified(t *testing.T) {
	t.Parallel()

	registry := newCatalogRegistry(t)

	routes, err := routelint.ProtectedRoutes(apiDir, handlersDir)
	require.NoError(t, err)

	unclassified := make([]string, 0)
	for _, route := range routes {
		policy := registry.PolicyForRoute(route.Method, route.Path)
		if policy.AccessClass == platformcatalog.RouteAccessClassUnclassified {
			unclassified = append(unclassified, route.Package+": "+route.Key())
		}
	}
	sort.Strings(unclassified)

	require.Emptyf(
		t,
		unclassified,
		"every protected route must be owned by a platform feature or listed as an account "+
			"shell route; unclassified routes fail open and cannot be sold as part of a pack:\n%s",
		strings.Join(unclassified, "\n"),
	)
}

func TestClassifiedProductRoutesResolveToKnownFeatures(t *testing.T) {
	t.Parallel()

	registry := newCatalogRegistry(t)

	routes, err := routelint.ProtectedRoutes(apiDir, handlersDir)
	require.NoError(t, err)

	for _, route := range routes {
		policy := registry.PolicyForRoute(route.Method, route.Path)
		if policy.AccessClass != platformcatalog.RouteAccessClassProduct {
			continue
		}
		_, ok := registry.GetFeature(policy.FeatureKey)
		require.Truef(t, ok, "route %s maps to unknown feature %q", route.Key(), policy.FeatureKey)
	}
}

func TestCatalogRoutesReferenceRegisteredRoutes(t *testing.T) {
	t.Parallel()

	routes, err := routelint.ProtectedRoutes(apiDir, handlersDir)
	require.NoError(t, err)

	registered := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		registered[route.Key()] = struct{}{}
	}

	stale := make([]string, 0)
	for _, feature := range catalog.NewStaticProvider().Features() {
		for _, ref := range feature.Routes {
			key := strings.ToUpper(ref.Method) + " " + ref.Path
			if _, ok := registered[key]; !ok {
				stale = append(stale, string(feature.Key)+": "+key)
			}
		}
	}
	sort.Strings(stale)

	require.Emptyf(
		t,
		stale,
		"catalog route refs must match routes the router actually registers; "+
			"these refs match no registered route:\n%s",
		strings.Join(stale, "\n"),
	)
}
