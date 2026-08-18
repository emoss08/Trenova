package ratematrix

// MaxDimensions caps a matrix at four axes.
//
// Four covers every tariff shape that occurs in practice — origin zone by
// destination zone by weight break by equipment is the widest real example, and
// class rating substitutes class for equipment rather than adding to it. The
// cap is what lets a cell be a fixed set of columns with one composite index
// behind it, instead of a row-per-axis join that no index can serve cheaply. A
// fifth axis is a sign the tariff wants to be two matrices.
const MaxDimensions = 4

// ValueKind says what a cell's number means, which is what tells the engine how
// to turn it into money.
type ValueKind string

const (
	ValueKindFlatRate    = ValueKind("FlatRate")
	ValueKindPerMile     = ValueKind("PerMile")
	ValueKindPerCwt      = ValueKind("PerCwt")
	ValueKindPerPiece    = ValueKind("PerPiece")
	ValueKindPerStop     = ValueKind("PerStop")
	ValueKindPercent     = ValueKind("Percent")
	ValueKindDiscount    = ValueKind("Discount")
	ValueKindMinimumOnly = ValueKind("MinimumOnly")
)

func (vk ValueKind) String() string {
	return string(vk)
}

func (vk ValueKind) IsValid() bool {
	switch vk {
	case ValueKindFlatRate,
		ValueKindPerMile,
		ValueKindPerCwt,
		ValueKindPerPiece,
		ValueKindPerStop,
		ValueKindPercent,
		ValueKindDiscount,
		ValueKindMinimumOnly:
		return true
	default:
		return false
	}
}

// DimensionKind names what one axis of a matrix varies by.
type DimensionKind string

const (
	DimensionKindZone          = DimensionKind("Zone")
	DimensionKindZip3          = DimensionKind("Zip3")
	DimensionKindZip5          = DimensionKind("Zip5")
	DimensionKindState         = DimensionKind("State")
	DimensionKindCountry       = DimensionKind("Country")
	DimensionKindWeightBreak   = DimensionKind("WeightBreak")
	DimensionKindDistance      = DimensionKind("Distance")
	DimensionKindPieceCount    = DimensionKind("PieceCount")
	DimensionKindLinearFeet    = DimensionKind("LinearFeet")
	DimensionKindFreightClass  = DimensionKind("FreightClass")
	DimensionKindEquipmentType = DimensionKind("EquipmentType")
	DimensionKindServiceType   = DimensionKind("ServiceType")
	DimensionKindCustom        = DimensionKind("Custom")
	// DimensionKindQuantity is a bare number with no shipment fact behind it.
	// It exists for matrices addressed from a formula's lookup() call, where
	// the expression supplies the quantity itself; a lane cannot price against
	// it because there is nothing on the shipment to read.
	DimensionKindQuantity = DimensionKind("Quantity")
)

func (dk DimensionKind) String() string {
	return string(dk)
}

func (dk DimensionKind) IsValid() bool {
	switch dk {
	case DimensionKindZone,
		DimensionKindZip3,
		DimensionKindZip5,
		DimensionKindState,
		DimensionKindCountry,
		DimensionKindWeightBreak,
		DimensionKindDistance,
		DimensionKindPieceCount,
		DimensionKindLinearFeet,
		DimensionKindFreightClass,
		DimensionKindEquipmentType,
		DimensionKindServiceType,
		DimensionKindCustom,
		DimensionKindQuantity:
		return true
	default:
		return false
	}
}

// IsNumeric reports whether values on this axis are quantities, which is what
// decides whether the axis can be banded into ranges.
func (dk DimensionKind) IsNumeric() bool {
	switch dk { //nolint:exhaustive // every other axis is keyed, not banded
	case DimensionKindWeightBreak,
		DimensionKindDistance,
		DimensionKindPieceCount,
		DimensionKindLinearFeet,
		DimensionKindQuantity:
		return true
	default:
		return false
	}
}

// MatchMode says whether an axis is looked up by equality or by which band a
// quantity falls into.
type MatchMode string

const (
	MatchModeExact = MatchMode("Exact")
	MatchModeRange = MatchMode("Range")
)

func (mm MatchMode) String() string {
	return string(mm)
}

func (mm MatchMode) IsValid() bool {
	switch mm {
	case MatchModeExact, MatchModeRange:
		return true
	default:
		return false
	}
}
