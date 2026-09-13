package platformcatalog

var graphQLShellSources = map[GraphQLSource]struct{}{
	"attention.graphqls":          {},
	"home_layout.graphqls":        {},
	"notification.graphqls":       {},
	"select_options.graphqls":     {},
	"shared.graphqls":             {},
	"sidebar_preference.graphqls": {},
	"us_state.graphqls":           {},
}

var selectOptionResourceOwners = map[string]FeatureKey{
	"ACCESSORIAL_CHARGE":           FeatureBilling,
	"ORGANIZATION":                 FeatureAdministration,
	"ROLE":                         FeatureAdministration,
	"USER":                         FeatureAdministration,
	"ACCOUNT_TYPE":                 FeatureAccounting,
	"BENEFIT_PLAN":                 FeatureWorkforceBenefits,
	"CARRIER":                      FeatureCoreTMS,
	"COMMODITY":                    FeatureCoreTMS,
	"CUSTOMER":                     FeatureCoreTMS,
	"DETENTION_POLICY":             FeatureDispatch,
	"DISTANCE_PROFILE":             FeatureDispatch,
	"DOCUMENT_TYPE":                FeatureDocumentManagement,
	"EDI_COMMUNICATION_PROFILE":    FeatureEDIIntegration,
	"EDI_CONNECTION":               FeatureEDIIntegration,
	"EDI_DOCUMENT_TYPE":            FeatureEDIIntegration,
	"EDI_MAPPING_PROFILE":          FeatureEDIIntegration,
	"EDI_PARTNER":                  FeatureEDIIntegration,
	"EDI_PARTNER_DOCUMENT_PROFILE": FeatureEDIIntegration,
	"EDI_TEMPLATE":                 FeatureEDIIntegration,
	"EDI_TRANSACTION_SET":          FeatureEDIIntegration,
	"EDI_TRANSFER":                 FeatureEDIIntegration,
	"EMAIL_PROFILE":                FeatureAdministration,
	"EQUIPMENT_MANUFACTURER":       FeatureFleetMaintenance,
	"EQUIPMENT_TYPE":               FeatureFleetMaintenance,
	"FISCAL_PERIOD":                FeatureAccounting,
	"FISCAL_YEAR":                  FeatureAccounting,
	"FLEET_CODE":                   FeatureFleetMaintenance,
	"FORMULA_TEMPLATE":             FeatureBilling,
	"FUEL_CARD":                    FeatureFleetMaintenance,
	"FUEL_INDEX":                   FeatureBilling,
	"FUEL_SURCHARGE_PROGRAM":       FeatureBilling,
	"GL_ACCOUNT":                   FeatureAccounting,
	"HAZARDOUS_MATERIAL":           FeatureCoreTMS,
	"IFTA_FUEL_TYPE":               FeatureAccounting,
	"IFTA_JURISDICTION":            FeatureAccounting,
	"JOB_POSITION":                 FeatureWorkforceCore,
	"LOCATION":                     FeatureDispatch,
	"LOCATION_CATEGORY":            FeatureDispatch,
	"ORDER":                        FeatureDispatch,
	"PAY_CODE":                     FeatureSettlement,
	"PAY_PROFILE":                  FeatureSettlement,
	"PERFORMANCE_REVIEW_TEMPLATE":  FeatureWorkforceTalent,
	"PTO_POLICY":                   FeatureWorkforceTimeOff,
	"RATE_AGREEMENT":               FeatureBilling,
	"RATE_MATRIX":                  FeatureBilling,
	"RATE_ZONE":                    FeatureBilling,
	"SERVICE_FAILURE_REASON_CODE":  FeatureDispatch,
	"SERVICE_TYPE":                 FeatureCoreTMS,
	"SHIFT_TEMPLATE":               FeatureWorkforceTimeTracking,
	"SHIPMENT":                     FeatureDispatch,
	"SHIPMENT_TYPE":                FeatureDispatch,
	"TRACTOR":                      FeatureFleetMaintenance,
	"TRAILER":                      FeatureFleetMaintenance,
	"TRAINING_COURSE":              FeatureWorkforceTalent,
	"WORKER":                       FeatureWorkforceCore,
	"WORKER_CREDENTIAL_TYPE":       FeatureWorkforceCompliance,
	"WORKER_POLICY":                FeatureWorkforceSelfService,
}

var selectOptionShellResources = map[string]struct{}{
	"US_STATE": {},
}

func graphQLSourceIsShell(source GraphQLSource) bool {
	_, ok := graphQLShellSources[source]
	return ok
}

func PolicyForSelectOptionResource(resource string) RoutePolicy {
	if _, ok := selectOptionShellResources[resource]; ok {
		return RoutePolicy{AccessClass: RouteAccessClassAccountShell}
	}
	if featureKey, ok := selectOptionResourceOwners[resource]; ok {
		return RoutePolicy{AccessClass: RouteAccessClassProduct, FeatureKey: featureKey}
	}

	return RoutePolicy{AccessClass: RouteAccessClassUnclassified}
}

func SelectOptionResources() []string {
	resources := make([]string, 0, len(selectOptionResourceOwners)+len(selectOptionShellResources))
	for resource := range selectOptionResourceOwners {
		resources = append(resources, resource)
	}
	for resource := range selectOptionShellResources {
		resources = append(resources, resource)
	}

	return resources
}
