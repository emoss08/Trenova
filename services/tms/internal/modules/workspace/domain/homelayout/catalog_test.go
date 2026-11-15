package homelayout_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryWidgetDefinitionNamesTheResourceOrTheFeatureItNeeds(t *testing.T) {
	t.Parallel()

	for _, definition := range homelayout.WidgetCatalog() {
		assert.Truef(
			t,
			definition.NamesGate(),
			"widget %q names neither a permission resource nor a feature; every widget must say "+
				"what it needs so a plan that leaves it out can lock it",
			definition.Key,
		)
	}
}

func TestEveryMetricDefinitionNamesTheResourceOrTheFeatureItNeeds(t *testing.T) {
	t.Parallel()

	for _, metric := range homelayout.MetricCatalog() {
		assert.Truef(
			t,
			metric.NamesGate(),
			"metric %q names neither a permission resource nor a feature; every metric must say "+
				"what it needs so a plan that leaves it out can lock it before anything is read",
			metric.Key,
		)
	}
}

func TestEveryMetricResourceIsKnown(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.Default()

	for _, metric := range homelayout.MetricCatalog() {
		if metric.Resource == "" {
			continue
		}
		_, known := registry.FeatureOf(metric.Resource.String())
		assert.Truef(t, known, "metric %q names resource %q the permission registry does not know",
			metric.Key, metric.Resource)
	}
}

func TestTheCostingFiguresNeedTheFeatureTheirProviderChecks(t *testing.T) {
	t.Parallel()

	costing, known := permissioncatalog.Default().FeatureOf(permission.ResourceCostingControl.String())
	require.True(t, known)

	for _, key := range []string{"marginPercent", "costPerMile"} {
		metric, ok := homelayout.MetricDefinitionFor(key)
		require.True(t, ok)
		assert.Equal(t, costing, metric.Feature,
			"%s is computed from the costing setup, so it needs the costing setup's feature", key)
	}
}

func TestTheDetentionWatchNeedsDetentionReadAndItsFeature(t *testing.T) {
	t.Parallel()

	definition, ok := homelayout.WidgetDefinitionFor(homelayout.WidgetDetentionWatch)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceShipment, definition.Resource)
	assert.Equal(t, []permission.Resource{permission.ResourceDetention}, definition.AlsoReads)

	feature, known := permissioncatalog.Default().FeatureOf(permission.ResourceDetention.String())
	require.True(t, known)
	assert.Equal(t, platformcatalog.FeatureDetention, feature)
}

func TestTheFleetMapNeedsTheMapsFeature(t *testing.T) {
	t.Parallel()

	definition, ok := homelayout.WidgetDefinitionFor(homelayout.WidgetMap)
	require.True(t, ok)
	assert.Equal(t, platformcatalog.FeatureGoogleMapsIntegration, definition.Feature)
}

func TestEveryWidgetResourceAndFeatureIsKnown(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.Default()

	for _, definition := range homelayout.WidgetCatalog() {
		if definition.Resource != "" {
			_, known := registry.FeatureOf(definition.Resource.String())
			assert.Truef(t, known,
				"widget %q names resource %q the permission registry does not know",
				definition.Key, definition.Resource)
		}
		for _, resource := range definition.AlsoReads {
			_, known := registry.FeatureOf(resource.String())
			assert.Truef(t, known,
				"widget %q also reads resource %q the permission registry does not know",
				definition.Key, resource)
		}
		if definition.Feature != "" {
			_, known := platformcatalog.Lookup(definition.Feature)
			assert.Truef(t, known,
				"widget %q names feature %q the platform catalog does not know",
				definition.Key, definition.Feature)
		}
	}
}

func TestEveryMetricFeatureIsKnownAndTheCostingFiguresNameAccounting(t *testing.T) {
	t.Parallel()

	for _, metric := range homelayout.MetricCatalog() {
		if metric.Feature == "" {
			continue
		}
		_, known := platformcatalog.Lookup(metric.Feature)
		assert.Truef(t, known, "metric %q names an unknown feature %q", metric.Key, metric.Feature)
	}

	for _, key := range []string{"marginPercent", "costPerMile"} {
		metric, ok := homelayout.MetricDefinitionFor(key)
		assert.True(t, ok)
		assert.Equal(t, platformcatalog.FeatureAccounting, metric.Feature, key)
	}
}
