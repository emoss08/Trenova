package ratematrix

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
)

var errCellOutOfRange = errors.New("the rules module picked a cell that was not sent")

// DimensionValue is the input for one axis of a lookup. Exact axes carry Keys,
// most preferred first, because one shipment can sit in several zones at once.
type DimensionValue struct {
	Keys     []string
	Quantity decimal.NullDecimal
}

// LookupValues holds one DimensionValue per axis, indexed by position.
type LookupValues [MaxDimensions]DimensionValue

// Match is the cell that priced a lookup, together with why it won.
type Match struct {
	Cell        *RateMatrixCell
	Preference  int
	MatchedKeys [MaxDimensions]string
}

func LookupCell(
	ctx context.Context,
	dimensions []*RateMatrixDimension,
	cells []*RateMatrixCell,
	values *LookupValues,
) (*Match, error) {
	req := &rules.MatrixSelectRequest{
		Dimensions: make([]rules.MatrixDimension, 0, len(dimensions)),
		Cells:      make([]rules.MatrixCell, 0, len(cells)),
		Values:     make([]rules.MatrixValue, 0, MaxDimensions),
	}
	for _, dimension := range dimensions {
		if dimension == nil {
			continue
		}
		req.Dimensions = append(req.Dimensions, rules.MatrixDimension{
			Position:  int64(dimension.Position),
			MatchMode: string(dimension.MatchMode),
			Kind:      string(dimension.Kind),
			Label:     dimension.Label,
		})
	}
	sent := make([]*RateMatrixCell, 0, len(cells))
	for _, cell := range cells {
		if cell == nil {
			continue
		}
		sent = append(sent, cell)
		req.Cells = append(req.Cells, rules.MatrixCell{
			ID:    cell.ID.String(),
			D0Key: cell.D0Key,
			D1Key: cell.D1Key,
			D2Key: cell.D2Key,
			D3Key: cell.D3Key,
			D0Min: cell.D0Min,
			D0Max: cell.D0Max,
			D1Min: cell.D1Min,
			D1Max: cell.D1Max,
			D2Min: cell.D2Min,
			D2Max: cell.D2Max,
			D3Min: cell.D3Min,
			D3Max: cell.D3Max,
		})
	}
	for _, value := range values {
		req.Values = append(req.Values, rules.MatrixValue{
			Keys:     value.Keys,
			Quantity: value.Quantity,
		})
	}

	selection, err := rules.SelectMatrixCell(ctx, req)
	if err != nil {
		return nil, err
	}
	if selection.Index < 0 || selection.Index >= len(sent) {
		return nil, fmt.Errorf("%w: %d of %d", errCellOutOfRange, selection.Index, len(sent))
	}

	return &Match{
		Cell:        sent[selection.Index],
		Preference:  selection.Preference,
		MatchedKeys: selection.MatchedKeys,
	}, nil
}
