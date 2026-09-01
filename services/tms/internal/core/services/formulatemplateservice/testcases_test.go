package formulatemplateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestCase(name string, expected float64, tolerance float64) *formulatemplate.TestCase {
	return &formulatemplate.TestCase{
		ID:             pulid.MustNew("ftc_"),
		TemplateID:     pulid.MustNew("ft_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           name,
		Variables:      map[string]any{"totalDistance": 100.0},
		ExpectedAmount: decimal.NewFromFloat(expected),
		Tolerance:      decimal.NewFromFloat(tolerance),
		CreatedByID:    pulid.MustNew("usr_"),
	}
}

func TestRunTestCases_PassFailAndTolerance(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{
		newTestCase("exact match", 250, 0.01),
		newTestCase("within tolerance", 250.005, 0.01),
		newTestCase("outside tolerance", 260, 0.01),
	}

	result, err := deps.svc.RunTestCases(t.Context(), &RunTestCasesRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, result.Total)
	assert.Equal(t, 2, result.Passed)
	assert.Equal(t, 1, result.Failed)

	byName := make(map[string]*TestCaseResult, len(result.Results))
	for _, item := range result.Results {
		byName[item.Name] = item
	}

	require.NotNil(t, byName["exact match"])
	assert.True(t, byName["exact match"].Passed)
	assert.True(t, byName["exact match"].ActualAmount.Equal(decimal.NewFromInt(250)))

	require.NotNil(t, byName["within tolerance"])
	assert.True(t, byName["within tolerance"].Passed)

	require.NotNil(t, byName["outside tolerance"])
	assert.False(t, byName["outside tolerance"].Passed)
	assert.True(t, byName["outside tolerance"].Difference.Equal(decimal.NewFromInt(-10)))
}

func TestRunTestCases_GuardrailsApplyToScenarios(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.MinCharge = decimal.NewNullDecimal(decimal.NewFromInt(300))
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{
		newTestCase("clamped to minimum", 300, 0.01),
	}

	result, err := deps.svc.RunTestCases(t.Context(), &RunTestCasesRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Passed)
	assert.True(t, result.Results[0].ActualAmount.Equal(decimal.NewFromInt(300)))
}

func TestRunTestCases_CandidateContentOverridesSaved(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{
		newTestCase("uses candidate expression", 500, 0.01),
	}

	result, err := deps.svc.RunTestCases(t.Context(), &RunTestCasesRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		Candidate: &TestCaseCandidate{
			Expression: "totalDistance * 5",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Passed)
}

func TestRunTestCases_CaseVariablesOverrideCandidateDefaults(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	scenario := newTestCase("overrides fuel default", 275, 0.01)
	scenario.Variables = map[string]any{"totalDistance": 100.0, "fuelPct": 10.0}
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{scenario}

	result, err := deps.svc.RunTestCases(t.Context(), &RunTestCasesRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		Candidate: &TestCaseCandidate{
			Expression: "totalDistance * 2.5 * (1 + fuelPct / 100)",
			VariableDefinitions: []*formulatypes.VariableDefinition{
				{
					Name:         "fuelPct",
					Type:         formulatypes.VariableValueTypeNumber,
					DefaultValue: 99.0,
				},
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	assert.Equal(t, 1, result.Passed, "case variables must beat candidate defaults: %+v", result.Results[0])
}

func TestRunTestCases_InvalidExpressionRecordsError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{
		newTestCase("broken candidate", 250, 0.01),
	}

	result, err := deps.svc.RunTestCases(t.Context(), &RunTestCasesRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		Candidate:  &TestCaseCandidate{Expression: "totalDistance +* 2"},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Failed)
	assert.NotEmpty(t, result.Results[0].Error)
}

func TestApprove_BlockedByFailingScenarios(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	tenant := newTenantInfo()
	submitterID := pulid.MustNew("usr_")
	submittedAt := int64(1700000000)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusInReview
	template.SubmittedByID = &submitterID
	template.SubmittedAt = &submittedAt

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{
		newTestCase("this one fails", 999, 0.01),
	}

	result, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: tenant,
		EntityID:   template.ID,
		Comment:    "approving",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "this one fails")
	deps.repo.AssertNotCalled(t, "Update")
}

func TestApprove_PassesWithGreenScenarios(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	tenant := newTenantInfo()
	submitterID := pulid.MustNew("usr_")
	submittedAt := int64(1700000000)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusInReview
	template.SubmittedByID = &submitterID
	template.SubmittedAt = &submittedAt

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{
		newTestCase("green scenario", 250, 0.01),
	}

	result, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: tenant,
		EntityID:   template.ID,
		Comment:    "approving",
	})

	require.NoError(t, err)
	assert.Equal(t, formulatemplate.StatusActive, result.Status)
}

func TestCreateTestCase_DefaultsToleranceAndAudits(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	created, err := deps.svc.CreateTestCase(t.Context(), &CreateTestCaseRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		TestCaseInput: TestCaseInput{
			Name:           "500 mile load",
			Variables:      map[string]any{"totalDistance": 500.0},
			ExpectedAmount: decimal.NewFromInt(1250),
		},
	})

	require.NoError(t, err)
	assert.True(t, created.Tolerance.Equal(decimal.NewFromFloat(0.01)))
	require.Len(t, deps.testCaseRepo.created, 1)
}

func TestCreateTestCase_RequiresName(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	_, err := deps.svc.CreateTestCase(t.Context(), &CreateTestCaseRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		TestCaseInput: TestCaseInput{
			ExpectedAmount: decimal.NewFromInt(100),
		},
	})

	require.Error(t, err)
	assert.Empty(t, deps.testCaseRepo.created)
}

func TestDeleteTestCase_Delegates(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	caseID := pulid.MustNew("ftc_")
	err := deps.svc.DeleteTestCase(t.Context(), repositories.GetTestCaseByIDRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: pulid.MustNew("ft_"),
		TestCaseID: caseID,
	})

	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{caseID}, deps.testCaseRepo.deleted)
}
