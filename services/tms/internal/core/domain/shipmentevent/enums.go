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

	TypeCarrierAssigned   = Type("CarrierAssigned")
	TypeCarrierUnassigned = Type("CarrierUnassigned")

	TypeTenderOffered         = Type("TenderOffered")
	TypeTenderAccepted        = Type("TenderAccepted")
	TypeTenderDeclined        = Type("TenderDeclined")
	TypeTenderExpired         = Type("TenderExpired")
	TypeTenderWithdrawn       = Type("TenderWithdrawn")
	TypeTenderNeedsReview     = Type("TenderNeedsReview")
	TypeRoutingGuideExhausted = Type("RoutingGuideExhausted")
	TypeTenderLateResponse    = Type("TenderLateResponse")
	TypeTenderDeliveryFailed  = Type("TenderDeliveryFailed")

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

func (v Type) IsValid() bool {
	switch v {
	case TypeShipmentCreated,
		TypeShipmentUpdated,
		TypeStatusChanged,
		TypeShipmentCanceled,
		TypeShipmentUncanceled,
		TypeOwnershipTransferred,
		TypeMoveStatusChanged,
		TypeMoveDeparted,
		TypeMoveArrived,
		TypeStopCompleted,
		TypeDriverAssigned,
		TypeDriverReassigned,
		TypeDriverUnassigned,
		TypeCarrierAssigned,
		TypeCarrierUnassigned,
		TypeTenderOffered,
		TypeTenderAccepted,
		TypeTenderDeclined,
		TypeTenderExpired,
		TypeTenderWithdrawn,
		TypeTenderNeedsReview,
		TypeRoutingGuideExhausted,
		TypeTenderLateResponse,
		TypeTenderDeliveryFailed,
		TypeHoldPlaced,
		TypeHoldUpdated,
		TypeHoldReleased,
		TypeCommentPosted:
		return true
	default:
		return false
	}
}

func (v Severity) IsValid() bool {
	switch v {
	case SeverityDanger,
		SeveritySuccess,
		SeverityBrand,
		SeverityInfo,
		SeverityMuted:
		return true
	default:
		return false
	}
}

func (v ActorType) IsValid() bool {
	switch v {
	case ActorUser,
		ActorAPIKey,
		ActorSystem,
		ActorEDI:
		return true
	default:
		return false
	}
}
