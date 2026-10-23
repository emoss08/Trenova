package formulatemplatehandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type problem struct {
	Status int `json:"status"`
	Errors []struct {
		Field string `json:"field"`
		Range *struct {
			From int `json:"from"`
			To   int `json:"to"`
		} `json:"range"`
	} `json:"errors"`
}

func returnsTemplate(_ context.Context, entity *formulatemplate.FormulaTemplate) *formulatemplate.FormulaTemplate {
	return entity
}

func returnsVersion(
	_ context.Context,
	version *formulatemplate.FormulaTemplateVersion,
) *formulatemplate.FormulaTemplateVersion {
	return version
}

func TestList_ReturnsTheTemplatesWithTheirSource(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	f.repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*formulatemplate.FormulaTemplate]{
			Items: []*formulatemplate.FormulaTemplate{{
				ID:             pulid.MustNew("ft_"),
				OrganizationID: testutil.TestOrgID,
				BusinessUnitID: testutil.TestBuID,
				Name:           "Per Mile",
				Source:         perMileSource,
				Status:         formulatemplate.StatusActive,
			}},
			Total: 1,
		}, nil)

	ginCtx := f.serve(http.MethodGet, "/api/v1/formula-templates/", nil)
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp.Results, 1)
	assert.Equal(t, perMileSource, resp.Results[0]["source"])
	assert.NotContains(t, resp.Results[0], "expression")
}

func TestCreate_StoresTheCheckedSource(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	f.repo.On("Create", mock.Anything, mock.Anything).Return(returnsTemplate, nil)
	f.versions.On("Create", mock.Anything, mock.Anything).Return(returnsVersion, nil)

	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/", map[string]any{
		"name":        "Per Mile",
		"description": "Linehaul per loaded mile",
		"source":      perMileSource,
	})
	require.Equal(t, http.StatusCreated, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())

	var created map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&created))
	assert.Equal(t, "Draft", created["status"])
	assert.EqualValues(t, 1, created["testCount"])
	inputs, ok := created["inputs"].([]any)
	require.True(t, ok)
	require.Len(t, inputs, 2)
	first := inputs[0].(map[string]any)
	assert.Equal(t, "rate", first["name"])
	assert.Equal(t, "Money/mi", first["type"])
	assert.Equal(t, true, first["required"])
	assert.NotNil(t, first["descriptor"])
}

func TestCreate_ASourceThatDoesNotCheckIsRefusedAtItsRange(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/", map[string]any{
		"name":   "Per Mile",
		"source": brokenSource,
	})
	require.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())

	var resp problem
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.NotEmpty(t, resp.Errors)
	assert.Equal(t, "source", resp.Errors[0].Field)
	require.NotNil(t, resp.Errors[0].Range, "the editor underlines the finding")
	assert.Greater(t, resp.Errors[0].Range.To, resp.Errors[0].Range.From)
	f.repo.AssertNotCalled(t, "Create")
}

func TestCreate_NeedsPermission(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: map[string]bool{"formula_template:read": true}})
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/", map[string]any{
		"name":   "Per Mile",
		"source": perMileSource,
	})
	assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode())
}

func TestPreview_AnswersDiagnosticsBillAndTests(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	f.rater.result = &tariffports.RateResult{
		Bill: &tarifftypes.Bill{Total: decimal.NewFromInt(285), Lines: []tarifftypes.Line{}},
		WireBill: &rules.TariffBill{
			Lines:            []rules.TariffLine{},
			FreightTotal:     rules.TariffQuantity{Value: decimal.NewFromInt(285), Unit: "$"},
			AccessorialTotal: rules.TariffQuantity{Value: decimal.Zero, Unit: "$"},
			Total:            rules.TariffQuantity{Value: decimal.NewFromInt(285), Unit: "$"},
		},
	}

	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/test", map[string]any{
		"source": perMileSource,
		"inputs": map[string]any{"rate": map[string]any{"$q": "2.85", "u": "$/mi"}},
	})
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, true, resp["valid"])
	assert.Empty(t, resp["diagnostics"])
	require.NotNil(t, resp["bill"])
	bill, ok := resp["bill"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, map[string]any{"$q": "285", "u": "$"}, bill["total"],
		"the client reads the engine's bill: totals are quantities with their unit")
	tests, ok := resp["tests"].([]any)
	require.True(t, ok)
	require.Len(t, tests, 1)
	assert.Equal(t, "passed", tests[0].(map[string]any)["status"])
	require.Len(t, f.rater.requests, 1)
	assert.Contains(t, f.rater.requests[0].Inputs, "rate")
	assert.Empty(t, f.records.requests)
}

func TestPreview_ASourceThatDoesNotCheckAnswersItsDiagnostics(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/test", map[string]any{
		"source": brokenSource,
	})
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, false, resp["valid"])
	assert.NotEmpty(t, resp["diagnostics"])
	assert.Nil(t, resp["bill"])
	assert.Empty(t, f.rater.requests)
}

func TestPreview_ARealShipmentNeedsShipmentRead(t *testing.T) {
	t.Parallel()

	grants := authorGrants()
	delete(grants, "shipment:read")
	f := newFixture(t, fixtureOptions{granted: grants})
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/test", map[string]any{
		"source":     perMileSource,
		"shipmentId": pulid.MustNew("shp_").String(),
	})
	assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode())
	assert.Empty(t, f.records.requests)

	allowed := newFixture(t, fixtureOptions{granted: authorGrants()})
	allowed.rater.result = &tariffports.RateResult{Bill: &tarifftypes.Bill{Lines: []tarifftypes.Line{}}}
	shipmentID := pulid.MustNew("shp_")
	ginCtx = allowed.serve(http.MethodPost, "/api/v1/formula-templates/test", map[string]any{
		"source":     perMileSource,
		"shipmentId": shipmentID.String(),
	})
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())
	require.Len(t, allowed.records.requests, 1)
	assert.Equal(t, shipmentID, allowed.records.requests[0].ShipmentID)
}

func TestScaffoldTest_RefusesARequestWithoutAName(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/scaffold-test", map[string]any{
		"source":     perMileSource,
		"shipmentId": pulid.MustNew("shp_").String(),
	})
	require.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())

	var resp problem
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.NotEmpty(t, resp.Errors)
	assert.Equal(t, "name", resp.Errors[0].Field)
	assert.Empty(t, f.records.requests)
}

func TestContext_ServesTheOrganisationsTariffContext(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	ginCtx := f.serve(http.MethodGet, "/api/v1/formula-templates/context", nil)
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	tariffCtx, ok := resp["context"].(map[string]any)
	require.True(t, ok)
	settings := tariffCtx["settings"].(map[string]any)
	assert.Equal(t, "mi", settings["distanceUnit"])
	assert.NotNil(t, resp["libraries"])
}

func TestContextCells_AnswersTheStampAndTheNeededTables(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: map[string]bool{"formula_template:read": true}})
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/context/cells", map[string]any{
		"needs": []map[string]any{{"table": "Missing", "keys": []string{"A"}}},
	})
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Contains(t, resp, "stamp")
	assert.Equal(t, []any{}, resp["tables"])
}

func TestContextCells_RefusesARequestWithoutNeeds(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: map[string]bool{"formula_template:read": true}})
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/context/cells", map[string]any{
		"needs": []any{},
	})
	require.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())

	var resp problem
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.NotEmpty(t, resp.Errors)
	assert.Equal(t, "needs", resp.Errors[0].Field)
}

func TestContextCells_NeedsFormulaTemplateRead(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: map[string]bool{}})
	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/context/cells", map[string]any{
		"needs": []map[string]any{{"table": "Grid", "keys": []string{"A"}}},
	})
	assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode())
}

func TestStandards_ListsTariffSources(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	ginCtx := f.serve(http.MethodGet, "/api/v1/formula-templates/standards", nil)
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.NotEmpty(t, resp)
	for _, standard := range resp {
		assert.NotEmpty(t, standard["source"], standard["name"])
	}
}

func TestGetUsage_PassesTheTenant(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	templateID := pulid.MustNew("ft_")
	f.repo.On("CountUsages", mock.Anything, mock.MatchedBy(
		func(req *formulatemplateports.GetTemplateUsageRequest) bool {
			return req.TemplateID == templateID && req.TenantInfo.OrgID == testutil.TestOrgID
		},
	)).Return(&formulatemplateports.GetTemplateUsageResponse{}, nil)

	ginCtx := f.serve(http.MethodGet, "/api/v1/formula-templates/"+templateID.String()+"/usage", nil)
	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())
}
