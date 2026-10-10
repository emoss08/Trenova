package sim

import "slices"

const (
	webhookEventAddressCreated = "AddressCreated"
	webhookEventAddressDeleted = "AddressDeleted"
	webhookEventAddressUpdated = "AddressUpdated"
)

var webhookEventTypes = []string{
	"AddressCreated",
	"AddressDeleted",
	"AddressUpdated",
	"AlertIncident",
	"AlertObjectEvent",
	"DocumentSubmitted",
	"DriverCreated",
	"DriverUpdated",
	"DvirSubmitted",
	"EngineFaultOff",
	"EngineFaultOn",
	"FormSubmitted",
	"FormUpdated",
	"GatewayUnplugged",
	"GeofenceEntry",
	"GeofenceExit",
	"IssueCreated",
	"MissingDvirPastDue",
	"PredictiveMaintenanceAlert",
	"RouteStopArrival",
	"RouteStopDeparture",
	"RouteStopEarlyLateArrival",
	"RouteStopEtaUpdated",
	"RouteStopResequence",
	"SevereSpeedingEnded",
	"SevereSpeedingStarted",
	"ShipmentTrackingEvent",
	"SpeedingEventEnded",
	"SpeedingEventStarted",
	"SuddenFuelLevelDrop",
	"SuddenFuelLevelRise",
	"VehicleCreated",
	"VehicleUpdated",
	"VisualSearchMatch",
	"WorkOrderCreatedOrChanged",
}

func isWebhookEventType(eventType string) bool {
	_, found := slices.BinarySearch(webhookEventTypes, eventType)
	return found
}
