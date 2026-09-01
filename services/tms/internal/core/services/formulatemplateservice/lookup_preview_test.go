package formulatemplateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/ratetypes"
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

func TestTestExpression_AppliesRoundingPolicy(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	halfUp := deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "10 / 3",
		SchemaID:   "shipment",
		Variables:  map[string]any{},
		TenantInfo: newTenantInfo(),
	})
	assert.True(t, decimal.RequireFromString("3.33").Equal(resultAmount(t, halfUp)), halfUp.Result)
	require.NotNil(t, halfUp.Rounding)
	assert.True(t, halfUp.Rounding.Applied)
	assert.Equal(t, "HalfUp", halfUp.Rounding.Mode)

	wholeUp := deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression:        "10 / 3",
		SchemaID:          "shipment",
		Variables:         map[string]any{},
		TenantInfo:        newTenantInfo(),
		RoundingMode:      ratetypes.RoundingModeUp,
		RoundingPrecision: 0,
	})
	assert.True(t, decimal.NewFromInt(4).Equal(resultAmount(t, wholeUp)), wholeUp.Result)
}

func TestTestExpression_GuardrailFloorIsNotRoundedAway(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result := deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression:        "249.996",
		SchemaID:          "shipment",
		Variables:         map[string]any{},
		TenantInfo:        newTenantInfo(),
		MinCharge:         decimal.NewNullDecimal(decimal.RequireFromString("250.00")),
		RoundingMode:      ratetypes.RoundingModeDown,
		RoundingPrecision: 0,
	})

	assert.True(t, decimal.NewFromInt(250).Equal(resultAmount(t, result)), result.Result)
	require.NotNil(t, result.Guardrail)
	assert.True(t, result.Guardrail.Applied)
}

func TestTestExpression_BooleanExpressionIsRejected(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result := deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "totalDistance > 100",
		SchemaID:   "shipment",
		Variables:  map[string]any{"totalDistance": 500.0},
		TenantInfo: newTenantInfo(),
	})

	assert.False(t, result.Valid)
	assert.Contains(t, result.Error, "true/false")
}

func TestRunTestCases_ScenarioUsesCandidateRoundingPolicy(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Expression = "totalDistance / 3"
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{
		newTestCase("rounded up to whole dollars", 34, 0.001),
	}

	result, err := deps.svc.RunTestCases(t.Context(), &RunTestCasesRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		Candidate: &TestCaseCandidate{
			Expression:        template.Expression,
			RoundingMode:      ratetypes.RoundingModeUp,
			RoundingPrecision: 0,
		},
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 1)
	assert.True(t, result.Results[0].Passed, result.Results[0].Error)
	assert.True(t, decimal.NewFromInt(34).Equal(result.Results[0].ActualAmount))
}

func TestTestExpression_WarnsAboutUnguardedNullableFields(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result := deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "weight * 0.5",
		SchemaID:   "shipment",
		Variables:  map[string]any{"weight": 1000.0},
		TenantInfo: newTenantInfo(),
		Breakdowns: []*formulatypes.BreakdownDefinition{
			{Name: "perPound", Label: "Per pound", Expression: "weight * 0.5"},
			{Name: "safe", Label: "Guarded", Expression: "coalesce(weight, 0) * 0.5"},
		},
	})

	require.True(t, result.Valid, result.Error)
	require.Len(t, result.Warnings, 2)
	assert.Equal(t, "expression", result.Warnings[0].Scope)
	assert.Equal(t, "weight", result.Warnings[0].Field)
	assert.Equal(t, "coalesce(weight, 0)", result.Warnings[0].Suggestion)
	assert.Equal(t, "breakdownDefinitions[0].expression", result.Warnings[1].Scope)
}

func TestTestExpression_GuardedNullableFieldHasNoWarning(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result := deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "coalesce(weight, 0) * 0.5 + totalDistance",
		SchemaID:   "shipment",
		Variables:  map[string]any{"totalDistance": 10.0},
		TenantInfo: newTenantInfo(),
	})

	require.True(t, result.Valid, result.Error)
	assert.Empty(t, result.Warnings)
}

func readinessByKey(resp *ReadinessResponse) map[string]ReadinessCheck {
	byKey := make(map[string]ReadinessCheck, len(resp.Checks))
	for _, check := range resp.Checks {
		byKey[check.Key] = check
	}
	return byKey
}

func TestReadiness_FailingScenarioBlocksBothSteps(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusInReview
	template.Expression = "totalDistance * 2"
	submitter := pulid.MustNew("usr_")
	template.SubmittedByID = &submitter
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{
		newTestCase("doubles distance", 200, 0.01),
		newTestCase("wrong expectation", 999, 0.01),
	}

	resp, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	assert.False(t, resp.CanSubmit)
	assert.False(t, resp.CanApprove)
	assert.Equal(t, 2, resp.ScenarioTotal)
	assert.Equal(t, 1, resp.ScenarioPassed)
	assert.Equal(t, []string{"wrong expectation"}, resp.ScenarioFailing)

	checks := readinessByKey(resp)
	assert.Equal(t, ReadinessFail, checks[ReadinessCheckScenarios].Status)
	assert.Equal(t, ReadinessPass, checks[ReadinessCheckReviewer].Status)
}

func TestReadiness_SubmitterCannotApproveButOthersCan(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	tenant := newTenantInfo()
	template := newTestTemplate()
	template.Status = formulatemplate.StatusInReview
	template.Expression = "totalDistance * 2"
	template.SubmittedByID = &tenant.UserID
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{newTestCase("ok", 200, 0.01)}

	asSubmitter, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: tenant,
		TemplateID: template.ID,
	})
	require.NoError(t, err)
	assert.False(t, asSubmitter.CanApprove)
	assert.Equal(t, ReadinessFail, readinessByKey(asSubmitter)[ReadinessCheckReviewer].Status)

	other := newTenantInfo()
	asReviewer, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: other,
		TemplateID: template.ID,
	})
	require.NoError(t, err)
	assert.True(t, asReviewer.CanApprove)
	assert.False(t, asReviewer.CanSubmit, "an in-review template is not submittable")
}

func TestReadiness_WarnsAboutUnguardedFieldsAndMissingDescription(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft
	template.Description = ""
	template.Expression = "weight * 0.5"
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	resp, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	assert.True(t, resp.CanSubmit, "warnings do not block")
	checks := readinessByKey(resp)
	assert.Equal(t, ReadinessWarn, checks[ReadinessCheckNullables].Status)
	assert.Contains(t, checks[ReadinessCheckNullables].Detail, "weight")
	assert.Equal(t, ReadinessWarn, checks[ReadinessCheckDescription].Status)
	assert.Equal(t, ReadinessWarn, checks[ReadinessCheckScenarios].Status)
}

func TestReadiness_BrokenExpressionBlocks(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft
	template.Expression = "totalDistance +* 2"
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	resp, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	assert.False(t, resp.CanSubmit)
	assert.Equal(t, ReadinessFail, readinessByKey(resp)[ReadinessCheckExpression].Status)
}

func TestTestExpression_ReturnsAReceipt(t *testing.T) {
	t.Parallel()
	deps := setupTestWithMatrices(t, []*repositories.RateMatrixLookupData{
		rangeMatrixFixture("miles", [][3]string{{"0", "500", "2.5"}, {"500", "", "2.0"}}),
	})

	result := deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "totalDistance * 3",
		SchemaID:   "shipment",
		Variables:  map[string]any{"totalDistance": 600.0},
		TenantInfo: newTenantInfo(),
		MinCharge:  decimal.NewNullDecimal(decimal.NewFromInt(5000)),
		Breakdowns: []*formulatypes.BreakdownDefinition{
			{Name: "linehaul", Label: "Linehaul", Expression: `lookup("miles", totalDistance) * totalDistance`},
		},
	})

	require.True(t, result.Valid, result.Error)
	require.NotNil(t, result.Receipt)
	assert.True(t, decimal.NewFromInt(1800).Equal(result.Receipt.RawAmount),
		"the receipt keeps the pre-guardrail amount")

	var distance *formulatypes.VariableProvenance
	for i := range result.Receipt.Variables {
		if result.Receipt.Variables[i].Name == "totalDistance" {
			distance = &result.Receipt.Variables[i]
		}
	}
	require.NotNil(t, distance)
	assert.Equal(t, formulatypes.ValueSourceSample, distance.Source)

	require.Len(t, result.Receipt.Lookups, 1)
	assert.Equal(t, "linehaul", result.Receipt.Lookups[0].Scope)
	assert.Equal(t, "miles", result.Receipt.Lookups[0].Table)
	require.NotNil(t, result.Receipt.Lookups[0].Match)
	assert.True(t, result.Receipt.Lookups[0].Match.BandMin.Equal(decimal.NewFromInt(500)))
}
