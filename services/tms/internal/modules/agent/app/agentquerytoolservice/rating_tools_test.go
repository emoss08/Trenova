package agentquerytoolservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRateMatrices struct {
	matrix *ratematrix.RateMatrix
	cells  []*ratematrix.RateMatrixCell
	pages  int
}

func (f *fakeRateMatrices) GetByID(
	context.Context,
	*ratematrixports.GetRateMatrixByIDRequest,
) (*ratematrix.RateMatrix, error) {
	return f.matrix, nil
}

func (f *fakeRateMatrices) ListCells(
	_ context.Context,
	req *ratematrixports.ListRateMatrixCellsRequest,
) (*pagination.ListResult[*ratematrix.RateMatrixCell], error) {
	f.pages++
	offset := min(req.Filter.Pagination.Offset, len(f.cells))
	end := min(offset+req.Filter.Pagination.Limit, len(f.cells))

	return &pagination.ListResult[*ratematrix.RateMatrixCell]{
		Items: f.cells[offset:end],
		Total: len(f.cells),
	}, nil
}

func TestGetRateMatrix_ReturnsAtMostTwoHundredCells(t *testing.T) {
	t.Parallel()

	cells := make([]*ratematrix.RateMatrixCell, 0, 450)
	for idx := range 450 {
		cells = append(cells, &ratematrix.RateMatrixCell{
			D0Key: fmt.Sprintf("Z%d", idx),
			D1Min: decimal.NewNullDecimal(decimal.NewFromInt(0)),
			D1Max: decimal.NewNullDecimal(decimal.NewFromInt(500)),
			Value: decimal.RequireFromString("2.15"),
		})
	}
	fake := &fakeRateMatrices{
		matrix: &ratematrix.RateMatrix{
			ID:   pulid.MustNew("rmx_"),
			Code: "LTL-70",
			Dimensions: []*ratematrix.RateMatrixDimension{
				{Position: 0, Kind: ratematrix.DimensionKindZone},
				{Position: 1, Kind: ratematrix.DimensionKindWeightBreak},
			},
		},
		cells: cells,
	}
	tool := newGetRateMatrixTool(fake, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(
		map[string]any{"rateMatrixId": fake.matrix.ID.String()},
		permission.SensitivityRestricted,
	))
	require.NoError(t, err)

	view := result.(rateMatrixView)
	assert.Len(t, view.Cells, maxMatrixCells)
	assert.Equal(t, 450, view.CellCount)
	assert.True(t, view.CellsHasMore)
	require.NotNil(t, view.NextCellOffset)
	assert.Equal(t, maxMatrixCells, *view.NextCellOffset)
	assert.Equal(t, []string{"Z0", ""}, view.Cells[0].Keys)
	assert.Equal(t, []string{"", "0 to 500"}, view.Cells[0].Ranges)
	assert.Equal(t, "2.15", view.Cells[0].Value)

	result, err = tool.Query(t.Context(), agentParams(map[string]any{
		"rateMatrixId": fake.matrix.ID.String(),
		"cellOffset":   float64(400),
	}, ""))
	require.NoError(t, err)
	last := result.(rateMatrixView)
	assert.Len(t, last.Cells, 50)
	assert.False(t, last.CellsHasMore)
	assert.Empty(t, last.Cells[0].Value)
	assert.Contains(t, last.Withheld, "cells.value")
}
