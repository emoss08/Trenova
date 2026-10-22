package formulatemplateservice

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
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

type stubRoles struct {
	permissionports.RoleRepository
	users []permissionports.PermittedUser
	asked []permissionports.ListUsersWithPermissionRequest
}

func (s *stubRoles) ListUsersWithPermission(
	_ context.Context,
	req permissionports.ListUsersWithPermissionRequest,
) ([]permissionports.PermittedUser, error) {
	s.asked = append(s.asked, req)
	return s.users, nil
}

func fieldsOf(t *testing.T, err error) map[string]string {
	t.Helper()

	multiErr, ok := err.(*errortypes.MultiError)
	require.True(t, ok, "expected field errors, got %v", err)
	fields := make(map[string]string, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields[fieldErr.Field] = fieldErr.Error()
	}
	return fields
}

func decimalPtr(value string) *string {
	return &value
}

func TestTestExpression_TracesStatementsOnRequest(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	request := &TestExpressionRequest{
		Expression: "let base = totalDistance * rate; let fuel = base * 0.1; base + fuel",
		SchemaID:   "shipment",
		Variables:  map[string]any{"totalDistance": 100.0, "rate": 2.0},
		TenantInfo: newTenantInfo(),
		MinCharge:  decimal.NewNullDecimal(decimal.NewFromInt(500)),
	}

	untraced := must(t)(deps.svc.TestExpression(t.Context(), request))
	require.True(t, untraced.Valid, failureOf(untraced))
	assert.Empty(t, untraced.Statements, "the live preview leaves tracing off")

	request.TraceStatements = true
	traced := must(t)(deps.svc.TestExpression(t.Context(), request))
	require.True(t, traced.Valid, failureOf(traced))
	require.Len(t, traced.Statements, 3)
	assert.Equal(t, "200", *TraceValueText(traced.Statements[0].Value))
	assert.Equal(t, "20", *TraceValueText(traced.Statements[1].Value))
	assert.Equal(t, formulatemplatetypes.StatementResult, traced.Statements[2].Kind)
	assert.Equal(t, "220", *TraceValueText(traced.Statements[2].Value), "the result is the raw amount")
	assert.True(t, traced.Result.Equal(decimal.NewFromInt(500)))
}

func TestTestExpression_TraceSurvivesARuntimeFailure(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result := must(t)(deps.svc.TestExpression(t.Context(), &TestExpressionRequest{
		Expression:      "let fine = totalDistance * 2; fine / (totalDistance - 100)",
		SchemaID:        "shipment",
		Variables:       map[string]any{"totalDistance": 100.0},
		TenantInfo:      newTenantInfo(),
		TraceStatements: true,
	}))

	assert.False(t, result.Valid)
	require.NotNil(t, result.Error)
	assert.Equal(t, "divideByZero", result.Error.Err.Code)
	require.Len(t, result.Statements, 2)
	assert.Equal(t, "200", *TraceValueText(result.Statements[0].Value))
	require.NotNil(t, result.Statements[1].Error)
	assert.Equal(t, "divideByZero", result.Statements[1].Error.Code)
}

func TestCreateTestCase_RecordsTheCandidateItWasPricedOn(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	tenant := newTenantInfo()
	template := newTestTemplate()
	template.CurrentVersionNumber = 4
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{VersionNumber: 3}, nil)

	candidate := &TestCaseCandidate{Expression: "totalDistance * 3"}
	created, err := deps.svc.CreateTestCase(t.Context(), &CreateTestCaseRequest{
		TenantInfo: tenant,
		TemplateID: template.ID,
		TestCaseInput: TestCaseInput{
			Name:           "Hundred miles",
			Variables:      map[string]any{"totalDistance": 100.0},
			ExpectedAmount: decimal.NewFromInt(300),
			Candidate:      candidate,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, created.RecordedOnVersion)
	assert.EqualValues(t, 4, *created.RecordedOnVersion)
	require.NotNil(t, created.RecordedByID)
	assert.Equal(t, tenant.UserID, *created.RecordedByID)
	require.NotNil(t, created.RecordedAt)
	assert.Positive(t, *created.RecordedAt)
	require.NotNil(t, created.RecordedContent)
	assert.Equal(t, "totalDistance * 3", created.RecordedContent.Expression)
}

func TestCreateTestCase_RecordsTheSavedContentBeforeAnyApproval(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).Return(nil, nil)

	created, err := deps.svc.CreateTestCase(t.Context(), &CreateTestCaseRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		TestCaseInput: TestCaseInput{
			Name:           "Hundred miles",
			Variables:      map[string]any{"totalDistance": 100.0},
			ExpectedAmount: decimal.NewFromInt(250),
		},
	})

	require.NoError(t, err)
	require.NotNil(t, created.RecordedOnVersion)
	assert.EqualValues(t, 0, *created.RecordedOnVersion)
	require.NotNil(t, created.RecordedContent)
	assert.Equal(t, template.Expression, created.RecordedContent.Expression)
}

func TestCreateTestCase_RefusesAnOversizedCandidate(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	_, err := deps.svc.CreateTestCase(t.Context(), &CreateTestCaseRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		TestCaseInput: TestCaseInput{
			Name:           "Bad",
			ExpectedAmount: decimal.NewFromInt(1),
			Candidate: &TestCaseCandidate{
				Expression: "1",
				MinCharge:  decimal.NewNullDecimal(decimal.NewFromInt(10)),
				MaxCharge:  decimal.NewNullDecimal(decimal.NewFromInt(5)),
			},
		},
	})

	require.Error(t, err)
	assert.Contains(t, fieldsOf(t, err), "candidate.minCharge")
	assert.Empty(t, deps.testCaseRepo.created)
}

func TestUpdateTestCase_RerecordsOnlyWithACandidate(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	original := newTestCase("Hundred miles", 250, 0.01)
	original.TemplateID = template.ID
	firstRecorder := pulid.MustNew("usr_")
	original.Record(template.ScenarioContent(), 0, firstRecorder, 100)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{original}
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything).Return(nil)

	editor := newTenantInfo()
	updated, err := deps.svc.UpdateTestCase(t.Context(), &UpdateTestCaseRequest{
		TenantInfo: editor,
		TemplateID: template.ID,
		TestCaseID: original.ID,
		TestCaseInput: TestCaseInput{
			Name:           "Renamed",
			Variables:      map[string]any{"totalDistance": 100.0},
			ExpectedAmount: decimal.NewFromInt(250),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, updated.RecordedByID)
	assert.Equal(t, firstRecorder, *updated.RecordedByID, "no candidate leaves the recording")
	assert.EqualValues(t, 100, *updated.RecordedAt)

	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).Return(nil, nil)
	rerecorded, err := deps.svc.UpdateTestCase(t.Context(), &UpdateTestCaseRequest{
		TenantInfo: editor,
		TemplateID: template.ID,
		TestCaseID: original.ID,
		TestCaseInput: TestCaseInput{
			Name:           "Renamed",
			Variables:      map[string]any{"totalDistance": 100.0},
			ExpectedAmount: decimal.NewFromInt(300),
			Candidate:      &TestCaseCandidate{Expression: "totalDistance * 3"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, editor.UserID, *rerecorded.RecordedByID)
	assert.Greater(t, *rerecorded.RecordedAt, int64(100))
	assert.Equal(t, "totalDistance * 3", rerecorded.RecordedContent.Expression)
}

func TestCreateWithTestCases_SavesHeldScenariosWithTheTemplate(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestTemplate()
	entity.ID = pulid.Nil
	deps.repo.On("Create", mock.Anything, mock.Anything).
		Return(func(_ context.Context, created *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			created.ID = pulid.MustNew("ft_")
			return created, nil
		})
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	userID := pulid.MustNew("usr_")
	created, err := deps.svc.CreateWithTestCases(t.Context(), entity, userID, []*TestCaseInput{
		{
			Name:           "Hundred miles",
			Variables:      map[string]any{"totalDistance": 100.0},
			ExpectedAmount: decimal.NewFromInt(250),
		},
		{
			Name:           "Two hundred miles",
			Variables:      map[string]any{"totalDistance": 200.0},
			ExpectedAmount: decimal.NewFromInt(600),
			Candidate:      &TestCaseCandidate{Expression: "totalDistance * 3"},
		},
	})

	require.NoError(t, err)
	require.Len(t, deps.testCaseRepo.created, 2)
	for _, testCase := range deps.testCaseRepo.created {
		assert.Equal(t, created.ID, testCase.TemplateID)
		assert.Equal(t, created.OrganizationID, testCase.OrganizationID)
		require.NotNil(t, testCase.RecordedOnVersion)
		assert.EqualValues(t, 0, *testCase.RecordedOnVersion)
		assert.Equal(t, userID, *testCase.RecordedByID)
	}
	assert.Equal(t, "totalDistance * 2.5", deps.testCaseRepo.created[0].RecordedContent.Expression)
	assert.Equal(t, "totalDistance * 3", deps.testCaseRepo.created[1].RecordedContent.Expression)
}

func TestCreateWithTestCases_RefusesDuplicateAndTooManyScenarios(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestTemplate()
	_, err := deps.svc.CreateWithTestCases(t.Context(), entity, pulid.MustNew("usr_"),
		[]*TestCaseInput{
			{Name: "Same", ExpectedAmount: decimal.NewFromInt(1)},
			{Name: "Same", ExpectedAmount: decimal.NewFromInt(2)},
			{Name: "", ExpectedAmount: decimal.NewFromInt(2)},
		})
	require.Error(t, err)
	fields := fieldsOf(t, err)
	assert.Contains(t, fields, "testCases[1].name")
	assert.Contains(t, fields, "testCases[2].name")

	tooMany := make([]*TestCaseInput, 0, MaxHeldTestCases+1)
	for range MaxHeldTestCases + 1 {
		tooMany = append(tooMany, &TestCaseInput{Name: pulid.MustNew("x_").String()})
	}
	_, err = deps.svc.CreateWithTestCases(t.Context(), entity, pulid.MustNew("usr_"), tooMany)
	require.Error(t, err)
	assert.Contains(t, fieldsOf(t, err), "testCases")
	deps.repo.AssertNotCalled(t, "Create")
}

func TestReadiness_ChecksMirrorTheStudio(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft
	template.Expression = "totalDistance * 2.5 + 100"
	template.BreakdownDefinitions = []*formulatypes.BreakdownDefinition{
		{Name: "linehaul", Label: "Linehaul", Expression: "totalDistance * 2.5"},
		{Name: "handling", Label: "Handling", Expression: "60"},
	}
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{newTestCase("exact", 350, 0.01)}

	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	resp, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})
	require.NoError(t, err)

	keys := make([]string, 0, len(resp.Checks))
	labels := make(map[string]string, len(resp.Checks))
	for _, check := range resp.Checks {
		keys = append(keys, check.Key)
		labels[check.Key] = check.Label
	}
	assert.Equal(t, []string{
		ReadinessCheckDetails,
		ReadinessCheckExpression,
		ReadinessCheckNullables,
		ReadinessCheckScenarios,
		ReadinessCheckBreakdown,
		ReadinessCheckStatus,
		ReadinessCheckReviewer,
		ReadinessCheckAge,
	}, keys)
	assert.Equal(t, "Name and description", labels[ReadinessCheckDetails])
	assert.Equal(t, "Expression is valid", labels[ReadinessCheckExpression])
	assert.Equal(t, "Safe on every shipment", labels[ReadinessCheckNullables])
	assert.Equal(t, "Scenarios pass", labels[ReadinessCheckScenarios])
	assert.Equal(t, "Breakdown reconciles", labels[ReadinessCheckBreakdown])

	breakdown := readinessByKey(resp)[ReadinessCheckBreakdown]
	assert.Equal(t, ReadinessWarn, breakdown.Status)
	assert.Equal(
		t,
		"Lines don't add up to the formula: they come to $60.00 and the formula to $100.00",
		breakdown.Detail,
	)
	assert.True(t, resp.CanSubmit, "a breakdown warning does not block")
}

func TestReadiness_BreakdownReconcilesThroughTheRoundingRemainder(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft
	template.Expression = "10.001 + 20.001 + 30.001"
	template.RoundingMode = ratetypes.RoundingModeUp
	template.RoundingPrecision = 2
	template.BreakdownDefinitions = []*formulatypes.BreakdownDefinition{
		{Name: "linehaul", Label: "Linehaul", Expression: "10.001"},
		{Name: "fuel", Label: "Fuel", Expression: "20.001"},
		{Name: "handling", Label: "Handling", Expression: "30.001"},
	}
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	resp, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})
	require.NoError(t, err)

	breakdown := readinessByKey(resp)[ReadinessCheckBreakdown]
	assert.Equal(t, ReadinessPass, breakdown.Status, breakdown.Detail)
	assert.Equal(t, "Lines explain the total", breakdown.Detail)
}

func TestReadiness_BreakdownWarnsWhenAGuardrailMovesTheTotal(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft
	template.Expression = "50"
	template.MinCharge = decimal.NewNullDecimal(decimal.NewFromInt(100))
	template.BreakdownDefinitions = []*formulatypes.BreakdownDefinition{
		{Name: "flat", Label: "Flat", Expression: "50"},
	}
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	resp, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
	})
	require.NoError(t, err)

	breakdown := readinessByKey(resp)[ReadinessCheckBreakdown]
	assert.Equal(t, ReadinessWarn, breakdown.Status)
	assert.Contains(t, breakdown.Detail, "minimum")
}

func TestSubmit_RefusesWhatReadinessWouldFail(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft
	template.Description = "  "
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	_, err := deps.svc.Submit(t.Context(), &SubmitRequest{
		ApprovalActionRequest: ApprovalActionRequest{
			TenantInfo: newTenantInfo(),
			EntityID:   template.ID,
		},
	})

	require.Error(t, err)
	fields := fieldsOf(t, err)
	assert.Equal(t, "Description is missing", fields[ReadinessCheckDetails])
	assert.Contains(t, fields[ReadinessCheckScenarios], "at least one")
	assert.NotContains(t, fields, ReadinessCheckExpression)
	deps.repo.AssertNotCalled(t, "Update")
}

func TestApprove_RequiresAtLeastOneScenario(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	submitter := pulid.MustNew("usr_")
	submittedAt := timeutils.NowUnix() - 60
	template := newTestTemplate()
	template.Status = formulatemplate.StatusInReview
	template.SubmittedByID = &submitter
	template.SubmittedAt = &submittedAt
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	deps.versionRepo.On("ListAuthorIDsSince", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	_, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   template.ID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one test scenario")
}

func TestSubmit_ValidatesAndRecordsTheChosenReviewer(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	captured := withNotifications(t, deps)
	reviews := primeReviews(deps, nil)

	tenant := newTenantInfo()
	reviewer := pulid.MustNew("usr_")
	roles := &stubRoles{users: []permissionports.PermittedUser{
		{UserID: tenant.UserID, Name: "Me"},
		{UserID: reviewer, Name: "Priya Raman", Username: "praman"},
	}}
	deps.svc.roles = roles

	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{newTestCase("exact", 250, 0.01)}
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).Return(nil, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Submit(t.Context(), &SubmitRequest{
		ApprovalActionRequest: ApprovalActionRequest{TenantInfo: tenant, EntityID: template.ID},
		ReviewerID:            tenant.UserID,
	})
	require.Error(t, err)
	assert.Contains(t, fieldsOf(t, err), "reviewerId", "the submitter cannot review their own work")

	_, err = deps.svc.Submit(t.Context(), &SubmitRequest{
		ApprovalActionRequest: ApprovalActionRequest{TenantInfo: tenant, EntityID: template.ID},
		ReviewerID:            pulid.MustNew("usr_"),
	})
	require.Error(t, err)
	assert.Contains(t, fieldsOf(t, err), "reviewerId")

	_, err = deps.svc.Submit(t.Context(), &SubmitRequest{
		ApprovalActionRequest: ApprovalActionRequest{
			TenantInfo: tenant,
			EntityID:   template.ID,
			Comment:    "ready",
		},
		ReviewerID:   reviewer,
		ReviewerNote: "  Raised the per-mile rate.  ",
	})
	require.NoError(t, err)

	require.Len(t, reviews.created, 1)
	entry := reviews.created[0]
	require.NotNil(t, entry.RequestedReviewerID)
	assert.Equal(t, reviewer, *entry.RequestedReviewerID)
	assert.Equal(t, "Raised the per-mile rate.", entry.ReviewerNote)

	notified := *captured
	require.NotNil(t, notified)
	require.NotNil(t, notified.TargetUserID)
	assert.Equal(t, reviewer, *notified.TargetUserID)
	assert.Equal(t, eventTemplateReviewRequested, notified.EventType)
	assert.Equal(t, templateReviewLink(template.ID), notified.Data["link"])
}

func TestSubmit_RefusesAnOverlongReviewerNote(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{newTestCase("exact", 250, 0.01)}
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	note := make([]byte, formulatemplate.MaxReviewerNoteLength+1)
	for index := range note {
		note[index] = 'a'
	}
	_, err := deps.svc.Submit(t.Context(), &SubmitRequest{
		ApprovalActionRequest: ApprovalActionRequest{
			TenantInfo: newTenantInfo(),
			EntityID:   template.ID,
		},
		ReviewerNote: string(note),
	})

	require.Error(t, err)
	assert.Contains(t, fieldsOf(t, err), "reviewerNote")
}

func TestListReviewers_ExcludesTheCallerAndNamesWithoutEmail(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	tenant := newTenantInfo()
	other := pulid.MustNew("usr_")
	deps.svc.roles = &stubRoles{users: []permissionports.PermittedUser{
		{UserID: tenant.UserID, Name: "Me"},
		{
			UserID:        other,
			Name:          "Priya Raman",
			Username:      "praman",
			EmailAddress:  "priya@example.com",
			ProfilePicURL: "https://example.com/p.png",
		},
	}}
	template := newTestTemplate()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	reviewers, err := deps.svc.ListReviewers(t.Context(), &ListReviewersRequest{
		TenantInfo: tenant,
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	require.Len(t, reviewers, 1)
	assert.Equal(t, Reviewer{
		ID:            other,
		Name:          "Priya Raman",
		Username:      "praman",
		ProfilePicURL: "https://example.com/p.png",
	}, reviewers[0])

	encoded, err := sonic.Marshal(reviewers[0])
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "priya@example.com")
}

func scenarioTraceFixture(
	t *testing.T,
	recorded *formulatemplate.ScenarioContent,
	expected int64,
) (*testDeps, *formulatemplate.FormulaTemplate, *formulatemplate.TestCase) {
	t.Helper()
	deps := setupTest(t)

	template := newTestTemplate()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)

	testCase := newTestCase("Hundred miles", float64(expected), 0.01)
	testCase.TemplateID = template.ID
	testCase.Record(recorded, 3, pulid.MustNew("usr_"), 100)
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{testCase}

	return deps, template, testCase
}

func rateContent(expression string, rate float64) *formulatemplate.ScenarioContent {
	return &formulatemplate.ScenarioContent{
		Expression: expression,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "rate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: rate},
		},
	}
}

func rowsByName(
	trace *formulatemplateports.ScenarioTrace,
) map[string]formulatemplateports.ScenarioTraceRow {
	rows := make(map[string]formulatemplateports.ScenarioTraceRow, len(trace.Rows))
	for _, row := range trace.Rows {
		rows[string(row.Kind)+":"+row.Name] = row
	}
	return rows
}

func TestScenarioTrace_ARateChangeLooksIntended(t *testing.T) {
	t.Parallel()

	expression := "let base = totalDistance * rate; base"
	deps, template, testCase := scenarioTraceFixture(t, rateContent(expression, 2.5), 250)

	trace, err := deps.svc.ScenarioTrace(t.Context(), &ScenarioTraceRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		TestCaseID: testCase.ID,
		Candidate:  rateContent(expression, 3),
	})
	require.NoError(t, err)

	assert.Equal(t, "250", trace.Expected)
	assert.Equal(t, decimalPtr("300"), trace.Actual)
	assert.Equal(t, decimalPtr("50"), trace.Delta)
	assert.Equal(t, "v3", trace.BaselineLabel)
	assert.EqualValues(t, 3, *trace.RecordedOnVersion)
	assert.Empty(t, trace.Error)

	require.NotEmpty(t, trace.Rows)
	assert.Equal(t, formulatemplateports.ScenarioRowInput, trace.Rows[0].Kind, "inputs first")
	rows := rowsByName(trace)
	assert.Equal(t, decimalPtr("2.5"), rows["input:rate"].Before)
	assert.Equal(t, decimalPtr("3"), rows["input:rate"].After)
	assert.Equal(t, decimalPtr("250"), rows["term:base"].Before)
	assert.Equal(t, decimalPtr("300"), rows["term:base"].After)
	assert.Contains(t, rows, "formula:formula")
	assert.Contains(t, rows, "billed:billed")
	assert.NotContains(t, rows, "policy:minimum")

	assert.Equal(t, formulatemplateports.ScenarioVerdictIntended, trace.FallbackVerdict)
	assert.Contains(t, trace.FallbackWhy, "`rate`")
	assert.Contains(t, trace.FallbackWhy, "v3")
}

func TestScenarioTrace_AnExpressionEditIsUnclear(t *testing.T) {
	t.Parallel()

	deps, template, testCase := scenarioTraceFixture(
		t,
		rateContent("let base = totalDistance * rate; base", 2.5),
		250,
	)

	trace, err := deps.svc.ScenarioTrace(t.Context(), &ScenarioTraceRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		TestCaseID: testCase.ID,
		Candidate:  rateContent("let base = totalDistance * rate * 2; base", 2.5),
	})
	require.NoError(t, err)

	assert.Equal(t, formulatemplateports.ScenarioVerdictUnclear, trace.FallbackVerdict)
	assert.Contains(t, trace.FallbackWhy, "`base`")
	assert.Contains(t, trace.FallbackWhy, "edit to the expression")
}

func TestScenarioTrace_ARuntimeFailureIsABug(t *testing.T) {
	t.Parallel()

	deps, template, testCase := scenarioTraceFixture(
		t,
		rateContent("totalDistance * rate", 2.5),
		250,
	)

	trace, err := deps.svc.ScenarioTrace(t.Context(), &ScenarioTraceRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		TestCaseID: testCase.ID,
		Candidate:  rateContent("totalDistance > rate", 2.5),
	})
	require.NoError(t, err)

	assert.NotEmpty(t, trace.Error)
	assert.Nil(t, trace.Actual)
	assert.Nil(t, trace.Delta)
	assert.Equal(t, formulatemplateports.ScenarioVerdictBug, trace.FallbackVerdict)
	assert.Contains(t, trace.FallbackWhy, "fails on this load")
}

func TestScenarioTrace_HittingANewMinimumIsUnclear(t *testing.T) {
	t.Parallel()

	deps, template, testCase := scenarioTraceFixture(
		t,
		rateContent("totalDistance * rate", 2.5),
		250,
	)
	candidate := rateContent("totalDistance * rate", 2.5)
	candidate.MinCharge = decimal.NewNullDecimal(decimal.NewFromInt(400))

	trace, err := deps.svc.ScenarioTrace(t.Context(), &ScenarioTraceRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		TestCaseID: testCase.ID,
		Candidate:  candidate,
	})
	require.NoError(t, err)

	rows := rowsByName(trace)
	require.Contains(t, rows, "policy:minimum")
	assert.Nil(t, rows["policy:minimum"].Before)
	assert.Equal(t, decimalPtr("400"), rows["policy:minimum"].After)
	assert.NotContains(t, rows, "formula:formula", "the raw amount did not move")
	assert.Equal(t, formulatemplateports.ScenarioVerdictUnclear, trace.FallbackVerdict)
	assert.Contains(t, trace.FallbackWhy, "$400.00")
	assert.Contains(t, trace.FallbackWhy, "$250.00")
}

func TestScenarioTrace_NothingChangedBlamesTheExpectation(t *testing.T) {
	t.Parallel()

	deps, template, testCase := scenarioTraceFixture(
		t,
		rateContent("totalDistance * rate", 2.5),
		999,
	)

	trace, err := deps.svc.ScenarioTrace(t.Context(), &ScenarioTraceRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		TestCaseID: testCase.ID,
		Candidate:  rateContent("totalDistance * rate", 2.5),
	})
	require.NoError(t, err)

	assert.Empty(t, trace.Rows)
	assert.Equal(t, formulatemplateports.ScenarioVerdictIntended, trace.FallbackVerdict)
	assert.Contains(t, trace.FallbackWhy, "$999.00")
}

func TestScenarioTrace_LegacyScenarioComparesWithTheSavedVersion(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	template := newTestTemplate()
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	testCase := newTestCase("Legacy", 250, 0.01)
	testCase.TemplateID = template.ID
	deps.testCaseRepo.cases = []*formulatemplate.TestCase{testCase}

	trace, err := deps.svc.ScenarioTrace(t.Context(), &ScenarioTraceRequest{
		TenantInfo: newTenantInfo(),
		TemplateID: template.ID,
		TestCaseID: testCase.ID,
		Candidate:  template.ScenarioContent(),
	})
	require.NoError(t, err)

	assert.Equal(t, "the saved version", trace.BaselineLabel)
	assert.Nil(t, trace.RecordedOnVersion)
	assert.Nil(t, trace.RecordedBy)
}

func TestPriceScenario_AppliesTheChargePolicy(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	content := rateContent("totalDistance * rate", 2.5)
	content.MinCharge = decimal.NewNullDecimal(decimal.NewFromInt(400))

	price := deps.svc.PriceScenario(t.Context(), &PriceScenarioRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		SchemaID: "shipment",
		Content:  content,
		Inputs:   map[string]any{"totalDistance": 100.0},
	})

	require.True(t, price.Valid, price.Error)
	assert.Equal(t, "400", price.Amount)
	assert.Equal(t, 2.5, price.Variables["rate"])
}
