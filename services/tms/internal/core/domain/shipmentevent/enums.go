package shipmentevent

type Type string

const (
	TypeShipmentCreated      = Type("ShipmentCreated")
	TypeShipmentUpdated      = Type("ShipmentUpdated")
	TypeStatusChanged        = Type("StatusChanged")
	TypeShipmentCanceled     = Type("ShipmentCanceled")
	TypeShipmentUncanceled   = Type("ShipmentUncanceled")
	TypeOwnershipTransferred = Type("OwnershipTransferred")

	TypeMoveStatusChanged = Type("MoveStatusChanged")
	TypeMoveDeparted      = Type("MoveDeparted")
	TypeMoveArrived       = Type("MoveArrived")

	TypeStopCompleted = Type("StopCompleted")

	TypeDriverAssigned   = Type("DriverAssigned")
	TypeDriverReassigned = Type("DriverReassigned")
	TypeDriverUnassigned = Type("DriverUnassigned")

	TypeHoldPlaced   = Type("HoldPlaced")
	TypeHoldUpdated  = Type("HoldUpdated")
	TypeHoldReleased = Type("HoldReleased")

	TypeCommentPosted = Type("CommentPosted")
)

type Severity string

const (
	SeverityDanger  = Severity("danger")
	SeveritySuccess = Severity("success")
	SeverityBrand   = Severity("brand")
	SeverityInfo    = Severity("info")
	SeverityMuted   = Severity("muted")
)

type ActorType string

const (
	ActorUser   = ActorType("user")
	ActorAPIKey = ActorType("apikey")
	ActorSystem = ActorType("system")
	ActorEDI    = ActorType("edi")
)
