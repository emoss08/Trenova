package platformcatalog

const (
	PackProfessional  = PackKey("professional")
	PackDispatchIntel = PackKey("dispatch-intel")
	PackVisibility    = PackKey("visibility")
	PackCompliance    = PackKey("compliance")
	PackDriverDash    = PackKey("driver-dash")
	PackSettlement    = PackKey("settlement")
	PackWorkforce     = PackKey("workforce")
)

func (p *StaticProvider) Packs() []Pack {
	return []Pack{
		{
			Key:         PackProfessional,
			Name:        "Professional",
			Description: "The core transportation management platform: shipments, dispatch, billing, accounting, fleet, and documents.",
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
			Key:         PackDriverDash,
			Name:        "Driver Dash",
			Description: "The driver-facing portal for loads, pay, hours of service, documents, and time-off requests.",
			Features: []FeatureKey{
				FeatureDriverPortal,
				FeatureWorkforceCore,
				FeatureWorkforceSelfService,
				FeatureWorkforceTimeOff,
				FeatureWorkforceTimeTracking,
			},
		},
		{
			Key:         PackSettlement,
			Name:        "Settlement",
			Description: "Driver and carrier settlement runs, escrow, advances, deductions, and dispute handling.",
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
	}
}
