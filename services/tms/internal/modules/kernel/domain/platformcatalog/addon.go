package platformcatalog

type AddOnKey string

const (
	AddOnBase                 = AddOnKey("BASE")
	AddOnAccounting           = AddOnKey("ACC")
	AddOnRating               = AddOnKey("RTE")
	AddOnFuel                 = AddOnKey("FUL")
	AddOnDispatchIntelligence = AddOnKey("DSP")
	AddOnVisibility           = AddOnKey("VIS")
	AddOnBrokerage            = AddOnKey("BRK")
	AddOnDriverPay            = AddOnKey("PAY")
	AddOnHR                   = AddOnKey("HRS")
	AddOnSafety               = AddOnKey("SAF")
	AddOnDocumentIntelligence = AddOnKey("DOC")
	AddOnAI                   = AddOnKey("AI")
	AddOnEDI                  = AddOnKey("EDI")
	AddOnSecurity             = AddOnKey("SEC")
)

type AddOn struct {
	Key  AddOnKey `json:"key"`
	Name string   `json:"name"`
}

var addOns = []AddOn{
	{Key: AddOnAccounting, Name: "Accounting & Cash"},
	{Key: AddOnRating, Name: "Rating & Revenue Recovery"},
	{Key: AddOnFuel, Name: "Fuel & IFTA"},
	{Key: AddOnDispatchIntelligence, Name: "Dispatch Intelligence"},
	{Key: AddOnVisibility, Name: "Visibility & Tracking"},
	{Key: AddOnBrokerage, Name: "Carrier & Brokerage"},
	{Key: AddOnDriverPay, Name: "Driver Pay"},
	{Key: AddOnHR, Name: "HR Suite"},
	{Key: AddOnSafety, Name: "Safety & Compliance"},
	{Key: AddOnDocumentIntelligence, Name: "Document Intelligence"},
	{Key: AddOnAI, Name: "Trenova AI"},
	{Key: AddOnEDI, Name: "EDI & Integrations"},
	{Key: AddOnSecurity, Name: "Security & Identity"},
}

var addOnIndex = indexAddOns(addOns)

func AddOns() []AddOn {
	out := make([]AddOn, len(addOns))
	copy(out, addOns)
	return out
}

func LookupAddOn(key AddOnKey) (AddOn, bool) {
	addOn, ok := addOnIndex[key]
	return addOn, ok
}

func AddOnFor(key FeatureKey) (AddOn, bool) {
	feature, ok := featureIndex[key]
	if !ok || feature.IsBase() {
		return AddOn{}, false
	}

	addOn, ok := addOnIndex[feature.AddOn]
	return addOn, ok
}

func indexAddOns(list []AddOn) map[AddOnKey]AddOn {
	index := make(map[AddOnKey]AddOn, len(list))
	for _, addOn := range list {
		index[addOn.Key] = addOn
	}
	return index
}
