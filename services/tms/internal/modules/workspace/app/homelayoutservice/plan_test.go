package homelayoutservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// planAwareBatch answers the way the permission engine does: a resource whose
// feature the plan leaves out is refused with the plan's denial, ahead of any
// role logic, and every other resource is allowed unless named in denied.
func planAwareBatch(
	h *harness,
	gate *plantest.Gate,
	denied ...permission.Resource,
) {
	deniedSet := make(map[string]struct{}, len(denied))
	for _, resource := range denied {
		deniedSet[resource.String()] = struct{}{}
	}
	registry := permissioncatalog.Default()

	h.engine.EXPECT().
		CheckBatch(mock.Anything, mock.Anything).
		RunAndReturn(func(
			ctx context.Context,
			req *permissionports.BatchPermissionCheckRequest,
		) (*permissionports.BatchPermissionCheckResult, error) {
			results := make([]permissionports.PermissionCheckResult, 0, len(req.Checks))
			for _, check := range req.Checks {
				if feature, known := registry.FeatureOf(check.Resource); known {
					if on, _ := gate.Enabled(ctx, feature); !on {
						denial := permissionports.NewPlanDenial(feature)
						results = append(results, permissionports.PermissionCheckResult{
							PlanDenial: &denial,
						})
						continue
					}
				}
				_, deny := deniedSet[check.Resource]
				results = append(results, permissionports.PermissionCheckResult{Allowed: !deny})
			}
			return &permissionports.BatchPermissionCheckResult{Results: results}, nil
		})
}

func expectPresetHome(h *harness, widgets ...homelayout.Widget) {
	assigned := preset("Plan", func(p *homelayout.Preset) {
		p.IsOrgDefault = true
		p.Layout = &homelayout.Layout{Widgets: widgets}
	})
	h.repo.EXPECT().GetPreference(mock.Anything, mock.Anything).Return(nil, false, nil)
	h.rbac.EXPECT().
		GetAuthorizedRoles(mock.Anything, mock.Anything, mock.Anything).
		Return([]*rbac.Role{role("")}, nil)
	h.repo.EXPECT().
		ListPresets(mock.Anything, mock.Anything).
		Return([]*homelayout.Preset{assigned}, nil)
}

func planWidgets() []homelayout.Widget {
	return []homelayout.Widget{
		{ID: "w_unassigned", Key: homelayout.WidgetUnassigned, W: 4, H: 5},
		{ID: "w_ar", Key: homelayout.WidgetARSnapshot, W: 6, H: 3},
		{ID: "w_ai", Key: homelayout.WidgetAIInsights, W: 4, H: 6},
		{ID: "w_edi", Key: homelayout.WidgetEDIAttention, W: 4, H: 3},
		{ID: "w_detention", Key: homelayout.WidgetDetentionWatch, W: 4, H: 5},
		{
			ID: "w_margin", Key: homelayout.WidgetKPI, W: 3, H: 2,
			Config: homelayout.WidgetConfig{Metric: "marginPercent"},
		},
		{
			ID: "w_onTime", Key: homelayout.WidgetKPI, W: 3, H: 2,
			Config: homelayout.WidgetConfig{Metric: "onTimePercent"},
		},
		{
			ID: "w_strip", Key: homelayout.WidgetKPIRow, W: 12, H: 2,
			Config: homelayout.WidgetConfig{
				Metrics: []string{"activeShipments", "arTotalOpen", "costPerMile"},
			},
		},
		{ID: "w_map", Key: homelayout.WidgetMap, W: 6, H: 6},
		{ID: "w_favorites", Key: homelayout.WidgetFavorites, W: 4, H: 4},
		{ID: "w_report", Key: homelayout.WidgetReport, W: 6, H: 5},
	}
}

func TestResolve_OnABaseOnlyPlanEveryAddOnWidgetIsLockedWithItsAddOnsName(t *testing.T) {
	t.Parallel()

	gate := plantest.BaseOnly()
	h := newHarnessOn(t, gate)
	expectPresetHome(h, planWidgets()...)
	planAwareBatch(h, gate)

	effective, err := h.svc.GetEffective(t.Context(), testRequest())
	require.NoError(t, err)

	assert.Len(t, effective.Layout.Widgets, len(planWidgets()), "a locked widget keeps its slot")

	addOnOf := func(id string) string {
		lock, locked := effective.Locks[id]
		if !locked || lock.Widget == nil {
			return ""
		}
		return lock.Widget.AddOnName
	}
	assert.Equal(t, "Accounting & Cash", addOnOf("w_ar"))
	assert.Equal(t, "Trenova AI", addOnOf("w_ai"))
	assert.Equal(t, "EDI & Integrations", addOnOf("w_edi"))
	assert.Equal(t, "Rating & Revenue Recovery", addOnOf("w_detention"), "the detention it also reads")
	assert.Equal(t, "Accounting & Cash", addOnOf("w_margin"), "the costing figure needs Accounting")
	assert.Equal(t, "Visibility & Tracking", addOnOf("w_map"), "the fleet map is drawn on Google Maps")
	assert.Empty(t, effective.Locks["w_margin"].Metrics,
		"a single figure off the plan locks its tile as a whole, so Home leaves the tile out")

	for _, id := range []string{"w_unassigned", "w_onTime", "w_favorites", "w_report"} {
		assert.NotContains(t, effective.Locks, id, "base widgets are unchanged")
	}
}

func TestResolve_OnABaseOnlyPlanAMetricStripLocksOnlyTheAddOnFigures(t *testing.T) {
	t.Parallel()

	gate := plantest.BaseOnly()
	h := newHarnessOn(t, gate)
	expectPresetHome(h, planWidgets()...)
	planAwareBatch(h, gate)

	effective, err := h.svc.GetEffective(t.Context(), testRequest())
	require.NoError(t, err)

	lock := effective.Locks["w_strip"]
	assert.Nil(t, lock.Widget, "the shipment figure on the strip is still on the plan")
	locked := make([]string, 0, len(lock.Metrics))
	for _, metric := range lock.Metrics {
		locked = append(locked, metric.Metric)
		assert.Equal(t, "Accounting & Cash", metric.Denial.AddOnName)
	}
	assert.Equal(t, []string{"arTotalOpen", "costPerMile"}, locked)
	for _, widget := range effective.Layout.Widgets {
		if widget.ID == "w_strip" {
			assert.Equal(
				t,
				[]string{"activeShipments", "arTotalOpen", "costPerMile"},
				widget.Config.Metrics,
				"the strip keeps its figures so saving the layout loses none",
			)
		}
	}
}

func TestResolve_AStripWhoseEveryFigureIsAnAddOnsIsLockedAsAWhole(t *testing.T) {
	t.Parallel()

	gate := plantest.BaseOnly()
	h := newHarnessOn(t, gate)
	expectPresetHome(h, homelayout.Widget{
		ID: "w_strip", Key: homelayout.WidgetKPIRow, W: 12, H: 2,
		Config: homelayout.WidgetConfig{Metrics: []string{"arTotalOpen", "marginPercent"}},
	})
	planAwareBatch(h, gate)

	effective, err := h.svc.GetEffective(t.Context(), testRequest())
	require.NoError(t, err)

	lock := effective.Locks["w_strip"]
	require.NotNil(t, lock.Widget)
	assert.Equal(t, "Accounting & Cash", lock.Widget.AddOnName)
	assert.Empty(t, lock.Metrics)
}

func TestResolve_WithEveryFeatureOnTheLayoutIsTodays(t *testing.T) {
	t.Parallel()

	gate := plantest.AllFeatures()
	h := newHarnessOn(t, gate)
	expectPresetHome(h, planWidgets()...)
	planAwareBatch(h, gate)

	effective, err := h.svc.GetEffective(t.Context(), testRequest())
	require.NoError(t, err)

	assert.Empty(t, effective.Locks)
	assert.Equal(t, widgetKeys(&homelayout.Layout{Widgets: planWidgets()}), widgetKeys(effective.Layout))
}

func TestResolve_AWidgetTheReaderLacksPermissionForIsStillDroppedOnAnyPlan(t *testing.T) {
	t.Parallel()

	gate := plantest.BaseOnly()
	h := newHarnessOn(t, gate)
	expectPresetHome(h, planWidgets()...)
	planAwareBatch(h, gate, permission.ResourceShipment)

	effective, err := h.svc.GetEffective(t.Context(), testRequest())
	require.NoError(t, err)

	keys := widgetKeys(effective.Layout)
	assert.NotContains(t, keys, homelayout.WidgetUnassigned)
	assert.NotContains(t, effective.Locks, "w_unassigned", "no read stays dropped, not locked")
}

func TestResolve_TheDetentionWatchIsDroppedForAReaderWithoutDetentionRead(t *testing.T) {
	t.Parallel()

	gate := plantest.AllFeatures()
	h := newHarnessOn(t, gate)
	expectPresetHome(h, planWidgets()...)
	planAwareBatch(h, gate, permission.ResourceDetention)

	effective, err := h.svc.GetEffective(t.Context(), testRequest())
	require.NoError(t, err)

	assert.NotContains(t, widgetKeys(effective.Layout), homelayout.WidgetDetentionWatch)
	assert.NotContains(t, effective.Locks, "w_detention", "no detention read is dropped, not locked")
	assert.Contains(t, widgetKeys(effective.Layout), homelayout.WidgetUnassigned)
}

func TestResolve_ASingleFigureOffThePlanLocksItsTileBeforeAnythingIsRead(t *testing.T) {
	t.Parallel()

	for _, metric := range []string{"marginPercent", "costPerMile"} {
		t.Run(metric, func(t *testing.T) {
			t.Parallel()

			gate := plantest.Without(platformcatalog.FeatureAccounting)
			h := newHarnessOn(t, gate)
			expectPresetHome(h, homelayout.Widget{
				ID: "w_kpi", Key: homelayout.WidgetKPI, W: 3, H: 2,
				Config: homelayout.WidgetConfig{Metric: metric},
			})
			planAwareBatch(h, gate)

			effective, err := h.svc.GetEffective(t.Context(), testRequest())
			require.NoError(t, err)

			lock := effective.Locks["w_kpi"]
			require.NotNil(t, lock.Widget)
			assert.Equal(t, platformcatalog.FeatureAccounting, lock.Widget.Feature)
			assert.Equal(t, "Accounting & Cash", lock.Widget.AddOnName)
		})
	}
}

func TestGetCatalog_TheDetentionWatchIsNotOfferedToAReaderWithoutDetentionRead(t *testing.T) {
	t.Parallel()

	gate := plantest.AllFeatures()
	h := newHarnessOn(t, gate)
	planAwareBatch(h, gate, permission.ResourceDetention)

	catalog, err := h.svc.GetCatalog(t.Context(), testRequest())
	require.NoError(t, err)

	for _, widget := range catalog.Widgets {
		assert.NotEqual(t, homelayout.WidgetDetentionWatch, widget.Key)
	}
	for _, widget := range catalog.LockedWidgets {
		assert.NotEqual(t, homelayout.WidgetDetentionWatch, widget.Definition.Key)
	}
}

func TestResolve_ALockedWidgetCostsNoMoreThanTheOneBatchedCheck(t *testing.T) {
	t.Parallel()

	gate := plantest.BaseOnly()
	h := newHarnessOn(t, gate)
	expectPresetHome(h, planWidgets()...)
	planAwareBatch(h, gate)

	_, err := h.svc.GetEffective(t.Context(), testRequest())
	require.NoError(t, err)

	h.engine.AssertNumberOfCalls(t, "CheckBatch", 1)
}

func TestGetCatalog_OnABaseOnlyPlanNamesTheWidgetsThatNeedAnAddOn(t *testing.T) {
	t.Parallel()

	gate := plantest.BaseOnly()
	h := newHarnessOn(t, gate)
	planAwareBatch(h, gate)

	catalog, err := h.svc.GetCatalog(t.Context(), testRequest())
	require.NoError(t, err)

	selectable := make(map[string]struct{}, len(catalog.Widgets))
	for _, widget := range catalog.Widgets {
		selectable[widget.Key] = struct{}{}
	}
	locked := make(map[string]string, len(catalog.LockedWidgets))
	for _, widget := range catalog.LockedWidgets {
		locked[widget.Definition.Key] = widget.Denial.AddOnName
		assert.NotContains(t, selectable, widget.Definition.Key, "a locked widget is not selectable")
	}

	assert.Equal(t, "Accounting & Cash", locked[homelayout.WidgetARSnapshot])
	assert.Equal(t, "Trenova AI", locked[homelayout.WidgetAIInsights])
	assert.Equal(t, "EDI & Integrations", locked[homelayout.WidgetEDIAttention])
	assert.Equal(t, "Rating & Revenue Recovery", locked[homelayout.WidgetDetentionWatch])
	assert.Contains(t, selectable, homelayout.WidgetAttention)
	assert.Contains(t, selectable, homelayout.WidgetUnassigned)

	lockedMetrics := make(map[string]string, len(catalog.LockedMetrics))
	for _, metric := range catalog.LockedMetrics {
		lockedMetrics[metric.Definition.Key] = metric.Denial.AddOnName
	}
	assert.Equal(t, "Accounting & Cash", lockedMetrics["arTotalOpen"])
	assert.Equal(t, "Accounting & Cash", lockedMetrics["marginPercent"])
	assert.Equal(t, "Accounting & Cash", lockedMetrics["costPerMile"])
	for _, metric := range catalog.Metrics {
		assert.NotContains(t, lockedMetrics, metric.Key)
	}
}

func TestGetCatalog_WithEveryFeatureOnLocksNothing(t *testing.T) {
	t.Parallel()

	gate := plantest.AllFeatures()
	h := newHarnessOn(t, gate)
	planAwareBatch(h, gate)

	catalog, err := h.svc.GetCatalog(t.Context(), testRequest())
	require.NoError(t, err)

	assert.Empty(t, catalog.LockedWidgets)
	assert.Empty(t, catalog.LockedMetrics)
	assert.Len(t, catalog.Widgets, len(homelayout.WidgetCatalog()))
}

func TestGetCatalog_AFeatureTheWidgetNamesLocksItWithoutAResourceCheck(t *testing.T) {
	t.Parallel()

	gate := plantest.Without(platformcatalog.FeatureRealtimeNotifications)
	h := newHarnessOn(t, gate)
	planAwareBatch(h, gate)

	catalog, err := h.svc.GetCatalog(t.Context(), testRequest())
	require.NoError(t, err)

	found := false
	for _, widget := range catalog.LockedWidgets {
		if widget.Definition.Key == homelayout.WidgetNotifications {
			found = true
			assert.Equal(t, platformcatalog.FeatureRealtimeNotifications, widget.Denial.Feature)
		}
	}
	assert.True(t, found, "notifications has no resource and names its feature")
}
