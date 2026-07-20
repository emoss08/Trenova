package fuelsurcharge

import "errors"

type IndexSource string

const (
	IndexSourceEIA    = IndexSource("EIA")
	IndexSourceCustom = IndexSource("Custom")
)

func (s IndexSource) String() string {
	return string(s)
}

func IndexSourceFromString(v string) (IndexSource, error) {
	switch v {
	case "EIA":
		return IndexSourceEIA, nil
	case "Custom":
		return IndexSourceCustom, nil
	default:
		return "", errors.New("invalid fuel index source")
	}
}

type FuelType string

const (
	FuelTypeDiesel   = FuelType("Diesel")
	FuelTypeGasoline = FuelType("Gasoline")
)

func (f FuelType) String() string {
	return string(f)
}

func FuelTypeFromString(v string) (FuelType, error) {
	switch v {
	case "Diesel":
		return FuelTypeDiesel, nil
	case "Gasoline":
		return FuelTypeGasoline, nil
	default:
		return "", errors.New("invalid fuel type")
	}
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

func ProgramMethodFromString(v string) (ProgramMethod, error) {
	switch v {
	case "PerMileStep":
		return ProgramMethodPerMileStep, nil
	case "PerMileMPG":
		return ProgramMethodPerMileMPG, nil
	case "TablePerMile":
		return ProgramMethodTablePerMile, nil
	case "TablePercent":
		return ProgramMethodTablePercent, nil
	case "TableFlat":
		return ProgramMethodTableFlat, nil
	default:
		return "", errors.New("invalid fuel surcharge program method")
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

func PercentBasisFromString(v string) (PercentBasis, error) {
	switch v {
	case "Linehaul":
		return PercentBasisLinehaul, nil
	case "LinehaulPlusAccessorials":
		return PercentBasisLinehaulPlusAccessorials, nil
	default:
		return "", errors.New("invalid fuel surcharge percent basis")
	}
}

type DateBasis string

const (
	DateBasisPickupDate = DateBasis("PickupDate")
	DateBasisTenderDate = DateBasis("TenderDate")
)

func (d DateBasis) String() string {
	return string(d)
}

func DateBasisFromString(v string) (DateBasis, error) {
	switch v {
	case "PickupDate":
		return DateBasisPickupDate, nil
	case "TenderDate":
		return DateBasisTenderDate, nil
	default:
		return "", errors.New("invalid fuel surcharge date basis")
	}
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

func StepRoundingFromString(v string) (StepRounding, error) {
	switch v {
	case "Up":
		return StepRoundingUp, nil
	case "Down":
		return StepRoundingDown, nil
	case "Nearest":
		return StepRoundingNearest, nil
	default:
		return "", errors.New("invalid fuel surcharge step rounding")
	}
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

func RateRoundingFromString(v string) (RateRounding, error) {
	switch v {
	case "HalfUp":
		return RateRoundingHalfUp, nil
	case "Up":
		return RateRoundingUp, nil
	case "Down":
		return RateRoundingDown, nil
	default:
		return "", errors.New("invalid fuel surcharge rate rounding")
	}
}

type MissingPriceFallback string

const (
	FallbackUseLatestAvailable = MissingPriceFallback("UseLatestAvailable")
	FallbackSkip               = MissingPriceFallback("Skip")
)

func (f MissingPriceFallback) String() string {
	return string(f)
}

func MissingPriceFallbackFromString(v string) (MissingPriceFallback, error) {
	switch v {
	case "UseLatestAvailable":
		return FallbackUseLatestAvailable, nil
	case "Skip":
		return FallbackSkip, nil
	default:
		return "", errors.New("invalid fuel surcharge missing price fallback")
	}
}

type ProgramStatus string

const (
	ProgramStatusActive   = ProgramStatus("Active")
	ProgramStatusInactive = ProgramStatus("Inactive")
)

func (s ProgramStatus) String() string {
	return string(s)
}

func ProgramStatusFromString(v string) (ProgramStatus, error) {
	switch v {
	case "Active":
		return ProgramStatusActive, nil
	case "Inactive":
		return ProgramStatusInactive, nil
	default:
		return "", errors.New("invalid fuel surcharge program status")
	}
}
