package formulatemplateservice

import (
	"time"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func exactMatrixFixture(
	code string,
	entries map[string]string,
) *ratematrixports.RateMatrixLookupData {
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

	return &ratematrixports.RateMatrixLookupData{
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

func rangeMatrixFixture(code string, bands [][3]string) *ratematrixports.RateMatrixLookupData {
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

	return &ratematrixports.RateMatrixLookupData{
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
	require.True(t, result.Valid, failureOf(result))
	require.NotNil(t, result.Result)
	return *result.Result
}

func TestTestExpression_PreviewUsesRealRateTables(t *testing.T) {
	t.Parallel()
	deps := setupTestWithMatrices(t, []*ratematrixports.RateMatrixLookupData{
		exactMatrixFixture("fsc", map[string]string{"DIESEL": "0.35"}),
	})

	result := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: `lookup("fsc", "DIESEL") * totalDistance`,
		SchemaID:   "shipment",
		Variables:  map[string]any{"totalDistance": 100.0},
		TenantInfo: newTenantInfo(),
	}))

	assert.True(t, decimal.NewFromFloat(35).Equal(resultAmount(t, result)), result.Result)
}

func TestTestExpression_BreakdownLinesUseRealRateTables(t *testing.T) {
	t.Parallel()
	deps := setupTestWithMatrices(t, []*ratematrixports.RateMatrixLookupData{
		rangeMatrixFixture("miles", [][3]string{{"0", "500", "2.5"}, {"500", "", "2.0"}}),
	})

	result := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
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
	}))

	require.True(t, result.Valid, failureOf(result))
	require.Len(t, result.Breakdown, 1)
	assert.Empty(t, result.Breakdown[0].Error)
	assert.True(t, decimal.NewFromInt(1200).Equal(result.Breakdown[0].Amount))
}

func TestTestExpression_LookupOrFallsBackOnlyWhenKeyMisses(t *testing.T) {
	t.Parallel()
	deps := setupTestWithMatrices(t, []*ratematrixports.RateMatrixLookupData{
		exactMatrixFixture("fsc", map[string]string{"DIESEL": "0.35"}),
	})

	miss := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: `lookupOr("fsc", "GAS", 9)`,
		SchemaID:   "shipment",
		Variables:  map[string]any{},
		TenantInfo: newTenantInfo(),
	}))
	assert.True(t, decimal.NewFromInt(9).Equal(resultAmount(t, miss)))

	missingTable := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: `lookupOr("nope", "GAS", 9)`,
		SchemaID:   "shipment",
		Variables:  map[string]any{},
		TenantInfo: newTenantInfo(),
	}))
	assert.False(t, missingTable.Valid, "a missing table is no miss to fall back on")
	refusal := missingTable.FirstErrorDiagnostic()
	require.NotNil(t, refusal)
	assert.Equal(t, "expression", refusal.Scope)
	assert.Equal(t, "rate-table", refusal.Code)
	assert.Contains(t, failureOf(missingTable), "nope")
}

func TestTestExpression_WithoutTableReferencesNeverLoadsTables(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "totalDistance * 2",
		SchemaID:   "shipment",
		Variables:  map[string]any{"totalDistance": 10.0},
		TenantInfo: newTenantInfo(),
	}))

	assert.True(t, decimal.NewFromInt(20).Equal(resultAmount(t, result)))
}

func TestRunTestCases_ScenarioUsesRealRateTable(t *testing.T) {
	t.Parallel()
	deps := setupTestWithMatrices(t, []*ratematrixports.RateMatrixLookupData{
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

	halfUp := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "10 / 3",
		SchemaID:   "shipment",
		Variables:  map[string]any{},
		TenantInfo: newTenantInfo(),
	}))
	assert.True(t, decimal.RequireFromString("3.33").Equal(resultAmount(t, halfUp)), halfUp.Result)
	require.NotNil(t, halfUp.Rounding)
	assert.True(t, halfUp.Rounding.Applied)
	assert.Equal(t, "HalfUp", halfUp.Rounding.Mode)

	wholeUp := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression:        "10 / 3",
		SchemaID:          "shipment",
		Variables:         map[string]any{},
		TenantInfo:        newTenantInfo(),
		RoundingMode:      ratetypes.RoundingModeUp,
		RoundingPrecision: 0,
	}))
	assert.True(t, decimal.NewFromInt(4).Equal(resultAmount(t, wholeUp)), wholeUp.Result)
}

func TestTestExpression_GuardrailFloorIsNotRoundedAway(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression:        "249.996",
		SchemaID:          "shipment",
		Variables:         map[string]any{},
		TenantInfo:        newTenantInfo(),
		MinCharge:         decimal.NewNullDecimal(decimal.RequireFromString("250.00")),
		RoundingMode:      ratetypes.RoundingModeDown,
		RoundingPrecision: 0,
	}))

	assert.True(t, decimal.NewFromInt(250).Equal(resultAmount(t, result)), result.Result)
	require.NotNil(t, result.Guardrail)
	assert.True(t, result.Guardrail.Applied)
}

func TestTestExpression_BooleanExpressionIsRejected(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "totalDistance > 100",
		SchemaID:   "shipment",
		Variables:  map[string]any{"totalDistance": 500.0},
		TenantInfo: newTenantInfo(),
	}))

	assert.False(t, result.Valid)
	assert.Contains(t, failureOf(result), "true/false")
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

func TestTestExpression_ReportsUnguardedNullableFieldsAsScopedDiagnostics(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "weight * 0.5",
		SchemaID:   "shipment",
		Variables:  map[string]any{"weight": 1000.0},
		TenantInfo: newTenantInfo(),
		Breakdowns: []*formulatypes.BreakdownDefinition{
			{Name: "perPound", Label: "Per pound", Expression: "weight * 0.5"},
			{Name: "safe", Label: "Guarded", Expression: "coalesce(weight, 0) * 0.5"},
		},
	}))

	require.True(t, result.Valid, failureOf(result))
	require.Len(t, result.Diagnostics, 2)
	assert.Equal(t, "expression", result.Diagnostics[0].Scope)
	assert.Equal(t, "nullable", result.Diagnostics[0].Code)
	assert.Equal(t, "warning", result.Diagnostics[0].Severity)
	assert.Equal(t, "weight", result.Diagnostics[0].Name)
	assert.Equal(t, "coalesce(weight, 0)", result.Diagnostics[0].Fixes[0].Changes[0].Insert)
	assert.Equal(t, "breakdowns[0].expression", result.Diagnostics[1].Scope)
	assert.Equal(t, []string{"weight"}, result.Unguarded())
}

func TestTestExpression_GuardedNullableFieldHasNoWarning(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "coalesce(weight, 0) * 0.5 + totalDistance",
		SchemaID:   "shipment",
		Variables:  map[string]any{"totalDistance": 10.0},
		TenantInfo: newTenantInfo(),
	}))

	require.True(t, result.Valid, failureOf(result))
	assert.Empty(t, result.Diagnostics)
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

	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
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

	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
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

func TestReadiness_WarnsAboutUnguardedFieldsAndFailsWithoutDescriptionOrScenarios(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft
	template.Description = ""
	template.Expression = "weight * 0.5"
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	resp, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	assert.False(t, resp.CanSubmit)
	checks := readinessByKey(resp)
	assert.Equal(t, ReadinessWarn, checks[ReadinessCheckNullables].Status)
	assert.Equal(t, "1 value can be empty: weight", checks[ReadinessCheckNullables].Detail)
	assert.Equal(t, ReadinessFail, checks[ReadinessCheckDetails].Status)
	assert.Equal(t, "Description is missing", checks[ReadinessCheckDetails].Detail)
	assert.Equal(t, ReadinessFail, checks[ReadinessCheckScenarios].Status)
	assert.Equal(t, "Add at least one", checks[ReadinessCheckScenarios].Detail)
}

func TestReadiness_WarningsDoNotBlock(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft
	template.Expression = "(weight ?? 0) * 0 + totalDistance * 2.5"
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{newTestCase("exact", 250, 0.01)}

	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	resp, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	assert.True(t, resp.CanSubmit)
	checks := readinessByKey(resp)
	assert.Equal(t, ReadinessPass, checks[ReadinessCheckDetails].Status)
	assert.Equal(t, ReadinessPass, checks[ReadinessCheckScenarios].Status)
	assert.Equal(t, "1 of 1 pass", checks[ReadinessCheckScenarios].Detail)
}

func TestReadiness_BrokenExpressionBlocks(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft
	template.Expression = "totalDistance +* 2"
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
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
	deps := setupTestWithMatrices(t, []*ratematrixports.RateMatrixLookupData{
		rangeMatrixFixture("miles", [][3]string{{"0", "500", "2.5"}, {"500", "", "2.0"}}),
	})

	result := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "totalDistance * 3",
		SchemaID:   "shipment",
		Variables:  map[string]any{"totalDistance": 600.0},
		TenantInfo: newTenantInfo(),
		MinCharge:  decimal.NewNullDecimal(decimal.NewFromInt(5000)),
		Breakdowns: []*formulatypes.BreakdownDefinition{
			{
				Name:       "linehaul",
				Label:      "Linehaul",
				Expression: `lookup("miles", totalDistance) * totalDistance`,
			},
		},
	}))

	require.True(t, result.Valid, failureOf(result))
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

func TestReviewDiff_ComparesLastApprovedSnapshotToPendingContent(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusInReview
	template.Expression = "totalDistance * 3"
	template.CurrentVersionNumber = 4
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.MatchedBy(
		func(req *formulatemplateports.GetLatestVersionByStatusRequest) bool {
			return req.Status == formulatemplate.StatusActive && req.TemplateID == template.ID
		},
	)).Return(newApprovedSnapshot(template, 2, "totalDistance * 2"), nil)

	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	diff, err := deps.svc.ReviewDiff(t.Context(), &ReviewDiffRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	assert.True(t, diff.HasApprovedBase)
	assert.Equal(t, int64(2), diff.BaseVersion)
	assert.Equal(t, int64(4), diff.CurrentVersion)
	assert.Equal(t, "totalDistance * 2", diff.BaseExpression)
	assert.Equal(t, "totalDistance * 3", diff.CurrentExpression)
	assert.Contains(t, diff.Changes, "expression")
	assert.Positive(t, diff.ChangeCount)
}

func TestReviewDiff_WithoutApprovedHistoryShowsEverythingAsNew(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusInReview
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).Return(nil, nil)

	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	diff, err := deps.svc.ReviewDiff(t.Context(), &ReviewDiffRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	assert.False(t, diff.HasApprovedBase)
	assert.Equal(t, int64(0), diff.BaseVersion)
	assert.Empty(t, diff.BaseExpression)
	assert.Equal(t, template.Expression, diff.CurrentExpression)
}

func TestListStandards_ExposesTheCatalog(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	standards, err := deps.svc.ListStandards()

	require.NoError(t, err)
	require.NotEmpty(t, standards)
	for _, standard := range standards {
		assert.NotEmpty(t, standard.Name)
		assert.NotEmpty(t, standard.Expression)
		assert.NotEmpty(t, standard.SchemaID)
		assert.NotEmpty(t, standard.Type)
	}
}

func TestBuildBacktestSummary_CountsClampsAndFailuresSeparately(t *testing.T) {
	t.Parallel()

	results := []*BacktestResult{
		{
			CurrentAmount:    decimal.NewFromInt(100),
			CandidateAmount:  decimal.NewFromInt(150),
			GuardrailApplied: true,
		},
		{CurrentAmount: decimal.NewFromInt(100), CandidateAmount: decimal.NewFromInt(100)},
		{CurrentError: "no rating detail"},
		{CandidateError: "boom", GuardrailApplied: true},
		{CurrentError: "x", CandidateError: "y"},
	}

	summary := buildBacktestSummary(results)

	assert.Equal(t, 1, summary.GuardrailCount, "clamps count only on rows that evaluated")
	assert.Equal(t, 2, summary.CurrentErrorCount)
	assert.Equal(t, 2, summary.CandidateErrorCount)
	assert.Equal(t, 3, summary.ErrorCount)
}

type stubContextProvider struct {
	variables map[string]any
}

func (s *stubContextProvider) ContextVariables(
	_ context.Context,
	_ pagination.TenantInfo,
) (map[string]any, error) {
	return s.variables, nil
}

func TestTestExpression_PreviewSeesProvidedContextVariables(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.svc.formulaService = newFormulaServiceWithProviders(
		t,
		&stubContextProvider{variables: map[string]any{"fuelPrice": 3.85}},
	)

	resp := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "100 * fuelPrice",
		SchemaID:   "shipment",
		TenantInfo: newTenantInfo(),
	}))

	require.True(t, resp.Valid, failureOf(resp))
	assert.InDelta(t, 385, resultAmount(t, resp).InexactFloat64(), 0.0001)
}

func TestReadiness_WarnsWhenAFreightChargeReadsTheStoredTotal(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.svc.formulaService = ratingtest.NewFormulaService(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft
	template.Type = formulatemplate.TemplateTypeFreightCharge
	template.Expression = "max(currentTotalCharge * 0.1, 50)"
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	resp, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	check := readinessByKey(resp)[ReadinessCheckNullables]
	assert.Equal(t, ReadinessWarn, check.Status)
	assert.Contains(t, check.Detail, "currentTotalCharge")
}

func TestTestExpression_LoadsTablesNamedInsideTheEnginesOwnFunctions(t *testing.T) {
	t.Parallel()
	deps := setupTestWithMatrices(t, []*ratematrixports.RateMatrixLookupData{
		exactMatrixFixture("fsc", map[string]string{"DIESEL": "0.35"}),
	})

	result := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: `sum(lookup("fsc", "DIESEL") * totalDistance, 10, 25)`,
		SchemaID:   "shipment",
		Variables:  map[string]any{"totalDistance": 100.0},
		TenantInfo: newTenantInfo(),
	}))

	require.True(t, result.Valid, failureOf(result))
	assert.True(t, decimal.NewFromFloat(70).Equal(resultAmount(t, result)), result.Result)
}

func TestReviewDiffAndReadiness_ListPendingSchedules(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	pending := timeutils.NowUnix() + 10*86_400
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).Return(nil, nil)
	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplateVersion{
			{VersionNumber: 6, Status: formulatemplate.StatusActive, EffectiveFrom: &pending},
		}, nil)

	diff, err := deps.svc.ReviewDiff(t.Context(), &ReviewDiffRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, []ScheduledWindow{{VersionNumber: 6, EffectiveFrom: pending}}, diff.PendingSchedules)

	readiness, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, []ScheduledWindow{{VersionNumber: 6, EffectiveFrom: pending}}, readiness.PendingSchedules)
}

func TestTestExpression_EnforcesTheSaveTimeCaps(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	_, err := deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: strings.Repeat("1 + ", formulatemplate.MaxExpressionLength/4) + "1",
		SchemaID:   "shipment",
		TenantInfo: newTenantInfo(),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Contains(t, err.Error(), "10,000")

	breakdowns := make([]*formulatypes.BreakdownDefinition, formulatypes.MaxBreakdownDefinitions+1)
	for i := range breakdowns {
		breakdowns[i] = &formulatypes.BreakdownDefinition{Name: "line", Label: "Line", Expression: "1"}
	}
	_, err = deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "1",
		SchemaID:   "shipment",
		TenantInfo: newTenantInfo(),
		Breakdowns: breakdowns,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "20 breakdown")
}

func must(t *testing.T) func(*TestExpressionResponse, error) *TestExpressionResponse {
	t.Helper()
	return func(resp *TestExpressionResponse, err error) *TestExpressionResponse {
		t.Helper()
		require.NoError(t, err)
		require.NotNil(t, resp)
		return resp
	}
}

func failureOf(resp *TestExpressionResponse) string {
	return resp.FailureText(i18n.Default)
}

type stalledContextProvider struct{}

func (stalledContextProvider) ContextVariables(
	ctx context.Context,
	_ pagination.TenantInfo,
) (map[string]any, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestTestExpression_ThePreviewStopsAtItsDeadlineAndSaysSo(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.svc.formulaService = newFormulaServiceWithProviders(t, stalledContextProvider{})

	started := time.Now()
	resp := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression: "100",
		SchemaID:   "shipment",
		TenantInfo: newTenantInfo(),
	}))

	assert.Less(t, time.Since(started), previewEvaluationTimeout+time.Second)
	assert.False(t, resp.Valid)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "expression", resp.Error.Scope)
	assert.Equal(t, CodeTimeout, resp.Error.Err.Code)
	assert.Contains(t, failureOf(resp), "2s")
}
