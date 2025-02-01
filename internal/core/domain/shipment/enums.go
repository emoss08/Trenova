package shipment

type Status string

const (
	StatusNew       = Status("New")
	StatusInTransit = Status("InTransit")
	StatusDelayed   = Status("Delayed")
	StatusCompleted = Status("Completed")
	StatusBilled    = Status("Billed")
	StatusCanceled  = Status("Canceled")
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
