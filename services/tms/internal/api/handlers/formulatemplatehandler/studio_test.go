package formulatemplatehandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/formulatemplatehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingStudio struct {
	explained []*agentports.ExplainFormulaRequest
	scenarios []*agentports.ExplainFormulaScenarioRequest
	notes     []*agentports.DraftFormulaReviewNoteRequest
	actors    []*permissionports.RequestActor
}

func (s *recordingStudio) ExplainFormula(
	_ context.Context,
	req *agentports.ExplainFormulaRequest,
	actor *permissionports.RequestActor,
) (*agentports.ExplainFormulaResponse, error) {
	s.explained = append(s.explained, req)
	s.actors = append(s.actors, actor)
	return &agentports.ExplainFormulaResponse{
		Items: []agentports.FormulaStatementNote{
			{StatementIndex: 0, Says: "It charges by the mile."},
		},
	}, nil
}

func (s *recordingStudio) DescribeInputs(
	context.Context,
	*agentports.DescribeFormulaInputsRequest,
	*permissionports.RequestActor,
) (*agentports.DescribeFormulaInputsResponse, error) {
	return &agentports.DescribeFormulaInputsResponse{Descriptions: map[string]string{}}, nil
}

func (s *recordingStudio) SuggestScenarios(
	context.Context,
	*agentports.SuggestFormulaScenariosRequest,
	*permissionports.RequestActor,
) (*agentports.SuggestFormulaScenariosResponse, error) {
	return &agentports.SuggestFormulaScenariosResponse{}, nil
}

func (s *recordingStudio) ExplainScenario(
	_ context.Context,
	req *agentports.ExplainFormulaScenarioRequest,
	_ *permissionports.RequestActor,
) (*agentports.ExplainFormulaScenarioResponse, error) {
	s.scenarios = append(s.scenarios, req)
	return &agentports.ExplainFormulaScenarioResponse{Verdict: "unclear"}, nil
}

func (s *recordingStudio) DraftReviewNote(
	_ context.Context,
	req *agentports.DraftFormulaReviewNoteRequest,
	_ *permissionports.RequestActor,
) (*agentports.DraftFormulaReviewNoteResponse, error) {
	s.notes = append(s.notes, req)
	return &agentports.DraftFormulaReviewNoteResponse{
		Note:      "Raised the rate.",
		Generated: true,
	}, nil
}

type studioFixture struct {
	handler   *formulatemplatehandler.Handler
	studio    *recordingStudio
	testCases *stubTestCaseRepo
}

func newStudioFixture(
	t *testing.T,
	granted map[string]bool,
	repo *mockFormulaTemplateRepo,
	withStudio bool,
) *studioFixture {
	t.Helper()

	logger := zap.NewNop()
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: &config.Config{App: config.AppConfig{Debug: true}},
	})
	engine := &grantedEngine{granted: granted}
	studio := &recordingStudio{}
	testCases := &stubTestCaseRepo{}
	params := formulatemplatehandler.Params{
		Service: formulatemplateservice.New(formulatemplateservice.Params{
			Logger:         logger,
			DB:             testDBConnection{},
			Repo:           repo,
			VersionRepo:    &mockVersionRepo{},
			TestCaseRepo:   testCases,
			ReviewRepo:     &stubReviewRepo{},
			ShipmentRepo:   &mockShipmentRepo{},
			FormulaService: newTestFormulaService(t),
			AuditService:   &mocks.NoopAuditService{},
		}),
		ErrorHandler: errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: engine,
				ErrorHandler:     errorHandler,
			},
		),
		PermissionEngine: engine,
	}
	if withStudio {
		params.StudioAssist = studio
	}

	return &studioFixture{
		handler:   formulatemplatehandler.New(params),
		studio:    studio,
		testCases: testCases,
	}
}

func (f *studioFixture) serve(method, path string, body any) *testutil.GinTestContext {
	ginCtx := testutil.NewGinTestContext().
		WithMethod(method).
		WithPath(path).
		WithDefaultAuthContext()
	if body != nil {
		ginCtx = ginCtx.WithJSONBody(body)
	}
	f.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	return ginCtx
}

func studioGrants() map[string]bool {
	return map[string]bool{
		"formula_template:read":   true,
		"formula_template:create": true,
		"formula_template:submit": true,
		"assistant:create":        true,
	}
}

func TestStudioExplain_ParsesTheDraftAndAnswers(t *testing.T) {
	t.Parallel()

	f := newStudioFixture(t, studioGrants(), &mockFormulaTemplateRepo{}, true)
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/ai/explain", map[string]any{
		"schemaId":          "shipment",
		"expression":        "totalDistance * rate",
		"minCharge":         "250",
		"maxCharge":         nil,
		"roundingMode":      "Up",
		"roundingPrecision": 0,
		"variableDefinitions": []map[string]any{
			{"name": "rate", "type": "Number", "defaultValue": 2.5},
		},
		"sampleValues": map[string]any{"totalDistance": 100},
	})

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	require.Len(t, f.studio.explained, 1)
	draft := f.studio.explained[0].Draft
	assert.Equal(t, "totalDistance * rate", draft.Content.Expression)
	assert.True(t, draft.Content.MinCharge.Valid)
	assert.Equal(t, "250", draft.Content.MinCharge.Decimal.String())
	assert.False(t, draft.Content.MaxCharge.Valid)
	assert.EqualValues(t, "Up", draft.Content.RoundingMode)
	assert.EqualValues(t, 0, draft.Content.RoundingPrecision)
	require.Len(t, f.studio.actors, 1)
	assert.Equal(t, testutil.TestUserID, f.studio.actors[0].UserID)

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	items, ok := resp["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 1)
}

func TestStudioExplain_RefusesABadPolicyAndAMissingAssistant(t *testing.T) {
	t.Parallel()

	f := newStudioFixture(t, studioGrants(), &mockFormulaTemplateRepo{}, true)
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/ai/explain", map[string]any{
		"expression": "1",
		"minCharge":  "10",
		"maxCharge":  "5",
	})
	assert.GreaterOrEqual(t, ginCtx.ResponseCode(), 400)
	assert.Empty(t, f.studio.explained)

	missing := newStudioFixture(t, studioGrants(), &mockFormulaTemplateRepo{}, false)
	ginCtx = missing.serve(http.MethodPost, "/api/v1/formula-templates/ai/explain", map[string]any{
		"expression": "1",
	})
	assert.GreaterOrEqual(t, ginCtx.ResponseCode(), 400)
}

func TestStudioRoutes_NeedTheAssistant(t *testing.T) {
	t.Parallel()

	f := newStudioFixture(
		t,
		map[string]bool{"formula_template:read": true},
		&mockFormulaTemplateRepo{},
		true,
	)
	ginCtx := f.serve(
		http.MethodPost,
		"/api/v1/formula-templates/ai/describe-inputs",
		map[string]any{
			"expression": "1",
		},
	)

	assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode())
}

func TestStudioExplainScenario_PassesTheCandidate(t *testing.T) {
	t.Parallel()

	f := newStudioFixture(t, studioGrants(), &mockFormulaTemplateRepo{}, true)
	templateID := pulid.MustNew("ft_")
	testCaseID := pulid.MustNew("ftc_")
	path := "/api/v1/formula-templates/" + templateID.String() + "/test-cases/" +
		testCaseID.String() + "/explain"

	ginCtx := f.serve(http.MethodPost, path, map[string]any{})
	assert.GreaterOrEqual(t, ginCtx.ResponseCode(), 400, "the candidate is required")

	ginCtx = f.serve(http.MethodPost, path, map[string]any{
		"candidate": map[string]any{"expression": "totalDistance * 3", "minCharge": "100"},
	})
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	require.Len(t, f.studio.scenarios, 1)
	assert.Equal(t, templateID, f.studio.scenarios[0].TemplateID)
	assert.Equal(t, testCaseID, f.studio.scenarios[0].TestCaseID)
	assert.Equal(t, "totalDistance * 3", f.studio.scenarios[0].Candidate.Expression)
}

func TestStudioReviewNote_IncludesTheBacktestOnlyForShipmentReaders(t *testing.T) {
	t.Parallel()

	f := newStudioFixture(t, studioGrants(), &mockFormulaTemplateRepo{}, true)
	path := "/api/v1/formula-templates/" + pulid.MustNew("ft_").String() + "/review-note"
	ginCtx := f.serve(http.MethodPost, path, map[string]any{})
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	require.Len(t, f.studio.notes, 1)
	assert.False(t, f.studio.notes[0].IncludeBacktest)

	grants := studioGrants()
	grants["shipment:read"] = true
	f = newStudioFixture(t, grants, &mockFormulaTemplateRepo{}, true)
	ginCtx = f.serve(http.MethodPost, path, nil)
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	assert.True(t, f.studio.notes[0].IncludeBacktest)
}

func TestCreate_SavesHeldScenariosWithTheTemplate(t *testing.T) {
	t.Parallel()

	repo := &mockFormulaTemplateRepo{
		createFunc: func(_ context.Context, entity *formulatemplate.FormulaTemplate) (*formulatemplate.FormulaTemplate, error) {
			entity.ID = pulid.MustNew("ft_")
			return entity, nil
		},
	}
	f := newStudioFixture(t, studioGrants(), repo, true)

	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/", map[string]any{
		"name":        "Per mile",
		"description": "Prices a load by the mile",
		"type":        "FreightCharge",
		"schemaId":    "shipment",
		"expression":  "totalDistance * 2.5",
		"testCases": []map[string]any{
			{
				"name":           "Hundred miles",
				"variables":      map[string]any{"totalDistance": 100},
				"expectedAmount": "250",
				"candidate":      map[string]any{"expression": "totalDistance * 2.5"},
			},
		},
	})

	require.Equal(t, http.StatusCreated, ginCtx.ResponseCode())
	require.Len(t, f.testCases.created, 1)
	created := f.testCases.created[0]
	assert.Equal(t, "Hundred miles", created.Name)
	assert.Equal(t, testutil.TestOrgID, created.OrganizationID)
	require.NotNil(t, created.RecordedOnVersion)
	assert.EqualValues(t, 0, *created.RecordedOnVersion)
	require.NotNil(t, created.RecordedByID)
	assert.Equal(t, testutil.TestUserID, *created.RecordedByID)
}

func TestSubmit_RefusesAMalformedReviewer(t *testing.T) {
	t.Parallel()

	repo, ftID := newApprovalTemplateRepo(formulatemplate.StatusDraft)
	f := newStudioFixture(t, studioGrants(), repo, true)

	ginCtx := f.serve(
		http.MethodPost,
		"/api/v1/formula-templates/"+ftID.String()+"/submit",
		map[string]any{"reviewerId": "not-an-id"},
	)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
	assert.Contains(t, ginCtx.Recorder.Body.String(), "reviewerId")
}

var _ formulatemplateports.FormulaTemplateTestCaseRepository = (*stubTestCaseRepo)(nil)

func TestTestExpression_TracesStatementsWhenAsked(t *testing.T) {
	t.Parallel()

	f := newStudioFixture(t, studioGrants(), &mockFormulaTemplateRepo{}, false)
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/test", map[string]any{
		"expression":      "let base = totalDistance * 2; base + 10",
		"schemaId":        "shipment",
		"variables":       map[string]any{"totalDistance": 100},
		"traceStatements": true,
	})
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	statements, ok := resp["statements"].([]any)
	require.True(t, ok, "statements are returned when traced")
	require.Len(t, statements, 2)

	first := statements[0].(map[string]any)
	assert.Equal(t, "let", first["kind"])
	assert.Equal(t, "base", first["name"])
	assert.Equal(t, "let base = totalDistance * 2", first["code"])
	assert.EqualValues(t, 0, first["start"])
	assert.EqualValues(t, 28, first["end"])
	assert.Equal(t, "200", first["value"])

	result := statements[1].(map[string]any)
	assert.Equal(t, "result", result["kind"])
	assert.NotContains(t, result, "name")
	assert.Equal(t, "210", result["value"])

	ginCtx = f.serve(http.MethodPost, "/api/v1/formula-templates/test", map[string]any{
		"expression": "let base = totalDistance * 2; base + 10",
		"schemaId":   "shipment",
		"variables":  map[string]any{"totalDistance": 100},
	})
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var untraced map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&untraced))
	assert.NotContains(t, untraced, "statements")
}
