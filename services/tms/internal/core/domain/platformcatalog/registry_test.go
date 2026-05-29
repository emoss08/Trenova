package platformcatalog

import (
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
			wantFeature:  FeatureFleetMaintenance,
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

func TestStaticProvider_ProductRoutePrefixesCovered(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(RegistryParams{
		Providers: []CatalogProvider{NewStaticProvider()},
	})
	require.NoError(t, err)

	for _, route := range protectedProductRouteFeatures() {
		t.Run(route.pattern, func(t *testing.T) {
			policy := registry.PolicyForRoute(route.method, route.pattern)
			require.Equal(t, RouteAccessClassProduct, policy.AccessClass)
			require.Equal(t, route.featureKey, policy.FeatureKey)
		})
	}
}

func TestStaticProvider_RouteRefsReferenceProtectedProductRoutes(t *testing.T) {
	t.Parallel()

	protectedProductRoutes := make(map[string]FeatureKey, len(protectedProductRouteFeatures()))
	for _, route := range protectedProductRouteFeatures() {
		protectedProductRoutes[route.method+" "+route.pattern] = route.featureKey
	}

	for _, feature := range NewStaticProvider().Features() {
		for _, route := range feature.Routes {
			t.Run(string(feature.Key)+" "+route.Path, func(t *testing.T) {
				wantFeatureKey, ok := protectedProductRoutes[route.Method+" "+route.Path]
				require.True(t, ok)
				require.Equal(t, wantFeatureKey, feature.Key)
			})
		}
	}
}

type protectedProductRouteFeature struct {
	method     string
	pattern    string
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

func protectedProductRouteFeatures() []protectedProductRouteFeature {
	return []protectedProductRouteFeature{
		{method: "GET", pattern: "/api/v1/accessorial-charges/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accessorial-charges/:accessorialChargeID/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/accessorial-charges/", featureKey: FeatureBilling},
		{method: "PUT", pattern: "/api/v1/accessorial-charges/:accessorialChargeID/", featureKey: FeatureBilling},
		{method: "PATCH", pattern: "/api/v1/accessorial-charges/:accessorialChargeID/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accessorial-charges/select-options/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accessorial-charges/select-options/:accessorialChargeID/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accounting-controls/", featureKey: FeatureAccounting},
		{method: "PUT", pattern: "/api/v1/accounting-controls/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/accounting/accounts-receivable/aging/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/accounting/accounts-receivable/open-items/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/accounting/accounts-receivable/customers/:customerID/ledger/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/accounting/accounts-receivable/customers/:customerID/statement/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/account-types/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/account-types/:accountTypeID", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/account-types/", featureKey: FeatureAccounting},
		{method: "PUT", pattern: "/api/v1/account-types/:accountTypeID/", featureKey: FeatureAccounting},
		{method: "PATCH", pattern: "/api/v1/account-types/:accountTypeID/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/account-types/bulk-update-status/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/account-types/select-options/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/account-types/select-options/:accountTypeID/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/analytics/", featureKey: FeatureAnalytics},
		{method: "GET", pattern: "/api/v1/api-keys/", featureKey: FeatureAPIKeys},
		{method: "POST", pattern: "/api/v1/api-keys/", featureKey: FeatureAPIKeys},
		{method: "GET", pattern: "/api/v1/api-keys/allowed-resources", featureKey: FeatureAPIKeys},
		{method: "GET", pattern: "/api/v1/api-keys/:apiKeyID/", featureKey: FeatureAPIKeys},
		{method: "PUT", pattern: "/api/v1/api-keys/:apiKeyID/", featureKey: FeatureAPIKeys},
		{method: "POST", pattern: "/api/v1/api-keys/:apiKeyID/rotate/", featureKey: FeatureAPIKeys},
		{method: "POST", pattern: "/api/v1/api-keys/:apiKeyID/revoke/", featureKey: FeatureAPIKeys},
		{method: "GET", pattern: "/api/v1/assignments/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/assignments/:assignmentID/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/assignments/check-worker-compliance/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipment-moves/:moveID/assignment/", featureKey: FeatureDispatch},
		{method: "PUT", pattern: "/api/v1/shipment-moves/:moveID/assignment/", featureKey: FeatureDispatch},
		{method: "DELETE", pattern: "/api/v1/shipment-moves/:moveID/assignment/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/audit-entries/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/audit-entries/:auditEntryID/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/audit-entries/resource/:resourceID/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/accounting/bank-receipt-batches/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accounting/bank-receipt-batches/:batchID/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/accounting/bank-receipt-batches/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accounting/bank-receipt-batches/select-options/sources/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accounting/bank-receipts/summary/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accounting/bank-receipts/exceptions/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accounting/bank-receipts/:receiptID/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accounting/bank-receipts/:receiptID/suggestions/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/accounting/bank-receipts/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/accounting/bank-receipts/:receiptID/match/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accounting/bank-receipt-work-items/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accounting/bank-receipt-work-items/:workItemID/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/accounting/bank-receipt-work-items/:workItemID/assign/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/accounting/bank-receipt-work-items/:workItemID/start-review/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/accounting/bank-receipt-work-items/:workItemID/resolve/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/accounting/bank-receipt-work-items/:workItemID/dismiss/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing-controls/", featureKey: FeatureBilling},
		{method: "PUT", pattern: "/api/v1/billing-controls/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing-queue/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing-queue/stats/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing-queue/:itemID/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/billing-queue/transfer/", featureKey: FeatureBilling},
		{method: "PUT", pattern: "/api/v1/billing-queue/:itemID/assign/", featureKey: FeatureBilling},
		{method: "PUT", pattern: "/api/v1/billing-queue/:itemID/status/", featureKey: FeatureBilling},
		{method: "PUT", pattern: "/api/v1/billing-queue/:itemID/charges/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing-queue/filter-presets/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/billing-queue/filter-presets/", featureKey: FeatureBilling},
		{method: "PUT", pattern: "/api/v1/billing-queue/filter-presets/:presetId/", featureKey: FeatureBilling},
		{method: "DELETE", pattern: "/api/v1/billing-queue/filter-presets/:presetId/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/commodities/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/commodities/:commodityID", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/commodities/", featureKey: FeatureCoreTMS},
		{method: "PUT", pattern: "/api/v1/commodities/:commodityID/", featureKey: FeatureCoreTMS},
		{method: "PATCH", pattern: "/api/v1/commodities/:commodityID/", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/commodities/bulk-update-status/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/commodities/select-options/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/commodities/select-options/:commodityID/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/customers/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/customers/:customerID", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/customers/:customerID/billing-profile/", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/customers/", featureKey: FeatureCoreTMS},
		{method: "PUT", pattern: "/api/v1/customers/:customerID/", featureKey: FeatureCoreTMS},
		{method: "PATCH", pattern: "/api/v1/customers/:customerID/", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/customers/bulk-update-status/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/customers/select-options/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/customers/select-options/:customerID/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/accounting/customer-payments/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accounting/customer-payments/:paymentID/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/accounting/customer-payments/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/accounting/customer-payments/:paymentID/apply/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/accounting/customer-payments/:paymentID/reverse/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/custom-fields/definitions/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/custom-fields/definitions/:definitionID/", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/custom-fields/definitions/", featureKey: FeatureCoreTMS},
		{method: "PUT", pattern: "/api/v1/custom-fields/definitions/:definitionID/", featureKey: FeatureCoreTMS},
		{method: "PATCH", pattern: "/api/v1/custom-fields/definitions/:definitionID/", featureKey: FeatureCoreTMS},
		{method: "DELETE", pattern: "/api/v1/custom-fields/definitions/:definitionID/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/custom-fields/resource-types/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/custom-fields/resources/:resourceType/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/admin/database-sessions/", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/admin/database-sessions/:pid/terminate/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/data-entry-controls/", featureKey: FeatureCoreTMS},
		{method: "PUT", pattern: "/api/v1/data-entry-controls/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/dispatch-controls/", featureKey: FeatureDispatch},
		{method: "PUT", pattern: "/api/v1/dispatch-controls/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/distance-overrides/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/distance-overrides/:distanceOverrideID", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/distance-overrides/", featureKey: FeatureDispatch},
		{method: "PUT", pattern: "/api/v1/distance-overrides/:distanceOverrideID/", featureKey: FeatureDispatch},
		{method: "PATCH", pattern: "/api/v1/distance-overrides/:distanceOverrideID/", featureKey: FeatureDispatch},
		{method: "DELETE", pattern: "/api/v1/distance-overrides/:distanceOverrideID/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/distance-profiles/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/distance-profiles/:distanceProfileID/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/distance-profiles/select-options/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/distance-profiles/select-options/:distanceProfileID", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/distance-profiles/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/distance-profiles/:distanceProfileID/set-default/", featureKey: FeatureDispatch},
		{method: "PUT", pattern: "/api/v1/distance-profiles/:distanceProfileID/", featureKey: FeatureDispatch},
		{method: "PATCH", pattern: "/api/v1/distance-profiles/:distanceProfileID/", featureKey: FeatureDispatch},
		{method: "DELETE", pattern: "/api/v1/distance-profiles/:distanceProfileID/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/document-controls/", featureKey: FeatureDocumentManagement},
		{method: "PUT", pattern: "/api/v1/document-controls/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/documents/uploads/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/uploads/active/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/uploads/:uploadSessionID/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/documents/uploads/:uploadSessionID/parts/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/documents/uploads/:uploadSessionID/complete/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/documents/uploads/:uploadSessionID/cancel/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/select-options/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/select-options/:documentID", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/:documentID/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/:documentID/content/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/:documentID/versions/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/documents/:documentID/restore/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/:documentID/shipment-draft/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/documents/:documentID/shipment-draft/reextract/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/documents/:documentID/attach-to-shipment/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/documents/:documentID/import-assistant/chat/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/documents/:documentID/import-assistant/chat-stream/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/:documentID/import-assistant/history/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/:documentID/download/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/:documentID/view/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/:documentID/preview/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/documents/upload/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/documents/upload-bulk/", featureKey: FeatureDocumentManagement},
		{method: "DELETE", pattern: "/api/v1/documents/:documentID/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/documents/bulk-delete/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/resource/:resourceType/:resourceID/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/documents/resource/:resourceType/:resourceID/packet-summary/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/admin/document-operations/:documentID/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/admin/document-operations/:documentID/reextract/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/admin/document-operations/:documentID/regenerate-preview/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/admin/document-operations/:documentID/resync-search/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/document-packet-rules/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/document-packet-rules/", featureKey: FeatureDocumentManagement},
		{method: "PUT", pattern: "/api/v1/document-packet-rules/:ruleID/", featureKey: FeatureDocumentManagement},
		{method: "DELETE", pattern: "/api/v1/document-packet-rules/:ruleID/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/document-parsing-rules/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/document-parsing-rules/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/document-parsing-rules/:ruleSetID/", featureKey: FeatureDocumentManagement},
		{method: "PUT", pattern: "/api/v1/document-parsing-rules/:ruleSetID/", featureKey: FeatureDocumentManagement},
		{method: "DELETE", pattern: "/api/v1/document-parsing-rules/:ruleSetID/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/document-parsing-rules/:ruleSetID/versions/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/document-parsing-rules/:ruleSetID/versions/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/document-parsing-rules/versions/:versionID/", featureKey: FeatureDocumentManagement},
		{method: "PUT", pattern: "/api/v1/document-parsing-rules/versions/:versionID/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/document-parsing-rules/versions/:versionID/publish/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/document-parsing-rules/versions/:versionID/simulate/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/document-parsing-rules/:ruleSetID/fixtures/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/document-parsing-rules/:ruleSetID/fixtures/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/document-parsing-rules/fixtures/:fixtureID/", featureKey: FeatureDocumentManagement},
		{method: "PUT", pattern: "/api/v1/document-parsing-rules/fixtures/:fixtureID/", featureKey: FeatureDocumentManagement},
		{method: "DELETE", pattern: "/api/v1/document-parsing-rules/fixtures/:fixtureID/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/document-types/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/document-types/:docTypeID/", featureKey: FeatureDocumentManagement},
		{method: "POST", pattern: "/api/v1/document-types/", featureKey: FeatureDocumentManagement},
		{method: "PUT", pattern: "/api/v1/document-types/:docTypeID/", featureKey: FeatureDocumentManagement},
		{method: "PATCH", pattern: "/api/v1/document-types/:docTypeID/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/document-types/select-options/", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/document-types/select-options/:docTypeID", featureKey: FeatureDocumentManagement},
		{method: "GET", pattern: "/api/v1/dot-hazmat-references/select-options/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/dot-hazmat-references/select-options/:dotHazmatReferenceID", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/edi/partners/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/partners/select-options/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/partners/internal-pairs/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/partners/:partnerID/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/partners/", featureKey: FeatureEDIIntegration},
		{method: "PUT", pattern: "/api/v1/edi/partners/:partnerID/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/partners/:partnerID/mapping-profile/", featureKey: FeatureEDIIntegration},
		{method: "PUT", pattern: "/api/v1/edi/partners/:partnerID/mapping-profile/", featureKey: FeatureEDIIntegration},
		{method: "DELETE", pattern: "/api/v1/edi/partners/:partnerID/mapping-profile/items/:mappingItemID/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/mapping-profiles/select-options/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/mapping-profiles/select-options/:profileID", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/mapping-profiles/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/mapping-profiles/:profileID/", featureKey: FeatureEDIIntegration},
		{method: "PUT", pattern: "/api/v1/edi/mapping-profiles/:profileID/items/", featureKey: FeatureEDIIntegration},
		{method: "DELETE", pattern: "/api/v1/edi/mapping-profiles/:profileID/items/:mappingItemID/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/connections/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/connections/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/connections/:connectionID/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/connections/:connectionID/accept/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/connections/:connectionID/reject/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/connections/:connectionID/suspend/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/connections/:connectionID/revoke/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/communication-profiles/select-options/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/communication-profiles/select-options/:profileID", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/communication-profiles/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/communication-profiles/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/communication-profiles/:profileID/", featureKey: FeatureEDIIntegration},
		{method: "PUT", pattern: "/api/v1/edi/communication-profiles/:profileID/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/catalog/document-types/select-options/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/catalog/document-types/select-options/:documentTypeID", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/catalog/document-types/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/catalog/source-context/schemas/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/catalog/source-context/schemas/:schemaID/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/catalog/source-context/schemas/:schemaID/fields/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/catalog/source-context/fields/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/catalog/source-context/fields/select-options/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/catalog/partner-settings/schemas/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/catalog/partner-settings/schemas/:schemaID/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/catalog/partner-settings/schemas/:schemaID/fields/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/catalog/partner-settings/fields/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/catalog/partner-settings/fields/select-options/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/catalog/partner-settings/validate/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/templates/select-options/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/templates/select-options/:templateID", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/templates/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/templates/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/templates/:templateID/", featureKey: FeatureEDIIntegration},
		{method: "PUT", pattern: "/api/v1/edi/templates/:templateID/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/templates/:templateID/draft/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/templates/:templateID/versions/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/templates/:templateID/versions/:versionID/", featureKey: FeatureEDIIntegration},
		{method: "PUT", pattern: "/api/v1/edi/templates/:templateID/versions/:versionID/", featureKey: FeatureEDIIntegration},
		{method: "PUT", pattern: "/api/v1/edi/templates/:templateID/versions/:versionID/segments/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/templates/:templateID/versions/:versionID/script-libraries/", featureKey: FeatureEDIIntegration},
		{method: "PUT", pattern: "/api/v1/edi/templates/:templateID/versions/:versionID/script-libraries/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/templates/:templateID/versions/:versionID/validate/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/templates/:templateID/versions/:versionID/certify/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/templates/:templateID/versions/:versionID/activate/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/templates/:templateID/versions/:versionID/archive/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/templates/:templateID/versions/:versionID/rollback/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/document-profiles/select-options/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/document-profiles/select-options/:profileID", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/document-profiles/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/document-profiles/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/document-profiles/:profileID/", featureKey: FeatureEDIIntegration},
		{method: "PUT", pattern: "/api/v1/edi/document-profiles/:profileID/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/documents/preview/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/documents/generate/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/messages/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/messages/:messageID/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/messages/:messageID/inspect/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/x12/inspect/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/test-cases/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/test-cases/:testCaseID/preview/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/load-tenders/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/transfers/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/transfers/:transferID/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/transfers/:transferID/mapping-preview/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/transfers/:transferID/approve/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/transfers/:transferID/reject/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/transfers/:transferID/cancel/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/transfers/:transferID/expire/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/shipment-links/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/shipment-links/:linkID/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/transfer-changes/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/edi/transfer-changes/:changeID/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/transfer-changes/:changeID/apply/", featureKey: FeatureEDIIntegration},
		{method: "POST", pattern: "/api/v1/edi/transfer-changes/:changeID/reject/", featureKey: FeatureEDIIntegration},
		{method: "GET", pattern: "/api/v1/equipment-manufacturers/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/equipment-manufacturers/:equipManufacturerID/", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/equipment-manufacturers/", featureKey: FeatureFleetMaintenance},
		{method: "PUT", pattern: "/api/v1/equipment-manufacturers/:equipManufacturerID/", featureKey: FeatureFleetMaintenance},
		{method: "PATCH", pattern: "/api/v1/equipment-manufacturers/:equipManufacturerID/", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/equipment-manufacturers/bulk-update-status/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/equipment-manufacturers/select-options/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/equipment-manufacturers/select-options/:equipManufacturerID", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/equipment-types/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/equipment-types/:equipTypeID/", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/equipment-types/", featureKey: FeatureFleetMaintenance},
		{method: "PUT", pattern: "/api/v1/equipment-types/:equipTypeID/", featureKey: FeatureFleetMaintenance},
		{method: "PATCH", pattern: "/api/v1/equipment-types/:equipTypeID/", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/equipment-types/bulk-update-status/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/equipment-types/select-options/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/equipment-types/select-options/:equipTypeID", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/exchange-rates/convert", featureKey: FeatureExchangeRateIntegration},
		{method: "GET", pattern: "/api/v1/exchange-rates/latest", featureKey: FeatureExchangeRateIntegration},
		{method: "POST", pattern: "/api/v1/exchange-rates/refresh", featureKey: FeatureExchangeRateIntegration},
		{method: "GET", pattern: "/api/v1/fiscal-periods/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/fiscal-periods/:fiscalPeriodID", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/fiscal-periods/", featureKey: FeatureAccounting},
		{method: "PUT", pattern: "/api/v1/fiscal-periods/:fiscalPeriodID/", featureKey: FeatureAccounting},
		{method: "PATCH", pattern: "/api/v1/fiscal-periods/:fiscalPeriodID/", featureKey: FeatureAccounting},
		{method: "DELETE", pattern: "/api/v1/fiscal-periods/:fiscalPeriodID/", featureKey: FeatureAccounting},
		{method: "PUT", pattern: "/api/v1/fiscal-periods/:fiscalPeriodID/close/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/fiscal-periods/:fiscalPeriodID/close-blockers/", featureKey: FeatureAccounting},
		{method: "PUT", pattern: "/api/v1/fiscal-periods/:fiscalPeriodID/reopen/", featureKey: FeatureAccounting},
		{method: "PUT", pattern: "/api/v1/fiscal-periods/:fiscalPeriodID/lock/", featureKey: FeatureAccounting},
		{method: "PUT", pattern: "/api/v1/fiscal-periods/:fiscalPeriodID/unlock/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/fiscal-years/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/fiscal-years/:fiscalYearID", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/fiscal-years/", featureKey: FeatureAccounting},
		{method: "PUT", pattern: "/api/v1/fiscal-years/:fiscalYearID/", featureKey: FeatureAccounting},
		{method: "PATCH", pattern: "/api/v1/fiscal-years/:fiscalYearID/", featureKey: FeatureAccounting},
		{method: "DELETE", pattern: "/api/v1/fiscal-years/:fiscalYearID/", featureKey: FeatureAccounting},
		{method: "PUT", pattern: "/api/v1/fiscal-years/:fiscalYearID/close/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/fiscal-years/:fiscalYearID/close-blockers/", featureKey: FeatureAccounting},
		{method: "PUT", pattern: "/api/v1/fiscal-years/:fiscalYearID/activate/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/fleet-codes/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/fleet-codes/:fleetCodeID", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/fleet-codes/", featureKey: FeatureFleetMaintenance},
		{method: "PUT", pattern: "/api/v1/fleet-codes/:fleetCodeID", featureKey: FeatureFleetMaintenance},
		{method: "PATCH", pattern: "/api/v1/fleet-codes/:fleetCodeID", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/fleet-codes/select-options/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/fleet-codes/select-options/:fleetCodeID", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/formula-templates/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/formula-templates/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/formula-templates/bulk-update-status", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/formula-templates/test", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/formula-templates/duplicate", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/formula-templates/:templateID/", featureKey: FeatureBilling},
		{method: "PUT", pattern: "/api/v1/formula-templates/:templateID/", featureKey: FeatureBilling},
		{method: "PATCH", pattern: "/api/v1/formula-templates/:templateID/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/formula-templates/:templateID/usage", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/formula-templates/:templateID/versions", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/formula-templates/:templateID/versions/:versionNumber", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/formula-templates/:templateID/versions", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/formula-templates/:templateID/rollback", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/formula-templates/:templateID/fork", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/formula-templates/:templateID/compare", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/formula-templates/:templateID/lineage", featureKey: FeatureBilling},
		{method: "PATCH", pattern: "/api/v1/formula-templates/:templateID/versions/:versionNumber/tags", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/formula-templates/select-options/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/formula-templates/select-options/:templateID", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/gl-accounts/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/gl-accounts/:glAccountID", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/gl-accounts/", featureKey: FeatureAccounting},
		{method: "PUT", pattern: "/api/v1/gl-accounts/:glAccountID/", featureKey: FeatureAccounting},
		{method: "PATCH", pattern: "/api/v1/gl-accounts/:glAccountID/", featureKey: FeatureAccounting},
		{method: "DELETE", pattern: "/api/v1/gl-accounts/:glAccountID/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/gl-accounts/bulk-update-status/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/gl-accounts/select-options/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/gl-accounts/select-options/:glAccountID/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/accounting/trial-balance/:fiscalPeriodID/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/accounting/statements/income-statement/:fiscalPeriodID/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/accounting/statements/balance-sheet/:fiscalPeriodID/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/google-maps/autocomplete/", featureKey: FeatureGoogleMapsIntegration},
		{method: "GET", pattern: "/api/v1/hazardous-materials/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/hazardous-materials/:hazardousMaterialID", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/hazardous-materials/", featureKey: FeatureCoreTMS},
		{method: "PUT", pattern: "/api/v1/hazardous-materials/:hazardousMaterialID/", featureKey: FeatureCoreTMS},
		{method: "PATCH", pattern: "/api/v1/hazardous-materials/:hazardousMaterialID/", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/hazardous-materials/bulk-update-status/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/hazardous-materials/select-options/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/hazardous-materials/select-options/:hazardousMaterialID/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/hazmat-segregation-rules/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/hazmat-segregation-rules/:hazmatSegregationRuleID", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/hazmat-segregation-rules/", featureKey: FeatureCoreTMS},
		{method: "PUT", pattern: "/api/v1/hazmat-segregation-rules/:hazmatSegregationRuleID/", featureKey: FeatureCoreTMS},
		{method: "PATCH", pattern: "/api/v1/hazmat-segregation-rules/:hazmatSegregationRuleID/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/hold-reasons/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/hold-reasons/:holdReasonID", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/hold-reasons/", featureKey: FeatureDispatch},
		{method: "PUT", pattern: "/api/v1/hold-reasons/:holdReasonID/", featureKey: FeatureDispatch},
		{method: "PATCH", pattern: "/api/v1/hold-reasons/:holdReasonID/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/hold-reasons/select-options/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/hold-reasons/select-options/:holdReasonID/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/integrations/catalog/", featureKey: FeatureSamsaraIntegration},
		{method: "GET", pattern: "/api/v1/integrations/:type/config/", featureKey: FeatureSamsaraIntegration},
		{method: "PUT", pattern: "/api/v1/integrations/:type/config/", featureKey: FeatureSamsaraIntegration},
		{method: "POST", pattern: "/api/v1/integrations/:type/test-connection/", featureKey: FeatureSamsaraIntegration},
		{method: "GET", pattern: "/api/v1/integrations/:type/runtime-config/", featureKey: FeatureSamsaraIntegration},
		{method: "GET", pattern: "/api/v1/integrations/samsara/workers/sync/readiness/", featureKey: FeatureSamsaraIntegration},
		{method: "GET", pattern: "/api/v1/integrations/samsara/workers/sync/drift/", featureKey: FeatureSamsaraIntegration},
		{method: "POST", pattern: "/api/v1/integrations/samsara/workers/sync/drift/detect/", featureKey: FeatureSamsaraIntegration},
		{method: "POST", pattern: "/api/v1/integrations/samsara/workers/sync/drift/repair/", featureKey: FeatureSamsaraIntegration},
		{method: "POST", pattern: "/api/v1/integrations/samsara/workers/sync/", featureKey: FeatureSamsaraIntegration},
		{method: "GET", pattern: "/api/v1/integrations/samsara/workers/sync/:workflowID/", featureKey: FeatureSamsaraIntegration},
		{method: "GET", pattern: "/api/v1/invoice-adjustment-controls/", featureKey: FeatureAccounting},
		{method: "PUT", pattern: "/api/v1/invoice-adjustment-controls/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/billing/invoice-adjustments/drafts/", featureKey: FeatureBilling},
		{method: "PATCH", pattern: "/api/v1/billing/invoice-adjustments/drafts/:adjustmentID/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/billing/invoice-adjustments/drafts/:adjustmentID/preview/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/billing/invoice-adjustments/drafts/:adjustmentID/submit/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/billing/invoice-adjustments/preview/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/billing/invoice-adjustments/submit/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/billing/invoice-adjustments/bulk-preview/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/billing/invoice-adjustments/bulk-submit/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing/invoice-adjustments/summary/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing/invoice-adjustments/approvals/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing/invoice-adjustments/reconciliation-exceptions/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing/invoice-adjustments/batches/:batchID/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing/invoice-adjustments/batches/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing/invoice-adjustments/correction-groups/:correctionGroupID/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing/invoice-adjustments/:adjustmentID/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/billing/invoice-adjustments/:adjustmentID/approve/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/billing/invoice-adjustments/:adjustmentID/reject/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing/invoice-adjustments/:adjustmentID/lineage/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing/invoices/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/billing/invoices/:invoiceID/", featureKey: FeatureBilling},
		{method: "POST", pattern: "/api/v1/billing/invoices/:invoiceID/post/", featureKey: FeatureBilling},
		{method: "GET", pattern: "/api/v1/accounting/journal-entries/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/accounting/journal-entries/:journalEntryID/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/accounting/journal-entries/source/:sourceObjectType/:sourceObjectID/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/accounting/journal-reversals/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/accounting/journal-reversals/:reversalID/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/accounting/journal-reversals/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/accounting/journal-reversals/:reversalID/approve/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/accounting/journal-reversals/:reversalID/post/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/accounting/journal-reversals/:reversalID/reject/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/accounting/journal-reversals/:reversalID/cancel/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/location-categories/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/location-categories/:locationCategoryID/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/location-categories/", featureKey: FeatureDispatch},
		{method: "PUT", pattern: "/api/v1/location-categories/:locationCategoryID/", featureKey: FeatureDispatch},
		{method: "PATCH", pattern: "/api/v1/location-categories/:locationCategoryID/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/location-categories/select-options/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/location-categories/select-options/:locationCategoryID", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/locations/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/locations/:locationID", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/locations/", featureKey: FeatureDispatch},
		{method: "PUT", pattern: "/api/v1/locations/:locationID/", featureKey: FeatureDispatch},
		{method: "PATCH", pattern: "/api/v1/locations/:locationID/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/locations/bulk-update-status/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/locations/select-options/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/locations/select-options/:locationID", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/accounting/manual-journals/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/accounting/manual-journals/:requestID/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/accounting/manual-journals/drafts/", featureKey: FeatureAccounting},
		{method: "PUT", pattern: "/api/v1/accounting/manual-journals/drafts/:requestID/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/accounting/manual-journals/:requestID/submit/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/accounting/manual-journals/:requestID/approve/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/accounting/manual-journals/:requestID/post/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/accounting/manual-journals/:requestID/reject/", featureKey: FeatureAccounting},
		{method: "POST", pattern: "/api/v1/accounting/manual-journals/:requestID/cancel/", featureKey: FeatureAccounting},
		{method: "GET", pattern: "/api/v1/organizations/select-options/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/permissions/resources", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/permissions/operations", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/role-assignments/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/role-assignments/:roleAssignmentID", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/role-assignments/select-options/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/role-assignments/select-options/:roleAssignmentID", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/roles/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/roles/:roleID", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/roles/", featureKey: FeatureCoreTMS},
		{method: "PUT", pattern: "/api/v1/roles/:roleID", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/roles/:roleID/impact", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/roles/:roleID/permissions", featureKey: FeatureCoreTMS},
		{method: "PUT", pattern: "/api/v1/roles/:roleID/permissions/:permID", featureKey: FeatureCoreTMS},
		{method: "DELETE", pattern: "/api/v1/roles/:roleID/permissions/:permID", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/roles/:roleID/assignments", featureKey: FeatureCoreTMS},
		{method: "DELETE", pattern: "/api/v1/roles/assignments/:assignmentID", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/roles/select-options/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/roles/select-options/:roleID", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/search/global/", featureKey: FeatureGlobalSearch},
		{method: "GET", pattern: "/api/v1/sequence-configs/", featureKey: FeatureCoreTMS},
		{method: "PUT", pattern: "/api/v1/sequence-configs/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/service-types/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/service-types/:serviceTypeID", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/service-types/", featureKey: FeatureCoreTMS},
		{method: "PUT", pattern: "/api/v1/service-types/:serviceTypeID/", featureKey: FeatureCoreTMS},
		{method: "PATCH", pattern: "/api/v1/service-types/:serviceTypeID/", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/service-types/bulk-update-status/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/service-types/select-options/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/service-types/select-options/:serviceTypeID/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/shipment-controls/", featureKey: FeatureDispatch},
		{method: "PUT", pattern: "/api/v1/shipment-controls/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipment-events/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipments/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipments/ui-policy/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipments/:shipmentID/billing-readiness/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipments/:shipmentID", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/calculate-totals/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/duplicate/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/check-for-duplicate-bols/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/check-hazmat-segregation/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/loading-optimization/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/previous-rates/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipments/delayed/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipments/unassigned/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/delay/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipments/auto-cancel/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/auto-cancel/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipments/:shipmentID/comments/count/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipments/:shipmentID/holds/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipments/:shipmentID/holds/:holdID/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/:shipmentID/holds/", featureKey: FeatureDispatch},
		{method: "PUT", pattern: "/api/v1/shipments/:shipmentID/holds/:holdID/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/:shipmentID/holds/:holdID/release/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipments/:shipmentID/comments/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/:shipmentID/comments/", featureKey: FeatureDispatch},
		{method: "PUT", pattern: "/api/v1/shipments/:shipmentID/comments/:commentID/", featureKey: FeatureDispatch},
		{method: "DELETE", pattern: "/api/v1/shipments/:shipmentID/comments/:commentID/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/:shipmentID/cancel/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/:shipmentID/uncancel/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/:shipmentID/transfer-ownership/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/:shipmentID/transfer-to-billing/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipments/bulk-transfer-to-billing/", featureKey: FeatureDispatch},
		{method: "PUT", pattern: "/api/v1/shipments/:shipmentID/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipment-moves/bulk-update-status/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipment-moves/:moveID/update-status/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipment-moves/:moveID/split/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipment-types/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipment-types/:shipmentTypeID", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipment-types/", featureKey: FeatureDispatch},
		{method: "PUT", pattern: "/api/v1/shipment-types/:shipmentTypeID/", featureKey: FeatureDispatch},
		{method: "PATCH", pattern: "/api/v1/shipment-types/:shipmentTypeID/", featureKey: FeatureDispatch},
		{method: "POST", pattern: "/api/v1/shipment-types/bulk-update-status/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipment-types/select-options/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/shipment-types/select-options/:shipmentTypeID/", featureKey: FeatureDispatch},
		{method: "GET", pattern: "/api/v1/tca/allowlisted-tables/", featureKey: FeatureTableChangeAlerts},
		{method: "GET", pattern: "/api/v1/tca/subscriptions/", featureKey: FeatureTableChangeAlerts},
		{method: "GET", pattern: "/api/v1/tca/subscriptions/:id", featureKey: FeatureTableChangeAlerts},
		{method: "POST", pattern: "/api/v1/tca/subscriptions/", featureKey: FeatureTableChangeAlerts},
		{method: "PUT", pattern: "/api/v1/tca/subscriptions/:id", featureKey: FeatureTableChangeAlerts},
		{method: "DELETE", pattern: "/api/v1/tca/subscriptions/:id", featureKey: FeatureTableChangeAlerts},
		{method: "PATCH", pattern: "/api/v1/tca/subscriptions/:id/pause", featureKey: FeatureTableChangeAlerts},
		{method: "PATCH", pattern: "/api/v1/tca/subscriptions/:id/resume", featureKey: FeatureTableChangeAlerts},
		{method: "GET", pattern: "/api/v1/table-configurations/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/table-configurations/default", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/table-configurations/:id", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/table-configurations/", featureKey: FeatureCoreTMS},
		{method: "PUT", pattern: "/api/v1/table-configurations/:id", featureKey: FeatureCoreTMS},
		{method: "PATCH", pattern: "/api/v1/table-configurations/:id", featureKey: FeatureCoreTMS},
		{method: "DELETE", pattern: "/api/v1/table-configurations/:id", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/table-configurations/:id/set-default", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/tractors/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/tractors/:tractorID/", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/tractors/", featureKey: FeatureFleetMaintenance},
		{method: "PUT", pattern: "/api/v1/tractors/:tractorID/", featureKey: FeatureFleetMaintenance},
		{method: "PATCH", pattern: "/api/v1/tractors/:tractorID/", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/tractors/bulk-update-status/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/tractors/select-options/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/tractors/select-options/:tractorID", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/trailers/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/trailers/:trailerID/", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/trailers/", featureKey: FeatureFleetMaintenance},
		{method: "PUT", pattern: "/api/v1/trailers/:trailerID/", featureKey: FeatureFleetMaintenance},
		{method: "PATCH", pattern: "/api/v1/trailers/:trailerID/", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/trailers/bulk-update-status/", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/trailers/:trailerID/locate/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/trailers/select-options/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/trailers/select-options/:trailerID", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/users/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/users/:userID/", featureKey: FeatureCoreTMS},
		{method: "PUT", pattern: "/api/v1/users/:userID/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/users/:userID/role-assignments/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/users/:userID/effective-permissions/", featureKey: FeatureCoreTMS},
		{method: "PATCH", pattern: "/api/v1/users/:userID/", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/users/bulk-update-status/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/users/:userID/organization-memberships/", featureKey: FeatureCoreTMS},
		{method: "PUT", pattern: "/api/v1/users/:userID/organization-memberships/", featureKey: FeatureCoreTMS},
		{method: "POST", pattern: "/api/v1/users/:userID/permissions/simulate/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/users/:userID/profile-picture/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/users/select-options/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/users/select-options/:userID", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/weather-alerts/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/weather-alerts/:alertID/", featureKey: FeatureCoreTMS},
		{method: "GET", pattern: "/api/v1/workers/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/workers/:workerID/", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/workers/", featureKey: FeatureFleetMaintenance},
		{method: "PUT", pattern: "/api/v1/workers/:workerID/", featureKey: FeatureFleetMaintenance},
		{method: "PATCH", pattern: "/api/v1/workers/:workerID/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/workers/select-options/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/workers/select-options/:workerID", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/worker-pto/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/worker-pto/upcoming/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/worker-pto/:ptoID/", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/worker-pto/", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/worker-pto/:ptoID/approve/", featureKey: FeatureFleetMaintenance},
		{method: "POST", pattern: "/api/v1/worker-pto/:ptoID/reject/", featureKey: FeatureFleetMaintenance},
		{method: "GET", pattern: "/api/v1/worker-pto/chart/", featureKey: FeatureFleetMaintenance},
	}
}
