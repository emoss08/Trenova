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
			Routes:      coreTMSRouteRefs(),
		},
		{
			Key:              FeatureDispatch,
			ProductKey:       ProductTMS,
			Name:             "Dispatch",
			Description:      "Shipment movement planning and execution.",
			RequiresFeatures: []FeatureKey{FeatureCoreTMS},
			Routes:           dispatchRouteRefs(),
		},
		{
			Key:              FeatureBilling,
			ProductKey:       ProductTMS,
			Name:             "Billing",
			Description:      "Invoicing, billing queues, and customer payments.",
			RequiresFeatures: []FeatureKey{FeatureCoreTMS},
			Routes:           billingRouteRefs(),
		},
		{
			Key:              FeatureAccounting,
			ProductKey:       ProductTMS,
			Name:             "Accounting",
			Description:      "General ledger, journal entries, and accounting controls.",
			RequiresFeatures: []FeatureKey{FeatureBilling},
			Routes:           accountingRouteRefs(),
		},
		{
			Key:              FeatureFleetMaintenance,
			ProductKey:       ProductTMS,
			Name:             "Fleet",
			Description:      "Equipment, workers, and fleet reference data.",
			RequiresFeatures: []FeatureKey{FeatureCoreTMS},
			Routes:           fleetRouteRefs(),
		},
		{
			Key:              FeatureDocumentManagement,
			ProductKey:       ProductTMS,
			Name:             "Document Management",
			Description:      "Document upload, storage, packets, and parsing rules.",
			RequiresFeatures: []FeatureKey{FeatureCoreTMS},
			Routes:           documentManagementRouteRefs(),
			Meters:           []MeterKey{MeterDocumentUploads},
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
			Routes:      globalSearchRouteRefs(),
			Meters:      []MeterKey{MeterGlobalSearchQueries},
		},
		{
			Key:         FeatureAnalytics,
			ProductKey:  ProductAnalytics,
			Name:        "Analytics Workspace",
			Description: "Operational analytics pages and query providers.",
			Routes:      analyticsRouteRefs(),
			Meters:      []MeterKey{MeterAnalyticsQueries},
		},
		{
			Key:         FeatureAPIKeys,
			ProductKey:  ProductPlatform,
			Name:        "API Keys",
			Description: "Tenant-scoped API keys and API usage tracking.",
			Routes:      apiKeyRouteRefs(),
			Meters:      []MeterKey{MeterAPIRequests},
		},
		{
			Key:         FeatureTableChangeAlerts,
			ProductKey:  ProductPlatform,
			Name:        "Table Change Alerts",
			Description: "Table change subscriptions and delivery events.",
			Routes:      tableChangeAlertRouteRefs(),
			Meters:      []MeterKey{MeterTableChangeEvents},
		},
		{
			Key:         FeatureEDIIntegration,
			ProductKey:  ProductIntegrations,
			Name:        "EDI Integration",
			Description: "EDI partner, profile, document, transfer, and X12 workflows.",
			Routes:      ediRouteRefs(),
		},
		{
			Key:         FeatureExchangeRateIntegration,
			ProductKey:  ProductIntegrations,
			Name:        "Exchange Rate Integration",
			Description: "Exchange-rate provider requests and cached exchange-rate data.",
			Routes:      exchangeRateRouteRefs(),
		},
		{
			Key:         FeatureSamsaraIntegration,
			ProductKey:  ProductIntegrations,
			Name:        "Samsara Integration",
			Description: "Samsara vehicle and routing integration.",
			Routes:      samsaraIntegrationRouteRefs(),
			Meters:      []MeterKey{MeterIntegrationSyncRuns},
		},
		{
			Key:         FeatureGoogleMapsIntegration,
			ProductKey:  ProductIntegrations,
			Name:        "Google Maps Integration",
			Description: "Google Maps-backed location and routing helpers.",
			Routes:      googleMapsRouteRefs(),
		},
		{
			Key:         FeatureRealtimeNotifications,
			ProductKey:  ProductPlatform,
			Name:        "Realtime Notifications",
			Description: "Realtime messaging and notifications.",
		},
	}
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
