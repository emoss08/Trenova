package formulatemplateservice

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func returnsTemplate(_ context.Context, entity *formulatemplate.FormulaTemplate) *formulatemplate.FormulaTemplate {
	return entity
}

func returnsVersion(
	_ context.Context,
	version *formulatemplate.FormulaTemplateVersion,
) *formulatemplate.FormulaTemplateVersion {
	return version
}

func validationFields(t *testing.T, err error) map[string]*errortypes.Error {
	t.Helper()
	fields := map[string]*errortypes.Error{}
	var multi *errortypes.MultiError
	if errors.As(err, &multi) {
		for _, fieldErr := range multi.Errors {
			fields[fieldErr.Field] = fieldErr
		}
		return fields
	}
	var single *errortypes.Error
	require.ErrorAs(t, err, &single)
	fields[single.Field] = single
	return fields
}

func TestCreate_StoresTheSourceWithWhatTheCheckerDerived(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	var snapshot *formulatemplate.FormulaTemplateVersion
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(returnsTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			snapshot = args.Get(1).(*formulatemplate.FormulaTemplateVersion)
		}).
		Return(returnsVersion, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusActive
	approver := pulid.MustNew("usr_")
	template.ApprovedByID = &approver
	created, err := deps.svc.Create(t.Context(), template, newTenantInfo().UserID)
	require.NoError(t, err)

	assert.Equal(t, formulatemplate.StatusDraft, created.Status, "a new template starts as a draft")
	assert.Nil(t, created.ApprovedByID)
	assert.EqualValues(t, 1, created.TestCount)
	assert.Empty(t, created.Imports)
	require.Len(t, created.Inputs, 2)
	assert.Equal(t, "rate", created.Inputs[0].Name)
	assert.Equal(t, "Money/mi", created.Inputs[0].Type)
	assert.True(t, created.Inputs[0].Required)
	assert.Equal(t, "What the contract pays per loaded mile.", created.Inputs[0].Doc)
	assert.Equal(t, "minimumCharge", created.Inputs[1].Name)
	assert.False(t, created.Inputs[1].Required)
	assert.NotEmpty(t, created.Inputs[1].Default)

	require.NotNil(t, snapshot)
	assert.EqualValues(t, 1, snapshot.VersionNumber)
	assert.Equal(t, perMileSource, snapshot.Source)
	assert.Len(t, snapshot.Inputs, 2)
}

func TestCreate_RefusesSourceThatDoesNotCheckAtItsRange(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Source = brokenSource
	_, err := deps.svc.Create(t.Context(), template, newTenantInfo().UserID)
	require.Error(t, err)

	fields := validationFields(t, err)
	refusal, ok := fields[fieldSource]
	require.True(t, ok, "the refusal names the source: %v", err)
	require.NotNil(t, refusal.Range, "the editor can underline it")
	assert.Greater(t, refusal.Range.To, refusal.Range.From)
	deps.repo.AssertNotCalled(t, "Create")
}

func TestUpdate_ASourceChangeRevertsApprovalAndSnapshotsAVersion(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	approver := pulid.MustNew("usr_")
	approvedAt := timeutils.NowUnix() - 3600
	original := newTestTemplate()
	original.ApprovedByID = &approver
	original.ApprovedAt = &approvedAt
	original.Inputs = checkedInputs(t, perMileSource)
	edited := *original
	edited.Source = perMileRetypedSource

	var snapshot *formulatemplate.FormulaTemplateVersion
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(original, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(returnsTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			snapshot = args.Get(1).(*formulatemplate.FormulaTemplateVersion)
		}).
		Return(returnsVersion, nil)
	deps.versionRepo.On("ClearScheduled", mock.Anything, mock.Anything).Return(int64(0), nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	updated, err := deps.svc.Update(t.Context(), &edited, newTenantInfo().UserID)
	require.NoError(t, err)

	assert.Equal(t, formulatemplate.StatusDraft, updated.Status, "a changed source needs a new review")
	assert.Nil(t, updated.ApprovedByID)
	require.Len(t, updated.Inputs, 2)
	assert.Equal(t, "Money/cwt", updated.Inputs[0].Type, "the declarations follow the source")
	require.NotNil(t, snapshot)
	assert.Equal(t, perMileRetypedSource, snapshot.Source)
}

func TestSubmit_RefusesATemplateWhoseTestsFail(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft
	template.Source = perMileFailingSource
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	_, err := deps.svc.Submit(t.Context(), &SubmitRequest{ApprovalActionRequest: ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   template.ID,
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test")
	deps.repo.AssertNotCalled(t, "Update")
}

func retypedForApproval(t *testing.T, deps *testDeps) *formulatemplate.FormulaTemplate {
	t.Helper()
	template := approvableTemplate(deps)
	template.Source = perMileRetypedSource
	deps.versionRepo.ExpectedCalls = nil
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).
		Return(approvedVersionOf(t, template, 1, perMileSource), nil)
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
	deps.dependents.sets = []*rateagreementports.TemplateInputs{{
		Kind:        rateagreementports.TemplateInputsKind(DependentRule),
		ExampleID:   ruleID,
		ExampleName: "Acme 2026, lane 3",
		Count:       4,
		Rule: tarifftypes.Inputs{
			"rate": rules.QuantityValue(decimal.RequireFromString("2.85"), "$/mi"),
		},
	}}

	_, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   template.ID,
	})
	require.Error(t, err)
	fields := validationFields(t, err)
	refusal, ok := fields[fieldInputs]
	require.True(t, ok, "the gate refuses on inputs: %v", err)
	_, args := refusal.LocalizedMessage()
	assert.Contains(t, fmt.Sprint(args...), "Acme 2026, lane 3")
	require.Len(t, deps.dependents.requests, 1)
	assert.Equal(t, template.ID, deps.dependents.requests[0].TemplateID)
	deps.repo.AssertNotCalled(t, "Update")
}

func TestApprove_DependentsWhoseInputsStillFitAreApprovedAndPinned(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)
	template := retypedForApproval(t, deps)

	deps.dependents.sets = []*rateagreementports.TemplateInputs{{
		Kind:  rateagreementports.TemplateInputsKind(DependentRule),
		Count: 2,
		Rule: tarifftypes.Inputs{
			"rate": rules.QuantityValue(decimal.NewFromInt(10), "$/cwt"),
		},
	}}
	var snapshot *formulatemplate.FormulaTemplateVersion
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(returnsTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			snapshot = args.Get(1).(*formulatemplate.FormulaTemplateVersion)
		}).
		Return(returnsVersion, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	approved, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   template.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, formulatemplate.StatusActive, approved.Status)
	assert.Equal(t, "Money/cwt", approved.Inputs[0].Type)
	require.NotNil(t, snapshot)
	assert.Equal(t, perMileRetypedSource, snapshot.Source)
	assert.NotNil(t, snapshot.Imports, "the pins are stored, empty when nothing is imported")
}

func TestApprove_UnchangedDeclarationsSkipTheDependents(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)
	template := approvableTemplate(deps)
	deps.versionRepo.ExpectedCalls = nil
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).
		Return(approvedVersionOf(t, template, 1, perMileSource), nil)
	deps.versionRepo.On("ListAuthorIDsSince", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).Return(returnsVersion, nil)
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(returnsTemplate, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   template.ID,
	})
	require.NoError(t, err)
	assert.Empty(t, deps.dependents.requests, "the same declarations fit every contract")
}

func TestPreview_ASourceThatDoesNotCheckIsNotRated(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result, err := deps.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: newTenantInfo(),
		Source:     brokenSource,
	})
	require.NoError(t, err)
	assert.False(t, result.Valid)
	require.NotEmpty(t, result.Diagnostics)
	assert.Equal(t, rules.SeverityError, result.Diagnostics[0].Severity)
	assert.Nil(t, result.Bill)
	assert.Empty(t, deps.rater.requests)
}

func TestPreview_RatesTheSampleRecordAndRunsTheTests(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.rater.result = &tariffports.RateResult{
		Bill:  &tarifftypes.Bill{Total: decimal.NewFromInt(285)},
		Trace: rules.RawJSON(`[{"kind":"charge"}]`),
	}
	inputs := tarifftypes.Inputs{"rate": rules.QuantityValue(decimal.RequireFromString("2.85"), "$/mi")}

	result, err := deps.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: newTenantInfo(),
		Source:     perMileSource,
		Inputs:     inputs,
	})
	require.NoError(t, err)
	assert.True(t, result.Valid)
	require.NotNil(t, result.Bill)
	assert.Equal(t, "285", result.Bill.Total.String())
	assert.JSONEq(t, `[{"kind":"charge"}]`, string(result.Trace))
	require.Len(t, result.Tests, 1)
	assert.Equal(t, rules.TariffTestPassed, result.Tests[0].Status)
	require.Len(t, result.Inputs, 2)

	require.Len(t, deps.rater.requests, 1)
	request := deps.rater.requests[0]
	assert.Equal(t, perMileSource, request.Template.Source)
	assert.True(t, request.Trace)
	assert.Equal(t, inputs, request.Inputs)
	assert.Empty(t, deps.records.requests, "no shipment rates the sample record")
}

func TestPreview_RatesARealShipmentAndReportsARefusal(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.records.record = rules.RawJSON(`{"shipment":{}}`)
	deps.rater.result = &tariffports.RateResult{
		Refusal: &rules.TariffRefusal{Text: "This rate needs the trip's miles"},
	}
	shipmentID := pulid.MustNew("shp_")

	result, err := deps.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: newTenantInfo(),
		Source:     perMileSource,
		ShipmentID: &shipmentID,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Refusal)
	assert.Equal(t, "This rate needs the trip's miles", result.Refusal.Text)
	require.Len(t, deps.records.requests, 1)
	assert.Equal(t, shipmentID, deps.records.requests[0].ShipmentID)
	assert.JSONEq(t, `{"shipment":{}}`, string(deps.rater.requests[0].Record))
}

func TestPreview_AModuleErrorIsReportedNotReturned(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.rater.err = errortypes.NewBusinessError("A weight is needed").WithParam("code", "none")

	result, err := deps.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: newTenantInfo(),
		Source:     perMileSource,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Error)
	assert.Equal(t, "none", result.Error.Code)

	deps.rater.err = errors.New("the database is gone")
	_, err = deps.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: newTenantInfo(),
		Source:     perMileSource,
	})
	require.Error(t, err, "an infrastructure failure is not a preview result")
}

func TestScaffoldTest_NeedsANameAndAShipment(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	_, err := deps.svc.ScaffoldTest(t.Context(), &ScaffoldTestRequest{
		TenantInfo: newTenantInfo(),
		Source:     perMileSource,
		ShipmentID: pulid.MustNew("shp_"),
	})
	assert.Contains(t, validationFields(t, err), "name")

	_, err = deps.svc.ScaffoldTest(t.Context(), &ScaffoldTestRequest{
		TenantInfo: newTenantInfo(),
		Source:     perMileSource,
		Name:       "from shipment 1042",
	})
	assert.Contains(t, validationFields(t, err), "shipmentId")
	assert.Empty(t, deps.records.requests)
}

func TestTestSource_RunsTheFilesTests(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	passing, err := deps.svc.TestSource(t.Context(), &TestSourceRequest{
		TenantInfo: newTenantInfo(),
		Source:     perMileSource,
	})
	require.NoError(t, err)
	assert.True(t, passing.Check.OK)
	assert.True(t, passing.Tests.Passed())
	assert.Len(t, passing.Inputs, 2)

	failing, err := deps.svc.TestSource(t.Context(), &TestSourceRequest{
		TenantInfo: newTenantInfo(),
		Source:     perMileFailingSource,
	})
	require.NoError(t, err)
	assert.False(t, failing.Tests.Passed())
	require.Len(t, failing.Tests.Tests, 1)
	assert.Equal(t, rules.TariffTestFailed, failing.Tests.Tests[0].Status)
}

func TestBacktest_TakesASourceOrAVersionNotBoth(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	_, err := deps.svc.Backtest(t.Context(), &BacktestRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: pulid.MustNew("ft_"),
	})
	assert.Contains(t, validationFields(t, err), fieldSource)

	version := int64(2)
	_, err = deps.svc.Backtest(t.Context(), &BacktestRequest{
		TenantInfo:    newTenantInfo(),
		TemplateID:    pulid.MustNew("ft_"),
		Source:        perMileSource,
		VersionNumber: &version,
	})
	assert.Contains(t, validationFields(t, err), fieldSource)
}

func TestImport_ChecksEachSourceAndCreatesDrafts(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	deps.repo.On("FindByNames", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplate{}, nil)
	deps.repo.On("Create", mock.Anything, mock.Anything).Return(returnsTemplate, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).Return(returnsVersion, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: ExportVersion,
		TenantInfo:    newTenantInfo(),
		Template: &ImportTemplatePayload{
			Name:        "Per Mile (imported)",
			Description: "From another organisation",
			Source:      perMileSource,
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Created, 1)
	assert.Equal(t, formulatemplate.StatusDraft, result.Created[0].Status)
	assert.Equal(t, perMileSource, result.Created[0].Source)
	assert.Len(t, result.Created[0].Inputs, 2)

	_, err = deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: ExportVersion,
		TenantInfo:    newTenantInfo(),
		Template:      &ImportTemplatePayload{Name: "Broken", Source: brokenSource},
	})
	require.Error(t, err, "an import that does not check is refused whole")

	_, err = deps.svc.Import(t.Context(), &ImportTemplatesRequest{
		ExportVersion: "1.0",
		TenantInfo:    newTenantInfo(),
		Template:      &ImportTemplatePayload{Name: "Old", Source: perMileSource},
	})
	assert.Contains(t, validationFields(t, err), "exportVersion")
}
