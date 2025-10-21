package shipment

import "errors"

type Status string

const (
	StatusNew                = Status("New")
	StatusPartiallyAssigned  = Status("PartiallyAssigned")
	StatusAssigned           = Status("Assigned")
	StatusInTransit          = Status("InTransit")
	StatusDelayed            = Status("Delayed")
	StatusPartiallyCompleted = Status("PartiallyCompleted")
	StatusCompleted          = Status("Completed")
	StatusReadyToBill        = Status("ReadyToBill")
	StatusReviewRequired     = Status("ReviewRequired")
	StatusBilled             = Status("Billed")
	StatusCanceled           = Status("Canceled")
)

func StatusFromString(status string) (Status, error) {
	switch status {
	case "New":
		return StatusNew, nil
	case "PartiallyAssigned":
		return StatusPartiallyAssigned, nil
	case "Assigned":
		return StatusAssigned, nil
	case "InTransit":
		return StatusInTransit, nil
	case "Delayed":
		return StatusDelayed, nil
	case "PartiallyCompleted":
		return StatusPartiallyCompleted, nil
	case "Completed":
		return StatusCompleted, nil
	case "Billed":
		return StatusBilled, nil
	case "Canceled":
		return StatusCanceled, nil
	case "ReadyToBill":
		return StatusReadyToBill, nil
	case "ReviewRequired":
		return StatusReviewRequired, nil
	default:
		return "", errors.New("invalid shipment status")
	}
}

type RatingMethod string

const (
	RatingMethodFlatRate        = RatingMethod("FlatRate")
	RatingMethodPerMile         = RatingMethod("PerMile")
	RatingMethodPerStop         = RatingMethod("PerStop")
	RatingMethodPerPound        = RatingMethod("PerPound")
	RatingMethodPerPallet       = RatingMethod("PerPallet")
	RatingMethodPerLinearFoot   = RatingMethod("PerLinearFoot")
	RatingMethodOther           = RatingMethod("Other")
	RatingMethodFormulaTemplate = RatingMethod("FormulaTemplate")
)

type EntryMethod string

const (
	EntryMethodManual     = EntryMethod("Manual")
	EntryMethodElectronic = EntryMethod("Electronic")
)

type StopType string

const (
	StopTypePickup        = StopType("Pickup")
	StopTypeDelivery      = StopType("Delivery")
	StopTypeSplitPickup   = StopType("SplitPickup")
	StopTypeSplitDelivery = StopType("SplitDelivery")
)

type StopStatus string

const (
	StopStatusNew       = StopStatus("New")
	StopStatusInTransit = StopStatus("InTransit")
	StopStatusCompleted = StopStatus("Completed")
	StopStatusCanceled  = StopStatus("Canceled")
)

type MoveStatus string

const (
	MoveStatusNew       = MoveStatus("New")
	MoveStatusAssigned  = MoveStatus("Assigned")
	MoveStatusInTransit = MoveStatus("InTransit")
	MoveStatusCompleted = MoveStatus("Completed")
	MoveStatusCanceled  = MoveStatus("Canceled")
)

type AssignmentStatus string

const (
	AssignmentStatusNew        = AssignmentStatus("New")
	AssignmentStatusInProgress = AssignmentStatus("InProgress")
	AssignmentStatusCompleted  = AssignmentStatus("Completed")
	AssignmentStatusCanceled   = AssignmentStatus("Canceled")
)

type AutoAssignmentStrategy string

const (
	AutoAssignmentStrategyProximity     = AutoAssignmentStrategy("Proximity")
	AutoAssignmentStrategyAvailability  = AutoAssignmentStrategy("Availability")
	AutoAssignmentStrategyLoadBalancing = AutoAssignmentStrategy("LoadBalancing")
)

type ComplianceEnforcementLevel string

const (
	ComplianceEnforcementLevelWarning = ComplianceEnforcementLevel("Warning")
	ComplianceEnforcementLevelBlock   = ComplianceEnforcementLevel("Block")
	ComplianceEnforcementLevelAudit   = ComplianceEnforcementLevel("Audit")
)

type HoldType string

const (
	HoldOperational = HoldType("OperationalHold")
	HoldCompliance  = HoldType("ComplianceHold")
	HoldCustomer    = HoldType("CustomerHold")
	HoldFinance     = HoldType("FinanceHold")
)

type HoldSeverity string

const (
	SeverityInfo     = HoldSeverity("Informational")
	SeverityAdvisory = HoldSeverity("Advisory")
	SeverityBlocking = HoldSeverity("Blocking")
)

type HoldSource string

const (
	SourceUser = HoldSource("User")
	SourceRule = HoldSource("Rule")
	SourceAPI  = HoldSource("API")
	SourceELD  = HoldSource("ELD")
	SourceEDI  = HoldSource("EDI")
)
