package platformcatalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()

	registry, err := NewRegistry(RegistryParams{
		Providers: []CatalogProvider{NewStaticProvider()},
	})
	require.NoError(t, err)

	return registry
}

func TestRegistry_PacksCoverPricedCatalog(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	wantPacks := []PackKey{
		PackCompliance,
		PackDispatchIntel,
		PackDriverDash,
		PackProfessional,
		PackSettlement,
		PackVisibility,
		PackWorkforce,
	}

	packs := registry.ListPacks()
	require.Len(t, packs, len(wantPacks))
	for i, pack := range packs {
		require.Equal(t, wantPacks[i], pack.Key)
		require.NotEmpty(t, pack.Name)
		require.NotEmpty(t, pack.Description)
		require.NotEmpty(t, pack.Features)
	}
}

func TestRegistry_WorkforcePackIsSelfContained(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	pack, ok := registry.GetPack(PackWorkforce)
	require.True(t, ok)
	require.True(t, pack.Standalone)

	closure, ok := registry.PackFeatureClosure(PackWorkforce)
	require.True(t, ok)
	require.Len(t, closure, len(pack.Features))

	for _, featureKey := range closure {
		require.NotEqual(t, FeatureCoreTMS, featureKey)
		require.NotEqual(t, FeatureDispatch, featureKey)
		require.NotEqual(t, FeatureBilling, featureKey)
		require.NotEqual(t, FeatureAccounting, featureKey)
		require.NotEqual(t, FeatureFleetMaintenance, featureKey)
		require.NotEqual(t, FeatureSettlement, featureKey)
	}
}

func TestRegistry_WorkforceFeaturesDoNotRequireTMS(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	for _, feature := range registry.FeaturesByProduct(ProductWorkforce) {
		t.Run(string(feature.Key), func(t *testing.T) {
			closure := make(map[FeatureKey]struct{})
			registry.collectRequiredFeatures(feature.Key, closure)

			for required := range closure {
				requiredFeature, ok := registry.GetFeature(required)
				require.True(t, ok)
				require.NotEqual(t, ProductTMS, requiredFeature.ProductKey)
			}
		})
	}
}

func TestRegistry_PackValidationRejectsMissingFeature(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		products: map[ProductKey]Product{},
		features: map[FeatureKey]Feature{},
		meters:   map[MeterKey]Meter{},
		packs: map[PackKey]Pack{
			"broken": {Key: "broken", Features: []FeatureKey{"missing.feature"}},
		},
	}

	err := registry.validatePacks()
	require.ErrorContains(t, err, "references missing feature")
}

func TestRegistry_PackValidationRejectsEmptyPack(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		packs: map[PackKey]Pack{"empty": {Key: "empty"}},
	}

	err := registry.validatePacks()
	require.ErrorContains(t, err, "must reference at least one feature")
}

func TestRegistry_PackValidationRejectsDuplicateFeature(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		features: map[FeatureKey]Feature{FeatureCoreTMS: {Key: FeatureCoreTMS}},
		packs: map[PackKey]Pack{
			"dupe": {Key: "dupe", Features: []FeatureKey{FeatureCoreTMS, FeatureCoreTMS}},
		},
	}

	err := registry.validatePacks()
	require.ErrorContains(t, err, "more than once")
}

func TestRegistry_StandalonePackMustIncludeRequiredFeatures(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		features: map[FeatureKey]Feature{
			"a.core": {Key: "a.core"},
			"a.addon": {
				Key:              "a.addon",
				RequiresFeatures: []FeatureKey{"a.core"},
			},
		},
		packs: map[PackKey]Pack{
			"partial": {Key: "partial", Standalone: true, Features: []FeatureKey{"a.addon"}},
		},
	}

	err := registry.validatePacks()
	require.ErrorContains(t, err, "requires missing feature")
}

func TestRegistry_AuthorizingFeaturesIncludesLegacyGrant(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	require.Equal(
		t,
		[]FeatureKey{FeatureWorkforceCore, FeatureFleetMaintenance},
		registry.AuthorizingFeatures(FeatureWorkforceCore),
	)
	require.Equal(
		t,
		[]FeatureKey{FeatureDispatch},
		registry.AuthorizingFeatures(FeatureDispatch),
	)
	require.Equal(
		t,
		[]FeatureKey{FeatureKey("not.a.feature")},
		registry.AuthorizingFeatures(FeatureKey("not.a.feature")),
	)
}

func TestRegistry_LegacyGrantValidationRejectsSelfReference(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		features: map[FeatureKey]Feature{FeatureCoreTMS: {Key: FeatureCoreTMS}},
	}

	err := registry.validateLegacyGrantingFeatures(
		FeatureCoreTMS,
		[]FeatureKey{FeatureCoreTMS},
	)
	require.ErrorContains(t, err, "cannot be legacy granted by itself")
}

func TestRegistry_LegacyGrantValidationRejectsMissingFeature(t *testing.T) {
	t.Parallel()

	registry := &Registry{features: map[FeatureKey]Feature{}}

	err := registry.validateLegacyGrantingFeatures(
		FeatureWorkforceCore,
		[]FeatureKey{FeatureKey("missing")},
	)
	require.ErrorContains(t, err, "legacy granted by missing feature")
}
