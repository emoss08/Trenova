package platformcatalog

type StaticProvider struct{}

func NewStaticProvider() *StaticProvider {
	return &StaticProvider{}
}

func (p *StaticProvider) Products() []Product {
	return []Product{
		{
			Key:         ProductTMS,
			Name:        "Transportation Management",
			Description: "Core shipment, dispatch, billing, accounting, and fleet workflows.",
			Features: []FeatureKey{
				FeatureCoreTMS,
				FeatureDispatch,
				FeatureBilling,
				FeatureAccounting,
				FeatureFleetMaintenance,
				FeatureDocumentManagement,
			},
		},
		{
			Key:         ProductDocumentIntelligence,
			Name:        "Document Intelligence",
			Description: "Document OCR, classification, extraction, and review workflows.",
			Features: []FeatureKey{
				FeatureDocumentIntelligence,
			},
		},
		{
			Key:         ProductIntegrations,
			Name:        "Integrations",
			Description: "External data and routing integrations.",
			Features: []FeatureKey{
				FeatureEDIIntegration,
				FeatureExchangeRateIntegration,
				FeatureSamsaraIntegration,
				FeatureGoogleMapsIntegration,
			},
		},
		{
			Key:         ProductAnalytics,
			Name:        "Analytics",
			Description: "Operational analytics and reporting views.",
			Features: []FeatureKey{
				FeatureAnalytics,
			},
		},
		{
			Key:         ProductPlatform,
			Name:        "Platform",
			Description: "Cross-cutting platform capabilities.",
			Features: []FeatureKey{
				FeatureGlobalSearch,
				FeatureAPIKeys,
				FeatureTableChangeAlerts,
				FeatureRealtimeNotifications,
			},
		},
	}
}

func (p *StaticProvider) Features() []Feature {
	return append(tmsFeatures(), platformFeatures()...)
}

func tmsFeatures() []Feature {
	return []Feature{
		{
			Key:         FeatureCoreTMS,
			ProductKey:  ProductTMS,
			Name:        "Core TMS",
			Description: "Foundational tenant, organization, customer, location, and shipment data.",
			Routes: routeRefs(
				"/api/v1/organizations/",
				"/api/v1/users/",
				"/api/v1/audit-entries/",
				"/api/v1/table-configurations/",
				"/api/v1/page-favorites/",
				"/api/v1/me/",
				"/api/v1/permissions/",
				"/api/v1/platform-catalog/",
				"/api/v1/roles/",
				"/api/v1/role-assignments/",
				"/api/v1/us-states/",
				"/api/v1/custom-fields/",
				"/api/v1/admin/database-sessions/",
				"/api/v1/data-entry-controls/",
				"/api/v1/service-types/",
				"/api/v1/sequence-configs/",
				"/api/v1/hazardous-materials/",
				"/api/v1/hazmat-segregation-rules/",
				"/api/v1/dot-hazmat-references/",
				"/api/v1/commodities/",
				"/api/v1/customers/",
				"/api/v1/weather-alerts/",
				"/api/v1/notifications/",
			),
		},
		{
			Key:              FeatureDispatch,
			ProductKey:       ProductTMS,
			Name:             "Dispatch",
			Description:      "Shipment movement planning and execution.",
			RequiresFeatures: []FeatureKey{FeatureCoreTMS},
			Routes: routeRefs(
				"/api/v1/assignments/",
				"/api/v1/dispatch-controls/",
				"/api/v1/distance-overrides/",
				"/api/v1/hold-reasons/",
				"/api/v1/locations/",
				"/api/v1/location-categories/",
				"/api/v1/shipments/",
				"/api/v1/shipment-moves/",
				"/api/v1/shipment-events/",
				"/api/v1/shipment-types/",
				"/api/v1/shipment-controls/",
			),
		},
		{
			Key:              FeatureBilling,
			ProductKey:       ProductTMS,
			Name:             "Billing",
			Description:      "Invoicing, billing queues, and customer payments.",
			RequiresFeatures: []FeatureKey{FeatureCoreTMS},
			Routes: routeRefs(
				"/api/v1/accessorial-charges/",
				"/api/v1/accounting/bank-receipt-batches/",
				"/api/v1/accounting/bank-receipts/",
				"/api/v1/accounting/bank-receipt-work-items/",
				"/api/v1/accounting/customer-payments/",
				"/api/v1/billing-controls/",
				"/api/v1/billing-queue/",
				"/api/v1/formula-templates/",
				"/api/v1/billing/invoice-adjustments/",
				"/api/v1/billing/invoices/",
			),
		},
		{
			Key:              FeatureAccounting,
			ProductKey:       ProductTMS,
			Name:             "Accounting",
			Description:      "General ledger, journal entries, and accounting controls.",
			RequiresFeatures: []FeatureKey{FeatureBilling},
			Routes: routeRefs(
				"/api/v1/accounting-control/",
				"/api/v1/accounting-controls/",
				"/api/v1/accounting/accounts-receivable/",
				"/api/v1/accounting/journal-entries/",
				"/api/v1/accounting/journal-reversals/",
				"/api/v1/accounting/manual-journals/",
				"/api/v1/accounting/statements/",
				"/api/v1/accounting/trial-balance/",
				"/api/v1/account-types/",
				"/api/v1/gl-accounts/",
				"/api/v1/fiscal-years/",
				"/api/v1/fiscal-periods/",
				"/api/v1/invoice-adjustment-controls/",
			),
		},
		{
			Key:              FeatureFleetMaintenance,
			ProductKey:       ProductTMS,
			Name:             "Fleet",
			Description:      "Equipment, workers, and fleet reference data.",
			RequiresFeatures: []FeatureKey{FeatureCoreTMS},
			Routes: routeRefs(
				"/api/v1/equipment-manufacturers/",
				"/api/v1/equipment-types/",
				"/api/v1/fleet-codes/",
				"/api/v1/tractors/",
				"/api/v1/trailers/",
				"/api/v1/workers/",
				"/api/v1/worker-pto/",
			),
		},
		{
			Key:              FeatureDocumentManagement,
			ProductKey:       ProductTMS,
			Name:             "Document Management",
			Description:      "Document upload, storage, packets, and parsing rules.",
			RequiresFeatures: []FeatureKey{FeatureCoreTMS},
			Routes: routeRefs(
				"/api/v1/admin/document-operations/",
				"/api/v1/document-controls/",
				"/api/v1/document-packet-rules/",
				"/api/v1/document-parsing-rules/",
				"/api/v1/document-types/",
				"/api/v1/documents/",
			),
			Meters: []MeterKey{MeterDocumentUploads},
		},
	}
}

func platformFeatures() []Feature {
	return []Feature{
		{
			Key:              FeatureDocumentIntelligence,
			ProductKey:       ProductDocumentIntelligence,
			Name:             "AI Document Intelligence",
			Description:      "OCR-backed document classification and extraction.",
			RequiresFeatures: []FeatureKey{FeatureDocumentManagement},
			Routes:           routeRefs("/api/v1/document-intelligence/"),
			Meters: []MeterKey{
				MeterDocumentAIClassifications,
				MeterDocumentAIExtractions,
			},
		},
		{
			Key:         FeatureGlobalSearch,
			ProductKey:  ProductPlatform,
			Name:        "Global Search",
			Description: "Cross-entity search backed by configured search infrastructure.",
			Routes:      routeRefs("/api/v1/search/"),
			Meters:      []MeterKey{MeterGlobalSearchQueries},
		},
		{
			Key:         FeatureAnalytics,
			ProductKey:  ProductAnalytics,
			Name:        "Analytics Workspace",
			Description: "Operational analytics pages and query providers.",
			Routes:      routeRefs("/api/v1/analytics/"),
			Meters:      []MeterKey{MeterAnalyticsQueries},
		},
		{
			Key:         FeatureAPIKeys,
			ProductKey:  ProductPlatform,
			Name:        "API Keys",
			Description: "Tenant-scoped API keys and API usage tracking.",
			Routes:      routeRefs("/api/v1/api-keys/"),
			Meters:      []MeterKey{MeterAPIRequests},
		},
		{
			Key:         FeatureTableChangeAlerts,
			ProductKey:  ProductPlatform,
			Name:        "Table Change Alerts",
			Description: "Table change subscriptions and delivery events.",
			Routes:      routeRefs("/api/v1/tca/"),
			Meters:      []MeterKey{MeterTableChangeEvents},
		},
		{
			Key:         FeatureEDIIntegration,
			ProductKey:  ProductIntegrations,
			Name:        "EDI Integration",
			Description: "EDI partner, profile, document, transfer, and X12 workflows.",
			Routes:      routeRefs("/api/v1/edi/"),
		},
		{
			Key:         FeatureExchangeRateIntegration,
			ProductKey:  ProductIntegrations,
			Name:        "Exchange Rate Integration",
			Description: "Exchange-rate provider requests and cached exchange-rate data.",
			Routes:      routeRefs("/api/v1/exchange-rates/"),
		},
		{
			Key:         FeatureSamsaraIntegration,
			ProductKey:  ProductIntegrations,
			Name:        "Samsara Integration",
			Description: "Samsara vehicle and routing integration.",
			Routes: routeRefs(
				"/api/v1/integrations/",
				"/api/v1/integrations/samsara/",
			),
			Meters: []MeterKey{MeterIntegrationSyncRuns},
		},
		{
			Key:         FeatureGoogleMapsIntegration,
			ProductKey:  ProductIntegrations,
			Name:        "Google Maps Integration",
			Description: "Google Maps-backed location and routing helpers.",
			Routes:      routeRefs("/api/v1/google-maps/"),
		},
		{
			Key:         FeatureRealtimeNotifications,
			ProductKey:  ProductPlatform,
			Name:        "Realtime Notifications",
			Description: "Realtime messaging and notifications.",
			Routes:      routeRefs("/api/v1/realtime/"),
		},
	}
}

func routeRefs(paths ...string) []RouteRef {
	routes := make([]RouteRef, 0, len(paths))
	for i := range paths {
		routes = append(routes, RouteRef{
			Method: "*",
			Path:   paths[i],
		})
	}
	return routes
}

func (p *StaticProvider) Meters() []Meter {
	return []Meter{
		{
			Key:         MeterAPIRequests,
			ProductKey:  ProductPlatform,
			FeatureKey:  FeatureAPIKeys,
			Name:        "API Requests",
			Description: "Authenticated API requests made by tenant principals.",
			Unit:        "request",
		},
		{
			Key:         MeterDocumentUploads,
			ProductKey:  ProductTMS,
			FeatureKey:  FeatureDocumentManagement,
			Name:        "Document Uploads",
			Description: "Documents uploaded into tenant storage.",
			Unit:        "document",
		},
		{
			Key:         MeterDocumentAIClassifications,
			ProductKey:  ProductDocumentIntelligence,
			FeatureKey:  FeatureDocumentIntelligence,
			Name:        "Document AI Classifications",
			Description: "Document classification operations.",
			Unit:        "classification",
		},
		{
			Key:         MeterDocumentAIExtractions,
			ProductKey:  ProductDocumentIntelligence,
			FeatureKey:  FeatureDocumentIntelligence,
			Name:        "Document AI Extractions",
			Description: "Document extraction operations.",
			Unit:        "extraction",
		},
		{
			Key:         MeterGlobalSearchQueries,
			ProductKey:  ProductPlatform,
			FeatureKey:  FeatureGlobalSearch,
			Name:        "Global Search Queries",
			Description: "Global search queries executed by tenant users.",
			Unit:        "query",
		},
		{
			Key:         MeterTableChangeEvents,
			ProductKey:  ProductPlatform,
			FeatureKey:  FeatureTableChangeAlerts,
			Name:        "Table Change Events",
			Description: "Table change alert events emitted for delivery.",
			Unit:        "event",
		},
		{
			Key:         MeterIntegrationSyncRuns,
			ProductKey:  ProductIntegrations,
			FeatureKey:  FeatureSamsaraIntegration,
			Name:        "Integration Sync Runs",
			Description: "External integration synchronization runs.",
			Unit:        "run",
		},
		{
			Key:         MeterAnalyticsQueries,
			ProductKey:  ProductAnalytics,
			FeatureKey:  FeatureAnalytics,
			Name:        "Analytics Queries",
			Description: "Analytics provider queries.",
			Unit:        "query",
		},
	}
}
