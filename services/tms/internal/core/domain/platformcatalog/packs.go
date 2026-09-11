package platformcatalog

const (
	PackProfessional  = PackKey("professional")
	PackDispatchIntel = PackKey("dispatch-intel")
	PackVisibility    = PackKey("visibility")
	PackCompliance    = PackKey("compliance")
	PackDriverDash    = PackKey("driver-dash")
	PackSettlement    = PackKey("settlement")
	PackWorkforce     = PackKey("workforce")
	PackEDI           = PackKey("edi")
	PackDocumentAI    = PackKey("document-intelligence")
)

func (p *StaticProvider) Packs() []Pack {
	return []Pack{
		{
			Key:         PackProfessional,
			Name:        "Professional",
			Description: "The core transportation management platform: shipments, dispatch, billing, accounting, fleet, documents, and currency conversion.",
			Features: []FeatureKey{
				FeatureCoreTMS,
				FeatureDispatch,
				FeatureBilling,
				FeatureAccounting,
				FeatureFleetMaintenance,
				FeatureDocumentManagement,
				FeatureWorkforceCore,
				FeatureAdministration,
				FeatureGlobalSearch,
				FeatureAPIKeys,
				FeatureExchangeRateIntegration,
				FeatureRealtimeNotifications,
			},
		},
		{
			Key:         PackDispatchIntel,
			Name:        "Dispatch Intelligence",
			Description: "Analytics, agent automation, and decision support layered on the dispatch board.",
			Features: []FeatureKey{
				FeatureAnalytics,
				FeatureAgentAutomation,
			},
		},
		{
			Key:         PackVisibility,
			Name:        "Visibility",
			Description: "Telematics-backed tracking, geofencing, and customer-facing shipment visibility.",
			Features: []FeatureKey{
				FeatureSamsaraIntegration,
				FeatureGoogleMapsIntegration,
				FeatureTableChangeAlerts,
			},
		},
		{
			Key:         PackCompliance,
			Name:        "Compliance",
			Description: "DOT qualification files, drug and alcohol program administration, and workplace safety records.",
			Features: []FeatureKey{
				FeatureWorkforceCore,
				FeatureWorkforceCompliance,
				FeatureWorkforceSafety,
				FeatureDocumentManagement,
			},
		},
		{
			Key:           PackDriverDash,
			Name:          "Driver Dash",
			Description:   "The driver-facing portal for loads, pay, hours of service, documents, and time-off requests.",
			RequiresPacks: []PackKey{PackProfessional},
			Features: []FeatureKey{
				FeatureDriverPortal,
				FeatureWorkforceCore,
				FeatureWorkforceSelfService,
				FeatureWorkforceTimeOff,
				FeatureWorkforceTimeTracking,
			},
		},
		{
			Key:           PackSettlement,
			Name:          "Settlement",
			Description:   "Driver and carrier settlement runs, escrow, advances, deductions, and dispute handling.",
			RequiresPacks: []PackKey{PackProfessional},
			Features: []FeatureKey{
				FeatureSettlement,
				FeatureWorkforceCore,
			},
		},
		{
			Key:         PackWorkforce,
			Name:        "Workforce",
			Description: "The complete workforce management suite, sellable without any transportation management feature.",
			Standalone:  true,
			Features: []FeatureKey{
				FeatureWorkforceCore,
				FeatureWorkforceCompliance,
				FeatureWorkforceTimeOff,
				FeatureWorkforceTimeTracking,
				FeatureWorkforceTalent,
				FeatureWorkforceSafety,
				FeatureWorkforceBenefits,
				FeatureWorkforceSelfService,
				FeatureDocumentManagement,
				FeatureAdministration,
				FeatureGlobalSearch,
				FeatureRealtimeNotifications,
			},
		},
		{
			Key:           PackEDI,
			Name:          "EDI",
			Description:   "Trading-partner EDI: partners, mapping profiles, templates, transfers, and X12 document exchange.",
			RequiresPacks: []PackKey{PackProfessional},
			Features: []FeatureKey{
				FeatureEDIIntegration,
			},
		},
		{
			Key:         PackDocumentAI,
			Name:        "Document Intelligence",
			Description: "OCR-backed document classification, extraction, and review.",
			Features: []FeatureKey{
				FeatureDocumentManagement,
				FeatureDocumentIntelligence,
			},
		},
	}
}
