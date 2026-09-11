package platformaudit

import (
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/routelint"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/stretchr/testify/require"
)

const (
	apiDir      = "../"
	handlersDir = "../handlers"
)

func legacyEntitlements() []platformcatalog.FeatureKey {
	return []platformcatalog.FeatureKey{
		platformcatalog.FeatureAnalytics,
		platformcatalog.FeatureAccounting,
		platformcatalog.FeatureAPIKeys,
		platformcatalog.FeatureBilling,
		platformcatalog.FeatureCoreTMS,
		platformcatalog.FeatureDispatch,
		platformcatalog.FeatureDocumentIntelligence,
		platformcatalog.FeatureDocumentManagement,
		platformcatalog.FeatureEDIIntegration,
		platformcatalog.FeatureExchangeRateIntegration,
		platformcatalog.FeatureFleetMaintenance,
		platformcatalog.FeatureGlobalSearch,
		platformcatalog.FeatureGoogleMapsIntegration,
		platformcatalog.FeatureRealtimeNotifications,
		platformcatalog.FeatureSamsaraIntegration,
		platformcatalog.FeatureTableChangeAlerts,
	}
}

func runAudit(t *testing.T) Findings {
	t.Helper()

	registry, err := platformcatalog.NewRegistry(platformcatalog.RegistryParams{
		Providers: []platformcatalog.CatalogProvider{platformcatalog.NewStaticProvider()},
	})
	require.NoError(t, err)

	routes, err := routelint.ProtectedRoutes(apiDir, handlersDir)
	require.NoError(t, err)

	return Audit(Input{
		Registry:      registry,
		Routes:        routes,
		LegacyEntitle: legacyEntitlements(),
	})
}

func TestEveryFeatureIsSellableThroughAPack(t *testing.T) {
	t.Parallel()

	findings := runAudit(t)

	names := make([]string, 0, len(findings.UnsellableFeatures))
	for _, featureKey := range findings.UnsellableFeatures {
		names = append(names, string(featureKey))
	}

	require.Emptyf(
		t,
		names,
		"every feature must belong to at least one pack; a feature in no pack cannot be "+
			"provisioned and its surface is unreachable once access is enforced: %s",
		strings.Join(names, ", "),
	)
}

func TestNoProtectedRouteIsStrandedBehindAnUnsellableFeature(t *testing.T) {
	t.Parallel()

	findings := runAudit(t)

	stranded := make([]string, 0)
	for featureKey, routes := range findings.UnreachableRoutes {
		stranded = append(
			stranded,
			string(featureKey)+" ("+strings.Join(routes[:1], "")+", +more)",
		)
	}
	sort.Strings(stranded)

	require.Emptyf(
		t,
		stranded,
		"these routes are owned by features no pack sells:\n%s",
		strings.Join(stranded, "\n"),
	)
}

func TestNoGraphQLSourceIsStrandedBehindAnUnsellableFeature(t *testing.T) {
	t.Parallel()

	findings := runAudit(t)

	stranded := make([]string, 0)
	for featureKey, sources := range findings.UnreachableGraphQL {
		for _, source := range sources {
			stranded = append(stranded, string(featureKey)+": "+string(source))
		}
	}
	sort.Strings(stranded)

	require.Emptyf(
		t,
		stranded,
		"these GraphQL sources are owned by features no pack sells:\n%s",
		strings.Join(stranded, "\n"),
	)
}

func TestExistingTenantsKeepEveryRouteTheyCanReachToday(t *testing.T) {
	t.Parallel()

	findings := runAudit(t)

	lost := make([]string, 0)
	for featureKey, routes := range findings.MigrationLostRoutes {
		lost = append(lost, string(featureKey)+": "+strings.Join(routes, ", "))
	}
	sort.Strings(lost)

	require.Emptyf(
		t,
		lost,
		"a tenant holding the pre-split entitlement set would lose these REST routes; "+
			"either give the owning feature a legacy grant or provision the new key "+
			"before shipping:\n%s",
		strings.Join(lost, "\n"),
	)
}

func TestGraphQLSurfaceLostOnEnforceIsReviewed(t *testing.T) {
	t.Parallel()

	findings := runAudit(t)

	lost := make([]string, 0)
	for featureKey, sources := range findings.MigrationLostGraphQL {
		for _, source := range sources {
			lost = append(lost, string(featureKey)+": "+string(source))
		}
	}
	sort.Strings(lost)

	require.ElementsMatchf(
		t,
		[]string{
			"workforce.benefits: benefits.graphqls",
			"workforce.compliance: worker_credential.graphqls",
			"workforce.compliance: worker_dqf.graphqls",
			"workforce.compliance: worker_drug_alcohol.graphqls",
			"workforce.safety: fleet_safety.graphqls",
			"workforce.safety: worker_injury.graphqls",
			"workforce.safety: worker_safety.graphqls",
			"workforce.self_service: self_service.graphqls",
			"workforce.talent: performance_review.graphqls",
			"workforce.talent: worker_training.graphqls",
			"workforce.time_off: pto_policy.graphqls",
			"workforce.time_off: worker_leave.graphqls",
			"workforce.time_tracking: scheduling.graphqls",
			"workforce.time_tracking: timesheet.graphqls",
		},
		lost,
		"turning graphqlAccessMode to enforce denies these schemas to a tenant holding only "+
			"the pre-split entitlement set; each one is a workforce module that is now sold "+
			"separately, so the list must stay deliberate rather than grow by accident",
	)
}

func TestLegacyGrantsMatchTheMigrationPlan(t *testing.T) {
	t.Parallel()

	findings := runAudit(t)

	require.ElementsMatch(
		t,
		[]platformcatalog.FeatureKey{
			platformcatalog.FeatureAdministration,
			platformcatalog.FeatureAgentAutomation,
			platformcatalog.FeatureDriverPortal,
			platformcatalog.FeatureSettlement,
			platformcatalog.FeatureWorkforceCore,
		},
		findings.LegacyOnlyFeatures,
		"legacy grants exist only to keep already-provisioned tenants working while the "+
			"control plane learns the new feature keys; adding or removing one is a "+
			"deliberate migration decision",
	)
}

func TestEveryPackGrantsSomeSurface(t *testing.T) {
	t.Parallel()

	findings := runAudit(t)
	require.NotEmpty(t, findings.PackReach)

	for _, reach := range findings.PackReach {
		t.Run(string(reach.Pack.Key), func(t *testing.T) {
			require.NotEmpty(t, reach.Features)
			require.Greaterf(
				t,
				reach.RouteCount+reach.GraphQLSources,
				0,
				"pack %s grants no reachable surface", reach.Pack.Key,
			)
		})
	}
}

func TestWorkforcePackNeedsNoTransportManagementFeature(t *testing.T) {
	t.Parallel()

	findings := runAudit(t)

	for _, reach := range findings.PackReach {
		if reach.Pack.Key != platformcatalog.PackWorkforce {
			continue
		}
		require.True(t, reach.Pack.Standalone)
		require.Empty(t, reach.Pack.RequiresPacks)
		require.Greater(t, reach.RouteCount, 0)
		require.Greater(t, reach.GraphQLSources, 0)

		return
	}

	t.Fatalf("workforce pack not found in audit output")
}
