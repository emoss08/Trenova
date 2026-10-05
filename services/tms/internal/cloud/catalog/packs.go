package catalog

import "github.com/emoss08/trenova/internal/core/domain/platformcatalog"

const (
	PackProfessional  = platformcatalog.PackKey("professional")
	PackDispatchIntel = platformcatalog.PackKey("dispatch-intel")
	PackVisibility    = platformcatalog.PackKey("visibility")
	PackCompliance    = platformcatalog.PackKey("compliance")
	PackDriverDash    = platformcatalog.PackKey("driver-dash")
	PackSettlement    = platformcatalog.PackKey("settlement")
	PackWorkforce     = platformcatalog.PackKey("workforce")
	PackEDI           = platformcatalog.PackKey("edi")
	PackDocumentAI    = platformcatalog.PackKey("document-intelligence")
)

func (p *StaticProvider) Packs() []platformcatalog.Pack {
	return append(platformPacks(), addOnPacks()...)
}

func platformPacks() []platformcatalog.Pack {
	return []platformcatalog.Pack{
		{
			Key:         PackProfessional,
			Name:        "Professional",
			Description: "The core transportation management platform: shipments, dispatch, billing, accounting, fleet, documents, and currency conversion.",
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureCoreTMS,
				platformcatalog.FeatureDispatch,
				platformcatalog.FeatureBilling,
				platformcatalog.FeatureAccounting,
				platformcatalog.FeatureFleetMaintenance,
				platformcatalog.FeatureDocumentManagement,
				platformcatalog.FeatureWorkforceCore,
				platformcatalog.FeatureAdministration,
				platformcatalog.FeatureGlobalSearch,
				platformcatalog.FeatureAPIKeys,
				platformcatalog.FeatureExchangeRateIntegration,
				platformcatalog.FeatureRealtimeNotifications,
			},
		},
		{
			Key:         PackWorkforce,
			Name:        "Workforce",
			Description: "The complete workforce management suite, sellable without any transportation management feature.",
			Standalone:  true,
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureWorkforceCore,
				platformcatalog.FeatureWorkforceCompliance,
				platformcatalog.FeatureWorkforceTimeOff,
				platformcatalog.FeatureWorkforceTimeTracking,
				platformcatalog.FeatureWorkforceTalent,
				platformcatalog.FeatureWorkforceSafety,
				platformcatalog.FeatureWorkforceBenefits,
				platformcatalog.FeatureWorkforceSelfService,
				platformcatalog.FeatureDocumentManagement,
				platformcatalog.FeatureAdministration,
				platformcatalog.FeatureGlobalSearch,
				platformcatalog.FeatureRealtimeNotifications,
			},
		},
	}
}

func addOnPacks() []platformcatalog.Pack {
	return []platformcatalog.Pack{
		{
			Key:         PackDispatchIntel,
			Name:        "Dispatch Intelligence",
			Description: "Analytics, agent automation, and decision support layered on the dispatch board.",
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureAnalytics,
				platformcatalog.FeatureAgentAutomation,
			},
		},
		{
			Key:         PackVisibility,
			Name:        "Visibility",
			Description: "Telematics-backed tracking, geofencing, and customer-facing shipment visibility.",
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureSamsaraIntegration,
				platformcatalog.FeatureGoogleMapsIntegration,
				platformcatalog.FeatureTableChangeAlerts,
			},
		},
		{
			Key:         PackCompliance,
			Name:        "Compliance",
			Description: "DOT qualification files, drug and alcohol program administration, and workplace safety records.",
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureWorkforceCore,
				platformcatalog.FeatureWorkforceCompliance,
				platformcatalog.FeatureWorkforceSafety,
				platformcatalog.FeatureDocumentManagement,
			},
		},
		{
			Key:           PackDriverDash,
			Name:          "Driver Dash",
			Description:   "The driver-facing portal for loads, pay, hours of service, documents, and time-off requests.",
			RequiresPacks: []platformcatalog.PackKey{PackProfessional},
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureDriverPortal,
				platformcatalog.FeatureWorkforceCore,
				platformcatalog.FeatureWorkforceSelfService,
				platformcatalog.FeatureWorkforceTimeOff,
				platformcatalog.FeatureWorkforceTimeTracking,
			},
		},
		{
			Key:           PackSettlement,
			Name:          "Settlement",
			Description:   "Driver and carrier settlement runs, escrow, advances, deductions, and dispute handling.",
			RequiresPacks: []platformcatalog.PackKey{PackProfessional},
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureSettlement,
				platformcatalog.FeatureWorkforceCore,
			},
		},
		{
			Key:           PackEDI,
			Name:          "EDI",
			Description:   "Trading-partner EDI: partners, mapping profiles, templates, transfers, and X12 document exchange.",
			RequiresPacks: []platformcatalog.PackKey{PackProfessional},
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureEDIIntegration,
			},
		},
		{
			Key:         PackDocumentAI,
			Name:        "Document Intelligence",
			Description: "OCR-backed document classification, extraction, and review.",
			Features: []platformcatalog.FeatureKey{
				platformcatalog.FeatureDocumentManagement,
				platformcatalog.FeatureDocumentIntelligence,
			},
		},
	}
}
