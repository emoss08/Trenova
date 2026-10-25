package ratematrix

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

// RateMatrixCell is one priced intersection of a matrix's axes.
//
// The four key/range column pairs are positional: pair zero belongs to the
// dimension at position zero, and so on. Exact axes fill the key column and
// leave the bounds null; range axes do the reverse. Flattening the axes into
// fixed columns is what makes a single composite index able to answer a lookup,
// which a row-per-axis design could not do at tariff scale.
type RateMatrixCell struct {
	keel.Table `keel:"rate_matrix_cells"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:rmc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	RateMatrixID   pulid.ID `json:"rateMatrixId"   keel:"rate_matrix_id"`
	VersionNumber  int64    `json:"versionNumber"  keel:"version_number"`

	D0Key string `json:"d0Key" keel:"d0_key"`
	D1Key string `json:"d1Key" keel:"d1_key"`
	D2Key string `json:"d2Key" keel:"d2_key"`
	D3Key string `json:"d3Key" keel:"d3_key"`

	D0KeyTo string `json:"d0KeyTo" keel:"d0_key_to"`
	D1KeyTo string `json:"d1KeyTo" keel:"d1_key_to"`
	D2KeyTo string `json:"d2KeyTo" keel:"d2_key_to"`
	D3KeyTo string `json:"d3KeyTo" keel:"d3_key_to"`

	D0Min decimal.NullDecimal `json:"d0Min" keel:"d0_min"`
	D0Max decimal.NullDecimal `json:"d0Max" keel:"d0_max"`
	D1Min decimal.NullDecimal `json:"d1Min" keel:"d1_min"`
	D1Max decimal.NullDecimal `json:"d1Max" keel:"d1_max"`
	D2Min decimal.NullDecimal `json:"d2Min" keel:"d2_min"`
	D2Max decimal.NullDecimal `json:"d2Max" keel:"d2_max"`
	D3Min decimal.NullDecimal `json:"d3Min" keel:"d3_min"`
	D3Max decimal.NullDecimal `json:"d3Max" keel:"d3_max"`

	Value           decimal.Decimal     `json:"value"           keel:"value"`
	MinCharge       decimal.NullDecimal `json:"minCharge"       keel:"min_charge"`
	DeficitEligible bool                `json:"deficitEligible" keel:"deficit_eligible"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	RateMatrix *RateMatrix `json:"-" keel:"belongs:rate_matrix_id"`
}

// KeyAt reads the exact key stored for the given axis.
func (rmc *RateMatrixCell) KeyAt(position int16) string {
	switch position {
	case 0:
		return rmc.D0Key
	case 1:
		return rmc.D1Key
	case 2:
		return rmc.D2Key
	case 3:
		return rmc.D3Key
	default:
		return ""
	}
}

func (rmc *RateMatrixCell) KeyToAt(position int16) string {
	switch position {
	case 0:
		return rmc.D0KeyTo
	case 1:
		return rmc.D1KeyTo
	case 2:
		return rmc.D2KeyTo
	case 3:
		return rmc.D3KeyTo
	default:
		return ""
	}
}

func (rmc *RateMatrixCell) SetKeyToAt(position int16, key string) {
	switch position {
	case 0:
		rmc.D0KeyTo = key
	case 1:
		rmc.D1KeyTo = key
	case 2:
		rmc.D2KeyTo = key
	case 3:
		rmc.D3KeyTo = key
	}
}

func (rmc *RateMatrixCell) PostalRangeAt(position int16) PostalRange {
	return NewPostalRange(rmc.KeyAt(position), rmc.KeyToAt(position))
}

// BoundsAt reads the range stored for the given axis.
func (rmc *RateMatrixCell) BoundsAt(position int16) (minimum, maximum decimal.NullDecimal) {
	switch position {
	case 0:
		return rmc.D0Min, rmc.D0Max
	case 1:
		return rmc.D1Min, rmc.D1Max
	case 2:
		return rmc.D2Min, rmc.D2Max
	case 3:
		return rmc.D3Min, rmc.D3Max
	default:
		return decimal.NullDecimal{}, decimal.NullDecimal{}
	}
}

// SetKeyAt writes the exact key for the given axis.
func (rmc *RateMatrixCell) SetKeyAt(position int16, key string) {
	switch position {
	case 0:
		rmc.D0Key = key
	case 1:
		rmc.D1Key = key
	case 2:
		rmc.D2Key = key
	case 3:
		rmc.D3Key = key
	}
}

// SetBoundsAt writes the range for the given axis.
func (rmc *RateMatrixCell) SetBoundsAt(position int16, minimum, maximum decimal.NullDecimal) {
	switch position {
	case 0:
		rmc.D0Min, rmc.D0Max = minimum, maximum
	case 1:
		rmc.D1Min, rmc.D1Max = minimum, maximum
	case 2:
		rmc.D2Min, rmc.D2Max = minimum, maximum
	case 3:
		rmc.D3Min, rmc.D3Max = minimum, maximum
	}
}

func (rmc *RateMatrixCell) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(rmc,
		validation.Field(&rmc.D0Key,
			validation.Length(0, 120).Error("Key cannot be longer than 120 characters"),
		),
		validation.Field(&rmc.D1Key,
			validation.Length(0, 120).Error("Key cannot be longer than 120 characters"),
		),
		validation.Field(&rmc.D2Key,
			validation.Length(0, 120).Error("Key cannot be longer than 120 characters"),
		),
		validation.Field(&rmc.D3Key,
			validation.Length(0, 120).Error("Key cannot be longer than 120 characters"),
		),
		validation.Field(&rmc.D0KeyTo,
			validation.Length(0, 120).Error("Key cannot be longer than 120 characters"),
		),
		validation.Field(&rmc.D1KeyTo,
			validation.Length(0, 120).Error("Key cannot be longer than 120 characters"),
		),
		validation.Field(&rmc.D2KeyTo,
			validation.Length(0, 120).Error("Key cannot be longer than 120 characters"),
		),
		validation.Field(&rmc.D3KeyTo,
			validation.Length(0, 120).Error("Key cannot be longer than 120 characters"),
		),
	))

	if rmc.Value.IsNegative() {
		multiErr.Add("value", errortypes.ErrInvalid, "Value cannot be negative")
	}

	if rmc.MinCharge.Valid && rmc.MinCharge.Decimal.IsNegative() {
		multiErr.Add("minCharge", errortypes.ErrInvalid, "Minimum charge cannot be negative")
	}

	rmc.validateBounds(multiErr)
}

var boundFieldNames = [MaxDimensions]string{"d0Max", "d1Max", "d2Max", "d3Max"}

func (rmc *RateMatrixCell) validateBounds(multiErr *errortypes.MultiError) {
	for position := int16(0); position < MaxDimensions; position++ {
		minimum, maximum := rmc.BoundsAt(position)
		if !minimum.Valid || !maximum.Valid {
			continue
		}

		if maximum.Decimal.LessThanOrEqual(minimum.Decimal) {
			multiErr.Add(
				boundFieldNames[position],
				errortypes.ErrInvalid,
				"Range maximum must be greater than range minimum",
			)
		}
	}
}

// ValidateAgainst checks the cell against the shape its matrix declares, which
// is the only place the two can be compared. A cell with a key on a range axis,
// or a band on an exact one, would be stored and never looked up.
func (rmc *RateMatrixCell) ValidateAgainst(
	dimensions []*RateMatrixDimension,
	multiErr *errortypes.MultiError,
) {
	rmc.Validate(multiErr)

	for _, dimension := range dimensions {
		if dimension == nil {
			continue
		}

		position := dimension.Position
		minimum, _ := rmc.BoundsAt(position)

		if dimension.IsPostalRange() {
			rmc.validatePostalRange(dimension, multiErr)
			continue
		}
		rmc.SetKeyToAt(position, "")

		switch dimension.MatchMode {
		case MatchModeExact:
			if rmc.KeyAt(position) == "" {
				multiErr.Add(
					keyFieldName(position),
					errortypes.ErrRequired,
					"{0} requires a value", dimension.DisplayLabel(),
				)
			}
		case MatchModeRange:
			if !minimum.Valid {
				multiErr.Add(
					minFieldName(position),
					errortypes.ErrRequired,
					"{0} requires a range minimum", dimension.DisplayLabel(),
				)
			}
		}
	}
}

func (rmc *RateMatrixCell) validatePostalRange(
	dimension *RateMatrixDimension,
	multiErr *errortypes.MultiError,
) {
	position := dimension.Position
	postal := rmc.PostalRangeAt(position)
	rmc.SetKeyAt(position, postal.From)
	rmc.SetKeyToAt(position, postal.To)
	rmc.SetBoundsAt(position, decimal.NullDecimal{}, decimal.NullDecimal{})

	switch {
	case postal.From == "" || postal.To == "":
		multiErr.Add(
			keyFieldName(position),
			errortypes.ErrRequired,
			"{0} requires the first and last postal code of its range", dimension.DisplayLabel(),
		)
	case postal.Length() != len([]rune(postal.To)):
		multiErr.Add(
			keyToFieldName(position),
			errortypes.ErrInvalid,
			"Both ends of a postal range have the same number of characters, as 30000 to 31999 do",
		)
	case postal.From > postal.To:
		multiErr.Add(
			keyToFieldName(position),
			errortypes.ErrInvalid,
			"A postal range cannot end before it starts",
		)
	}
}

func keyToFieldName(position int16) string {
	switch position {
	case 0:
		return "d0KeyTo"
	case 1:
		return "d1KeyTo"
	case 2:
		return "d2KeyTo"
	default:
		return "d3KeyTo"
	}
}

func keyFieldName(position int16) string {
	switch position {
	case 0:
		return "d0Key"
	case 1:
		return "d1Key"
	case 2:
		return "d2Key"
	default:
		return "d3Key"
	}
}

func minFieldName(position int16) string {
	switch position {
	case 0:
		return "d0Min"
	case 1:
		return "d1Min"
	case 2:
		return "d2Min"
	default:
		return "d3Min"
	}
}

func (rmc *RateMatrixCell) GetID() pulid.ID {
	return rmc.ID
}

func (rmc *RateMatrixCell) GetOrganizationID() pulid.ID {
	return rmc.OrganizationID
}

func (rmc *RateMatrixCell) GetBusinessUnitID() pulid.ID {
	return rmc.BusinessUnitID
}

func (rmc *RateMatrixCell) GetTableName() string {
	return "rate_matrix_cells"
}
