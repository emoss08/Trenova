package homelayout

import (
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/permission"
)

const (
	DocumentSchemaVersion = 1
	GridColumns           = 12
	MaxWidgets            = 24
	MaxWidgetSpan         = GridColumns
	MaxWidgetHeight       = 12
	MaxKPIRowMetrics      = 6
	MaxAnnouncementLength = 2000
	MaxWidgetLimit        = 50
	MaxWindowDays         = 365
)

const (
	CategoryWork        = "work"
	CategoryPulse       = "pulse"
	CategoryInsight     = "insight"
	CategoryOrientation = "orientation"
	CategoryComms       = "comms"
)

const (
	DensityComfortable = "comfortable"
	DensityCompact     = "compact"
)

// ConfigKind selects which WidgetConfig fields a widget requires. A widget of
// kind ConfigKindNone carries no configuration at all, so its config is
// discarded rather than validated.
type ConfigKind string

const (
	ConfigKindNone      = ConfigKind("none")
	ConfigKindQueue     = ConfigKind("queue")
	ConfigKindMetric    = ConfigKind("metric")
	ConfigKindMetricRow = ConfigKind("metricRow")
	ConfigKindTrend     = ConfigKind("trend")
	ConfigKindReport    = ConfigKind("report")
	ConfigKindDashboard = ConfigKind("dashboard")
	ConfigKindText      = ConfigKind("text")
)

const (
	WidgetAttention        = "attention"
	WidgetUnassigned       = "unassigned"
	WidgetExceptions       = "exceptions"
	WidgetDetentionWatch   = "detention-watch"
	WidgetTomorrowsPickups = "tomorrows-pickups"
	WidgetMyApprovals      = "my-approvals"
	WidgetBillingQueue     = "billing-queue"
	WidgetServiceFailures  = "service-failures"
	WidgetEDIAttention     = "edi-attention"
	//nolint:gosec // G101: a dashboard widget identifier, not a credential
	WidgetExpiringCredentials = "expiring-credentials"

	WidgetKPI          = "kpi"
	WidgetKPIRow       = "kpi-row"
	WidgetARSnapshot   = "ar-snapshot"
	WidgetRevenueTrend = "revenue-trend"
	WidgetFleetStatus  = "fleet-status"
	WidgetOnTimeGoal   = "on-time-goal"

	WidgetReport        = "report"
	WidgetDashboardLink = "dashboard-link"
	WidgetLaneHeatmap   = "lane-heatmap"
	WidgetCustomerMix   = "customer-mix"

	WidgetQuickActions  = "quick-actions"
	WidgetFavorites     = "favorites"
	WidgetJumpBackIn    = "jump-back-in"
	WidgetSavedViews    = "saved-views"
	WidgetActivity      = "activity"
	WidgetNotifications = "notifications"

	WidgetAnnouncement = "announcement"
	WidgetMap          = "map"
)

// Analytics include keys understood by the shipment analytics provider. An
// empty include means the widget reads the provider's default payload, which
// every metric widget shares — that is what keeps a full canvas to one request.
const (
	IncludeDefault          = ""
	IncludeTomorrowsPickups = "tomorrowsPickups"
)

type CategoryDefinition struct {
	Key         string
	Label       string
	Description string
}

// WidgetDefinition is the code-resident contract for one widget. Resource may
// be empty, which means the widget carries no gate of its own because its
// contents are individually permission-filtered as they render (the attention
// grid and the quick-action launcher both work this way).
type WidgetDefinition struct {
	Key              string
	Label            string
	Description      string
	Category         string
	Resource         permission.Resource
	Operation        permission.Operation
	ConfigKind       ConfigKind
	AnalyticsInclude string
	DefaultW         int
	DefaultH         int
	MinW             int
	MinH             int
	MaxW             int
	MaxH             int
}

// RequiresPermission reports whether the widget carries a gate of its own.
func (d WidgetDefinition) RequiresPermission() bool {
	return d.Resource != ""
}

// MetricDefinition names one number a KPI widget can display. Metrics come from
// the analytics providers that already compute them, so a KPI widget never
// introduces a query of its own.
type MetricDefinition struct {
	Key       string
	Label     string
	Resource  permission.Resource
	Operation permission.Operation
}

func CategoryCatalog() []CategoryDefinition {
	return []CategoryDefinition{
		{
			Key:         CategoryWork,
			Label:       "Work",
			Description: "Queues that need someone to act",
		},
		{
			Key:         CategoryPulse,
			Label:       "Pulse",
			Description: "How the operation is running right now",
		},
		{
			Key:         CategoryInsight,
			Label:       "Insight",
			Description: "Saved reports and dashboards on the canvas",
		},
		{
			Key:         CategoryOrientation,
			Label:       "Orientation",
			Description: "Shortcuts back to where you were",
		},
		{
			Key:         CategoryComms,
			Label:       "Comms",
			Description: "Announcements and the live fleet map",
		},
	}
}

func WidgetCatalog() []WidgetDefinition {
	return slices.Concat(
		workWidgets(),
		pulseWidgets(),
		insightWidgets(),
		orientationWidgets(),
		commsWidgets(),
	)
}

func workWidgets() []WidgetDefinition {
	return []WidgetDefinition{
		{
			Key:         WidgetAttention,
			Label:       "Needs Attention",
			Description: "Every open queue across billing, operations, and EDI in one grid",
			Category:    CategoryWork,
			ConfigKind:  ConfigKindNone,
			DefaultW:    4, DefaultH: 4, MinW: 3, MinH: 3, MaxW: 12, MaxH: 8,
		},
		{
			Key:         WidgetUnassigned,
			Label:       "Unassigned Loads",
			Description: "Active shipments with no usable assignment, and the revenue waiting on them",
			Category:    CategoryWork,
			Resource:    permission.ResourceShipment,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindQueue,
			DefaultW:    4,
			DefaultH:    5,
			MinW:        3,
			MinH:        3,
			MaxW:        12,
			MaxH:        10,
		},
		{
			Key:         WidgetExceptions,
			Label:       "Exceptions",
			Description: "At-risk shipments: ETA slip, weather exposure, and reefer alarms",
			Category:    CategoryWork,
			Resource:    permission.ResourceShipment,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindQueue,
			DefaultW:    4, DefaultH: 5, MinW: 3, MinH: 3, MaxW: 12, MaxH: 10,
		},
		{
			Key:         WidgetDetentionWatch,
			Label:       "Detention Watch",
			Description: "Stops dwelling past the detention threshold, longest first",
			Category:    CategoryWork,
			Resource:    permission.ResourceShipment,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindQueue,
			DefaultW:    4, DefaultH: 5, MinW: 3, MinH: 3, MaxW: 12, MaxH: 10,
		},
		{
			Key:              WidgetTomorrowsPickups,
			Label:            "Tomorrow's Pickups",
			Description:      "The next business day's pickup windows with driver coverage",
			Category:         CategoryWork,
			Resource:         permission.ResourceShipment,
			Operation:        permission.OpRead,
			ConfigKind:       ConfigKindQueue,
			AnalyticsInclude: IncludeTomorrowsPickups,
			DefaultW:         6, DefaultH: 5, MinW: 4, MinH: 3, MaxW: 12, MaxH: 10,
		},
		{
			Key:         WidgetMyApprovals,
			Label:       "My Approvals",
			Description: "Invoices and billing transfers waiting on your decision",
			Category:    CategoryWork,
			Resource:    permission.ResourceInvoice,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindQueue,
			DefaultW:    4, DefaultH: 3, MinW: 3, MinH: 3, MaxW: 12, MaxH: 10,
		},
		{
			Key:         WidgetBillingQueue,
			Label:       "Billing Queue",
			Description: "Delivered loads waiting to be reviewed and billed",
			Category:    CategoryWork,
			Resource:    permission.ResourceBillingQueue,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindQueue,
			DefaultW:    4, DefaultH: 3, MinW: 3, MinH: 3, MaxW: 12, MaxH: 10,
		},
		{
			Key:         WidgetServiceFailures,
			Label:       "Service Failures",
			Description: "Open service failures and the loads they belong to",
			Category:    CategoryWork,
			Resource:    permission.ResourceServiceFailure,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindQueue,
			DefaultW:    4, DefaultH: 3, MinW: 3, MinH: 3, MaxW: 12, MaxH: 10,
		},
		{
			Key:         WidgetEDIAttention,
			Label:       "EDI Needs Attention",
			Description: "Inbound and outbound EDI messages that failed or are held",
			Category:    CategoryWork,
			Resource:    permission.ResourceEDI,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindQueue,
			DefaultW:    4, DefaultH: 3, MinW: 3, MinH: 3, MaxW: 12, MaxH: 10,
		},
		{
			Key:         WidgetExpiringCredentials,
			Label:       "Expiring Credentials",
			Description: "Driver licenses and medical cards, plus equipment registrations, coming due",
			Category:    CategoryWork,
			Resource:    permission.ResourceWorker,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindQueue,
			DefaultW:    4,
			DefaultH:    5,
			MinW:        3,
			MinH:        3,
			MaxW:        12,
			MaxH:        10,
		},
	}
}

func pulseWidgets() []WidgetDefinition {
	return []WidgetDefinition{
		{
			Key:         WidgetKPI,
			Label:       "Metric",
			Description: "One number with its trend, target, and change since yesterday",
			Category:    CategoryPulse,
			Resource:    permission.ResourceShipment,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindMetric,
			DefaultW:    3, DefaultH: 2, MinW: 2, MinH: 2, MaxW: 6, MaxH: 4,
		},
		{
			Key:         WidgetKPIRow,
			Label:       "Metric Strip",
			Description: "Up to six metrics in one compact band",
			Category:    CategoryPulse,
			Resource:    permission.ResourceShipment,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindMetricRow,
			DefaultW:    12, DefaultH: 2, MinW: 6, MinH: 2, MaxW: 12, MaxH: 3,
		},
		{
			Key:         WidgetARSnapshot,
			Label:       "Receivables Snapshot",
			Description: "Open, overdue, unapplied cash, and days sales outstanding",
			Category:    CategoryPulse,
			Resource:    permission.ResourceAccountsReceivable,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindNone,
			DefaultW:    6, DefaultH: 3, MinW: 4, MinH: 2, MaxW: 12, MaxH: 5,
		},
		{
			Key:         WidgetRevenueTrend,
			Label:       "Revenue & Volume",
			Description: "Revenue against load count over a rolling window",
			Category:    CategoryPulse,
			Resource:    permission.ResourceShipment,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindTrend,
			DefaultW:    6, DefaultH: 4, MinW: 4, MinH: 3, MaxW: 12, MaxH: 8,
		},
		{
			Key:         WidgetFleetStatus,
			Label:       "Fleet Status",
			Description: "Tractor, trailer, and driver availability at a glance",
			Category:    CategoryPulse,
			Resource:    permission.ResourceTractor,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindNone,
			DefaultW:    4, DefaultH: 4, MinW: 3, MinH: 3, MaxW: 8, MaxH: 6,
		},
		{
			Key:         WidgetOnTimeGoal,
			Label:       "On-Time Goal",
			Description: "Service performance against target, with the seven-day trail",
			Category:    CategoryPulse,
			Resource:    permission.ResourceShipment,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindNone,
			DefaultW:    3, DefaultH: 3, MinW: 2, MinH: 2, MaxW: 6, MaxH: 5,
		},
	}
}

func insightWidgets() []WidgetDefinition {
	return []WidgetDefinition{
		{
			Key:         WidgetReport,
			Label:       "Report",
			Description: "Any saved or canned report, drawn as a chart, table, or single measure",
			Category:    CategoryInsight,
			Resource:    permission.ResourceReport,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindReport,
			DefaultW:    6, DefaultH: 5, MinW: 3, MinH: 2, MaxW: 12, MaxH: 12,
		},
		{
			Key:         WidgetDashboardLink,
			Label:       "Dashboard",
			Description: "A shortcut to a saved dashboard, showing its headline measure inline",
			Category:    CategoryInsight,
			Resource:    permission.ResourceDashboard,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindDashboard,
			DefaultW:    3, DefaultH: 2, MinW: 2, MinH: 2, MaxW: 6, MaxH: 4,
		},
		{
			Key:         WidgetLaneHeatmap,
			Label:       "Lane Heatmap",
			Description: "Where volume is concentrating, origin against destination",
			Category:    CategoryInsight,
			Resource:    permission.ResourceShipment,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindNone,
			DefaultW:    6, DefaultH: 5, MinW: 4, MinH: 3, MaxW: 12, MaxH: 10,
		},
		{
			Key:         WidgetCustomerMix,
			Label:       "Customer Mix",
			Description: "Top customers by revenue share, with direction of travel",
			Category:    CategoryInsight,
			Resource:    permission.ResourceCustomer,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindNone,
			DefaultW:    6, DefaultH: 5, MinW: 4, MinH: 3, MaxW: 12, MaxH: 10,
		},
	}
}

func orientationWidgets() []WidgetDefinition {
	return []WidgetDefinition{
		{
			Key:         WidgetQuickActions,
			Label:       "Quick Actions",
			Description: "Create anything you have permission to create",
			Category:    CategoryOrientation,
			ConfigKind:  ConfigKindNone,
			DefaultW:    4, DefaultH: 3, MinW: 3, MinH: 2, MaxW: 12, MaxH: 6,
		},
		{
			Key:         WidgetFavorites,
			Label:       "Favorites",
			Description: "The pages you starred",
			Category:    CategoryOrientation,
			ConfigKind:  ConfigKindNone,
			DefaultW:    4, DefaultH: 4, MinW: 3, MinH: 2, MaxW: 12, MaxH: 8,
		},
		{
			Key:         WidgetJumpBackIn,
			Label:       "Jump Back In",
			Description: "Records you touched most recently",
			Category:    CategoryOrientation,
			Resource:    permission.ResourceAuditLog,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindQueue,
			DefaultW:    4, DefaultH: 4, MinW: 3, MinH: 2, MaxW: 12, MaxH: 8,
		},
		{
			Key:         WidgetSavedViews,
			Label:       "Saved Views",
			Description: "Your pinned table views and report views",
			Category:    CategoryOrientation,
			Resource:    permission.ResourceTableConfiguration,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindQueue,
			DefaultW:    4, DefaultH: 4, MinW: 3, MinH: 2, MaxW: 12, MaxH: 8,
		},
		{
			Key:         WidgetActivity,
			Label:       "Activity",
			Description: "What changed across the organization, newest first",
			Category:    CategoryOrientation,
			Resource:    permission.ResourceAuditLog,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindQueue,
			DefaultW:    6, DefaultH: 5, MinW: 4, MinH: 3, MaxW: 12, MaxH: 10,
		},
		{
			Key:         WidgetNotifications,
			Label:       "Notifications",
			Description: "Unread notifications, most urgent first",
			Category:    CategoryOrientation,
			ConfigKind:  ConfigKindQueue,
			DefaultW:    4, DefaultH: 4, MinW: 3, MinH: 2, MaxW: 12, MaxH: 8,
		},
	}
}

func commsWidgets() []WidgetDefinition {
	return []WidgetDefinition{
		{
			Key:         WidgetAnnouncement,
			Label:       "Announcement",
			Description: "A note from an administrator, pinned to everyone assigned this home screen",
			Category:    CategoryComms,
			ConfigKind:  ConfigKindText,
			DefaultW:    12,
			DefaultH:    2,
			MinW:        3,
			MinH:        1,
			MaxW:        12,
			MaxH:        6,
		},
		{
			Key:         WidgetMap,
			Label:       "Fleet Map",
			Description: "Where the fleet is right now",
			Category:    CategoryComms,
			Resource:    permission.ResourceShipment,
			Operation:   permission.OpRead,
			ConfigKind:  ConfigKindNone,
			DefaultW:    6, DefaultH: 6, MinW: 4, MinH: 4, MaxW: 12, MaxH: 12,
		},
	}
}

// MetricCatalog lists the measures a KPI widget can point at. Every entry is
// already computed by an analytics provider — a KPI widget selects a number, it
// never defines one.
func MetricCatalog() []MetricDefinition {
	shipment := func(key, label string) MetricDefinition {
		return MetricDefinition{
			Key:       key,
			Label:     label,
			Resource:  permission.ResourceShipment,
			Operation: permission.OpRead,
		}
	}

	return []MetricDefinition{
		shipment("activeShipments", "Active Shipments"),
		shipment("revenueToday", "Revenue Today"),
		shipment("onTimePercent", "On-Time %"),
		shipment("emptyMilePercent", "Empty Mile %"),
		shipment("atRisk", "At Risk"),
		shipment("unassigned", "Unassigned"),
		shipment("readyToDispatch", "Ready to Dispatch"),
		shipment("marginPercent", "Margin %"),
		shipment("revenuePerMile", "Revenue per Mile"),
		shipment("costPerMile", "Cost per Mile"),
		{
			Key:       "arTotalOpen",
			Label:     "Open Receivables",
			Resource:  permission.ResourceAccountsReceivable,
			Operation: permission.OpRead,
		},
		{
			Key:       "arOverdue",
			Label:     "Overdue Receivables",
			Resource:  permission.ResourceAccountsReceivable,
			Operation: permission.OpRead,
		},
		{
			Key:       "arDaysSalesOutstanding",
			Label:     "Days Sales Outstanding",
			Resource:  permission.ResourceAccountsReceivable,
			Operation: permission.OpRead,
		},
	}
}

func Densities() []string {
	return []string{DensityComfortable, DensityCompact}
}

// The catalogs are code-resident and immutable, so the lookup maps are built
// once. Normalize and permission narrowing look up every widget and metric on
// every home-screen resolve; rebuilding the full catalog per lookup would make
// that O(widgets × catalog) allocations.
var widgetDefinitionsByKey = sync.OnceValue(func() map[string]WidgetDefinition {
	catalog := WidgetCatalog()
	byKey := make(map[string]WidgetDefinition, len(catalog))
	for _, widget := range catalog {
		byKey[widget.Key] = widget
	}
	return byKey
})

var metricDefinitionsByKey = sync.OnceValue(func() map[string]MetricDefinition {
	catalog := MetricCatalog()
	byKey := make(map[string]MetricDefinition, len(catalog))
	for _, metric := range catalog {
		byKey[metric.Key] = metric
	}
	return byKey
})

func widgetDefinition(key string) (WidgetDefinition, bool) {
	definition, ok := widgetDefinitionsByKey()[key]
	return definition, ok
}

// WidgetDefinitionFor exposes the catalog lookup to the service layer, which
// needs a widget's resource to decide whether the viewer may see it.
func WidgetDefinitionFor(key string) (WidgetDefinition, bool) {
	return widgetDefinition(key)
}

func metricDefinition(key string) (MetricDefinition, bool) {
	definition, ok := metricDefinitionsByKey()[key]
	return definition, ok
}

// MetricDefinitionFor exposes the metric lookup to the service layer so a KPI
// widget can be dropped when the viewer cannot read the measure behind it.
func MetricDefinitionFor(key string) (MetricDefinition, bool) {
	return metricDefinition(key)
}

func metricExists(key string) bool {
	_, ok := metricDefinition(key)
	return ok
}

func densityAllowed(density string) bool {
	return slices.Contains(Densities(), density)
}
