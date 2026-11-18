package platformcatalog

type Meter struct {
	Key   MeterKey `json:"key"`
	Label string   `json:"label"`
}

var meters = []Meter{
	{Key: MeterShipmentsTotal, Label: "Shipments"},
	{Key: MeterDocumentIntelligencePages, Label: "Document Intelligence pages"},
	{Key: MeterDocumentStorageBytes, Label: "Document storage"},
	{Key: MeterOrganizationsTotal, Label: "Organizations"},
	{Key: MeterUserSeats, Label: "User seats"},
	{Key: MeterRecurringShipmentSeries, Label: "Recurring shipment series"},
	{Key: MeterCustomersTotal, Label: "Customers"},
	{Key: MeterLocationsTotal, Label: "Locations"},
	{Key: MeterWorkersTotal, Label: "Workers"},
	{Key: MeterTractorsTotal, Label: "Tractors"},
	{Key: MeterTrailersTotal, Label: "Trailers"},
	{Key: MeterDocumentUploads, Label: "Document uploads"},
	{Key: MeterDocumentFileBytes, Label: "Largest upload"},
	{Key: MeterDocumentAIClassifications, Label: "Document classifications"},
	{Key: MeterAIAssistantMessages, Label: "Assistant messages"},
	{Key: MeterAPIRequests, Label: "API requests"},
	{Key: MeterGlobalSearchQueries, Label: "Searches"},
	{Key: MeterTableChangeEvents, Label: "Change alerts"},
	{Key: MeterIntegrationSyncRuns, Label: "Integration syncs"},
	{Key: MeterAnalyticsQueries, Label: "Report runs"},
	{Key: MeterWorkforceManagedWorkers, Label: "Managed workers"},
	{Key: MeterWorkforceComplianceScreens, Label: "Compliance screenings"},
}

var meterPositions = indexMeters(meters)

func Meters() []Meter {
	out := make([]Meter, len(meters))
	copy(out, meters)
	return out
}

func LookupMeter(key MeterKey) (meter Meter, position int, ok bool) {
	position, ok = meterPositions[key]
	if !ok {
		return Meter{}, 0, false
	}
	return meters[position], position, true
}

func indexMeters(list []Meter) map[MeterKey]int {
	index := make(map[MeterKey]int, len(list))
	for i, meter := range list {
		index[meter.Key] = i
	}
	return index
}
