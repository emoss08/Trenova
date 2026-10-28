package formulatemplatehandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/formulastudioports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingStudio struct {
	explained []*formulastudioports.ExplainFormulaRequest
	described []*formulastudioports.DescribeFormulaInputsRequest
	suggested []*formulastudioports.SuggestFormulaTestsRequest
	tested    []*formulastudioports.ExplainFormulaTestRequest
	notes     []*formulastudioports.DraftFormulaReviewNoteRequest
	actors    []*permissionports.RequestActor
}

func (s *recordingStudio) ExplainFormula(
	_ context.Context,
	req *formulastudioports.ExplainFormulaRequest,
	actor *permissionports.RequestActor,
) (*formulastudioports.ExplainFormulaResponse, error) {
	s.explained = append(s.explained, req)
	s.actors = append(s.actors, actor)
	return &formulastudioports.ExplainFormulaResponse{
		Items: []formulastudioports.FormulaTraceNote{{Index: 0, Says: "It charges by the mile."}},
	}, nil
}

func (s *recordingStudio) DescribeInputs(
	_ context.Context,
	req *formulastudioports.DescribeFormulaInputsRequest,
	_ *permissionports.RequestActor,
) (*formulastudioports.DescribeFormulaInputsResponse, error) {
	s.described = append(s.described, req)
	return &formulastudioports.DescribeFormulaInputsResponse{
		Descriptions: map[string]string{"rate": "What the contract pays per mile"},
	}, nil
}

func (s *recordingStudio) SuggestTests(
	_ context.Context,
	req *formulastudioports.SuggestFormulaTestsRequest,
	_ *permissionports.RequestActor,
) (*formulastudioports.SuggestFormulaTestsResponse, error) {
	s.suggested = append(s.suggested, req)
	return &formulastudioports.SuggestFormulaTestsResponse{
		Tests: []formulastudioports.SuggestedFormulaTest{{
			Name:   "Minimum reached",
			Block:  "test \"Minimum reached\" {\n  expect total >= $0\n}",
			Status: rules.TariffTestPassed,
		}},
	}, nil
}

func (s *recordingStudio) ExplainTest(
	_ context.Context,
	req *formulastudioports.ExplainFormulaTestRequest,
	_ *permissionports.RequestActor,
) (*formulastudioports.ExplainFormulaTestResponse, error) {
	s.tested = append(s.tested, req)
	return &formulastudioports.ExplainFormulaTestResponse{
		Verdict: formulastudioports.FormulaTestVerdictUnclear,
	}, nil
}

func (s *recordingStudio) DraftReviewNote(
	_ context.Context,
	req *formulastudioports.DraftFormulaReviewNoteRequest,
	_ *permissionports.RequestActor,
) (*formulastudioports.DraftFormulaReviewNoteResponse, error) {
	s.notes = append(s.notes, req)
	return &formulastudioports.DraftFormulaReviewNoteResponse{
		Note:      "Raised the rate.",
		Generated: true,
	}, nil
}

func newStudioFixture(t *testing.T, granted map[string]bool, studio *recordingStudio) *fixture {
	t.Helper()
	opts := fixtureOptions{granted: granted}
	if studio != nil {
		opts.studio = studio
	}
	return newFixture(t, opts)
}

func TestStudioExplain_PassesTheDraftInputsAndShipment(t *testing.T) {
	t.Parallel()

	studio := &recordingStudio{}
	f := newStudioFixture(t, authorGrants(), studio)
	shipmentID := pulid.MustNew("shp_")
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/ai/explain", map[string]any{
		"name":       "Per Mile",
		"source":     perMileSource,
		"inputs":     map[string]any{"rate": map[string]any{"$q": "2.85", "u": "$/mi"}},
		"shipmentId": shipmentID.String(),
	})

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())
	require.Len(t, studio.explained, 1)
	request := studio.explained[0]
	assert.Equal(t, perMileSource, request.Draft.Source)
	assert.Equal(t, "Per Mile", request.Draft.Name)
	assert.Equal(t, shipmentID, request.ShipmentID)
	assert.Contains(t, request.Inputs, "rate")
	require.Len(t, studio.actors, 1)
	assert.Equal(t, testutil.TestUserID, studio.actors[0].UserID)

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	items, ok := resp["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 1)
}

func TestStudioExplain_RefusesAnEmptySourceAndAMissingAssistant(t *testing.T) {
	t.Parallel()

	studio := &recordingStudio{}
	f := newStudioFixture(t, authorGrants(), studio)
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/ai/explain", map[string]any{
		"source": "",
	})
	assert.GreaterOrEqual(t, ginCtx.ResponseCode(), http.StatusBadRequest)
	assert.Empty(t, studio.explained)

	missing := newStudioFixture(t, authorGrants(), nil)
	ginCtx = missing.serve(http.MethodPost, "/api/v1/formula-templates/ai/explain", map[string]any{
		"source": perMileSource,
	})
	assert.GreaterOrEqual(t, ginCtx.ResponseCode(), http.StatusBadRequest)
}

func TestStudioRoutes_NeedTheAssistant(t *testing.T) {
	t.Parallel()

	studio := &recordingStudio{}
	f := newStudioFixture(t, map[string]bool{"formula_template:read": true}, studio)
	for _, path := range []string{
		"/api/v1/formula-templates/ai/describe-inputs",
		"/api/v1/formula-templates/ai/suggest-tests",
		"/api/v1/formula-templates/ai/explain-test",
	} {
		ginCtx := f.serve(http.MethodPost, path, map[string]any{"source": perMileSource})
		assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode(), path)
	}
	assert.Empty(t, studio.described)
	assert.Empty(t, studio.suggested)
}

func TestStudioSuggestTests_AnswersRunnableBlocks(t *testing.T) {
	t.Parallel()

	studio := &recordingStudio{}
	f := newStudioFixture(t, authorGrants(), studio)
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/ai/suggest-tests", map[string]any{
		"source": perMileSource,
	})
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())
	require.Len(t, studio.suggested, 1)
	assert.Equal(t, perMileSource, studio.suggested[0].Draft.Source)

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	tests := resp["tests"].([]any)
	require.Len(t, tests, 1)
	assert.Equal(t, "passed", tests[0].(map[string]any)["status"])
}

func TestStudioExplainTest_NeedsTheTestsName(t *testing.T) {
	t.Parallel()

	studio := &recordingStudio{}
	f := newStudioFixture(t, authorGrants(), studio)
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/ai/explain-test", map[string]any{
		"source": perMileSource,
	})
	assert.GreaterOrEqual(t, ginCtx.ResponseCode(), http.StatusBadRequest)
	assert.Empty(t, studio.tested)

	ginCtx = f.serve(http.MethodPost, "/api/v1/formula-templates/ai/explain-test", map[string]any{
		"source":   perMileSource,
		"testName": "a short haul bills its miles",
	})
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())
	require.Len(t, studio.tested, 1)
	assert.Equal(t, "a short haul bills its miles", studio.tested[0].TestName)
}

func TestStudioReviewNote_IncludesTheBacktestOnlyForShipmentReaders(t *testing.T) {
	t.Parallel()

	grants := authorGrants()
	delete(grants, "shipment:read")
	studio := &recordingStudio{}
	f := newStudioFixture(t, grants, studio)
	path := "/api/v1/formula-templates/" + pulid.MustNew("ft_").String() + "/review-note"
	ginCtx := f.serve(http.MethodPost, path, map[string]any{})
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	require.Len(t, studio.notes, 1)
	assert.False(t, studio.notes[0].IncludeBacktest)

	studio = &recordingStudio{}
	f = newStudioFixture(t, authorGrants(), studio)
	ginCtx = f.serve(http.MethodPost, path, nil)
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	require.Len(t, studio.notes, 1)
	assert.True(t, studio.notes[0].IncludeBacktest)
}

func TestSubmit_RefusesAMalformedReviewer(t *testing.T) {
	t.Parallel()

	f := newStudioFixture(t, authorGrants(), &recordingStudio{})
	templateID := pulid.MustNew("ft_")
	f.repo.On("GetByID", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplate{
			ID:     templateID,
			Name:   "Per Mile",
			Source: perMileSource,
			Status: formulatemplate.StatusDraft,
		}, nil).Maybe()

	ginCtx := f.serve(
		http.MethodPost,
		"/api/v1/formula-templates/"+templateID.String()+"/submit",
		map[string]any{"reviewerId": "not-an-id"},
	)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
	assert.Contains(t, ginCtx.Recorder.Body.String(), "reviewerId")
}
