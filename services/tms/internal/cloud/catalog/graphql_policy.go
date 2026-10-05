package catalog

import "github.com/emoss08/trenova/internal/core/domain/platformcatalog"

var graphQLShellSources = map[platformcatalog.GraphQLSource]struct{}{
	"attention.graphqls":          {},
	"home_layout.graphqls":        {},
	"notification.graphqls":       {},
	"select_options.graphqls":     {},
	"shared.graphqls":             {},
	"sidebar_preference.graphqls": {},
	"us_state.graphqls":           {},
}

var selectOptionResourceOwners = map[string]platformcatalog.FeatureKey{
	"ACCESSORIAL_CHARGE":           platformcatalog.FeatureBilling,
	"ORGANIZATION":                 platformcatalog.FeatureAdministration,
	"ROLE":                         platformcatalog.FeatureAdministration,
	"USER":                         platformcatalog.FeatureAdministration,
	"ACCOUNT_TYPE":                 platformcatalog.FeatureAccounting,
	"BENEFIT_PLAN":                 platformcatalog.FeatureWorkforceBenefits,
	"CARRIER":                      platformcatalog.FeatureCoreTMS,
	"COMMODITY":                    platformcatalog.FeatureCoreTMS,
	"CUSTOMER":                     platformcatalog.FeatureCoreTMS,
	"DETENTION_POLICY":             platformcatalog.FeatureDispatch,
	"DISTANCE_PROFILE":             platformcatalog.FeatureDispatch,
	"CAPTURE_DEVICE":               platformcatalog.FeatureDocumentManagement,
	"CAPTURE_PROFILE":              platformcatalog.FeatureDocumentManagement,
	"DOCUMENT_TYPE":                platformcatalog.FeatureDocumentManagement,
	"EDI_COMMUNICATION_PROFILE":    platformcatalog.FeatureEDIIntegration,
	"EDI_CONNECTION":               platformcatalog.FeatureEDIIntegration,
	"EDI_DOCUMENT_TYPE":            platformcatalog.FeatureEDIIntegration,
	"EDI_MAPPING_PROFILE":          platformcatalog.FeatureEDIIntegration,
	"EDI_PARTNER":                  platformcatalog.FeatureEDIIntegration,
	"EDI_PARTNER_DOCUMENT_PROFILE": platformcatalog.FeatureEDIIntegration,
	"EDI_TEMPLATE":                 platformcatalog.FeatureEDIIntegration,
	"EDI_TRANSACTION_SET":          platformcatalog.FeatureEDIIntegration,
	"EDI_TRANSFER":                 platformcatalog.FeatureEDIIntegration,
	"EMAIL_PROFILE":                platformcatalog.FeatureAdministration,
	"EQUIPMENT_MANUFACTURER":       platformcatalog.FeatureFleetMaintenance,
	"EQUIPMENT_TYPE":               platformcatalog.FeatureFleetMaintenance,
	"FISCAL_PERIOD":                platformcatalog.FeatureAccounting,
	"FISCAL_YEAR":                  platformcatalog.FeatureAccounting,
	"FLEET_CODE":                   platformcatalog.FeatureFleetMaintenance,
	"FORMULA_TEMPLATE":             platformcatalog.FeatureBilling,
	"FUEL_CARD":                    platformcatalog.FeatureFleetMaintenance,
	"FUEL_INDEX":                   platformcatalog.FeatureBilling,
	"FUEL_SURCHARGE_PROGRAM":       platformcatalog.FeatureBilling,
	"GL_ACCOUNT":                   platformcatalog.FeatureAccounting,
	"HAZARDOUS_MATERIAL":           platformcatalog.FeatureCoreTMS,
	"IFTA_FUEL_TYPE":               platformcatalog.FeatureAccounting,
	"IFTA_JURISDICTION":            platformcatalog.FeatureAccounting,
	"JOB_POSITION":                 platformcatalog.FeatureWorkforceCore,
	"LOCATION":                     platformcatalog.FeatureDispatch,
	"LOCATION_CATEGORY":            platformcatalog.FeatureDispatch,
	"ORDER":                        platformcatalog.FeatureDispatch,
	"PAY_CODE":                     platformcatalog.FeatureSettlement,
	"PAY_PROFILE":                  platformcatalog.FeatureSettlement,
	"PERFORMANCE_REVIEW_TEMPLATE":  platformcatalog.FeatureWorkforceTalent,
	"PTO_POLICY":                   platformcatalog.FeatureWorkforceTimeOff,
	"RATE_AGREEMENT":               platformcatalog.FeatureBilling,
	"RATE_MATRIX":                  platformcatalog.FeatureBilling,
	"RATE_ZONE":                    platformcatalog.FeatureBilling,
	"SERVICE_FAILURE_REASON_CODE":  platformcatalog.FeatureDispatch,
	"SERVICE_TYPE":                 platformcatalog.FeatureCoreTMS,
	"SHIFT_TEMPLATE":               platformcatalog.FeatureWorkforceTimeTracking,
	"SHIPMENT":                     platformcatalog.FeatureDispatch,
	"SHIPMENT_TYPE":                platformcatalog.FeatureDispatch,
	"TRACTOR":                      platformcatalog.FeatureFleetMaintenance,
	"TRAILER":                      platformcatalog.FeatureFleetMaintenance,
	"TRAINING_COURSE":              platformcatalog.FeatureWorkforceTalent,
	"WORKER":                       platformcatalog.FeatureWorkforceCore,
	"WORKER_CREDENTIAL_TYPE":       platformcatalog.FeatureWorkforceCompliance,
	"WORKER_POLICY":                platformcatalog.FeatureWorkforceSelfService,
}

var selectOptionShellResources = map[string]struct{}{
	"US_STATE": {},
}

func graphQLSourceIsShell(source platformcatalog.GraphQLSource) bool {
	_, ok := graphQLShellSources[source]
	return ok
}

func PolicyForSelectOptionResource(resource string) platformcatalog.RoutePolicy {
	if _, ok := selectOptionShellResources[resource]; ok {
		return platformcatalog.RoutePolicy{
			AccessClass: platformcatalog.RouteAccessClassAccountShell,
		}
	}
	if featureKey, ok := selectOptionResourceOwners[resource]; ok {
		return platformcatalog.RoutePolicy{
			AccessClass: platformcatalog.RouteAccessClassProduct,
			FeatureKey:  featureKey,
		}
	}

	return platformcatalog.RoutePolicy{AccessClass: platformcatalog.RouteAccessClassUnclassified}
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
