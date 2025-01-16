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

func (rm RatingMethod) String() string {
	return string(rm)
}

func (rm RatingMethod) IsValid() bool {
	switch rm {
	case FlatRate, PerMile, PerStop, PerPound, PerPallet, PerLinearFoot, Other:
		return true
	}
	return false
}

type EntryMethod string

const (
	EntryMethodManual     = EntryMethod("Manual")
	EntryMethodElectronic = EntryMethod("Electronic")
)
