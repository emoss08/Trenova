package ratematrix

import "strings"

// MaxDimensions caps a matrix at four axes.
//
// Four covers every tariff shape that occurs in practice — origin zone by
// destination zone by weight break by equipment is the widest real example, and
// class rating substitutes class for equipment rather than adding to it. The
// cap is what lets a cell be a fixed set of columns with one composite index
// behind it, instead of a row-per-axis join that no index can serve cheaply. A
// fifth axis is a sign the tariff wants to be two matrices.
const MaxDimensions = 4

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

// KeyNormalization is how an exact-match axis reads its keys before comparing
// them. Both the stored cell key and the value a formula looks up pass
// through the same rule, so a ZIP+4 finds a ZIP3 zone and a lowercase lane
// code finds an uppercase cell.
type KeyNormalization string

const (
	KeyNormalizationNone  = KeyNormalization("None")
	KeyNormalizationTrim  = KeyNormalization("Trim")
	KeyNormalizationUpper = KeyNormalization("Upper")
	KeyNormalizationZip3  = KeyNormalization("Zip3")
)

const zip3Length = 3

func (kn KeyNormalization) String() string {
	return string(kn)
}

func (kn KeyNormalization) IsValid() bool {
	switch kn {
	case KeyNormalizationNone, KeyNormalizationTrim, KeyNormalizationUpper, KeyNormalizationZip3:
		return true
	default:
		return false
	}
}

// Apply normalises a key by this rule. An unset mode behaves as None so a
// dimension created before the column existed keeps matching as it did.
func (kn KeyNormalization) Apply(key string) string {
	switch kn {
	case KeyNormalizationTrim:
		return strings.TrimSpace(key)
	case KeyNormalizationUpper:
		return strings.ToUpper(strings.TrimSpace(key))
	case KeyNormalizationZip3:
		trimmed := strings.ToUpper(strings.TrimSpace(key))
		runes := []rune(trimmed)
		if len(runes) > zip3Length {
			return string(runes[:zip3Length])
		}
		return trimmed
	default:
		return key
	}
}

// RangeOverflow is what a banded axis does with a quantity no band covers.
// Error is the strict default; ClampToTopBand prices anything past the last
// band at the last band, which is how most tariffs treat weights beyond
// their heaviest break; Nearest also covers quantities below the first band
// and gaps between bands.
type RangeOverflow string

const (
	RangeOverflowError          = RangeOverflow("Error")
	RangeOverflowClampToTopBand = RangeOverflow("ClampToTopBand")
	RangeOverflowNearest        = RangeOverflow("Nearest")
)

func (ro RangeOverflow) String() string {
	return string(ro)
}

func (ro RangeOverflow) IsValid() bool {
	switch ro {
	case RangeOverflowError, RangeOverflowClampToTopBand, RangeOverflowNearest:
		return true
	default:
		return false
	}
}
