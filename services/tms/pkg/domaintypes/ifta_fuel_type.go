package domaintypes

type IFTAFuelType string

const (
	IFTAFuelTypeDiesel      = IFTAFuelType("Diesel")
	IFTAFuelTypeGasoline    = IFTAFuelType("Gasoline")
	IFTAFuelTypeGasohol     = IFTAFuelType("Gasohol")
	IFTAFuelTypePropane     = IFTAFuelType("Propane")
	IFTAFuelTypeCNG         = IFTAFuelType("CNG")
	IFTAFuelTypeLNG         = IFTAFuelType("LNG")
	IFTAFuelTypeEthanol     = IFTAFuelType("Ethanol")
	IFTAFuelTypeMethanol    = IFTAFuelType("Methanol")
	IFTAFuelTypeE85         = IFTAFuelType("E85")
	IFTAFuelTypeM85         = IFTAFuelType("M85")
	IFTAFuelTypeA55         = IFTAFuelType("A55")
	IFTAFuelTypeBiodiesel   = IFTAFuelType("Biodiesel")
	IFTAFuelTypeElectricity = IFTAFuelType("Electricity")
	IFTAFuelTypeHydrogen    = IFTAFuelType("Hydrogen")
	IFTAFuelTypeDEF         = IFTAFuelType("DEF")
	IFTAFuelTypeReefer      = IFTAFuelType("Reefer")
	IFTAFuelTypeOther       = IFTAFuelType("Other")
)

var allIFTAFuelTypes = [...]IFTAFuelType{
	IFTAFuelTypeDiesel,
	IFTAFuelTypeGasoline,
	IFTAFuelTypeGasohol,
	IFTAFuelTypePropane,
	IFTAFuelTypeCNG,
	IFTAFuelTypeLNG,
	IFTAFuelTypeEthanol,
	IFTAFuelTypeMethanol,
	IFTAFuelTypeE85,
	IFTAFuelTypeM85,
	IFTAFuelTypeA55,
	IFTAFuelTypeBiodiesel,
	IFTAFuelTypeElectricity,
	IFTAFuelTypeHydrogen,
	IFTAFuelTypeDEF,
	IFTAFuelTypeReefer,
	IFTAFuelTypeOther,
}

func (f IFTAFuelType) String() string { return string(f) }

func (f IFTAFuelType) IsValid() bool {
	switch f {
	case IFTAFuelTypeDiesel, IFTAFuelTypeGasoline, IFTAFuelTypeGasohol, IFTAFuelTypePropane,
		IFTAFuelTypeCNG, IFTAFuelTypeLNG, IFTAFuelTypeEthanol, IFTAFuelTypeMethanol,
		IFTAFuelTypeE85, IFTAFuelTypeM85, IFTAFuelTypeA55, IFTAFuelTypeBiodiesel,
		IFTAFuelTypeElectricity, IFTAFuelTypeHydrogen, IFTAFuelTypeDEF, IFTAFuelTypeReefer,
		IFTAFuelTypeOther:
		return true
	default:
		return false
	}
}

func (f IFTAFuelType) Label() string {
	switch f {
	case IFTAFuelTypeDiesel:
		return "Diesel"
	case IFTAFuelTypeGasoline:
		return "Gasoline"
	case IFTAFuelTypeGasohol:
		return "Gasohol"
	case IFTAFuelTypePropane:
		return "Propane (LPG)"
	case IFTAFuelTypeCNG:
		return "Compressed natural gas"
	case IFTAFuelTypeLNG:
		return "Liquefied natural gas"
	case IFTAFuelTypeEthanol:
		return "Ethanol"
	case IFTAFuelTypeMethanol:
		return "Methanol"
	case IFTAFuelTypeE85:
		return "E-85"
	case IFTAFuelTypeM85:
		return "M-85"
	case IFTAFuelTypeA55:
		return "A-55"
	case IFTAFuelTypeBiodiesel:
		return "Biodiesel"
	case IFTAFuelTypeElectricity:
		return "Electricity"
	case IFTAFuelTypeHydrogen:
		return "Hydrogen"
	case IFTAFuelTypeDEF:
		return "Diesel exhaust fluid"
	case IFTAFuelTypeReefer:
		return "Reefer fuel"
	case IFTAFuelTypeOther:
		return "Other"
	default:
		return string(f)
	}
}

func (f IFTAFuelType) CountsForIFTA() bool {
	switch f {
	case IFTAFuelTypeDiesel, IFTAFuelTypeGasoline, IFTAFuelTypeGasohol, IFTAFuelTypePropane,
		IFTAFuelTypeCNG, IFTAFuelTypeLNG, IFTAFuelTypeEthanol, IFTAFuelTypeMethanol,
		IFTAFuelTypeE85, IFTAFuelTypeM85, IFTAFuelTypeA55, IFTAFuelTypeBiodiesel,
		IFTAFuelTypeElectricity, IFTAFuelTypeHydrogen:
		return true
	default:
		return false
	}
}

func (f IFTAFuelType) IsGaseous() bool {
	switch f {
	case IFTAFuelTypePropane, IFTAFuelTypeCNG, IFTAFuelTypeLNG, IFTAFuelTypeHydrogen:
		return true
	default:
		return false
	}
}

func IFTAFuelTypes() []IFTAFuelType {
	out := make([]IFTAFuelType, 0, len(allIFTAFuelTypes))
	for _, fuelType := range allIFTAFuelTypes {
		if fuelType.CountsForIFTA() {
			out = append(out, fuelType)
		}
	}
	return out
}

func AllIFTAFuelTypes() []IFTAFuelType {
	out := make([]IFTAFuelType, len(allIFTAFuelTypes))
	copy(out, allIFTAFuelTypes[:])
	return out
}
