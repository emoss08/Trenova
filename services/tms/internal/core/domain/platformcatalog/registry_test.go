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
			routePattern: "/api/v1/organizations/:id",
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
			method:       "PATCH",
			routePattern: "/api/v1/billing-queue/:id",
			want:         FeatureBilling,
		},
		{
			name:         "document route",
			method:       "POST",
			routePattern: "/api/v1/documents/upload",
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

func TestStaticProvider_ProtectedRoutePrefixesCovered(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(RegistryParams{
		Providers: []CatalogProvider{NewStaticProvider()},
	})
	require.NoError(t, err)

	protectedPrefixes := []string{
		"/api/v1/accessorial-charges/",
		"/api/v1/account-types/",
		"/api/v1/accounting-controls/",
		"/api/v1/accounting/accounts-receivable/",
		"/api/v1/accounting/bank-receipt-batches/",
		"/api/v1/accounting/bank-receipt-work-items/",
		"/api/v1/accounting/bank-receipts/",
		"/api/v1/accounting/customer-payments/",
		"/api/v1/accounting/journal-entries/",
		"/api/v1/accounting/journal-reversals/",
		"/api/v1/accounting/manual-journals/",
		"/api/v1/accounting/statements/",
		"/api/v1/accounting/trial-balance/",
		"/api/v1/admin/database-sessions/",
		"/api/v1/admin/document-operations/",
		"/api/v1/analytics/",
		"/api/v1/api-keys/",
		"/api/v1/assignments/",
		"/api/v1/audit-entries/",
		"/api/v1/billing-queue/",
		"/api/v1/billing/invoice-adjustments/",
		"/api/v1/billing/invoices/",
		"/api/v1/billing-controls/",
		"/api/v1/commodities/",
		"/api/v1/custom-fields/",
		"/api/v1/customers/",
		"/api/v1/data-entry-controls/",
		"/api/v1/dispatch-controls/",
		"/api/v1/distance-overrides/",
		"/api/v1/document-controls/",
		"/api/v1/document-packet-rules/",
		"/api/v1/document-parsing-rules/",
		"/api/v1/document-types/",
		"/api/v1/documents/",
		"/api/v1/dot-hazmat-references/",
		"/api/v1/edi/",
		"/api/v1/equipment-manufacturers/",
		"/api/v1/equipment-types/",
		"/api/v1/exchange-rates/",
		"/api/v1/fiscal-periods/",
		"/api/v1/fiscal-years/",
		"/api/v1/fleet-codes/",
		"/api/v1/formula-templates/",
		"/api/v1/gl-accounts/",
		"/api/v1/google-maps/",
		"/api/v1/hazardous-materials/",
		"/api/v1/hazmat-segregation-rules/",
		"/api/v1/hold-reasons/",
		"/api/v1/integrations/",
		"/api/v1/invoice-adjustment-controls/",
		"/api/v1/location-categories/",
		"/api/v1/locations/",
		"/api/v1/me/",
		"/api/v1/notifications/",
		"/api/v1/organizations/",
		"/api/v1/page-favorites/",
		"/api/v1/permissions/",
		"/api/v1/platform-catalog/",
		"/api/v1/realtime/",
		"/api/v1/role-assignments/",
		"/api/v1/roles/",
		"/api/v1/search/",
		"/api/v1/sequence-configs/",
		"/api/v1/service-types/",
		"/api/v1/shipment-controls/",
		"/api/v1/shipment-events/",
		"/api/v1/shipment-moves/",
		"/api/v1/shipment-types/",
		"/api/v1/shipments/",
		"/api/v1/table-configurations/",
		"/api/v1/tca/",
		"/api/v1/tractors/",
		"/api/v1/trailers/",
		"/api/v1/us-states/",
		"/api/v1/users/",
		"/api/v1/weather-alerts/",
		"/api/v1/worker-pto/",
		"/api/v1/workers/",
	}

	for _, prefix := range protectedPrefixes {
		t.Run(prefix, func(t *testing.T) {
			_, ok := registry.FeatureForRoute("GET", prefix)
			require.True(t, ok)
		})
	}
}
