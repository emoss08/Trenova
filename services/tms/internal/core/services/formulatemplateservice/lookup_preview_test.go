package formulatemplateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func exactMatrixFixture(code string, entries map[string]string) *repositories.RateMatrixLookupData {
	matrixID := pulid.MustNew("rmx_")

	cells := make([]*ratematrix.RateMatrixCell, 0, len(entries))
	for key, value := range entries {
		cells = append(cells, &ratematrix.RateMatrixCell{
			ID:           pulid.MustNew("rmc_"),
			RateMatrixID: matrixID,
			D0Key:        key,
			Value:        decimal.RequireFromString(value),
		})
	}

	return &repositories.RateMatrixLookupData{
		Matrix: &ratematrix.RateMatrix{
			ID:   matrixID,
			Code: code,
			Dimensions: []*ratematrix.RateMatrixDimension{{
				Position:  0,
				Kind:      ratematrix.DimensionKindCustom,
				MatchMode: ratematrix.MatchModeExact,
			}},
		},
		Cells: cells,
	}
}

func rangeMatrixFixture(code string, bands [][3]string) *repositories.RateMatrixLookupData {
	matrixID := pulid.MustNew("rmx_")

	cells := make([]*ratematrix.RateMatrixCell, 0, len(bands))
	for _, band := range bands {
		cell := &ratematrix.RateMatrixCell{
			ID:           pulid.MustNew("rmc_"),
			RateMatrixID: matrixID,
			D0Min:        decimal.NewNullDecimal(decimal.RequireFromString(band[0])),
			Value:        decimal.RequireFromString(band[2]),
		}
		if band[1] != "" {
			cell.D0Max = decimal.NewNullDecimal(decimal.RequireFromString(band[1]))
		}
		cells = append(cells, cell)
	}

	return &repositories.RateMatrixLookupData{
		Matrix: &ratematrix.RateMatrix{
			ID:   matrixID,
			Code: code,
			Dimensions: []*ratematrix.RateMatrixDimension{{
				Position:  0,
				Kind:      ratematrix.DimensionKindQuantity,
				MatchMode: ratematrix.MatchModeRange,
			}},
		},
		Cells: cells,
	}
}

func resultAmount(t *testing.T, result *TestExpressionResponse) decimal.Decimal {
	t.Helper()
	require.True(t, result.Valid, result.Error)
	amount, ok := result.Result.(decimal.Decimal)
	require.True(t, ok, "result should be a decimal, got %T", result.Result)
	return amount
}

func TestTestExpression_PreviewUsesRealRateTables(t *testing.T) {
	t.Parallel()
	deps := setupTestWithMatrices(t, []*repositories.RateMatrixLookupData{
		exactMatrixFixture("fsc", map[string]string{"DIESEL": "0.35"}),
	})

	result := deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: `lookup("fsc", "DIESEL") * totalDistance`,
		SchemaID:   "shipment",
		Variables:  map[string]any{"totalDistance": 100.0},
		TenantInfo: newTenantInfo(),
	})

	assert.True(t, decimal.NewFromFloat(35).Equal(resultAmount(t, result)), result.Result)
}

func TestTestExpression_BreakdownLinesUseRealRateTables(t *testing.T) {
	t.Parallel()
	deps := setupTestWithMatrices(t, []*repositories.RateMatrixLookupData{
		rangeMatrixFixture("miles", [][3]string{{"0", "500", "2.5"}, {"500", "", "2.0"}}),
	})

	result := deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "totalDistance * 3",
		SchemaID:   "shipment",
		Variables:  map[string]any{"totalDistance": 600.0},
		TenantInfo: newTenantInfo(),
		Breakdowns: []*formulatypes.BreakdownDefinition{
			{
				Name:       "linehaul",
				Label:      "Linehaul",
				Expression: `lookup("miles", totalDistance) * totalDistance`,
			},
		},
	})

	require.True(t, result.Valid, result.Error)
	require.Len(t, result.Breakdown, 1)
	assert.Empty(t, result.Breakdown[0].Error)
	assert.True(t, decimal.NewFromInt(1200).Equal(result.Breakdown[0].Amount))
}

func TestTestExpression_LookupOrFallsBackOnlyWhenKeyMisses(t *testing.T) {
	t.Parallel()
	deps := setupTestWithMatrices(t, []*repositories.RateMatrixLookupData{
		exactMatrixFixture("fsc", map[string]string{"DIESEL": "0.35"}),
	})

	miss := deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: `lookupOr("fsc", "GAS", 9)`,
		SchemaID:   "shipment",
		Variables:  map[string]any{},
		TenantInfo: newTenantInfo(),
	})
	assert.True(t, decimal.NewFromInt(9).Equal(resultAmount(t, miss)))

	missingTable := deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: `lookupOr("nope", "GAS", 9)`,
		SchemaID:   "shipment",
		Variables:  map[string]any{},
		TenantInfo: newTenantInfo(),
	})
	assert.False(t, missingTable.Valid)
	assert.Contains(t, missingTable.Error, "nope")
}

func TestTestExpression_WithoutTableReferencesNeverLoadsTables(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result := deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "totalDistance * 2",
		SchemaID:   "shipment",
		Variables:  map[string]any{"totalDistance": 10.0},
		TenantInfo: newTenantInfo(),
	})

	assert.True(t, decimal.NewFromInt(20).Equal(resultAmount(t, result)))
}

func TestRunTestCases_ScenarioUsesRealRateTable(t *testing.T) {
	t.Parallel()
	deps := setupTestWithMatrices(t, []*repositories.RateMatrixLookupData{
		rangeMatrixFixture("miles", [][3]string{{"0", "500", "2.5"}, {"500", "", "2.0"}}),
	})

	template := newTestTemplate()
	template.Expression = `lookup("miles", totalDistance) * totalDistance`
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{
		newTestCase("short haul at 2.50", 250, 0.01),
	}

	result, err := deps.svc.RunTestCases(t.Context(), &RunTestCasesRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 1)
	assert.True(t, result.Results[0].Passed, result.Results[0].Error)
	assert.True(t, decimal.NewFromInt(250).Equal(result.Results[0].ActualAmount))
}

func TestRunTestCases_ScenarioAgainstMissingTableFailsLoudly(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Expression = `lookup("nope", totalDistance) * totalDistance`
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{
		newTestCase("would have passed at zero", 0, 0.01),
	}

	result, err := deps.svc.RunTestCases(t.Context(), &RunTestCasesRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 1)
	assert.False(t, result.Results[0].Passed)
	assert.Contains(t, result.Results[0].Error, "nope")
	assert.Equal(t, 1, result.Failed)
}
