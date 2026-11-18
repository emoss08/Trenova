package homelayoutresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/app/homelayoutservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeWidgetsToModel_ALockedWidgetCarriesTheAddOnAndItsLabel(t *testing.T) {
	t.Parallel()

	denial := permissionports.NewPlanDenial(platformcatalog.FeatureAccounting)
	layout := &homelayout.Layout{Widgets: []homelayout.Widget{
		{ID: "w_ar", Key: homelayout.WidgetARSnapshot, W: 6, H: 3},
		{ID: "w_fav", Key: homelayout.WidgetFavorites, W: 4, H: 4},
		{ID: "w_strip", Key: homelayout.WidgetKPIRow, W: 12, H: 2},
	}}
	locks := map[string]homelayoutservice.WidgetLock{
		"w_ar": {Widget: &denial},
		"w_strip": {Metrics: []homelayoutservice.MetricLock{
			{Metric: "arTotalOpen", Denial: denial},
		}},
	}

	widgets := homeWidgetsToModel(i18n.EN, layout, locks)

	require.Len(t, widgets, 3)
	require.NotNil(t, widgets[0].PlanLock)
	assert.Equal(t, "Accounting & Cash", widgets[0].PlanLock.AddOnName)
	assert.Equal(t, string(platformcatalog.FeatureAccounting), widgets[0].PlanLock.Feature)
	assert.Equal(t, "Receivables snapshot", widgets[0].Label)
	assert.Empty(t, widgets[0].LockedMetrics)

	assert.Nil(t, widgets[1].PlanLock, "a base widget carries no lock")
	assert.NotNil(t, widgets[1].LockedMetrics, "the list is never null")

	require.Len(t, widgets[2].LockedMetrics, 1)
	assert.Equal(t, "arTotalOpen", widgets[2].LockedMetrics[0].Metric)
	assert.Equal(t, "Open receivables", widgets[2].LockedMetrics[0].Label)
	assert.Equal(t, "Accounting & Cash", widgets[2].LockedMetrics[0].Lock.AddOnName)
}

func TestHomePlanLockToModel_NamesTheFeatureAndAddOnInTheReadersLanguage(t *testing.T) {
	t.Parallel()

	denial := permissionports.NewPlanDenial(platformcatalog.FeatureAccounting)

	lock := homePlanLockToModel(i18n.ES, &denial)

	require.NotNil(t, lock)
	assert.Equal(t, "Contabilidad", lock.FeatureName)
	assert.Equal(t, "Contabilidad y efectivo", lock.AddOnName)
	assert.Equal(t, string(platformcatalog.AddOnAccounting), lock.AddOn)
	assert.Nil(t, homePlanLockToModel(i18n.ES, nil))
}

func TestHomeWidgetCatalogToModel_ListsLockedWidgetsAfterTheSelectableOnes(t *testing.T) {
	t.Parallel()

	denial := permissionports.NewPlanDenial(platformcatalog.FeatureAccounting)
	catalog := &homelayoutservice.WidgetCatalog{
		Widgets: []homelayout.WidgetDefinition{
			{Key: homelayout.WidgetAttention, Label: "Needs attention"},
		},
		LockedWidgets: []homelayoutservice.LockedWidget{{
			Definition: homelayout.WidgetDefinition{
				Key:   homelayout.WidgetARSnapshot,
				Label: "Receivables snapshot",
			},
			Denial: denial,
		}},
		LockedMetrics: []homelayoutservice.LockedMetric{{
			Definition: homelayout.MetricDefinition{Key: "marginPercent", Label: "Margin %"},
			Denial:     denial,
		}},
	}

	model := homeWidgetCatalogToModel(i18n.EN, catalog)

	require.Len(t, model.Widgets, 2)
	assert.Nil(t, model.Widgets[0].PlanLock)
	require.NotNil(t, model.Widgets[1].PlanLock)
	assert.Equal(t, "Accounting & Cash", model.Widgets[1].PlanLock.AddOnName)
	require.Len(t, model.Metrics, 1)
	require.NotNil(t, model.Metrics[0].PlanLock)
}
