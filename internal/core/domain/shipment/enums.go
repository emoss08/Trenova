package shipment

type Status string

const (
	New       = Status("New")
	InTransit = Status("InTransit")
	Delayed   = Status("Delayed")
	Completed = Status("Completed")
	Billed    = Status("Billed")
	Canceled  = Status("Canceled")
)

type RatingMethod string

const (
	// FlatRate is the cost per shipment
	FlatRate = RatingMethod("FlatRate")

	// PerMile is the cost per mile of the shipment
	PerMile = RatingMethod("PerMile")

	// PerStop is the cost per stop of the shipment
	PerStop = RatingMethod("PerStop")

	// PerPound is the cost per pound of the shipment
	PerPound = RatingMethod("PerPound")

	// PerPallet is the cost per pallet position used
	PerPallet = RatingMethod("PerPallet")

	// PerLinearFoot is the cost based on the linear feet of trailer space used.
	// This is commonly used for LTL shipments, Flatbed haulers, and specific
	// commodities that are measured in linear feet.
	PerLinearFoot = RatingMethod("PerLinearFoot")

	// Other takes the rating units and the rate and does multiplication
	// of the two to get the total cost
	Other = RatingMethod("Other")
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
