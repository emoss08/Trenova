package ratematrixcell

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type postalSighting struct {
	postal PostalRange
	index  int
}

func ValidatePostalAxes(
	dimensions []*ratematrix.RateMatrixDimension,
	cells []*RateMatrixCell,
	multiErr *errortypes.MultiError,
) {
	for _, dimension := range dimensions {
		if dimension == nil || !dimension.IsPostalRange() {
			continue
		}
		validatePostalAxis(dimension, cells, multiErr)
	}
}

func validatePostalAxis(
	dimension *ratematrix.RateMatrixDimension,
	cells []*RateMatrixCell,
	multiErr *errortypes.MultiError,
) {
	position := dimension.Position
	seen := make(map[PostalRange]struct{}, len(cells))
	sightings := make([]postalSighting, 0, len(cells))
	length := 0
	for index, cell := range cells {
		if cell == nil {
			continue
		}
		postal := cell.PostalRangeAt(position)
		if !postal.IsWellFormed() {
			continue
		}
		if length == 0 {
			length = postal.Length()
		}
		if postal.Length() != length {
			multiErr.WithIndex("cells", index).Add(
				keyFieldName(position),
				errortypes.ErrInvalid,
				"Every postal range on {0} has {1} characters; {2} has a different length",
				dimension.DisplayLabel(),
				length,
				postal.String(),
			)
			return
		}
		if _, dup := seen[postal]; dup {
			continue
		}
		seen[postal] = struct{}{}
		sightings = append(sightings, postalSighting{postal: postal, index: index})
	}

	slices.SortFunc(sightings, func(a, b postalSighting) int {
		return strings.Compare(a.postal.From, b.postal.From)
	})
	if len(sightings) == 0 {
		return
	}
	reach := sightings[0]
	for _, later := range sightings[1:] {
		if reach.postal.Overlaps(later.postal) {
			multiErr.WithIndex("cells", later.index).Add(
				keyFieldName(position),
				errortypes.ErrInvalid,
				"The postal ranges {0} and {1} on {2} overlap, so one postal code would match both",
				reach.postal.String(),
				later.postal.String(),
				dimension.DisplayLabel(),
			)
		}
		if later.postal.To > reach.postal.To {
			reach = later
		}
	}
}
