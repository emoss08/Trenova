package platformcatalog

type Feature struct {
	Key   FeatureKey `json:"key"`
	Name  string     `json:"name"`
	AddOn AddOnKey   `json:"addOn"`
}

func (f Feature) IsBase() bool {
	return f.AddOn == AddOnBase
}

var features = []Feature{
	{Key: FeatureCoreTMS, Name: "Freight", AddOn: AddOnBase},
	{Key: FeatureDispatch, Name: "Dispatch", AddOn: AddOnBase},
	{Key: FeatureFleetMaintenance, Name: "Fleet", AddOn: AddOnBase},
	{Key: FeatureBilling, Name: "Billing", AddOn: AddOnBase},
	{Key: FeatureWorkforceCore, Name: "People", AddOn: AddOnBase},
	{Key: FeatureDocumentManagement, Name: "Documents", AddOn: AddOnBase},
	{Key: FeatureAnalytics, Name: "Reports and dashboards", AddOn: AddOnBase},
	{Key: FeatureAdministration, Name: "Administration", AddOn: AddOnBase},
	{Key: FeatureAPIKeys, Name: "API keys", AddOn: AddOnBase},
	{Key: FeatureGlobalSearch, Name: "Search", AddOn: AddOnBase},
	{Key: FeatureRealtimeNotifications, Name: "Notifications", AddOn: AddOnBase},
	{Key: FeatureExchangeRateIntegration, Name: "Exchange rates", AddOn: AddOnBase},
	{Key: FeatureAccounting, Name: "Accounting", AddOn: AddOnAccounting},
	{Key: FeatureFactoring, Name: "Factoring", AddOn: AddOnAccounting},
	{Key: FeatureRating, Name: "Rating", AddOn: AddOnRating},
	{Key: FeatureDetention, Name: "Detention", AddOn: AddOnRating},
	{Key: FeatureFuelTax, Name: "Fuel and IFTA", AddOn: AddOnFuel},
	{Key: FeatureSettlement, Name: "Driver settlements", AddOn: AddOnDriverPay},
	{Key: FeatureDriverPortal, Name: "Driver portal", AddOn: AddOnDriverPay},
	{Key: FeatureWorkforceTimeOff, Name: "Time off", AddOn: AddOnHR},
	{Key: FeatureWorkforceTimeTracking, Name: "Time tracking", AddOn: AddOnHR},
	{Key: FeatureWorkforceTalent, Name: "Training and performance", AddOn: AddOnHR},
	{Key: FeatureWorkforceBenefits, Name: "Benefits", AddOn: AddOnHR},
	{Key: FeatureWorkforceSelfService, Name: "Employee self-service", AddOn: AddOnHR},
	{Key: FeatureWorkforceCompliance, Name: "Driver compliance", AddOn: AddOnSafety},
	{Key: FeatureWorkforceSafety, Name: "Safety events", AddOn: AddOnSafety},
	{Key: FeatureCarrierBrokerage, Name: "Carrier brokerage", AddOn: AddOnBrokerage},
	{Key: FeatureEDIIntegration, Name: "EDI", AddOn: AddOnEDI},
	{Key: FeatureTableChangeAlerts, Name: "Change alerts", AddOn: AddOnVisibility},
	{Key: FeatureSamsaraIntegration, Name: "Samsara", AddOn: AddOnVisibility},
	{Key: FeatureGoogleMapsIntegration, Name: "Google Maps", AddOn: AddOnVisibility},
	{Key: FeatureCustomerTracking, Name: "Customer tracking", AddOn: AddOnVisibility},
	{
		Key:   FeatureDispatchIntelligence,
		Name:  "Dispatch intelligence",
		AddOn: AddOnDispatchIntelligence,
	},
	{
		Key:   FeatureDocumentIntelligence,
		Name:  "Document intelligence",
		AddOn: AddOnDocumentIntelligence,
	},
	{Key: FeatureAIDesk, Name: "Desk", AddOn: AddOnAI},
	{Key: FeatureAIWatchtower, Name: "Watchtower", AddOn: AddOnAI},
	{Key: FeatureAITriage, Name: "Inbound triage", AddOn: AddOnAI},
	{Key: FeatureAgentAutomation, Name: "Agents", AddOn: AddOnAI},
	{Key: FeatureAIAuditTrail, Name: "AI audit trail", AddOn: AddOnAI},
	{Key: FeatureIdentity, Name: "Single sign-on and provisioning", AddOn: AddOnSecurity},
}

var featureIndex = indexFeatures(features)

func Features() []Feature {
	out := make([]Feature, len(features))
	copy(out, features)
	return out
}

func Lookup(key FeatureKey) (Feature, bool) {
	feature, ok := featureIndex[key]
	return feature, ok
}

func indexFeatures(list []Feature) map[FeatureKey]Feature {
	index := make(map[FeatureKey]Feature, len(list))
	for _, feature := range list {
		index[feature.Key] = feature
	}
	return index
}
