package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const matrixCellsPerRead = 100

// A matrix is read a page of cells at a time: the result says whether more
// cells follow and the offset to ask for next, and each cell's keys and
// ranges are along the matrix's own axes.
func TestGetRateMatrix_ReadsAPageOfCellsAndSaysWhereTheNextStarts(t *testing.T) {
	t.Parallel()

	cells := make([]any, 0, matrixCellsPerRead)
	for range matrixCellsPerRead {
		cells = append(cells, map[string]any{
			"d0Key": "Z1", "d1Key": "", "d2Key": "", "d3Key": "",
			"d1Min": "0", "d1Max": "500", "value": "2.15",
		})
	}
	reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
		"DeskRateMatrixCard": {Data: map[string]any{"rateMatrix": map[string]any{
			"id": "rmx_1", "code": "LTL-70", "cellCount": 450,
			"dimensions": []any{map[string]any{"kind": "Zone"}, map[string]any{"kind": "WeightBreak"}},
			"cells":      cells,
		}}},
	}}
	spec := rateMatrixGetSpec()
	tool, err := newGraphQLGetTool(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	id := pulid.MustNew("rmx_").String()
	result, err := tool.Query(t.Context(), testParams(map[string]any{
		paramRateMatrixID: id, "cellOffset": float64(200),
	}))
	require.NoError(t, err)

	matrix := encodedDocument(t, result)
	assert.Equal(t, true, matrix["cellsHasMore"])
	assert.EqualValues(t, 300, matrix["nextCellOffset"])
	first, _ := matrix["cells"].([]any)[0].(map[string]any)
	assert.Equal(t, []any{"Z1", ""}, first["keys"])
	assert.Equal(t, []any{"", "0 to 500"}, first["ranges"])
	assert.NotContains(t, first, "d0Key")
	sent := reader.sent("DeskRateMatrixCard").Variables
	assert.Equal(t, id, sent["id"])
	assert.EqualValues(t, 200, sent["cellOffset"])
}

func TestGetRateMatrix_SaysNoMoreOnTheLastPage(t *testing.T) {
	t.Parallel()

	record := map[string]any{"cellCount": 2, "cells": []any{map[string]any{}, map[string]any{}}}
	pages := rateMatrixGetSpec().pages
	pages.mark(record, map[string]any{})

	assert.Equal(t, false, record["cellsHasMore"])
	assert.NotContains(t, record, "nextCellOffset")
}
