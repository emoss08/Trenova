package homelayout

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
)

// Canned report keys the shipped presets draw on. These are compiled against
// the canned registry by a service-layer test, so a report removed from the
// library breaks the build rather than a customer's home screen.
const (
	cannedUnbilledDelivered = "unbilled-delivered-shipments"
	cannedARAgingByCustomer = "ar-aging-by-customer"
	cannedCustomerScorecard = "customer-scorecard"
)

// BuiltInLayout is the home screen a user sees before anyone has authored
// anything for them. Choosing by core responsibility means a new dispatcher and
// a new controller each land somewhere useful on their first login without an
// administrator having done any setup.
func BuiltInLayout(responsibility permission.CoreResponsibility) *Layout {
	switch responsibility {
	case permission.CoreResponsibilityOperations:
		return operationsLayout()
	case permission.CoreResponsibilityBilling:
		return billingLayout()
	case permission.CoreResponsibilityFinance:
		return financeLayout()
	case permission.CoreResponsibilityLeadership:
		return leadershipLayout()
	default:
		return fallbackLayout()
	}
}

// BuiltInLayoutName labels the shipped layout so the UI can say what the user
// is looking at, and what "reset" would return them to.
func BuiltInLayoutName(responsibility permission.CoreResponsibility) string {
	switch responsibility {
	case permission.CoreResponsibilityOperations:
		return "Operations Home"
	case permission.CoreResponsibilityBilling:
		return "Billing Home"
	case permission.CoreResponsibilityFinance:
		return "Finance Home"
	case permission.CoreResponsibilityLeadership:
		return "Leadership Home"
	default:
		return "Trenova Home"
	}
}

func operationsLayout() *Layout {
	return layoutOf(
		widget(WidgetKPIRow, 12, 2, WidgetConfig{
			Metrics: []string{
				"activeShipments",
				"unassigned",
				"atRisk",
				"onTimePercent",
				"readyToDispatch",
				"revenueToday",
			},
		}),
		widget(WidgetAttention, 4, 4, WidgetConfig{}),
		widget(WidgetUnassigned, 4, 4, WidgetConfig{Limit: 8}),
		widget(WidgetExceptions, 4, 4, WidgetConfig{Limit: 8}),
		widget(WidgetDetentionWatch, 4, 5, WidgetConfig{Limit: 8}),
		widget(WidgetTomorrowsPickups, 8, 5, WidgetConfig{Limit: 10}),
		widget(WidgetFleetStatus, 4, 4, WidgetConfig{}),
		widget(WidgetMap, 8, 4, WidgetConfig{}),
		widget(WidgetQuickActions, 12, 2, WidgetConfig{}),
	)
}

func billingLayout() *Layout {
	return layoutOf(
		widget(WidgetKPIRow, 12, 2, WidgetConfig{
			Metrics: []string{"arTotalOpen", "arOverdue", "arDaysSalesOutstanding", "revenueToday"},
		}),
		// The queue and approvals tiles are a single number with a destination;
		// three rows fit them, and five leaves a tile that is mostly air.
		widget(WidgetBillingQueue, 4, 3, WidgetConfig{Limit: 10}),
		widget(WidgetMyApprovals, 4, 3, WidgetConfig{Limit: 10}),
		widget(WidgetAttention, 4, 3, WidgetConfig{}),
		widget(WidgetARSnapshot, 6, 3, WidgetConfig{}),
		widget(WidgetReport, 6, 5, WidgetConfig{
			CannedKey: cannedUnbilledDelivered,
			Limit:     10,
		}),
		widget(WidgetActivity, 6, 5, WidgetConfig{Limit: 10}),
		widget(WidgetQuickActions, 6, 3, WidgetConfig{}),
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
		widget(WidgetARSnapshot, 6, 3, WidgetConfig{}),
		widget(WidgetRevenueTrend, 6, 4, WidgetConfig{WindowDays: 90}),
		widget(WidgetReport, 6, 5, WidgetConfig{
			CannedKey: cannedARAgingByCustomer,
			Limit:     10,
		}),
		widget(WidgetCustomerMix, 6, 5, WidgetConfig{}),
		// No dashboard-link here: a shipped preset has no dashboard to point
		// at, and a link widget without a target fails its own validation.
		widget(WidgetOnTimeGoal, 3, 4, WidgetConfig{}),
		widget(WidgetActivity, 9, 4, WidgetConfig{Limit: 10}),
	)
}

func leadershipLayout() *Layout {
	return layoutOf(
		widget(WidgetAnnouncement, 12, 2, WidgetConfig{
			Text: "Welcome to Trenova. Edit this announcement to speak to your team.",
		}),
		widget(WidgetKPIRow, 12, 2, WidgetConfig{
			Metrics: []string{
				"revenueToday",
				"onTimePercent",
				"emptyMilePercent",
				"marginPercent",
				"activeShipments",
				"atRisk",
			},
		}),
		widget(WidgetRevenueTrend, 5, 4, WidgetConfig{WindowDays: 90}),
		widget(WidgetOnTimeGoal, 3, 4, WidgetConfig{}),
		widget(WidgetARSnapshot, 4, 4, WidgetConfig{}),
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
		widget(WidgetAttention, 4, 4, WidgetConfig{}),
		widget(WidgetQuickActions, 4, 4, WidgetConfig{}),
		widget(WidgetFavorites, 4, 4, WidgetConfig{}),
		widget(WidgetJumpBackIn, 6, 4, WidgetConfig{Limit: 8}),
		widget(WidgetActivity, 6, 4, WidgetConfig{Limit: 10}),
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
