package fuelsurcharge

type IndexSource string

const (
	IndexSourceEIA    = IndexSource("EIA")
	IndexSourceCustom = IndexSource("Custom")
)

func (s IndexSource) String() string {
	return string(s)
}

type FuelType string

const (
	FuelTypeDiesel   = FuelType("Diesel")
	FuelTypeGasoline = FuelType("Gasoline")
)

func (f FuelType) String() string {
	return string(f)
}

type ProgramMethod string

const (
	ProgramMethodPerMileStep  = ProgramMethod("PerMileStep")
	ProgramMethodPerMileMPG   = ProgramMethod("PerMileMPG")
	ProgramMethodTablePerMile = ProgramMethod("TablePerMile")
	ProgramMethodTablePercent = ProgramMethod("TablePercent")
	ProgramMethodTableFlat    = ProgramMethod("TableFlat")
)

func (m ProgramMethod) String() string {
	return string(m)
}

func (m ProgramMethod) IsTableMethod() bool {
	switch m {
	case ProgramMethodTablePerMile, ProgramMethodTablePercent, ProgramMethodTableFlat:
		return true
	case ProgramMethodPerMileStep, ProgramMethodPerMileMPG:
		return false
	default:
		return false
	}
}

func (m ProgramMethod) IsPerMile() bool {
	switch m {
	case ProgramMethodPerMileStep, ProgramMethodPerMileMPG, ProgramMethodTablePerMile:
		return true
	case ProgramMethodTablePercent, ProgramMethodTableFlat:
		return false
	default:
		return false
	}
}

type PercentBasis string

const (
	PercentBasisLinehaul                 = PercentBasis("Linehaul")
	PercentBasisLinehaulPlusAccessorials = PercentBasis("LinehaulPlusAccessorials")
)

func (b PercentBasis) String() string {
	return string(b)
}

type DateBasis string

const (
	DateBasisPickupDate = DateBasis("PickupDate")
	DateBasisTenderDate = DateBasis("TenderDate")
)

func (d DateBasis) String() string {
	return string(d)
}

type StepRounding string

const (
	StepRoundingUp      = StepRounding("Up")
	StepRoundingDown    = StepRounding("Down")
	StepRoundingNearest = StepRounding("Nearest")
)

func (r StepRounding) String() string {
	return string(r)
}

type RateRounding string

const (
	RateRoundingHalfUp = RateRounding("HalfUp")
	RateRoundingUp     = RateRounding("Up")
	RateRoundingDown   = RateRounding("Down")
)

func (r RateRounding) String() string {
	return string(r)
}

type MissingPriceFallback string

const (
	FallbackUseLatestAvailable = MissingPriceFallback("UseLatestAvailable")
	FallbackSkip               = MissingPriceFallback("Skip")
)

func (f MissingPriceFallback) String() string {
	return string(f)
}

type ProgramStatus string

const (
	ProgramStatusActive   = ProgramStatus("Active")
	ProgramStatusInactive = ProgramStatus("Inactive")
)

func (s ProgramStatus) String() string {
	return string(s)
}
