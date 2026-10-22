package formula

import (
	"cmp"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/shopspring/decimal"
)

type wireTable struct {
	Name  string     `json:"name"`
	Axes  []wireAxis `json:"axes"`
	Cells []wireCell `json:"cells"`
}

type wireAxis struct {
	Match         ratematrix.MatchMode        `json:"match"`
	Kind          ratematrix.DimensionKind    `json:"kind,omitempty"`
	Normalization ratematrix.KeyNormalization `json:"normalization,omitempty"`
	Overflow      ratematrix.RangeOverflow    `json:"overflow,omitempty"`
}

type wireCell struct {
	At    []wireAt `json:"at"`
	Value string   `json:"value"`
}

type wireAt struct {
	Key string `json:"key,omitempty"`
	Min string `json:"min,omitempty"`
	Max string `json:"max,omitempty"`
}

type tableKey struct {
	name string
	axes int
}

func NewRateTables(
	data []*ratematrixports.RateMatrixLookupData,
	stamp string,
) (*formulatemplatetypes.RateTables, error) {
	byKey := make(map[tableKey]int, len(data))
	tables := make([]formulatemplatetypes.RateTable, 0, len(data))

	for _, item := range data {
		axes := tableAxes(item)
		if axes == nil {
			continue
		}

		encoded, err := sonic.Marshal(&wireTable{
			Name:  item.Matrix.Code,
			Axes:  wireAxes(axes),
			Cells: wireCells(axes, item.Cells),
		})
		if err != nil {
			return nil, err
		}

		table := formulatemplatetypes.RateTable{
			Name:    item.Matrix.Code,
			Axes:    len(axes),
			Banded:  axes[0].MatchMode == ratematrix.MatchModeRange,
			Encoded: encoded,
		}
		key := tableKey{name: table.Name, axes: table.Axes}
		if index, seen := byKey[key]; seen {
			tables[index] = table
			continue
		}
		byKey[key] = len(tables)
		tables = append(tables, table)
	}

	return &formulatemplatetypes.RateTables{Stamp: stamp, Tables: tables}, nil
}

func tableAxes(item *ratematrixports.RateMatrixLookupData) []*ratematrix.RateMatrixDimension {
	if item == nil || item.Matrix == nil {
		return nil
	}
	dimensions := item.Matrix.Dimensions
	if len(dimensions) < 1 || len(dimensions) > 2 || slices.Contains(dimensions, nil) {
		return nil
	}

	axes := slices.Clone(dimensions)
	slices.SortStableFunc(axes, func(a, b *ratematrix.RateMatrixDimension) int {
		return cmp.Compare(a.Position, b.Position)
	})
	return axes
}

func wireAxes(axes []*ratematrix.RateMatrixDimension) []wireAxis {
	wire := make([]wireAxis, 0, len(axes))
	for _, axis := range axes {
		wire = append(wire, wireAxis{
			Match:         axis.MatchMode,
			Kind:          axis.Kind,
			Normalization: axis.KeyNormalization,
			Overflow:      axis.RangeOverflow,
		})
	}
	return wire
}

func wireCells(
	axes []*ratematrix.RateMatrixDimension,
	cells []*ratematrix.RateMatrixCell,
) []wireCell {
	wire := make([]wireCell, 0, len(cells))
	for _, cell := range cells {
		if cell == nil {
			continue
		}
		at := make([]wireAt, 0, len(axes))
		for position, axis := range axes {
			at = append(at, cellPosition(axis, cell, position))
		}
		wire = append(wire, wireCell{At: at, Value: cell.Value.String()})
	}
	return wire
}

func cellPosition(
	axis *ratematrix.RateMatrixDimension,
	cell *ratematrix.RateMatrixCell,
	position int,
) wireAt {
	key, low, high := cell.D0Key, cell.D0Min, cell.D0Max
	if position == 1 {
		key, low, high = cell.D1Key, cell.D1Min, cell.D1Max
	}
	if axis.MatchMode != ratematrix.MatchModeRange {
		return wireAt{Key: key}
	}
	return wireAt{Min: decimalText(low), Max: decimalText(high)}
}

func decimalText(value decimal.NullDecimal) string {
	if !value.Valid {
		return ""
	}
	return value.Decimal.String()
}
