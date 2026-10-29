package ratequerytools

import (
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

const (
	matrixAxes      = 4
	fieldMatrixKeys = "keys"
	fieldRanges     = "ranges"
	paramCellOffset = "cellOffset"
)

// RateMatrixGetSpec reads one rate matrix through the Desk's matrix card, a
// page of its cells at a time.
func RateMatrixGetSpec() readspec.Get {
	return readspec.Get{
		Name:   "rate_get_matrix",
		Entity: "rate matrix",
		Summary: "Retrieve one rate matrix by id: a table a tariff looks values up in. It gives " +
			"the value type, the axes (zone, zip, weight break, freight class and so on) and up " +
			"to 100 cells with their keys, ranges and values. Take the id from the page you are on.",
		Notes: map[string]string{
			paramCellOffset: "When a result says cellsHasMore, call again with its nextCellOffset.",
		},
		Resource:  permission.ResourceRateMatrix,
		ParamName: querytool.ParamRateMatrixID,
		IDSource:  "from the page you are on",
		Pages:     &readspec.Pages{Offset: paramCellOffset, Items: "cells", Total: "cellCount"},
		Shape:     foldMatrixCells,
	}
}

// foldMatrixCells gives each cell its keys and ranges along the matrix's own
// axes, in place of a column per possible axis; a postal range's key reads
// "30000 to 31999".
func foldMatrixCells(record map[string]any, _ map[string]struct{}, _ filtercatalog.Clock) {
	dimensions, _ := record["dimensions"].([]any)
	axes := min(len(dimensions), matrixAxes)
	cells, _ := record["cells"].([]any)
	for _, raw := range cells {
		cell, _ := raw.(map[string]any)
		keys := make([]any, 0, axes)
		ranges := make([]any, 0, axes)
		keyed, ranged := false, false
		for axis := range matrixAxes {
			prefix := "d" + strconv.Itoa(axis)
			key, _ := cell[prefix+"Key"].(string)
			if keyTo, _ := cell[prefix+"KeyTo"].(string); keyTo != "" {
				key += " to " + keyTo
			}
			low, high := cell[prefix+"Min"], cell[prefix+"Max"]
			delete(cell, prefix+"Key")
			delete(cell, prefix+"KeyTo")
			delete(cell, prefix+"Min")
			delete(cell, prefix+"Max")
			if axis >= axes {
				continue
			}
			keys = append(keys, key)
			keyed = keyed || key != ""
			ranges = append(ranges, matrixRange(low, high))
			ranged = ranged || low != nil || high != nil
		}
		if keyed {
			cell[fieldMatrixKeys] = keys
		}
		if ranged {
			cell[fieldRanges] = ranges
		}
	}
}

func matrixRange(low, high any) string {
	if low == nil && high == nil {
		return ""
	}
	from, to := matrixBound(low), matrixBound(high)
	if to == "" {
		to = "and up"
	}

	return from + " to " + to
}

func matrixBound(value any) string {
	switch bound := value.(type) {
	case nil:
		return ""
	case string:
		return bound
	case float64:
		return strconv.FormatFloat(bound, 'f', -1, 64)
	default:
		return ""
	}
}
