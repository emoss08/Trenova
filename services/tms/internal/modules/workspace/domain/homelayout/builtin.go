package homelayout

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
)

// BuiltInLayout is the home screen a user sees before anyone has authored
// anything for them. Choosing by core responsibility means a new dispatcher and
// a new controller each land somewhere useful on their first login without an
// administrator having done any setup.
func BuiltInLayout(responsibility rbac.CoreResponsibility) *Layout {
	switch responsibility {
	case rbac.CoreResponsibilityOperations:
		return operationsLayout()
	case rbac.CoreResponsibilityBilling:
		return billingLayout()
	case rbac.CoreResponsibilityFinance:
		return financeLayout()
	case rbac.CoreResponsibilityLeadership:
		return leadershipLayout()
	default:
		return fallbackLayout()
	}
}

// BuiltInLayoutName labels the shipped layout so the UI can say what the user
// is looking at, and what "reset" would return them to.
func BuiltInLayoutName(responsibility rbac.CoreResponsibility) string {
	switch responsibility {
	case rbac.CoreResponsibilityOperations:
		return "Operations home"
	case rbac.CoreResponsibilityBilling:
		return "Billing home"
	case rbac.CoreResponsibilityFinance:
		return "Finance home"
	case rbac.CoreResponsibilityLeadership:
		return "Leadership home"
	default:
		return "Trenova home"
	}
}

func operationsLayout() *Layout {
	return layoutOf(
		widget(WidgetKPIRow, 12, 2, WidgetConfig{
			Metrics: []string{
				"activeShipments",
				"readyToDispatch",
				"movesLate",
				"onTimePercent",
				"emptyMilePercent",
				"revenueToday",
			},
		}),
		widget(WidgetUnassigned, 6, 5, WidgetConfig{Limit: 8}),
		widget(WidgetExceptions, 6, 5, WidgetConfig{Limit: 8}),
		widget(WidgetTomorrowsPickups, 8, 5, WidgetConfig{Limit: 10}),
		widget(WidgetDetentionWatch, 4, 5, WidgetConfig{Limit: 8}),
		widget(WidgetMap, 8, 5, WidgetConfig{}),
		widget(WidgetFleetStatus, 4, 3, WidgetConfig{}),
		widget(WidgetQuickActions, 4, 2, WidgetConfig{}),
	)
}

func billingLayout() *Layout {
	return layoutOf(
		widget(WidgetKPIRow, 12, 2, WidgetConfig{
			Metrics: []string{"arTotalOpen", "arOverdue", "arDaysSalesOutstanding", "revenueToday"},
		}),
		widget(WidgetReport, 6, 5, WidgetConfig{
			CannedKey: cannedUnbilledDelivered,
			Limit:     10,
		}),
		widget(WidgetReport, 6, 5, WidgetConfig{
			CannedKey: cannedARAgingByCustomer,
			Limit:     10,
		}),
		widget(WidgetActivity, 8, 3, WidgetConfig{Limit: 10}),
		widget(WidgetQuickActions, 4, 3, WidgetConfig{}),
	)
}

func financeLayout() *Layout {
	return layoutOf(
		widget(WidgetKPIRow, 12, 2, WidgetConfig{
			Metrics: []string{
				"arTotalOpen",
				"arOverdue",
				"arDaysSalesOutstanding",
				"revenuePerMile",
				"costPerMile",
				"marginPercent",
			},
		}),
		widget(WidgetRevenueTrend, 8, 4, WidgetConfig{WindowDays: 90}),
		widget(WidgetOnTimeGoal, 4, 4, WidgetConfig{}),
		// No dashboard-link here: a shipped preset has no dashboard to point
		// at, and a link widget without a target fails its own validation.
		widget(WidgetReport, 6, 5, WidgetConfig{
			CannedKey: cannedARAgingByCustomer,
			Limit:     10,
		}),
		widget(WidgetCustomerMix, 6, 5, WidgetConfig{}),
	)
}

func leadershipLayout() *Layout {
	return layoutOf(
		widget(WidgetKPIRow, 12, 2, WidgetConfig{
			Metrics: []string{
				"revenueToday",
				"marginPercent",
				"onTimePercent",
				"emptyMilePercent",
				"activeShipments",
				"arOverdue",
			},
		}),
		widget(WidgetRevenueTrend, 8, 4, WidgetConfig{WindowDays: 90}),
		widget(WidgetOnTimeGoal, 4, 4, WidgetConfig{}),
		widget(WidgetLaneHeatmap, 6, 5, WidgetConfig{}),
		widget(WidgetCustomerMix, 6, 5, WidgetConfig{}),
		widget(WidgetReport, 12, 5, WidgetConfig{
			CannedKey: cannedCustomerScorecard,
			Limit:     10,
		}),
	)
}

func fallbackLayout() *Layout {
	return layoutOf(
		widget(WidgetKPIRow, 12, 2, WidgetConfig{
			Metrics: []string{"activeShipments", "onTimePercent", "revenueToday", "arOverdue"},
		}),
		widget(WidgetUnassigned, 6, 5, WidgetConfig{Limit: 8}),
		widget(WidgetExceptions, 6, 5, WidgetConfig{Limit: 8}),
		widget(WidgetTomorrowsPickups, 8, 5, WidgetConfig{Limit: 10}),
		widget(WidgetQuickActions, 4, 2, WidgetConfig{}),
		widget(WidgetJumpBackIn, 4, 3, WidgetConfig{Limit: 6}),
	)
}

func layoutOf(widgets ...Widget) *Layout {
	for i := range widgets {
		widgets[i].ID = NextWidgetID(widgets[:i])
	}
	return &Layout{Widgets: widgets}
}

func widget(key string, w, h int, config WidgetConfig) Widget {
	return Widget{Key: key, W: w, H: h, Config: config}
}
