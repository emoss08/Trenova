package ratematrix

import (
	"errors"
	"sort"

	"github.com/shopspring/decimal"
)

var (
	// ErrDimensionValueMissing is returned when a lookup does not supply a value
	// for an axis the matrix declares.
	ErrDimensionValueMissing = errors.New("no value supplied for matrix dimension")

	// ErrNoMatchingCell is returned when nothing in the matrix covers the
	// supplied values.
	ErrNoMatchingCell = errors.New("no matrix cell matches the supplied values")
)

// DimensionValue is the input for one axis of a lookup.
//
// Exact axes carry Keys rather than a single key because one shipment can sit
// in several zones at once. The slice is ordered most preferred first, and that
// ordering is what breaks a tie between two cells that both match.
type DimensionValue struct {
	Keys     []string
	Quantity decimal.NullDecimal
}

// LookupValues holds one DimensionValue per axis, indexed by position.
type LookupValues [MaxDimensions]DimensionValue

// Match is a cell that satisfied a lookup, together with why it won.
type Match struct {
	Cell *RateMatrixCell
	// Preference is the summed index of each matched key within the candidate
	// list it came from. Zero means every axis matched its first choice.
	Preference int
	// MatchedKeys records the key actually used per axis, so a rating trace can
	// say which zone or class the cell was found under rather than only which
	// cell it was.
	MatchedKeys [MaxDimensions]string
}

// SelectCell picks the one cell that prices this lookup.
//
// Candidates come from the database already narrowed by the exact keys and, for
// banded axes, by the quantity's bounds. What is left for this function is the
// part SQL cannot express well: preferring the caller's earlier key choices
// when several zones matched, and settling any remaining tie the same way every
// time.
//
// The final tiebreak is the cell id. PULIDs sort by creation, so two otherwise
// indistinguishable cells always resolve to the older one — on every server,
// on every run. Without it the winner would follow whatever order the database
// happened to return, and the same shipment could rate two ways.
func SelectCell(
	dimensions []*RateMatrixDimension,
	cells []*RateMatrixCell,
	values LookupValues,
) (*Match, error) {
	if err := validateLookupInputs(dimensions, values); err != nil {
		return nil, err
	}

	matches := make([]*Match, 0, len(cells))

	for _, cell := range cells {
		if cell == nil {
			continue
		}

		if match, ok := matchCell(dimensions, cell, values); ok {
			matches = append(matches, match)
		}
	}

	if len(matches) == 0 {
		return nil, ErrNoMatchingCell
	}

	sort.Slice(matches, func(a, b int) bool {
		if matches[a].Preference != matches[b].Preference {
			return matches[a].Preference < matches[b].Preference
		}
		return matches[a].Cell.ID.String() < matches[b].Cell.ID.String()
	})

	return matches[0], nil
}

func validateLookupInputs(dimensions []*RateMatrixDimension, values LookupValues) error {
	for _, dimension := range dimensions {
		if dimension == nil {
			continue
		}

		value := values[dimension.Position]

		switch dimension.MatchMode {
		case MatchModeExact:
			if len(value.Keys) == 0 {
				return ErrDimensionValueMissing
			}
		case MatchModeRange:
			if !value.Quantity.Valid {
				return ErrDimensionValueMissing
			}
		}
	}

	return nil
}

func matchCell(
	dimensions []*RateMatrixDimension,
	cell *RateMatrixCell,
	values LookupValues,
) (*Match, bool) {
	match := &Match{Cell: cell}

	for _, dimension := range dimensions {
		if dimension == nil {
			continue
		}

		position := dimension.Position
		value := values[position]

		switch dimension.MatchMode {
		case MatchModeExact:
			rank, ok := keyRank(value.Keys, cell.KeyAt(position))
			if !ok {
				return nil, false
			}
			match.Preference += rank
			match.MatchedKeys[position] = cell.KeyAt(position)
		case MatchModeRange:
			if !cell.ContainsQuantity(position, value.Quantity.Decimal) {
				return nil, false
			}
			match.MatchedKeys[position] = value.Quantity.Decimal.String()
		}
	}

	return match, true
}

func keyRank(candidates []string, key string) (int, bool) {
	for i, candidate := range candidates {
		if candidate == key {
			return i, true
		}
	}

	return 0, false
}
