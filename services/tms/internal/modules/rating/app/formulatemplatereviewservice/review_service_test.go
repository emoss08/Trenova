package formulatemplatereviewservice

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatetest"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSubmit_RefusesATemplateWhoseTestsFail(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)

	template := formulatemplatetest.NewTemplate()
	template.Status = formulatemplate.StatusDraft
	template.Source = formulatemplatetest.PerMileFailingSource
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	_, err := deps.svc.Submit(
		t.Context(),
		&SubmitRequest{ApprovalActionRequest: ApprovalActionRequest{
			TenantInfo: formulatemplatetest.NewTenantInfo(),
			EntityID:   template.ID,
		}},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test")
	deps.repo.AssertNotCalled(t, "Update")
}

func retypedForApproval(t *testing.T, deps *testDeps) *formulatemplate.FormulaTemplate {
	t.Helper()
	template := approvableTemplate(deps)
	template.Source = formulatemplatetest.PerMileRetypedSource
	deps.versionRepo.ExpectedCalls = nil
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ApprovedVersionOf(t, template, 1, formulatemplatetest.PerMileSource), nil)
	deps.versionRepo.On("ListAuthorIDsSince", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	return template
}

func TestApprove_TheDependentsGateRefusesInputsTheNewDeclarationsNoLongerTake(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)
	template := retypedForApproval(t, deps)

	ruleID := pulid.MustNew("rar_")
	deps.dependents.Sets = []*rateagreementports.TemplateInputs{{
		Kind:        rateagreementports.TemplateInputsKind(DependentRule),
		ExampleID:   ruleID,
		ExampleName: "Acme 2026, lane 3",
		Count:       4,
		Rule: tarifftypes.Inputs{
			"rate": rules.QuantityValue(decimal.RequireFromString("2.85"), "$/mi"),
		},
	}}

	_, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		EntityID:   template.ID,
	})
	require.Error(t, err)
	fields := formulatemplatetest.ValidationFields(t, err)
	refusal, ok := fields[formulatemplateservice.FieldInputs]
	require.True(t, ok, "the gate refuses on inputs: %v", err)
	_, args := refusal.LocalizedMessage()
	assert.Contains(t, fmt.Sprint(args...), "Acme 2026, lane 3")
	require.Len(t, deps.dependents.Requests, 1)
	assert.Equal(t, template.ID, deps.dependents.Requests[0].TemplateID)
	deps.repo.AssertNotCalled(t, "Update")
}

func TestApprove_DependentsWhoseInputsStillFitAreApprovedAndPinned(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)
	template := retypedForApproval(t, deps)

	deps.dependents.Sets = []*rateagreementports.TemplateInputs{{
		Kind:  rateagreementports.TemplateInputsKind(DependentRule),
		Count: 2,
		Rule: tarifftypes.Inputs{
			"rate": rules.QuantityValue(decimal.NewFromInt(10), "$/cwt"),
		},
	}}
	var snapshot *formulatemplate.FormulaTemplateVersion
	deps.repo.On("Update", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			snapshot = args.Get(1).(*formulatemplate.FormulaTemplateVersion)
		}).
		Return(formulatemplatetest.ReturnsVersion, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	approved, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		EntityID:   template.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, formulatemplate.StatusActive, approved.Status)
	assert.Equal(t, "Money/cwt", approved.Inputs[0].Type)
	require.NotNil(t, snapshot)
	assert.Equal(t, formulatemplatetest.PerMileRetypedSource, snapshot.Source)
	assert.NotNil(t, snapshot.Imports, "the pins are stored, empty when nothing is imported")
}

func TestApprove_UnchangedDeclarationsSkipTheDependents(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)
	template := approvableTemplate(deps)
	deps.versionRepo.ExpectedCalls = nil
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ApprovedVersionOf(t, template, 1, formulatemplatetest.PerMileSource), nil)
	deps.versionRepo.On("ListAuthorIDsSince", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsVersion, nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).
		Return(formulatemplatetest.ReturnsTemplate, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		EntityID:   template.ID,
	})
	require.NoError(t, err)
	assert.Empty(t, deps.dependents.Requests, "the same declarations fit every contract")
}
