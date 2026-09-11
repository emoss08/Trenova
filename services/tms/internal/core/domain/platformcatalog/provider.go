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
			Description: "Core shipment, dispatch, billing, accounting, fleet, and settlement workflows.",
			Features: []FeatureKey{
				FeatureCoreTMS,
				FeatureDispatch,
				FeatureBilling,
				FeatureAccounting,
				FeatureFleetMaintenance,
				FeatureSettlement,
				FeatureDriverPortal,
			},
		},
		{
			Key:         ProductWorkforce,
			Name:        "Workforce Management",
			Description: "Driver and employee records, DOT compliance, time off, scheduling, talent, safety, and benefits.",
			Features: []FeatureKey{
				FeatureWorkforceCore,
				FeatureWorkforceCompliance,
				FeatureWorkforceTimeOff,
				FeatureWorkforceTimeTracking,
				FeatureWorkforceTalent,
				FeatureWorkforceSafety,
				FeatureWorkforceBenefits,
				FeatureWorkforceSelfService,
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
				FeatureDocumentManagement,
				FeatureGlobalSearch,
				FeatureAPIKeys,
				FeatureTableChangeAlerts,
				FeatureAgentAutomation,
				FeatureAdministration,
				FeatureRealtimeNotifications,
			},
		},
	}
}

func (p *StaticProvider) Features() []Feature {
	features := make([]Feature, 0, len(graphQLSourceOwners))
	features = append(features, tmsFeatures()...)
	features = append(features, workforceFeatures()...)
	features = append(features, platformFeatures()...)

	return withGraphQLOwnership(features)
}

func withGraphQLOwnership(features []Feature) []Feature {
	for i := range features {
		features[i].GraphQLSources = graphQLSourcesFor(features[i].Key)
		features[i].GraphQLRootFields = graphQLRootFieldsFor(features[i].Key)
	}

	return features
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
			RequiresFeatures: []FeatureKey{FeatureCoreTMS, FeatureWorkforceCore},
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
			Description:      "Equipment, tractors, trailers, and fleet reference data.",
			RequiresFeatures: []FeatureKey{FeatureCoreTMS},
			Routes:           fleetRouteRefs(),
		},
		{
			Key:                    FeatureSettlement,
			ProductKey:             ProductTMS,
			Name:                   "Settlement",
			Description:            "Driver and carrier settlement runs, escrow, advances, deductions, and disputes.",
			RequiresFeatures:       []FeatureKey{FeatureCoreTMS, FeatureWorkforceCore},
			LegacyGrantingFeatures: []FeatureKey{FeatureCoreTMS},
		},
		{
			Key:                    FeatureDriverPortal,
			ProductKey:             ProductTMS,
			Name:                   "Driver Portal",
			Description:            "Driver-facing loads, pay, hours of service, and document capture.",
			RequiresFeatures:       []FeatureKey{FeatureCoreTMS, FeatureWorkforceCore},
			LegacyGrantingFeatures: []FeatureKey{FeatureCoreTMS},
			Routes:                 driverPortalRouteRefs(),
		},
	}
}

func workforceFeatures() []Feature {
	return []Feature{
		{
			Key:                    FeatureWorkforceCore,
			ProductKey:             ProductWorkforce,
			Name:                   "Workforce Core",
			Description:            "Worker records, profiles, job positions, org structure, holidays, and onboarding checklists.",
			LegacyGrantingFeatures: []FeatureKey{FeatureFleetMaintenance},
			Routes:                 workforceCoreRouteRefs(),
			Meters:                 []MeterKey{MeterWorkforceManagedWorkers},
		},
		{
			Key:              FeatureWorkforceCompliance,
			ProductKey:       ProductWorkforce,
			Name:             "Workforce Compliance",
			Description:      "Driver qualification files, credentials, DOT testing, random pools, and Clearinghouse queries.",
			RequiresFeatures: []FeatureKey{FeatureWorkforceCore, FeatureDocumentManagement},
			Meters:           []MeterKey{MeterWorkforceComplianceScreens},
		},
		{
			Key:              FeatureWorkforceTimeOff,
			ProductKey:       ProductWorkforce,
			Name:             "Time Off",
			Description:      "PTO policies, accrual ledgers, balances, and leave case administration.",
			RequiresFeatures: []FeatureKey{FeatureWorkforceCore},
		},
		{
			Key:              FeatureWorkforceTimeTracking,
			ProductKey:       ProductWorkforce,
			Name:             "Time and Scheduling",
			Description:      "Timesheets, shift templates, swaps, and rota scheduling.",
			RequiresFeatures: []FeatureKey{FeatureWorkforceCore},
		},
		{
			Key:              FeatureWorkforceTalent,
			ProductKey:       ProductWorkforce,
			Name:             "Talent",
			Description:      "Training courses and records, performance reviews, discipline, and recognition.",
			RequiresFeatures: []FeatureKey{FeatureWorkforceCore},
		},
		{
			Key:              FeatureWorkforceSafety,
			ProductKey:       ProductWorkforce,
			Name:             "Workforce Safety",
			Description:      "Injury reporting, OSHA annual summaries, safety events, and fleet safety standing.",
			RequiresFeatures: []FeatureKey{FeatureWorkforceCore},
		},
		{
			Key:              FeatureWorkforceBenefits,
			ProductKey:       ProductWorkforce,
			Name:             "Benefits",
			Description:      "Benefit plan administration and worker enrollments.",
			RequiresFeatures: []FeatureKey{FeatureWorkforceCore},
		},
		{
			Key:              FeatureWorkforceSelfService,
			ProductKey:       ProductWorkforce,
			Name:             "Employee Self Service",
			Description:      "Worker portal access, policy acknowledgements, and profile change requests.",
			RequiresFeatures: []FeatureKey{FeatureWorkforceCore},
		},
	}
}

func platformFeatures() []Feature {
	return []Feature{
		{
			Key:         FeatureDocumentManagement,
			ProductKey:  ProductPlatform,
			Name:        "Document Management",
			Description: "Document upload, storage, packets, and parsing rules.",
			Routes:      documentManagementRouteRefs(),
			Meters:      []MeterKey{MeterDocumentUploads},
		},
		{
			Key:                    FeatureAgentAutomation,
			ProductKey:             ProductPlatform,
			Name:                   "Agent Automation",
			Description:            "Agent runs, proposals, and exception handling.",
			LegacyGrantingFeatures: []FeatureKey{FeatureCoreTMS},
			Routes:                 agentAutomationRouteRefs(),
		},
		{
			Key:                    FeatureAdministration,
			ProductKey:             ProductPlatform,
			Name:                   "Administration",
			Description:            "Users, roles, permissions, audit history, custom fields, and table configuration.",
			LegacyGrantingFeatures: []FeatureKey{FeatureCoreTMS},
			Routes:                 administrationRouteRefs(),
		},
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
			ProductKey:  ProductPlatform,
			FeatureKey:  FeatureDocumentManagement,
			Name:        "Document Uploads",
			Description: "Documents uploaded into tenant storage.",
			Unit:        "document",
		},
		{
			Key:         MeterWorkforceManagedWorkers,
			ProductKey:  ProductWorkforce,
			FeatureKey:  FeatureWorkforceCore,
			Name:        "Managed Workers",
			Description: "Active workers under workforce management, the per-seat billing unit.",
			Unit:        "worker",
		},
		{
			Key:         MeterWorkforceComplianceScreens,
			ProductKey:  ProductWorkforce,
			FeatureKey:  FeatureWorkforceCompliance,
			Name:        "Compliance Screenings",
			Description: "DOT test orders and Clearinghouse queries run against workers.",
			Unit:        "screening",
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
