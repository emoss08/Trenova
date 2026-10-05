package catalog

import "github.com/emoss08/trenova/internal/core/domain/platformcatalog"

type StaticProvider struct{}

func NewStaticProvider() *StaticProvider {
	return &StaticProvider{}
}

func (p *StaticProvider) Products() []platformcatalog.Product {
	return []platformcatalog.Product{
		{
			Key:         platformcatalog.ProductTMS,
			Name:        "Transportation Management",
			Description: "Core shipment, dispatch, billing, accounting, fleet, and settlement workflows.",
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureCoreTMS,
				platformcatalog.FeatureDispatch,
				platformcatalog.FeatureBilling,
				platformcatalog.FeatureAccounting,
				platformcatalog.FeatureFleetMaintenance,
				platformcatalog.FeatureSettlement,
				platformcatalog.FeatureDriverPortal,
			},
		},
		{
			Key:         platformcatalog.ProductWorkforce,
			Name:        "Workforce Management",
			Description: "Driver and employee records, DOT compliance, time off, scheduling, talent, safety, and benefits.",
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureWorkforceCore,
				platformcatalog.FeatureWorkforceCompliance,
				platformcatalog.FeatureWorkforceTimeOff,
				platformcatalog.FeatureWorkforceTimeTracking,
				platformcatalog.FeatureWorkforceTalent,
				platformcatalog.FeatureWorkforceSafety,
				platformcatalog.FeatureWorkforceBenefits,
				platformcatalog.FeatureWorkforceSelfService,
			},
		},
		{
			Key:         platformcatalog.ProductDocumentIntelligence,
			Name:        "Document Intelligence",
			Description: "Document OCR, classification, extraction, and review workflows.",
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureDocumentIntelligence,
			},
		},
		{
			Key:         platformcatalog.ProductIntegrations,
			Name:        "Integrations",
			Description: "External data and routing integrations.",
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureEDIIntegration,
				platformcatalog.FeatureExchangeRateIntegration,
				platformcatalog.FeatureSamsaraIntegration,
				platformcatalog.FeatureGoogleMapsIntegration,
			},
		},
		{
			Key:         platformcatalog.ProductAnalytics,
			Name:        "Analytics",
			Description: "Operational analytics and reporting views.",
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureAnalytics,
			},
		},
		{
			Key:         platformcatalog.ProductPlatform,
			Name:        "Platform",
			Description: "Cross-cutting platform capabilities.",
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureDocumentManagement,
				platformcatalog.FeatureGlobalSearch,
				platformcatalog.FeatureAPIKeys,
				platformcatalog.FeatureTableChangeAlerts,
				platformcatalog.FeatureAgentAutomation,
				platformcatalog.FeatureAdministration,
				platformcatalog.FeatureRealtimeNotifications,
			},
		},
	}
}

func (p *StaticProvider) Features() []platformcatalog.Feature {
	features := make([]platformcatalog.Feature, 0, len(graphQLSourceOwners))
	features = append(features, tmsFeatures()...)
	features = append(features, workforceFeatures()...)
	features = append(features, platformFeatures()...)

	return withGraphQLOwnership(features)
}

func withGraphQLOwnership(features []platformcatalog.Feature) []platformcatalog.Feature {
	for i := range features {
		features[i].GraphQLSources = graphQLSourcesFor(features[i].Key)
		features[i].GraphQLRootFields = graphQLRootFieldsFor(features[i].Key)
	}

	return features
}

func tmsFeatures() []platformcatalog.Feature {
	return []platformcatalog.Feature{
		{
			Key:         platformcatalog.FeatureCoreTMS,
			ProductKey:  platformcatalog.ProductTMS,
			Name:        "Core TMS",
			Description: "Foundational tenant, organization, customer, location, and shipment data.",
			Routes:      coreTMSRouteRefs(),
			Meters: []platformcatalog.MeterKey{
				platformcatalog.MeterShipmentsTotal,
				platformcatalog.MeterRecurringShipmentSeries,
				platformcatalog.MeterCustomersTotal,
				platformcatalog.MeterLocationsTotal,
				platformcatalog.MeterWorkersTotal,
				platformcatalog.MeterTractorsTotal,
				platformcatalog.MeterTrailersTotal,
			},
		},
		{
			Key:         platformcatalog.FeatureDispatch,
			ProductKey:  platformcatalog.ProductTMS,
			Name:        "Dispatch",
			Description: "Shipment movement planning and execution.",
			RequiresFeatures: []platformcatalog.FeatureKey{
				platformcatalog.FeatureCoreTMS,
				platformcatalog.FeatureWorkforceCore,
			},
			Routes: dispatchRouteRefs(),
		},
		{
			Key:              platformcatalog.FeatureBilling,
			ProductKey:       platformcatalog.ProductTMS,
			Name:             "Billing",
			Description:      "Invoicing, billing queues, and customer payments.",
			RequiresFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureCoreTMS},
			Routes:           billingRouteRefs(),
		},
		{
			Key:              platformcatalog.FeatureAccounting,
			ProductKey:       platformcatalog.ProductTMS,
			Name:             "Accounting",
			Description:      "General ledger, journal entries, and accounting controls.",
			RequiresFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureBilling},
			Routes:           accountingRouteRefs(),
		},
		{
			Key:              platformcatalog.FeatureFleetMaintenance,
			ProductKey:       platformcatalog.ProductTMS,
			Name:             "Fleet",
			Description:      "Equipment, tractors, trailers, and fleet reference data.",
			RequiresFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureCoreTMS},
			Routes:           fleetRouteRefs(),
		},
		{
			Key:         platformcatalog.FeatureSettlement,
			ProductKey:  platformcatalog.ProductTMS,
			Name:        "Settlement",
			Description: "Driver and carrier settlement runs, escrow, advances, deductions, and disputes.",
			RequiresFeatures: []platformcatalog.FeatureKey{
				platformcatalog.FeatureCoreTMS,
				platformcatalog.FeatureWorkforceCore,
			},
			LegacyGrantingFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureCoreTMS},
		},
		{
			Key:         platformcatalog.FeatureDriverPortal,
			ProductKey:  platformcatalog.ProductTMS,
			Name:        "Driver Portal",
			Description: "Driver-facing loads, pay, hours of service, and document capture.",
			RequiresFeatures: []platformcatalog.FeatureKey{
				platformcatalog.FeatureCoreTMS,
				platformcatalog.FeatureWorkforceCore,
			},
			LegacyGrantingFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureCoreTMS},
			Routes:                 driverPortalRouteRefs(),
		},
	}
}

func workforceFeatures() []platformcatalog.Feature {
	return []platformcatalog.Feature{
		{
			Key:         platformcatalog.FeatureWorkforceCore,
			ProductKey:  platformcatalog.ProductWorkforce,
			Name:        "Workforce Core",
			Description: "Worker records, profiles, job positions, org structure, holidays, and onboarding checklists.",
			LegacyGrantingFeatures: []platformcatalog.FeatureKey{
				platformcatalog.FeatureFleetMaintenance,
			},
			Routes: workforceCoreRouteRefs(),
			Meters: []platformcatalog.MeterKey{
				platformcatalog.MeterWorkforceManagedWorkers,
			},
		},
		{
			Key:         platformcatalog.FeatureWorkforceCompliance,
			ProductKey:  platformcatalog.ProductWorkforce,
			Name:        "Workforce Compliance",
			Description: "Driver qualification files, credentials, DOT testing, random pools, and Clearinghouse queries.",
			RequiresFeatures: []platformcatalog.FeatureKey{
				platformcatalog.FeatureWorkforceCore,
				platformcatalog.FeatureDocumentManagement,
			},
			Meters: []platformcatalog.MeterKey{
				platformcatalog.MeterWorkforceComplianceScreens,
			},
		},
		{
			Key:              platformcatalog.FeatureWorkforceTimeOff,
			ProductKey:       platformcatalog.ProductWorkforce,
			Name:             "Time Off",
			Description:      "PTO policies, accrual ledgers, balances, and leave case administration.",
			RequiresFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureWorkforceCore},
		},
		{
			Key:              platformcatalog.FeatureWorkforceTimeTracking,
			ProductKey:       platformcatalog.ProductWorkforce,
			Name:             "Time and Scheduling",
			Description:      "Timesheets, shift templates, swaps, and rota scheduling.",
			RequiresFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureWorkforceCore},
		},
		{
			Key:              platformcatalog.FeatureWorkforceTalent,
			ProductKey:       platformcatalog.ProductWorkforce,
			Name:             "Talent",
			Description:      "Training courses and records, performance reviews, discipline, and recognition.",
			RequiresFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureWorkforceCore},
		},
		{
			Key:              platformcatalog.FeatureWorkforceSafety,
			ProductKey:       platformcatalog.ProductWorkforce,
			Name:             "Workforce Safety",
			Description:      "Injury reporting, OSHA annual summaries, safety events, and fleet safety standing.",
			RequiresFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureWorkforceCore},
		},
		{
			Key:              platformcatalog.FeatureWorkforceBenefits,
			ProductKey:       platformcatalog.ProductWorkforce,
			Name:             "Benefits",
			Description:      "Benefit plan administration and worker enrollments.",
			RequiresFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureWorkforceCore},
		},
		{
			Key:              platformcatalog.FeatureWorkforceSelfService,
			ProductKey:       platformcatalog.ProductWorkforce,
			Name:             "Employee Self Service",
			Description:      "Worker portal access, policy acknowledgements, and profile change requests.",
			RequiresFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureWorkforceCore},
		},
	}
}

//nolint:funlen // static catalog data reads clearest as one literal
func platformFeatures() []platformcatalog.Feature {
	return []platformcatalog.Feature{
		{
			Key:         platformcatalog.FeatureDocumentManagement,
			ProductKey:  platformcatalog.ProductPlatform,
			Name:        "Document Management",
			Description: "Document upload, storage, packets, and parsing rules.",
			Routes:      documentManagementRouteRefs(),
			Meters: []platformcatalog.MeterKey{
				platformcatalog.MeterDocumentUploads,
				platformcatalog.MeterDocumentStorageBytes,
				platformcatalog.MeterDocumentFileBytes,
			},
		},
		{
			Key:                    platformcatalog.FeatureAgentAutomation,
			ProductKey:             platformcatalog.ProductPlatform,
			Name:                   "Agent Automation",
			Description:            "Agent runs, proposals, and exception handling.",
			LegacyGrantingFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureCoreTMS},
			Routes:                 agentAutomationRouteRefs(),
		},
		{
			Key:                    platformcatalog.FeatureAdministration,
			ProductKey:             platformcatalog.ProductPlatform,
			Name:                   "Administration",
			Description:            "Users, roles, permissions, audit history, custom fields, and table configuration.",
			LegacyGrantingFeatures: []platformcatalog.FeatureKey{platformcatalog.FeatureCoreTMS},
			Routes:                 administrationRouteRefs(),
			Meters:                 []platformcatalog.MeterKey{platformcatalog.MeterUserSeats},
		},
		{
			Key:         platformcatalog.FeatureDocumentIntelligence,
			ProductKey:  platformcatalog.ProductDocumentIntelligence,
			Name:        "AI Document Intelligence",
			Description: "OCR-backed document classification and extraction.",
			RequiresFeatures: []platformcatalog.FeatureKey{
				platformcatalog.FeatureDocumentManagement,
			},
			Meters: []platformcatalog.MeterKey{
				platformcatalog.MeterDocumentAIClassifications,
				platformcatalog.MeterDocumentAIExtractions,
			},
		},
		{
			Key:         platformcatalog.FeatureGlobalSearch,
			ProductKey:  platformcatalog.ProductPlatform,
			Name:        "Global Search",
			Description: "Cross-entity search backed by configured search infrastructure.",
			Routes:      globalSearchRouteRefs(),
			Meters:      []platformcatalog.MeterKey{platformcatalog.MeterGlobalSearchQueries},
		},
		{
			Key:         platformcatalog.FeatureAnalytics,
			ProductKey:  platformcatalog.ProductAnalytics,
			Name:        "Analytics Workspace",
			Description: "Operational analytics pages and query providers.",
			Routes:      analyticsRouteRefs(),
			Meters:      []platformcatalog.MeterKey{platformcatalog.MeterAnalyticsQueries},
		},
		{
			Key:         platformcatalog.FeatureAPIKeys,
			ProductKey:  platformcatalog.ProductPlatform,
			Name:        "API Keys",
			Description: "Tenant-scoped API keys and API usage tracking.",
			Routes:      apiKeyRouteRefs(),
			Meters:      []platformcatalog.MeterKey{platformcatalog.MeterAPIRequests},
		},
		{
			Key:         platformcatalog.FeatureTableChangeAlerts,
			ProductKey:  platformcatalog.ProductPlatform,
			Name:        "Table Change Alerts",
			Description: "Table change subscriptions and delivery events.",
			Routes:      tableChangeAlertRouteRefs(),
			Meters:      []platformcatalog.MeterKey{platformcatalog.MeterTableChangeEvents},
		},
		{
			Key:         platformcatalog.FeatureEDIIntegration,
			ProductKey:  platformcatalog.ProductIntegrations,
			Name:        "EDI Integration",
			Description: "EDI partner, profile, document, transfer, and X12 workflows.",
			Routes:      ediRouteRefs(),
		},
		{
			Key:         platformcatalog.FeatureExchangeRateIntegration,
			ProductKey:  platformcatalog.ProductIntegrations,
			Name:        "Exchange Rate Integration",
			Description: "Exchange-rate provider requests and cached exchange-rate data.",
			Routes:      exchangeRateRouteRefs(),
		},
		{
			Key:         platformcatalog.FeatureSamsaraIntegration,
			ProductKey:  platformcatalog.ProductIntegrations,
			Name:        "Samsara Integration",
			Description: "Samsara vehicle and routing integration.",
			Routes:      samsaraIntegrationRouteRefs(),
			Meters:      []platformcatalog.MeterKey{platformcatalog.MeterIntegrationSyncRuns},
		},
		{
			Key:         platformcatalog.FeatureGoogleMapsIntegration,
			ProductKey:  platformcatalog.ProductIntegrations,
			Name:        "Google Maps Integration",
			Description: "Google Maps-backed location and routing helpers.",
			Routes:      googleMapsRouteRefs(),
		},
		{
			Key:         platformcatalog.FeatureRealtimeNotifications,
			ProductKey:  platformcatalog.ProductPlatform,
			Name:        "Realtime Notifications",
			Description: "Realtime messaging and notifications.",
		},
	}
}

//nolint:funlen // static catalog data reads clearest as one literal
func (p *StaticProvider) Meters() []platformcatalog.Meter {
	return []platformcatalog.Meter{
		{
			Key:         platformcatalog.MeterAPIRequests,
			ProductKey:  platformcatalog.ProductPlatform,
			FeatureKey:  platformcatalog.FeatureAPIKeys,
			Name:        "API Requests",
			Description: "Authenticated API requests made by tenant principals.",
			Unit:        "request",
		},
		{
			Key:         platformcatalog.MeterDocumentUploads,
			ProductKey:  platformcatalog.ProductPlatform,
			FeatureKey:  platformcatalog.FeatureDocumentManagement,
			Name:        "Document Uploads",
			Description: "Documents uploaded into tenant storage.",
			Unit:        "document",
		},
		{
			Key:         platformcatalog.MeterWorkforceManagedWorkers,
			ProductKey:  platformcatalog.ProductWorkforce,
			FeatureKey:  platformcatalog.FeatureWorkforceCore,
			Name:        "Managed Workers",
			Description: "Active workers under workforce management, the per-seat billing unit.",
			Unit:        "worker",
		},
		{
			Key:         platformcatalog.MeterWorkforceComplianceScreens,
			ProductKey:  platformcatalog.ProductWorkforce,
			FeatureKey:  platformcatalog.FeatureWorkforceCompliance,
			Name:        "Compliance Screenings",
			Description: "DOT test orders and Clearinghouse queries run against workers.",
			Unit:        "screening",
		},
		{
			Key:         platformcatalog.MeterDocumentAIClassifications,
			ProductKey:  platformcatalog.ProductDocumentIntelligence,
			FeatureKey:  platformcatalog.FeatureDocumentIntelligence,
			Name:        "Document AI Classifications",
			Description: "Document classification operations.",
			Unit:        "classification",
		},
		{
			Key:         platformcatalog.MeterDocumentAIExtractions,
			ProductKey:  platformcatalog.ProductDocumentIntelligence,
			FeatureKey:  platformcatalog.FeatureDocumentIntelligence,
			Name:        "Document AI Extractions",
			Description: "Document extraction operations.",
			Unit:        "extraction",
		},
		{
			Key:         platformcatalog.MeterGlobalSearchQueries,
			ProductKey:  platformcatalog.ProductPlatform,
			FeatureKey:  platformcatalog.FeatureGlobalSearch,
			Name:        "Global Search Queries",
			Description: "Global search queries executed by tenant users.",
			Unit:        "query",
		},
		{
			Key:         platformcatalog.MeterTableChangeEvents,
			ProductKey:  platformcatalog.ProductPlatform,
			FeatureKey:  platformcatalog.FeatureTableChangeAlerts,
			Name:        "Table Change Events",
			Description: "Table change alert events emitted for delivery.",
			Unit:        "event",
		},
		{
			Key:         platformcatalog.MeterIntegrationSyncRuns,
			ProductKey:  platformcatalog.ProductIntegrations,
			FeatureKey:  platformcatalog.FeatureSamsaraIntegration,
			Name:        "Integration Sync Runs",
			Description: "External integration synchronization runs.",
			Unit:        "run",
		},
		{
			Key:         platformcatalog.MeterAnalyticsQueries,
			ProductKey:  platformcatalog.ProductAnalytics,
			FeatureKey:  platformcatalog.FeatureAnalytics,
			Name:        "Analytics Queries",
			Description: "Analytics provider queries.",
			Unit:        "query",
		},
		{
			Key:         platformcatalog.MeterShipmentsTotal,
			ProductKey:  platformcatalog.ProductTMS,
			FeatureKey:  platformcatalog.FeatureCoreTMS,
			Name:        "Shipments",
			Description: "Shipments an organization holds.",
			Unit:        "shipment",
		},
		{
			Key:         platformcatalog.MeterRecurringShipmentSeries,
			ProductKey:  platformcatalog.ProductTMS,
			FeatureKey:  platformcatalog.FeatureCoreTMS,
			Name:        "Recurring Shipment Series",
			Description: "Recurring shipment series an organization holds.",
			Unit:        "series",
		},
		{
			Key:         platformcatalog.MeterCustomersTotal,
			ProductKey:  platformcatalog.ProductTMS,
			FeatureKey:  platformcatalog.FeatureCoreTMS,
			Name:        "Customers",
			Description: "Customers an organization holds.",
			Unit:        "customer",
		},
		{
			Key:         platformcatalog.MeterLocationsTotal,
			ProductKey:  platformcatalog.ProductTMS,
			FeatureKey:  platformcatalog.FeatureCoreTMS,
			Name:        "Locations",
			Description: "Locations an organization holds.",
			Unit:        "location",
		},
		{
			Key:         platformcatalog.MeterWorkersTotal,
			ProductKey:  platformcatalog.ProductTMS,
			FeatureKey:  platformcatalog.FeatureCoreTMS,
			Name:        "Workers",
			Description: "Workers an organization holds.",
			Unit:        "worker",
		},
		{
			Key:         platformcatalog.MeterTractorsTotal,
			ProductKey:  platformcatalog.ProductTMS,
			FeatureKey:  platformcatalog.FeatureCoreTMS,
			Name:        "Tractors",
			Description: "Tractors an organization holds.",
			Unit:        "tractor",
		},
		{
			Key:         platformcatalog.MeterTrailersTotal,
			ProductKey:  platformcatalog.ProductTMS,
			FeatureKey:  platformcatalog.FeatureCoreTMS,
			Name:        "Trailers",
			Description: "Trailers an organization holds.",
			Unit:        "trailer",
		},
		{
			Key:         platformcatalog.MeterUserSeats,
			ProductKey:  platformcatalog.ProductPlatform,
			FeatureKey:  platformcatalog.FeatureAdministration,
			Name:        "User Seats",
			Description: "People who belong to an organization.",
			Unit:        "seat",
		},
		{
			Key:         platformcatalog.MeterDocumentStorageBytes,
			ProductKey:  platformcatalog.ProductPlatform,
			FeatureKey:  platformcatalog.FeatureDocumentManagement,
			Name:        "Document Storage",
			Description: "Bytes of documents an organization stores.",
			Unit:        "byte",
		},
		{
			Key:         platformcatalog.MeterDocumentFileBytes,
			ProductKey:  platformcatalog.ProductPlatform,
			FeatureKey:  platformcatalog.FeatureDocumentManagement,
			Name:        "Document File Size",
			Description: "Bytes in a single uploaded document.",
			Unit:        "byte",
		},
		{
			Key:         platformcatalog.MeterAIAssistantMessages,
			ProductKey:  platformcatalog.ProductPlatform,
			Name:        "Assistant Messages",
			Description: "Questions people ask the assistant in a calendar month.",
			Unit:        "message",
		},
		{
			Key:         platformcatalog.MeterAISpendCents,
			ProductKey:  platformcatalog.ProductPlatform,
			Name:        "AI Spend",
			Description: "Model spend in a calendar month.",
			Unit:        "cent",
		},
	}
}
