package costingcontrol

import "errors"

type CategoryType string

const (
	CategoryTypeDriverWages       = CategoryType("DriverWages")
	CategoryTypeDriverBenefits    = CategoryType("DriverBenefits")
	CategoryTypeFuel              = CategoryType("Fuel")
	CategoryTypeEquipmentPayments = CategoryType("EquipmentPayments")
	CategoryTypeMaintenance       = CategoryType("Maintenance")
	CategoryTypeInsurance         = CategoryType("Insurance")
	CategoryTypeTires             = CategoryType("Tires")
	CategoryTypeTolls             = CategoryType("Tolls")
	CategoryTypePermitsLicenses   = CategoryType("PermitsLicenses")
	CategoryTypeOverhead          = CategoryType("Overhead")
	CategoryTypeCustom            = CategoryType("Custom")
)

func (c CategoryType) String() string {
	return string(c)
}

func CategoryTypeFromString(v string) (CategoryType, error) {
	switch v {
	case "DriverWages":
		return CategoryTypeDriverWages, nil
	case "DriverBenefits":
		return CategoryTypeDriverBenefits, nil
	case "Fuel":
		return CategoryTypeFuel, nil
	case "EquipmentPayments":
		return CategoryTypeEquipmentPayments, nil
	case "Maintenance":
		return CategoryTypeMaintenance, nil
	case "Insurance":
		return CategoryTypeInsurance, nil
	case "Tires":
		return CategoryTypeTires, nil
	case "Tolls":
		return CategoryTypeTolls, nil
	case "PermitsLicenses":
		return CategoryTypePermitsLicenses, nil
	case "Overhead":
		return CategoryTypeOverhead, nil
	case "Custom":
		return CategoryTypeCustom, nil
	default:
		return "", errors.New("invalid cost category type")
	}
}

type CostBehavior string

const (
	CostBehaviorFixed    = CostBehavior("Fixed")
	CostBehaviorVariable = CostBehavior("Variable")
)

func (b CostBehavior) String() string {
	return string(b)
}

func CostBehaviorFromString(v string) (CostBehavior, error) {
	switch v {
	case "Fixed":
		return CostBehaviorFixed, nil
	case "Variable":
		return CostBehaviorVariable, nil
	default:
		return "", errors.New("invalid cost behavior")
	}
}

type RateSource string

const (
	RateSourceBenchmark = RateSource("Benchmark")
	RateSourceOverride  = RateSource("Override")
	RateSourceGLActual  = RateSource("GLActual")
)

func (s RateSource) String() string {
	return string(s)
}

func RateSourceFromString(v string) (RateSource, error) {
	switch v {
	case "Benchmark":
		return RateSourceBenchmark, nil
	case "Override":
		return RateSourceOverride, nil
	case "GLActual":
		return RateSourceGLActual, nil
	default:
		return "", errors.New("invalid cost rate source")
	}
}

type EffectiveRateSource string

const (
	EffectiveRateSourceBenchmark = EffectiveRateSource("Benchmark")
	EffectiveRateSourceOverride  = EffectiveRateSource("Override")
	EffectiveRateSourceGLActual  = EffectiveRateSource("GLActual")
	EffectiveRateSourceLiveIndex = EffectiveRateSource("LiveIndex")
)

func (s EffectiveRateSource) String() string {
	return string(s)
}
