package platformcatalog

const (
	ProductTMS                  ProductKey = "tms"
	ProductDocumentIntelligence ProductKey = "document_intelligence"
	ProductIntegrations         ProductKey = "integrations"
	ProductAnalytics            ProductKey = "analytics"
	ProductPlatform             ProductKey = "platform"
)

const (
	FeatureCoreTMS               FeatureKey = "tms.core"
	FeatureDispatch              FeatureKey = "tms.dispatch"
	FeatureBilling               FeatureKey = "tms.billing"
	FeatureAccounting            FeatureKey = "tms.accounting"
	FeatureFleetMaintenance      FeatureKey = "tms.fleet"
	FeatureDocumentManagement    FeatureKey = "document_management"
	FeatureDocumentIntelligence  FeatureKey = "document_intelligence.ai"
	FeatureGlobalSearch          FeatureKey = "platform.global_search"
	FeatureAnalytics             FeatureKey = "analytics.workspace"
	FeatureAPIKeys               FeatureKey = "platform.api_keys"
	FeatureTableChangeAlerts     FeatureKey = "platform.table_change_alerts"
	FeatureSamsaraIntegration    FeatureKey = "integrations.samsara"
	FeatureGoogleMapsIntegration FeatureKey = "integrations.google_maps"
	FeatureRealtimeNotifications FeatureKey = "platform.realtime_notifications"
)

const (
	MeterAPIRequests               MeterKey = "api.requests"
	MeterDocumentUploads           MeterKey = "documents.uploads"
	MeterDocumentAIClassifications MeterKey = "document_intelligence.classifications"
	MeterDocumentAIExtractions     MeterKey = "document_intelligence.extractions"
	MeterGlobalSearchQueries       MeterKey = "platform.search_queries"
	MeterTableChangeEvents         MeterKey = "platform.table_change_events"
	MeterIntegrationSyncRuns       MeterKey = "integrations.sync_runs"
	MeterAnalyticsQueries          MeterKey = "analytics.queries"
)
