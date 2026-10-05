package catalog

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/stretchr/testify/require"
)

type testProvider struct {
	products []platformcatalog.Product
	features []platformcatalog.Feature
	meters   []platformcatalog.Meter
}

func (p testProvider) Products() []platformcatalog.Product { return p.products }

func (p testProvider) Features() []platformcatalog.Feature { return p.features }

func (p testProvider) Meters() []platformcatalog.Meter { return p.meters }

func TestNewRegistry_ValidStaticProvider(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(RegistryParams{
		Providers: []platformcatalog.CatalogProvider{NewStaticProvider()},
	})

	require.NoError(t, err)
	require.NotEmpty(t, registry.ListProducts())
	require.NotEmpty(t, registry.ListFeatures())
	require.NotEmpty(t, registry.ListMeters())
}

func TestNewRegistry_ValidationFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider platformcatalog.CatalogProvider
		want     string
	}{
		{
			name: "duplicate products",
			provider: testProvider{
				products: []platformcatalog.Product{
					{Key: platformcatalog.ProductTMS},
					{Key: platformcatalog.ProductTMS},
				},
			},
			want: "duplicate product",
		},
		{
			name: "duplicate features",
			provider: testProvider{
				products: []platformcatalog.Product{{Key: platformcatalog.ProductTMS}},
				features: []platformcatalog.Feature{
					{Key: platformcatalog.FeatureCoreTMS, ProductKey: platformcatalog.ProductTMS},
					{Key: platformcatalog.FeatureCoreTMS, ProductKey: platformcatalog.ProductTMS},
				},
			},
			want: "duplicate feature",
		},
		{
			name: "duplicate meters",
			provider: testProvider{
				products: []platformcatalog.Product{{Key: platformcatalog.ProductTMS}},
				meters: []platformcatalog.Meter{
					{Key: platformcatalog.MeterAPIRequests, ProductKey: platformcatalog.ProductTMS},
					{Key: platformcatalog.MeterAPIRequests, ProductKey: platformcatalog.ProductTMS},
				},
			},
			want: "duplicate meter",
		},
		{
			name: "missing product reference",
			provider: testProvider{
				features: []platformcatalog.Feature{{Key: platformcatalog.FeatureCoreTMS, ProductKey: platformcatalog.ProductTMS}},
			},
			want: "references missing product",
		},
		{
			name: "missing required feature",
			provider: testProvider{
				products: []platformcatalog.Product{{Key: platformcatalog.ProductTMS}},
				features: []platformcatalog.Feature{
					{
						Key:              platformcatalog.FeatureDispatch,
						ProductKey:       platformcatalog.ProductTMS,
						RequiresFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureCoreTMS},
					},
				},
			},
			want: "requires missing feature",
		},
		{
			name: "self required feature",
			provider: testProvider{
				products: []platformcatalog.Product{{Key: platformcatalog.ProductTMS}},
				features: []platformcatalog.Feature{
					{
						Key:              platformcatalog.FeatureCoreTMS,
						ProductKey:       platformcatalog.ProductTMS,
						RequiresFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureCoreTMS},
					},
				},
			},
			want: "cannot require itself",
		},
		{
			name: "duplicate feature route refs",
			provider: testProvider{
				products: []platformcatalog.Product{{Key: platformcatalog.ProductTMS}},
				features: []platformcatalog.Feature{
					{
						Key:        platformcatalog.FeatureAccounting,
						ProductKey: platformcatalog.ProductTMS,
						Routes: []platformcatalog.RouteRef{{
							Method: "GET",
							Path:   "/api/v1/accounting-controls/",
						}},
					},
					{
						Key:        platformcatalog.FeatureBilling,
						ProductKey: platformcatalog.ProductTMS,
						Routes: []platformcatalog.RouteRef{{
							Method: "GET",
							Path:   "/api/v1/accounting-controls/",
						}},
					},
				},
			},
			want: "assigned to both feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRegistry(RegistryParams{
				Providers: []platformcatalog.CatalogProvider{tt.provider},
			})

			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestRegistry_FeatureForRoute(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(RegistryParams{
		Providers: []platformcatalog.CatalogProvider{NewStaticProvider()},
	})
	require.NoError(t, err)

	tests := []struct {
		name         string
		method       string
		routePattern string
		want         platformcatalog.FeatureKey
	}{
		{
			name:         "core route",
			method:       "GET",
			routePattern: "/api/v1/organizations/select-options/",
			want:         platformcatalog.FeatureCoreTMS,
		},
		{
			name:         "dispatch route",
			method:       "POST",
			routePattern: "/api/v1/shipments/",
			want:         platformcatalog.FeatureDispatch,
		},
		{
			name:         "billing route",
			method:       "GET",
			routePattern: "/api/v1/billing-queue/:itemID/",
			want:         platformcatalog.FeatureBilling,
		},
		{
			name:         "document route",
			method:       "POST",
			routePattern: "/api/v1/documents/upload/",
			want:         platformcatalog.FeatureDocumentManagement,
		},
		{
			name:         "table change alert route",
			method:       "GET",
			routePattern: "/api/v1/tca/subscriptions/",
			want:         platformcatalog.FeatureTableChangeAlerts,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feature, ok := registry.FeatureForRoute(tt.method, tt.routePattern)
			require.True(t, ok)
			require.Equal(t, tt.want, feature.Key)
		})
	}
}

func TestRegistry_FeatureForRouteUnknown(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(RegistryParams{
		Providers: []platformcatalog.CatalogProvider{NewStaticProvider()},
	})
	require.NoError(t, err)

	_, ok := registry.FeatureForRoute("GET", "/api/v1/unmapped/")
	require.False(t, ok)
}

func TestRegistry_PolicyForRouteAccountShell(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(RegistryParams{
		Providers: []platformcatalog.CatalogProvider{NewStaticProvider()},
	})
	require.NoError(t, err)

	for _, route := range accountShellRoutePatterns() {
		t.Run(route.name, func(t *testing.T) {
			policy := registry.PolicyForRoute(route.method, route.routePattern)
			require.Equal(t, platformcatalog.RouteAccessClassAccountShell, policy.AccessClass)
			require.Empty(t, policy.FeatureKey)
		})
	}
}

func TestRegistry_PolicyForRouteProductFeatures(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(RegistryParams{
		Providers: []platformcatalog.CatalogProvider{NewStaticProvider()},
	})
	require.NoError(t, err)

	routes := []struct {
		name         string
		method       string
		routePattern string
		wantFeature  platformcatalog.FeatureKey
	}{
		{
			name:         "dispatch shipment",
			method:       "GET",
			routePattern: "/api/v1/shipments/:id",
			wantFeature:  platformcatalog.FeatureDispatch,
		},
		{
			name:         "customer tenant data",
			method:       "GET",
			routePattern: "/api/v1/customers/",
			wantFeature:  platformcatalog.FeatureCoreTMS,
		},
		{
			name:         "fleet worker",
			method:       "POST",
			routePattern: "/api/v1/workers/",
			wantFeature:  platformcatalog.FeatureWorkforceCore,
		},
		{
			name:         "billing invoice",
			method:       "GET",
			routePattern: "/api/v1/billing/invoices/:invoiceID/",
			wantFeature:  platformcatalog.FeatureBilling,
		},
		{
			name:         "accounting journal entries",
			method:       "GET",
			routePattern: "/api/v1/accounting/journal-entries/",
			wantFeature:  platformcatalog.FeatureAccounting,
		},
		{
			name:         "table change alert subscription",
			method:       "PATCH",
			routePattern: "/api/v1/tca/subscriptions/:id/pause",
			wantFeature:  platformcatalog.FeatureTableChangeAlerts,
		},
	}

	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			policy := registry.PolicyForRoute(route.method, route.routePattern)
			require.Equal(t, platformcatalog.RouteAccessClassProduct, policy.AccessClass)
			require.Equal(t, route.wantFeature, policy.FeatureKey)
		})
	}
}

func TestRegistry_AppShellRoutesAreNotMappedToCommercialFeatures(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(RegistryParams{
		Providers: []platformcatalog.CatalogProvider{NewStaticProvider()},
	})
	require.NoError(t, err)

	for _, route := range accountShellRoutePatterns() {
		t.Run(route.name, func(t *testing.T) {
			_, ok := registry.FeatureForRoute(route.method, route.routePattern)
			require.False(t, ok)
		})
	}
}

func TestStaticProvider_RouteRefsReferenceProtectedProductRoutes(t *testing.T) {
	t.Parallel()

	prefixes := protectedProductRoutePrefixes()

	for _, feature := range NewStaticProvider().Features() {
		for _, route := range feature.Routes {
			t.Run(string(feature.Key)+" "+route.Path, func(t *testing.T) {
				owner, ok := longestPrefixOwner(prefixes, route.Path)
				require.Truef(
					t,
					ok,
					"route %s is not covered by any declared route prefix",
					route.Path,
				)
				require.Equal(t, owner, feature.Key)
			})
		}
	}
}

func TestStaticProvider_RoutePrefixesAreAllUsed(t *testing.T) {
	t.Parallel()

	used := make(map[string]struct{})
	for _, feature := range NewStaticProvider().Features() {
		for _, route := range feature.Routes {
			for _, prefix := range protectedProductRoutePrefixes() {
				if strings.HasPrefix(route.Path, prefix.prefix) {
					used[prefix.prefix] = struct{}{}
				}
			}
		}
	}

	for _, prefix := range protectedProductRoutePrefixes() {
		t.Run(prefix.prefix, func(t *testing.T) {
			_, ok := used[prefix.prefix]
			require.Truef(t, ok, "route prefix %s matches no catalog route", prefix.prefix)
		})
	}
}

func longestPrefixOwner(
	prefixes []protectedProductRoutePrefix,
	routePath string,
) (platformcatalog.FeatureKey, bool) {
	var (
		owner platformcatalog.FeatureKey
		best  int
	)
	for _, prefix := range prefixes {
		if !strings.HasPrefix(routePath, prefix.prefix) {
			continue
		}
		if len(prefix.prefix) > best {
			best = len(prefix.prefix)
			owner = prefix.featureKey
		}
	}

	return owner, best > 0
}

type protectedProductRoutePrefix struct {
	prefix     string
	featureKey platformcatalog.FeatureKey
}

type accountShellRoute struct {
	name         string
	method       string
	routePattern string
}

func accountShellRoutePatterns() []accountShellRoute {
	return []accountShellRoute{
		{
			name:         "current user",
			method:       "GET",
			routePattern: "/api/v1/users/me",
		},
		{
			name:         "current user trailing slash",
			method:       "GET",
			routePattern: "/api/v1/users/me/",
		},
		{
			name:         "current user organizations",
			method:       "GET",
			routePattern: "/api/v1/users/me/organizations/",
		},
		{
			name:         "current user switch organization",
			method:       "POST",
			routePattern: "/api/v1/users/me/switch-organization/",
		},
		{
			name:         "current user settings",
			method:       "PATCH",
			routePattern: "/api/v1/users/me/settings/",
		},
		{
			name:         "current user profile picture",
			method:       "POST",
			routePattern: "/api/v1/users/me/profile-picture/",
		},
		{
			name:         "current user delete profile picture",
			method:       "DELETE",
			routePattern: "/api/v1/users/me/profile-picture/",
		},
		{
			name:         "current user change password",
			method:       "POST",
			routePattern: "/api/v1/users/me/change-password/",
		},
		{
			name:         "permission manifest",
			method:       "GET",
			routePattern: "/api/v1/me/permissions",
		},
		{
			name:         "permission manifest trailing slash",
			method:       "GET",
			routePattern: "/api/v1/me/permissions/",
		},
		{
			name:         "permission version",
			method:       "GET",
			routePattern: "/api/v1/me/permissions/version",
		},
		{
			name:         "permission resource",
			method:       "GET",
			routePattern: "/api/v1/me/permissions/:resource",
		},
		{
			name:         "permission check",
			method:       "POST",
			routePattern: "/api/v1/me/permissions/check",
		},
		{
			name:         "billing shell",
			method:       "GET",
			routePattern: "/api/v1/me/billing",
		},
		{
			name:         "billing shell trailing slash",
			method:       "GET",
			routePattern: "/api/v1/me/billing/",
		},
		{
			name:         "platform catalog shell",
			method:       "GET",
			routePattern: "/api/v1/me/platform-catalog",
		},
		{
			name:         "platform catalog shell trailing slash",
			method:       "GET",
			routePattern: "/api/v1/me/platform-catalog/",
		},
		{
			name:         "entitlements shell",
			method:       "GET",
			routePattern: "/api/v1/me/entitlements",
		},
		{
			name:         "entitlements shell trailing slash",
			method:       "GET",
			routePattern: "/api/v1/me/entitlements/",
		},
		{
			name:         "organization read",
			method:       "GET",
			routePattern: "/api/v1/organizations/:id",
		},
		{
			name:         "organization update",
			method:       "PUT",
			routePattern: "/api/v1/organizations/:id",
		},
		{
			name:         "organization logo read",
			method:       "GET",
			routePattern: "/api/v1/organizations/:id/logo",
		},
		{
			name:         "organization logo",
			method:       "POST",
			routePattern: "/api/v1/organizations/:id/logo",
		},
		{
			name:         "organization logo delete",
			method:       "DELETE",
			routePattern: "/api/v1/organizations/:id/logo",
		},
		{
			name:         "organization microsoft sso read",
			method:       "GET",
			routePattern: "/api/v1/organizations/:id/microsoft-sso",
		},
		{
			name:         "organization microsoft sso update",
			method:       "PUT",
			routePattern: "/api/v1/organizations/:id/microsoft-sso",
		},
		{
			name:         "organization okta sso read",
			method:       "GET",
			routePattern: "/api/v1/organizations/:id/okta-sso",
		},
		{
			name:         "organization okta sso update",
			method:       "PUT",
			routePattern: "/api/v1/organizations/:id/okta-sso",
		},
		{
			name:         "state select options",
			method:       "GET",
			routePattern: "/api/v1/us-states/select-options/",
		},
		{
			name:         "state select option",
			method:       "GET",
			routePattern: "/api/v1/us-states/select-options/:usStateID",
		},
		{
			name:         "notification list",
			method:       "GET",
			routePattern: "/api/v1/notifications/",
		},
		{
			name:         "notifications",
			method:       "GET",
			routePattern: "/api/v1/notifications/unread-count",
		},
		{
			name:         "notifications mark read",
			method:       "PATCH",
			routePattern: "/api/v1/notifications/mark-read",
		},
		{
			name:         "notifications mark all read",
			method:       "PATCH",
			routePattern: "/api/v1/notifications/mark-all-read",
		},
		{
			name:         "page favorites list",
			method:       "GET",
			routePattern: "/api/v1/page-favorites/",
		},
		{
			name:         "page favorites",
			method:       "GET",
			routePattern: "/api/v1/page-favorites/check",
		},
		{
			name:         "page favorites toggle",
			method:       "POST",
			routePattern: "/api/v1/page-favorites/toggle",
		},
		{
			name:         "realtime stream",
			method:       "GET",
			routePattern: "/api/v1/realtime/stream/",
		},
		{
			name:         "platform catalog",
			method:       "GET",
			routePattern: "/api/v1/platform-catalog/products",
		},
		{
			name:         "platform catalog features",
			method:       "GET",
			routePattern: "/api/v1/platform-catalog/features",
		},
		{
			name:         "platform catalog meters",
			method:       "GET",
			routePattern: "/api/v1/platform-catalog/meters",
		},
		{
			name:         "platform catalog validate",
			method:       "GET",
			routePattern: "/api/v1/platform-catalog/validate",
		},
	}
}

func protectedProductRoutePrefixes() []protectedProductRoutePrefix {
	return []protectedProductRoutePrefix{
		{prefix: "/api/v1/accessorial-charges/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/account-types/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/accounting-controls/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/accounting/accounts-receivable/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/accounting/bank-receipt-batches/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/accounting/bank-receipt-work-items/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/accounting/bank-receipts/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/accounting/customer-payments/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/accounting/journal-entries/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/accounting/journal-reversals/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/accounting/manual-journals/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/accounting/statements/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/accounting/trial-balance/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/admin/database-sessions/", featureKey: platformcatalog.FeatureAdministration},
		{prefix: "/api/v1/admin/document-operations/", featureKey: platformcatalog.FeatureDocumentManagement},
		{prefix: "/api/v1/agent-controls/", featureKey: platformcatalog.FeatureAgentAutomation},
		{prefix: "/api/v1/agent-exceptions/", featureKey: platformcatalog.FeatureAgentAutomation},
		{prefix: "/api/v1/agent-plans/", featureKey: platformcatalog.FeatureAgentAutomation},
		{prefix: "/api/v1/agent-proposals/", featureKey: platformcatalog.FeatureAgentAutomation},
		{prefix: "/api/v1/agent-definitions/", featureKey: platformcatalog.FeatureAgentAutomation},
		{prefix: "/api/v1/agent-extensions/", featureKey: platformcatalog.FeatureAgentAutomation},
		{prefix: "/api/v1/agent-plans/", featureKey: platformcatalog.FeatureAgentAutomation},
		{prefix: "/api/v1/agent-runs/", featureKey: platformcatalog.FeatureAgentAutomation},
		{prefix: "/api/v1/ai-providers/", featureKey: platformcatalog.FeatureAgentAutomation},
		{prefix: "/api/v1/analytics/", featureKey: platformcatalog.FeatureAnalytics},
		{prefix: "/api/v1/api-keys/", featureKey: platformcatalog.FeatureAPIKeys},
		{prefix: "/api/v1/assignments/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/assistant/", featureKey: platformcatalog.FeatureAgentAutomation},
		{prefix: "/api/v1/billing-controls/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/billing-queue/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/billing/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/capture/", featureKey: platformcatalog.FeatureDocumentManagement},
		{prefix: "/api/v1/carriers/", featureKey: platformcatalog.FeatureCoreTMS},
		{prefix: "/api/v1/commodities/", featureKey: platformcatalog.FeatureCoreTMS},
		{prefix: "/api/v1/custom-fields/", featureKey: platformcatalog.FeatureAdministration},
		{prefix: "/api/v1/customers/", featureKey: platformcatalog.FeatureCoreTMS},
		{prefix: "/api/v1/data-entry-controls/", featureKey: platformcatalog.FeatureCoreTMS},
		{prefix: "/api/v1/data-retention/", featureKey: platformcatalog.FeatureAdministration},
		{prefix: "/api/v1/detention-policies/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/detention/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/dispatch-controls/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/distance-controls/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/distance-overrides/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/distance-profiles/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/document-controls/", featureKey: platformcatalog.FeatureDocumentManagement},
		{prefix: "/api/v1/document-packet-rules/", featureKey: platformcatalog.FeatureDocumentManagement},
		{prefix: "/api/v1/document-parsing-rules/", featureKey: platformcatalog.FeatureDocumentManagement},
		{prefix: "/api/v1/document-templates/", featureKey: platformcatalog.FeatureDocumentManagement},
		{prefix: "/api/v1/document-types/", featureKey: platformcatalog.FeatureDocumentManagement},
		{prefix: "/api/v1/documents/", featureKey: platformcatalog.FeatureDocumentManagement},
		{prefix: "/api/v1/dot-hazmat-references/", featureKey: platformcatalog.FeatureCoreTMS},
		{prefix: "/api/v1/edi/", featureKey: platformcatalog.FeatureEDIIntegration},
		{prefix: "/api/v1/email-logs/", featureKey: platformcatalog.FeatureAdministration},
		{prefix: "/api/v1/email-profiles/", featureKey: platformcatalog.FeatureAdministration},
		{prefix: "/api/v1/email-suppressions/", featureKey: platformcatalog.FeatureAdministration},
		{prefix: "/api/v1/equipment-manufacturers/", featureKey: platformcatalog.FeatureFleetMaintenance},
		{prefix: "/api/v1/equipment-types/", featureKey: platformcatalog.FeatureFleetMaintenance},
		{prefix: "/api/v1/exchange-rates/", featureKey: platformcatalog.FeatureExchangeRateIntegration},
		{prefix: "/api/v1/fiscal-periods/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/fiscal-years/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/fleet-codes/", featureKey: platformcatalog.FeatureFleetMaintenance},
		{prefix: "/api/v1/formula-templates/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/gl-accounts/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/google-maps/", featureKey: platformcatalog.FeatureGoogleMapsIntegration},
		{prefix: "/api/v1/hazardous-materials/", featureKey: platformcatalog.FeatureCoreTMS},
		{prefix: "/api/v1/hazmat-segregation-rules/", featureKey: platformcatalog.FeatureCoreTMS},
		{prefix: "/api/v1/hold-reasons/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/insights/", featureKey: platformcatalog.FeatureAgentAutomation},
		{prefix: "/api/v1/integrations/", featureKey: platformcatalog.FeatureSamsaraIntegration},
		{prefix: "/api/v1/invoice-adjustment-controls/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/jurisdiction-rule-overrides/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/jurisdiction-rules/", featureKey: platformcatalog.FeatureAccounting},
		{prefix: "/api/v1/location-categories/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/locations/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/orders/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/organizations/:id/", featureKey: platformcatalog.FeatureAdministration},
		{prefix: "/api/v1/organizations/select-options/", featureKey: platformcatalog.FeatureCoreTMS},
		{prefix: "/api/v1/permissions/", featureKey: platformcatalog.FeatureAdministration},
		{prefix: "/api/v1/portal/", featureKey: platformcatalog.FeatureDriverPortal},
		{prefix: "/api/v1/rate-agreements/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/rate-confirmations/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/rate-imports/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/rate-matrices/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/rate-quotes/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/rate-simulations/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/rate-zones/", featureKey: platformcatalog.FeatureBilling},
		{prefix: "/api/v1/recurring-shipments/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/reports/", featureKey: platformcatalog.FeatureAnalytics},
		{prefix: "/api/v1/role-assignments/", featureKey: platformcatalog.FeatureAdministration},
		{prefix: "/api/v1/roles/", featureKey: platformcatalog.FeatureAdministration},
		{prefix: "/api/v1/routing-guides/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/search/", featureKey: platformcatalog.FeatureGlobalSearch},
		{prefix: "/api/v1/sequence-configs/", featureKey: platformcatalog.FeatureCoreTMS},
		{prefix: "/api/v1/service-failure-reason-codes/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/service-failures/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/service-types/", featureKey: platformcatalog.FeatureCoreTMS},
		{prefix: "/api/v1/shipment-controls/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/shipment-events/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/shipment-moves/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/shipment-types/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/shipments/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/stored-mileages/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/tables/", featureKey: platformcatalog.FeatureAgentAutomation},
		{prefix: "/api/v1/tca/", featureKey: platformcatalog.FeatureTableChangeAlerts},
		{prefix: "/api/v1/tenders/", featureKey: platformcatalog.FeatureDispatch},
		{prefix: "/api/v1/tractors/", featureKey: platformcatalog.FeatureFleetMaintenance},
		{prefix: "/api/v1/trailers/", featureKey: platformcatalog.FeatureFleetMaintenance},
		{prefix: "/api/v1/users/", featureKey: platformcatalog.FeatureAdministration},
		{prefix: "/api/v1/weather-alerts/", featureKey: platformcatalog.FeatureCoreTMS},
		{prefix: "/api/v1/workers/", featureKey: platformcatalog.FeatureWorkforceCore},
	}
}
