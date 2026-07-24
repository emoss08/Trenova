package telematics

type DutyStatus string

const (
	DutyStatusOffDuty            = DutyStatus("offDuty")
	DutyStatusSleeperBed         = DutyStatus("sleeperBed")
	DutyStatusDriving            = DutyStatus("driving")
	DutyStatusOnDuty             = DutyStatus("onDuty")
	DutyStatusYardMove           = DutyStatus("yardMove")
	DutyStatusPersonalConveyance = DutyStatus("personalConveyance")
)

func (d DutyStatus) IsValid() bool {
	switch d {
	case DutyStatusOffDuty,
		DutyStatusSleeperBed,
		DutyStatusDriving,
		DutyStatusOnDuty,
		DutyStatusYardMove,
		DutyStatusPersonalConveyance:
		return true
	}
	return false
}

type EngineState string

const (
	EngineStateOn   = EngineState("On")
	EngineStateOff  = EngineState("Off")
	EngineStateIdle = EngineState("Idle")
)

type FeedType string

const (
	FeedTypeVehicleStats = FeedType("vehicleStats")
)

type EventType string

const (
	EventTypeGeofenceEntry = EventType("GeofenceEntry")
	EventTypeGeofenceExit  = EventType("GeofenceExit")
	EventTypeAlertIncident = EventType("AlertIncident")
)
