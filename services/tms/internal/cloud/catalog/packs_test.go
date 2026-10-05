package catalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/stretchr/testify/require"
)

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()

	registry, err := NewRegistry(RegistryParams{
		Providers: []platformcatalog.CatalogProvider{NewStaticProvider()},
	})
	require.NoError(t, err)

	return registry
}

func TestRegistry_PacksCoverPricedCatalog(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	wantPacks := []platformcatalog.PackKey{
		PackCompliance,
		PackDispatchIntel,
		PackDocumentAI,
		PackDriverDash,
		PackEDI,
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
		require.NotEqual(t, platformcatalog.FeatureCoreTMS, featureKey)
		require.NotEqual(t, platformcatalog.FeatureDispatch, featureKey)
		require.NotEqual(t, platformcatalog.FeatureBilling, featureKey)
		require.NotEqual(t, platformcatalog.FeatureAccounting, featureKey)
		require.NotEqual(t, platformcatalog.FeatureFleetMaintenance, featureKey)
		require.NotEqual(t, platformcatalog.FeatureSettlement, featureKey)
	}
}

func TestRegistry_WorkforceFeaturesDoNotRequireTMS(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	for _, feature := range registry.FeaturesByProduct(platformcatalog.ProductWorkforce) {
		t.Run(string(feature.Key), func(t *testing.T) {
			closure := make(map[platformcatalog.FeatureKey]struct{})
			registry.collectRequiredFeatures(feature.Key, closure)

			for required := range closure {
				requiredFeature, ok := registry.GetFeature(required)
				require.True(t, ok)
				require.NotEqual(t, platformcatalog.ProductTMS, requiredFeature.ProductKey)
			}
		})
	}
}

func TestRegistry_PackValidationRejectsMissingFeature(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		products: map[platformcatalog.ProductKey]platformcatalog.Product{},
		features: map[platformcatalog.FeatureKey]platformcatalog.Feature{},
		meters:   map[platformcatalog.MeterKey]platformcatalog.Meter{},
		packs: map[platformcatalog.PackKey]platformcatalog.Pack{
			"broken": {Key: "broken", Features: []platformcatalog.FeatureKey{"missing.feature"}},
		},
	}

	err := registry.validatePacks()
	require.ErrorContains(t, err, "references missing feature")
}

func TestRegistry_PackValidationRejectsEmptyPack(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		packs: map[platformcatalog.PackKey]platformcatalog.Pack{"empty": {Key: "empty"}},
	}

	err := registry.validatePacks()
	require.ErrorContains(t, err, "must reference at least one feature")
}

func TestRegistry_PackValidationRejectsDuplicateFeature(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		features: map[platformcatalog.FeatureKey]platformcatalog.Feature{platformcatalog.FeatureCoreTMS: {Key: platformcatalog.FeatureCoreTMS}},
		packs: map[platformcatalog.PackKey]platformcatalog.Pack{
			"dupe": {Key: "dupe", Features: []platformcatalog.FeatureKey{platformcatalog.FeatureCoreTMS, platformcatalog.FeatureCoreTMS}},
		},
	}

	err := registry.validatePacks()
	require.ErrorContains(t, err, "more than once")
}

func TestRegistry_StandalonePackMustIncludeRequiredFeatures(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		features: map[platformcatalog.FeatureKey]platformcatalog.Feature{
			"a.core": {Key: "a.core"},
			"a.addon": {
				Key:              "a.addon",
				RequiresFeatures: []platformcatalog.FeatureKey{"a.core"},
			},
		},
		packs: map[platformcatalog.PackKey]platformcatalog.Pack{
			"partial": {Key: "partial", Standalone: true, Features: []platformcatalog.FeatureKey{"a.addon"}},
		},
	}

	err := registry.validatePacks()
	require.ErrorContains(t, err, "which requires feature")
}

func TestRegistry_AddOnPackMaySatisfyRequirementsThroughRequiredPack(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		features: map[platformcatalog.FeatureKey]platformcatalog.Feature{
			"a.core":  {Key: "a.core"},
			"a.addon": {Key: "a.addon", RequiresFeatures: []platformcatalog.FeatureKey{"a.core"}},
		},
		packs: map[platformcatalog.PackKey]platformcatalog.Pack{
			"base": {Key: "base", Features: []platformcatalog.FeatureKey{"a.core"}},
			"addon": {
				Key:           "addon",
				RequiresPacks: []platformcatalog.PackKey{"base"},
				Features:      []platformcatalog.FeatureKey{"a.addon"},
			},
		},
	}

	require.NoError(t, registry.validatePacks())
}

func TestRegistry_PackValidationRejectsMissingRequiredPack(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		features: map[platformcatalog.FeatureKey]platformcatalog.Feature{"a.core": {Key: "a.core"}},
		packs: map[platformcatalog.PackKey]platformcatalog.Pack{
			"addon": {
				Key:           "addon",
				RequiresPacks: []platformcatalog.PackKey{"nope"},
				Features:      []platformcatalog.FeatureKey{"a.core"},
			},
		},
	}

	err := registry.validatePacks()
	require.ErrorContains(t, err, "requires missing pack")
}

func TestRegistry_PackValidationRejectsSelfRequiringPack(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		features: map[platformcatalog.FeatureKey]platformcatalog.Feature{"a.core": {Key: "a.core"}},
		packs: map[platformcatalog.PackKey]platformcatalog.Pack{
			"loop": {
				Key:           "loop",
				RequiresPacks: []platformcatalog.PackKey{"loop"},
				Features:      []platformcatalog.FeatureKey{"a.core"},
			},
		},
	}

	err := registry.validatePacks()
	require.ErrorContains(t, err, "cannot require itself")
}

func TestRegistry_StandalonePackCannotRequireOtherPacks(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		features: map[platformcatalog.FeatureKey]platformcatalog.Feature{"a.core": {Key: "a.core"}},
		packs: map[platformcatalog.PackKey]platformcatalog.Pack{
			"base": {Key: "base", Features: []platformcatalog.FeatureKey{"a.core"}},
			"solo": {
				Key:           "solo",
				Standalone:    true,
				RequiresPacks: []platformcatalog.PackKey{"base"},
				Features:      []platformcatalog.FeatureKey{"a.core"},
			},
		},
	}

	err := registry.validatePacks()
	require.ErrorContains(t, err, "cannot require other packs")
}

func TestRegistry_AuthorizingFeaturesIncludesLegacyGrant(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	require.Equal(
		t,
		[]platformcatalog.FeatureKey{platformcatalog.FeatureWorkforceCore, platformcatalog.FeatureFleetMaintenance},
		registry.AuthorizingFeatures(platformcatalog.FeatureWorkforceCore, true),
	)
	require.Equal(
		t,
		[]platformcatalog.FeatureKey{platformcatalog.FeatureDispatch},
		registry.AuthorizingFeatures(platformcatalog.FeatureDispatch, true),
	)
	require.Equal(
		t,
		[]platformcatalog.FeatureKey{platformcatalog.FeatureKey("not.a.feature")},
		registry.AuthorizingFeatures(platformcatalog.FeatureKey("not.a.feature"), true),
	)
}

func TestRegistry_PackValidationReportsIndirectCycles(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		features: map[platformcatalog.FeatureKey]platformcatalog.Feature{"a.core": {Key: "a.core"}},
		packs: map[platformcatalog.PackKey]platformcatalog.Pack{
			"alpha": {
				Key:           "alpha",
				RequiresPacks: []platformcatalog.PackKey{"beta"},
				Features:      []platformcatalog.FeatureKey{"a.core"},
			},
			"beta": {
				Key:           "beta",
				RequiresPacks: []platformcatalog.PackKey{"alpha"},
				Features:      []platformcatalog.FeatureKey{"a.core"},
			},
		},
	}

	err := registry.validatePacks()
	require.ErrorContains(t, err, "pack requirement cycle")
	require.NotContains(t, err.Error(), "cannot require itself")
}

func TestRegistry_PackGrantedFeaturesAreDeclaredNotInferred(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		features: map[platformcatalog.FeatureKey]platformcatalog.Feature{
			"a.core":  {Key: "a.core"},
			"a.addon": {Key: "a.addon", RequiresFeatures: []platformcatalog.FeatureKey{"a.core"}},
		},
		packs: map[platformcatalog.PackKey]platformcatalog.Pack{
			"solo": {Key: "solo", Features: []platformcatalog.FeatureKey{"a.addon"}},
		},
	}

	granted, ok := registry.PackGrantedFeatures("solo")
	require.True(t, ok)
	require.Equal(
		t,
		[]platformcatalog.FeatureKey{"a.addon"},
		granted,
		"a pack grants what it declares; a feature dependency is not an implicit grant",
	)

	closure, ok := registry.PackFeatureClosure("solo")
	require.True(t, ok)
	require.Equal(
		t,
		[]platformcatalog.FeatureKey{"a.addon", "a.core"},
		closure,
		"the dependency closure still records what the pack needs in order to work",
	)
}

func TestRegistry_PackGrantedFeaturesFollowRequiredPacks(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	granted, ok := registry.PackGrantedFeatures(PackSettlement)
	require.True(t, ok)
	require.Contains(t, granted, platformcatalog.FeatureSettlement)
	require.Contains(
		t,
		granted,
		platformcatalog.FeatureCoreTMS,
		"Settlement requires the Professional pack, so its buyer holds Core TMS through it",
	)

	standalone, ok := registry.PackGrantedFeatures(PackWorkforce)
	require.True(t, ok)
	require.NotContains(t, standalone, platformcatalog.FeatureCoreTMS)
	require.NotContains(t, standalone, platformcatalog.FeatureDispatch)
}

func TestRegistry_AuthorizingFeaturesCanIgnoreLegacyGrants(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	require.Equal(
		t,
		[]platformcatalog.FeatureKey{platformcatalog.FeatureWorkforceCore},
		registry.AuthorizingFeatures(platformcatalog.FeatureWorkforceCore, false),
	)
	require.Equal(
		t,
		[]platformcatalog.FeatureKey{platformcatalog.FeatureSettlement},
		registry.AuthorizingFeatures(platformcatalog.FeatureSettlement, false),
	)
}

func TestRegistry_LegacyGrantValidationRejectsSelfReference(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		features: map[platformcatalog.FeatureKey]platformcatalog.Feature{platformcatalog.FeatureCoreTMS: {Key: platformcatalog.FeatureCoreTMS}},
	}

	err := registry.validateLegacyGrantingFeatures(
		platformcatalog.FeatureCoreTMS,
		[]platformcatalog.FeatureKey{platformcatalog.FeatureCoreTMS},
	)
	require.ErrorContains(t, err, "cannot be legacy granted by itself")
}

func TestRegistry_LegacyGrantValidationRejectsMissingFeature(t *testing.T) {
	t.Parallel()

	registry := &Registry{features: map[platformcatalog.FeatureKey]platformcatalog.Feature{}}

	err := registry.validateLegacyGrantingFeatures(
		platformcatalog.FeatureWorkforceCore,
		[]platformcatalog.FeatureKey{platformcatalog.FeatureKey("missing")},
	)
	require.ErrorContains(t, err, "legacy granted by missing feature")
}
