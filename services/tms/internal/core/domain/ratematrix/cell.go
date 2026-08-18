package ratematrix

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RateMatrixCell)(nil)

// RateMatrixCell is one priced intersection of a matrix's axes.
//
// The four key/range column pairs are positional: pair zero belongs to the
// dimension at position zero, and so on. Exact axes fill the key column and
// leave the bounds null; range axes do the reverse. Flattening the axes into
// fixed columns is what makes a single composite index able to answer a lookup,
// which a row-per-axis design could not do at tariff scale.
type RateMatrixCell struct {
	bun.BaseModel `bun:"table:rate_matrix_cells,alias:rmc" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RateMatrixID   pulid.ID `json:"rateMatrixId"   bun:"rate_matrix_id,type:VARCHAR(100),notnull"`

	D0Key string `json:"d0Key" bun:"d0_key,type:VARCHAR(120),notnull,default:''"`
	D1Key string `json:"d1Key" bun:"d1_key,type:VARCHAR(120),notnull,default:''"`
	D2Key string `json:"d2Key" bun:"d2_key,type:VARCHAR(120),notnull,default:''"`
	D3Key string `json:"d3Key" bun:"d3_key,type:VARCHAR(120),notnull,default:''"`

	D0Min decimal.NullDecimal `json:"d0Min" bun:"d0_min,type:NUMERIC(19,4),nullzero"`
	D0Max decimal.NullDecimal `json:"d0Max" bun:"d0_max,type:NUMERIC(19,4),nullzero"`
	D1Min decimal.NullDecimal `json:"d1Min" bun:"d1_min,type:NUMERIC(19,4),nullzero"`
	D1Max decimal.NullDecimal `json:"d1Max" bun:"d1_max,type:NUMERIC(19,4),nullzero"`
	D2Min decimal.NullDecimal `json:"d2Min" bun:"d2_min,type:NUMERIC(19,4),nullzero"`
	D2Max decimal.NullDecimal `json:"d2Max" bun:"d2_max,type:NUMERIC(19,4),nullzero"`
	D3Min decimal.NullDecimal `json:"d3Min" bun:"d3_min,type:NUMERIC(19,4),nullzero"`
	D3Max decimal.NullDecimal `json:"d3Max" bun:"d3_max,type:NUMERIC(19,4),nullzero"`

	Value           decimal.Decimal     `json:"value"           bun:"value,type:NUMERIC(19,6),notnull"`
	MinCharge       decimal.NullDecimal `json:"minCharge"       bun:"min_charge,type:NUMERIC(19,4),nullzero"`
	DeficitEligible bool                `json:"deficitEligible" bun:"deficit_eligible,type:BOOLEAN,notnull,default:true"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	RateMatrix *RateMatrix `json:"-" bun:"rel:belongs-to,join:rate_matrix_id=id"`
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

// ContainsQuantity reports whether a quantity falls inside the cell's band on
// the given axis.
//
// Bands are half open: a shipment weighing exactly the upper bound belongs to
// the next band up. Tariffs are written that way — a "1000 to 2000" break means
// two thousand pounds rates at the next tier — and closed intervals would let
// two adjacent bands both claim the boundary.
func (rmc *RateMatrixCell) ContainsQuantity(position int16, quantity decimal.Decimal) bool {
	minimum, maximum := rmc.BoundsAt(position)

	if minimum.Valid && quantity.LessThan(minimum.Decimal) {
		return false
	}

	if maximum.Valid && quantity.GreaterThanOrEqual(maximum.Decimal) {
		return false
	}

	return true
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

		switch dimension.MatchMode {
		case MatchModeExact:
			if rmc.KeyAt(position) == "" {
				multiErr.Add(
					keyFieldName(position),
					errortypes.ErrRequired,
					dimension.DisplayLabel()+" requires a value",
				)
			}
		case MatchModeRange:
			if !minimum.Valid {
				multiErr.Add(
					minFieldName(position),
					errortypes.ErrRequired,
					dimension.DisplayLabel()+" requires a range minimum",
				)
			}
		}
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

func (rmc *RateMatrixCell) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rmc.ID.IsNil() {
			rmc.ID = pulid.MustNew("rmc_")
		}
		rmc.CreatedAt = now
		rmc.UpdatedAt = now
	case *bun.UpdateQuery:
		rmc.UpdatedAt = now
	}

	return nil
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
