package shipment

type Status string

const (
	// StatusNew indicates that the shipment has been created but not yet processed
	StatusNew = Status("New")

	// StatusPartiallyAssigned indicates that the shipment has multiple moves
	// but only some of them have been assigned to a worker
	StatusPartiallyAssigned = Status("PartiallyAssigned")

	// StatusAssigned indicates that all moves on the shipment have been assigned
	// to workers(s)
	StatusAssigned = Status("Assigned")

	// StatusInTransit indicates that the shipment is currently being processed
	StatusInTransit = Status("InTransit")

	// StatusDelayed indicates that the shipment is delayed
	StatusDelayed = Status("Delayed")

	// StatusPartiallyCompleted indicates that not all moves on the shipment
	// have been completed
	StatusPartiallyCompleted = Status("PartiallyCompleted")

	// StatusCompleted indicates that the shipment has been completed
	StatusCompleted = Status("Completed")

	// StatusBilled indicates that the shipment has been billed
	StatusBilled = Status("Billed")

	// StatusCanceled indicates that the shipment has been canceled
	StatusCanceled = Status("Canceled")
)

type RatingMethod string

const (
	// FlatRate is the cost per shipment
	RatingMethodFlatRate = RatingMethod("FlatRate")

	// PerMile is the cost per mile of the shipment
	RatingMethodPerMile = RatingMethod("PerMile")

	// PerStop is the cost per stop of the shipment
	RatingMethodPerStop = RatingMethod("PerStop")

	// PerPound is the cost per pound of the shipment
	RatingMethodPerPound = RatingMethod("PerPound")

	// PerPallet is the cost per pallet position used
	RatingMethodPerPallet = RatingMethod("PerPallet")

	// PerLinearFoot is the cost based on the linear feet of trailer space used.
	// This is commonly used for LTL shipments, Flatbed haulers, and specific
	// commodities that are measured in linear feet.
	RatingMethodPerLinearFoot = RatingMethod("PerLinearFoot")

	// Other takes the rating units and the rate and does multiplication
	// of the two to get the total cost
	RatingMethodOther = RatingMethod("Other")
)

type EntryMethod string

const (
	// EntryMethodManual is when a user manually enters the shipment
	EntryMethodManual = EntryMethod("Manual")

	// EntryMethodElectronic is when a the system automatically enters the shipment
	EntryMethodElectronic = EntryMethod("Electronic")
)

type StopType string

const (
	// StopTypePickup is when a user manually enters the shipment
	StopTypePickup = StopType("Pickup")

	// StopTypeDelivery is when a the system automatically enters the shipment
	StopTypeDelivery = StopType("Delivery")

	// StopTypeSplitPickup is when a user manually enters the shipment
	StopTypeSplitPickup = StopType("SplitPickup")

	// StopTypeSplitDelivery is when a the system automatically enters the shipment
	StopTypeSplitDelivery = StopType("SplitDelivery")
)

type StopStatus string

const (
	// StopStatusNew indicates that the stop has been created but not yet processed
	StopStatusNew = StopStatus("New")

	// StopStatusInTransit indicates that the stop is currently being processed
	StopStatusInTransit = StopStatus("InTransit")

	// StopStatusCompleted indicates that the stop has been completed
	StopStatusCompleted = StopStatus("Completed")

	// StopStatusCanceled indicates that the stop has been canceled
	StopStatusCanceled = StopStatus("Canceled")
)

type MoveStatus string

const (
	// MoveStatusNew indicates that the move has been created but not yet processed
	MoveStatusNew = MoveStatus("New")

	// MoveStatusAssigned indicates that the move has been assigned to a worker
	MoveStatusAssigned = MoveStatus("Assigned")

	// MoveStatusInTransit indicates that the move is currently being processed
	MoveStatusInTransit = MoveStatus("InTransit")
	MoveStatusCompleted = MoveStatus("Completed")

	// MoveStatusCanceled indicates that the move has been canceled
	MoveStatusCanceled = MoveStatus("Canceled")
)

type AssignmentStatus string

const (
	// AssignmentStatusNew indicates that the assignment has been created but not yet processed
	AssignmentStatusNew = AssignmentStatus("New")

	// AssignmentStatusInProgress indicates that the assignment is currently being processed
	AssignmentStatusInProgress = AssignmentStatus("InProgress")

	// AssignmentStatusCompleted indicates that the assignment has been completed
	AssignmentStatusCompleted = AssignmentStatus("Completed")

	// AssignmentStatusCanceled indicates that the assignment has been canceled
	AssignmentStatusCanceled = AssignmentStatus("Canceled")
)
