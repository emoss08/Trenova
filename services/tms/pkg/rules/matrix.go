package rules

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
)

const (
	OpMatrixSelect = "matrix.select"

	MatrixDimensions = 4

	MatrixErrorValueMissing   = "dimension_value_missing"
	MatrixErrorNoMatchingCell = "no_matching_cell"
)

var errNoSelection = errors.New(
	"the rules module answered matrix.select with neither a cell nor an error",
)

type MatrixDimension struct {
	Position  int64  `json:"position"`
	MatchMode string `json:"matchMode"`
	Kind      string `json:"kind"`
	Label     string `json:"label,omitempty"`
}

type MatrixCell struct {
	ID    string              `json:"id,omitempty"`
	D0Key string              `json:"d0Key"`
	D1Key string              `json:"d1Key"`
	D2Key string              `json:"d2Key"`
	D3Key string              `json:"d3Key"`
	D0Min decimal.NullDecimal `json:"d0Min"`
	D0Max decimal.NullDecimal `json:"d0Max"`
	D1Min decimal.NullDecimal `json:"d1Min"`
	D1Max decimal.NullDecimal `json:"d1Max"`
	D2Min decimal.NullDecimal `json:"d2Min"`
	D2Max decimal.NullDecimal `json:"d2Max"`
	D3Min decimal.NullDecimal `json:"d3Min"`
	D3Max decimal.NullDecimal `json:"d3Max"`
}

type MatrixValue struct {
	Keys     []string            `json:"keys"`
	Quantity decimal.NullDecimal `json:"quantity"`
}

type MatrixSelectRequest struct {
	Dimensions []MatrixDimension `json:"dimensions"`
	Cells      []MatrixCell      `json:"cells"`
	Values     []MatrixValue     `json:"values"`
}

type MatrixSelection struct {
	Index       int                      `json:"index"`
	ID          string                   `json:"id"`
	Preference  int                      `json:"preference"`
	MatchedKeys [MatrixDimensions]string `json:"matchedKeys"`
}

type MatrixSelectError struct {
	Code     string  `json:"code"`
	Position *int64  `json:"position"`
	Message  Message `json:"message"`
}

func (e *MatrixSelectError) Error() string {
	return e.Message.String()
}

func (e *MatrixSelectError) LocalizedMessage() (message string, args []any) {
	return e.Message.LocalizedMessage()
}

type matrixSelectOut struct {
	Cell  *MatrixSelection   `json:"cell"`
	Error *MatrixSelectError `json:"error"`
}

func SelectMatrixCell(ctx context.Context, req *MatrixSelectRequest) (*MatrixSelection, error) {
	var out matrixSelectOut
	if err := Call(ctx, OpMatrixSelect, req, &out); err != nil {
		return nil, err
	}
	switch {
	case out.Error != nil:
		return nil, out.Error
	case out.Cell != nil:
		return out.Cell, nil
	default:
		return nil, errNoSelection
	}
}
