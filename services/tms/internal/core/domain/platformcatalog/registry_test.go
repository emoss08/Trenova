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
