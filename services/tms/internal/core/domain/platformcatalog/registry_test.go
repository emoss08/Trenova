package platformcatalog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type testProvider struct {
	products []Product
	features []Feature
	meters   []Meter
}

func (p testProvider) Products() []Product { return p.products }

func (p testProvider) Features() []Feature { return p.features }

func (p testProvider) Meters() []Meter { return p.meters }

func TestNewRegistry_ValidStaticProvider(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(RegistryParams{
		Providers: []CatalogProvider{NewStaticProvider()},
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
		provider CatalogProvider
		want     string
	}{
		{
			name: "duplicate products",
			provider: testProvider{
				products: []Product{
					{Key: ProductTMS},
					{Key: ProductTMS},
				},
			},
			want: "duplicate product",
		},
		{
			name: "duplicate features",
			provider: testProvider{
				products: []Product{{Key: ProductTMS}},
				features: []Feature{
					{Key: FeatureCoreTMS, ProductKey: ProductTMS},
					{Key: FeatureCoreTMS, ProductKey: ProductTMS},
				},
			},
			want: "duplicate feature",
		},
		{
			name: "duplicate meters",
			provider: testProvider{
				products: []Product{{Key: ProductTMS}},
				meters: []Meter{
					{Key: MeterAPIRequests, ProductKey: ProductTMS},
					{Key: MeterAPIRequests, ProductKey: ProductTMS},
				},
			},
			want: "duplicate meter",
		},
		{
			name: "missing product reference",
			provider: testProvider{
				features: []Feature{{Key: FeatureCoreTMS, ProductKey: ProductTMS}},
			},
			want: "references missing product",
		},
		{
			name: "missing required feature",
			provider: testProvider{
				products: []Product{{Key: ProductTMS}},
				features: []Feature{
					{
						Key:              FeatureDispatch,
						ProductKey:       ProductTMS,
						RequiresFeatures: []FeatureKey{FeatureCoreTMS},
					},
				},
			},
			want: "requires missing feature",
		},
		{
			name: "self required feature",
			provider: testProvider{
				products: []Product{{Key: ProductTMS}},
				features: []Feature{
					{
						Key:              FeatureCoreTMS,
						ProductKey:       ProductTMS,
						RequiresFeatures: []FeatureKey{FeatureCoreTMS},
					},
				},
			},
			want: "cannot require itself",
		},
		{
			name: "duplicate feature route refs",
			provider: testProvider{
				products: []Product{{Key: ProductTMS}},
				features: []Feature{
					{
						Key:        FeatureAccounting,
						ProductKey: ProductTMS,
						Routes: []RouteRef{{
							Method: "GET",
							Path:   "/api/v1/accounting-controls/",
						}},
					},
					{
						Key:        FeatureBilling,
						ProductKey: ProductTMS,
						Routes: []RouteRef{{
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
				Providers: []CatalogProvider{tt.provider},
			})

			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestRegistry_FeatureForRoute(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(RegistryParams{
		Providers: []CatalogProvider{NewStaticProvider()},
	})
	require.NoError(t, err)

	tests := []struct {
		name         string
		method       string
		routePattern string
		want         FeatureKey
	}{
		{
			name:         "core route",
			method:       "GET",
			routePattern: "/api/v1/organizations/select-options/",
			want:         FeatureCoreTMS,
		},
		{
			name:         "dispatch route",
			method:       "POST",
			routePattern: "/api/v1/shipments/",
			want:         FeatureDispatch,
		},
		{
			name:         "billing route",
			method:       "GET",
			routePattern: "/api/v1/billing-queue/:itemID/",
			want:         FeatureBilling,
		},
		{
			name:         "document route",
			method:       "POST",
			routePattern: "/api/v1/documents/upload/",
			want:         FeatureDocumentManagement,
		},
		{
			name:         "table change alert route",
			method:       "GET",
			routePattern: "/api/v1/tca/subscriptions/",
			want:         FeatureTableChangeAlerts,
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
		Providers: []CatalogProvider{NewStaticProvider()},
	})
	require.NoError(t, err)

	_, ok := registry.FeatureForRoute("GET", "/api/v1/unmapped/")
	require.False(t, ok)
}

func TestRegistry_PolicyForRouteAccountShell(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(RegistryParams{
		Providers: []CatalogProvider{NewStaticProvider()},
	})
	require.NoError(t, err)

	for _, route := range accountShellRoutePatterns() {
		t.Run(route.name, func(t *testing.T) {
			policy := registry.PolicyForRoute(route.method, route.routePattern)
			require.Equal(t, RouteAccessClassAccountShell, policy.AccessClass)
			require.Empty(t, policy.FeatureKey)
		})
	}
}

func TestRegistry_PolicyForRouteProductFeatures(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(RegistryParams{
		Providers: []CatalogProvider{NewStaticProvider()},
	})
	require.NoError(t, err)

	routes := []struct {
		name         string
		method       string
		routePattern string
		wantFeature  FeatureKey
	}{
		{
			name:         "dispatch shipment",
			method:       "GET",
			routePattern: "/api/v1/shipments/:id",
			wantFeature:  FeatureDispatch,
		},
		{
			name:         "customer tenant data",
			method:       "GET",
			routePattern: "/api/v1/customers/",
			wantFeature:  FeatureCoreTMS,
		},
		{
			name:         "fleet worker",
			method:       "POST",
			routePattern: "/api/v1/workers/",
			wantFeature:  FeatureWorkforceCore,
		},
		{
			name:         "billing invoice",
			method:       "GET",
			routePattern: "/api/v1/billing/invoices/:invoiceID/",
			wantFeature:  FeatureBilling,
		},
		{
			name:         "accounting journal entries",
			method:       "GET",
			routePattern: "/api/v1/accounting/journal-entries/",
			wantFeature:  FeatureAccounting,
		},
		{
			name:         "table change alert subscription",
			method:       "PATCH",
			routePattern: "/api/v1/tca/subscriptions/:id/pause",
			wantFeature:  FeatureTableChangeAlerts,
		},
	}

	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			policy := registry.PolicyForRoute(route.method, route.routePattern)
			require.Equal(t, RouteAccessClassProduct, policy.AccessClass)
			require.Equal(t, route.wantFeature, policy.FeatureKey)
		})
	}
}

func TestRegistry_AppShellRoutesAreNotMappedToCommercialFeatures(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(RegistryParams{
		Providers: []CatalogProvider{NewStaticProvider()},
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
) (FeatureKey, bool) {
	var (
		owner FeatureKey
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
	featureKey FeatureKey
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
			name:         "realtime token request",
			method:       "GET",
			routePattern: "/api/v1/realtime/token-request/",
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
		{prefix: "/api/v1/accessorial-charges/", featureKey: FeatureBilling},
		{prefix: "/api/v1/account-types/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/accounting-controls/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/accounting/accounts-receivable/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/accounting/bank-receipt-batches/", featureKey: FeatureBilling},
		{prefix: "/api/v1/accounting/bank-receipt-work-items/", featureKey: FeatureBilling},
		{prefix: "/api/v1/accounting/bank-receipts/", featureKey: FeatureBilling},
		{prefix: "/api/v1/accounting/customer-payments/", featureKey: FeatureBilling},
		{prefix: "/api/v1/accounting/journal-entries/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/accounting/journal-reversals/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/accounting/manual-journals/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/accounting/statements/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/accounting/trial-balance/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/admin/database-sessions/", featureKey: FeatureAdministration},
		{prefix: "/api/v1/admin/document-operations/", featureKey: FeatureDocumentManagement},
		{prefix: "/api/v1/agent-controls/", featureKey: FeatureAgentAutomation},
		{prefix: "/api/v1/agent-exceptions/", featureKey: FeatureAgentAutomation},
		{prefix: "/api/v1/agent-proposals/", featureKey: FeatureAgentAutomation},
		{prefix: "/api/v1/agent-runs/", featureKey: FeatureAgentAutomation},
		{prefix: "/api/v1/analytics/", featureKey: FeatureAnalytics},
		{prefix: "/api/v1/api-keys/", featureKey: FeatureAPIKeys},
		{prefix: "/api/v1/assignments/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/billing-controls/", featureKey: FeatureBilling},
		{prefix: "/api/v1/billing-queue/", featureKey: FeatureBilling},
		{prefix: "/api/v1/billing/", featureKey: FeatureBilling},
		{prefix: "/api/v1/carriers/", featureKey: FeatureCoreTMS},
		{prefix: "/api/v1/commodities/", featureKey: FeatureCoreTMS},
		{prefix: "/api/v1/custom-fields/", featureKey: FeatureAdministration},
		{prefix: "/api/v1/customers/", featureKey: FeatureCoreTMS},
		{prefix: "/api/v1/data-entry-controls/", featureKey: FeatureCoreTMS},
		{prefix: "/api/v1/data-retention/", featureKey: FeatureAdministration},
		{prefix: "/api/v1/detention-policies/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/detention/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/dispatch-controls/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/distance-controls/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/distance-overrides/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/distance-profiles/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/document-controls/", featureKey: FeatureDocumentManagement},
		{prefix: "/api/v1/document-packet-rules/", featureKey: FeatureDocumentManagement},
		{prefix: "/api/v1/document-parsing-rules/", featureKey: FeatureDocumentManagement},
		{prefix: "/api/v1/document-templates/", featureKey: FeatureDocumentManagement},
		{prefix: "/api/v1/document-types/", featureKey: FeatureDocumentManagement},
		{prefix: "/api/v1/documents/", featureKey: FeatureDocumentManagement},
		{prefix: "/api/v1/dot-hazmat-references/", featureKey: FeatureCoreTMS},
		{prefix: "/api/v1/edi/", featureKey: FeatureEDIIntegration},
		{prefix: "/api/v1/email-logs/", featureKey: FeatureAdministration},
		{prefix: "/api/v1/email-profiles/", featureKey: FeatureAdministration},
		{prefix: "/api/v1/email-suppressions/", featureKey: FeatureAdministration},
		{prefix: "/api/v1/equipment-manufacturers/", featureKey: FeatureFleetMaintenance},
		{prefix: "/api/v1/equipment-types/", featureKey: FeatureFleetMaintenance},
		{prefix: "/api/v1/exchange-rates/", featureKey: FeatureExchangeRateIntegration},
		{prefix: "/api/v1/fiscal-periods/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/fiscal-years/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/fleet-codes/", featureKey: FeatureFleetMaintenance},
		{prefix: "/api/v1/formula-templates/", featureKey: FeatureBilling},
		{prefix: "/api/v1/gl-accounts/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/google-maps/", featureKey: FeatureGoogleMapsIntegration},
		{prefix: "/api/v1/hazardous-materials/", featureKey: FeatureCoreTMS},
		{prefix: "/api/v1/hazmat-segregation-rules/", featureKey: FeatureCoreTMS},
		{prefix: "/api/v1/hold-reasons/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/integrations/", featureKey: FeatureSamsaraIntegration},
		{prefix: "/api/v1/invoice-adjustment-controls/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/jurisdiction-rule-overrides/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/jurisdiction-rules/", featureKey: FeatureAccounting},
		{prefix: "/api/v1/location-categories/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/locations/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/orders/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/organizations/:id/", featureKey: FeatureAdministration},
		{prefix: "/api/v1/organizations/select-options/", featureKey: FeatureCoreTMS},
		{prefix: "/api/v1/permissions/", featureKey: FeatureAdministration},
		{prefix: "/api/v1/portal/", featureKey: FeatureDriverPortal},
		{prefix: "/api/v1/rate-agreements/", featureKey: FeatureBilling},
		{prefix: "/api/v1/rate-confirmations/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/rate-imports/", featureKey: FeatureBilling},
		{prefix: "/api/v1/rate-matrices/", featureKey: FeatureBilling},
		{prefix: "/api/v1/rate-quotes/", featureKey: FeatureBilling},
		{prefix: "/api/v1/rate-simulations/", featureKey: FeatureBilling},
		{prefix: "/api/v1/rate-zones/", featureKey: FeatureBilling},
		{prefix: "/api/v1/recurring-shipments/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/reports/", featureKey: FeatureAnalytics},
		{prefix: "/api/v1/role-assignments/", featureKey: FeatureAdministration},
		{prefix: "/api/v1/roles/", featureKey: FeatureAdministration},
		{prefix: "/api/v1/routing-guides/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/search/", featureKey: FeatureGlobalSearch},
		{prefix: "/api/v1/sequence-configs/", featureKey: FeatureCoreTMS},
		{prefix: "/api/v1/service-failure-reason-codes/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/service-failures/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/service-types/", featureKey: FeatureCoreTMS},
		{prefix: "/api/v1/shipment-controls/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/shipment-events/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/shipment-moves/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/shipment-types/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/shipments/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/stored-mileages/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/tca/", featureKey: FeatureTableChangeAlerts},
		{prefix: "/api/v1/tenders/", featureKey: FeatureDispatch},
		{prefix: "/api/v1/tractors/", featureKey: FeatureFleetMaintenance},
		{prefix: "/api/v1/trailers/", featureKey: FeatureFleetMaintenance},
		{prefix: "/api/v1/users/", featureKey: FeatureAdministration},
		{prefix: "/api/v1/weather-alerts/", featureKey: FeatureCoreTMS},
		{prefix: "/api/v1/workers/", featureKey: FeatureWorkforceCore},
	}
}
