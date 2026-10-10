package homelayout

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
	WidgetWorkerAttention     = "worker-attention"

	WidgetKPI          = "kpi"
	WidgetKPIRow       = "kpi-row"
	WidgetARSnapshot   = "ar-snapshot"
	WidgetRevenueTrend = "revenue-trend"
	WidgetFleetStatus  = "fleet-status"
	WidgetOnTimeGoal   = "on-time-goal"

	WidgetAIInsights    = "ai-insights"
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

// Canned report keys the shipped presets draw on. These are compiled against
// the canned registry by a service-layer test, so a report removed from the
// library breaks the build rather than a customer's home screen.
const (
	cannedUnbilledDelivered = "unbilled-delivered-shipments"
	cannedARAgingByCustomer = "ar-aging-by-customer"
	cannedCustomerScorecard = "customer-scorecard"
)

// Mode records where a user's home screen comes from. A user who has never
// touched the canvas stays on ModePreset, so an administrator's later edits to
// the assigned preset reach them; the moment they rearrange anything they move
// to ModeCustom and own their own layout.
type Mode string

const (
	ModePreset = Mode("preset")
	ModeCustom = Mode("custom")
)

const (
	MaxPresetPriority   = 1000
	maxPresetNameLength = 255
)
