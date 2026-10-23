package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratematrixservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	maxMatrixCells = 200
	matrixCellPage = 100
)

func ratingToolProviders() []any {
	return []any{
		provideGetRateMatrixTool,
	}
}

type rateMatrixReader interface {
	GetByID(
		ctx context.Context,
		req *ratematrixports.GetRateMatrixByIDRequest,
	) (*ratematrix.RateMatrix, error)
	ListCells(
		ctx context.Context,
		req *ratematrixports.ListRateMatrixCellsRequest,
	) (*pagination.ListResult[*ratematrix.RateMatrixCell], error)
}

func provideGetRateMatrixTool(
	matrices *ratematrixservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newGetRateMatrixTool(matrices, permissions)
}

func nullDecimalText(value decimal.NullDecimal) string {
	if !value.Valid {
		return ""
	}

	return value.Decimal.String()
}

func decimalPointerText(value *decimal.Decimal) string {
	if value == nil {
		return ""
	}

	return value.String()
}

type getRateMatrixTool struct {
	matrices rateMatrixReader
	access   fieldAccess
}

func newGetRateMatrixTool(
	matrices rateMatrixReader,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &getRateMatrixTool{matrices: matrices, access: newFieldAccess(permissions)}
}

func (t *getRateMatrixTool) Name() string { return "rate_get_matrix" }

func (t *getRateMatrixTool) Description() string {
	return "Retrieve one rate matrix by id: a table a tariff looks values up in. It gives " +
		"the value type, the axes (zone, zip, weight break, freight class and so on) and up " +
		"to 200 cells with their keys, ranges and values. Take the id from the page you are on."
}

func (t *getRateMatrixTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramRateMatrixID: agenttoolschema.RecordIDText(permission.ResourceRateMatrix,
			"The rate matrix's id, from the page you are on."),
		"cellOffset": intParam("How many cells to skip. When a result says cellsHasMore, " +
			"call again with its nextCellOffset."),
	}, paramRateMatrixID)
}

func (t *getRateMatrixTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceRateMatrix})
}

type matrixDimensionRow struct {
	Position         int16  `json:"position"`
	Kind             string `json:"kind"`
	MatchMode        string `json:"matchMode"`
	Label            string `json:"label,omitempty"`
	KeyNormalization string `json:"keyNormalization,omitempty"`
	RangeOverflow    string `json:"rangeOverflow,omitempty"`
}

type matrixCellRow struct {
	Keys            []string `json:"keys,omitempty"`
	Ranges          []string `json:"ranges,omitempty"`
	DeficitEligible bool     `json:"deficitEligible,omitempty"`
	Value           string   `json:"value,omitempty"`
	MinCharge       string   `json:"minCharge,omitempty"`
}

type rateMatrixView struct {
	ID             string               `json:"id"`
	Code           string               `json:"code"`
	Name           string               `json:"name"`
	Description    string               `json:"description,omitempty"`
	Status         string               `json:"status"`
	ValueType      string               `json:"valueType"`
	Dimensions     []matrixDimensionRow `json:"dimensions"`
	CellCount      int                  `json:"cellCount"`
	CellOffset     int                  `json:"cellOffset,omitempty"`
	Cells          []matrixCellRow      `json:"cells"`
	CellsHasMore   bool                 `json:"cellsHasMore"`
	NextCellOffset *int                 `json:"nextCellOffset,omitempty"`
	Withheld       []string             `json:"withheldByAccess,omitempty"`
}

func (t *getRateMatrixTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	id, err := requirePulid(params.Params, paramRateMatrixID)
	if err != nil {
		return nil, err
	}
	offset := max(optionalInt(params.Params, "cellOffset", 0), 0)

	tenant := tenantOf(params)
	matrix, err := t.matrices.GetByID(ctx, &ratematrixports.GetRateMatrixByIDRequest{
		RateMatrixID:      id,
		TenantInfo:        tenant,
		IncludeDimensions: true,
	})
	if err != nil {
		return nil, err
	}

	cells, total, err := t.cells(ctx, tenant, id, offset)
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceRateMatrix)
	view := rateMatrixView{
		ID:          matrix.ID.String(),
		Code:        matrix.Code,
		Name:        matrix.Name,
		Description: matrix.Description,
		Status:      string(matrix.Status),
		ValueType:   matrix.ValueType,
		Dimensions:  make([]matrixDimensionRow, 0, len(matrix.Dimensions)),
		CellCount:   total,
		CellOffset:  offset,
		Cells:       make([]matrixCellRow, 0, len(cells)),
	}
	axes := 0
	for _, dimension := range matrix.Dimensions {
		if dimension == nil {
			continue
		}
		axes = max(axes, int(dimension.Position)+1)
		view.Dimensions = append(view.Dimensions, matrixDimensionRow{
			Position:         dimension.Position,
			Kind:             string(dimension.Kind),
			MatchMode:        string(dimension.MatchMode),
			Label:            dimension.Label,
			KeyNormalization: string(dimension.KeyNormalization),
			RangeOverflow:    string(dimension.RangeOverflow),
		})
	}

	showValue := gate.show("value", "cells.value")
	showMin := gate.show("minCharge", "cells.minCharge")
	for _, cell := range cells {
		if cell == nil {
			continue
		}
		row := matrixCellRow{
			Keys:            cellKeys(cell, axes),
			Ranges:          cellRanges(cell, axes),
			DeficitEligible: cell.DeficitEligible,
		}
		if showValue {
			row.Value = cell.Value.String()
		}
		if showMin {
			row.MinCharge = nullDecimalText(cell.MinCharge)
		}
		view.Cells = append(view.Cells, row)
	}
	if offset+len(view.Cells) < total {
		view.CellsHasMore = true
		next := offset + len(view.Cells)
		view.NextCellOffset = &next
	}
	view.Withheld = gate.Withheld()

	return view, nil
}

func (t *getRateMatrixTool) cells(
	ctx context.Context,
	tenant pagination.TenantInfo,
	matrixID pulid.ID,
	offset int,
) ([]*ratematrix.RateMatrixCell, int, error) {
	cells := make([]*ratematrix.RateMatrixCell, 0, matrixCellPage)
	total := 0
	for len(cells) < maxMatrixCells {
		page, err := t.matrices.ListCells(ctx, &ratematrixports.ListRateMatrixCellsRequest{
			TenantInfo:   tenant,
			RateMatrixID: matrixID,
			Filter: &pagination.QueryOptions{
				TenantInfo: tenant,
				Pagination: pagination.Info{
					Limit:  min(matrixCellPage, maxMatrixCells-len(cells)),
					Offset: offset + len(cells),
				},
			},
		})
		if err != nil {
			return nil, 0, err
		}
		total = page.Total
		cells = append(cells, page.Items...)
		if len(page.Items) < matrixCellPage || offset+len(cells) >= total {
			break
		}
	}

	return cells, total, nil
}

func cellKeys(cell *ratematrix.RateMatrixCell, axes int) []string {
	keys := []string{cell.D0Key, cell.D1Key, cell.D2Key, cell.D3Key}[:min(max(axes, 0), 4)]
	for _, key := range keys {
		if key != "" {
			return keys
		}
	}

	return nil
}

func cellRanges(cell *ratematrix.RateMatrixCell, axes int) []string {
	bounds := [][2]decimal.NullDecimal{
		{cell.D0Min, cell.D0Max},
		{cell.D1Min, cell.D1Max},
		{cell.D2Min, cell.D2Max},
		{cell.D3Min, cell.D3Max},
	}[:min(max(axes, 0), 4)]

	ranges := make([]string, 0, len(bounds))
	found := false
	for _, bound := range bounds {
		low, high := nullDecimalText(bound[0]), nullDecimalText(bound[1])
		if low == "" && high == "" {
			ranges = append(ranges, "")
			continue
		}
		found = true
		if high == "" {
			high = "and up"
		}
		ranges = append(ranges, low+" to "+high)
	}
	if !found {
		return nil
	}

	return ranges
}
