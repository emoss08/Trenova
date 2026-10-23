package formulatemplatehandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const boundedSource = `tariff "Per Mile"

/// What the contract pays per loaded mile.
input rate: Money/mi where $0.50/mi..=$15/mi
/// The service the contract buys.
input service: Text = "Standard" where ["Standard", "Guaranteed"]

charge "Per Mile" (quantity: shipment.distance, rate: rate)
if service == "Guaranteed" {
  charge "Guarantee" = $50
}

test "a short haul bills its miles" {
  given input.rate = $2/mi
  given shipment.distance = 100 mi
  expect total == $200
}
`

var (
	rateBounds = map[string]any{"range": map[string]any{
		"min":          map[string]any{"$q": "0.5", "u": "$/mi"},
		"max":          map[string]any{"$q": "15", "u": "$/mi"},
		"maxInclusive": true,
	}}
	serviceChoices = map[string]any{"oneOf": []any{"Standard", "Guaranteed"}}
)

func inputsOf(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()
	raw, ok := body["inputs"].([]any)
	require.True(t, ok, "%v", body["inputs"])
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		input, isMap := item.(map[string]any)
		require.True(t, isMap)
		out = append(out, input)
	}
	return out
}

func TestCreate_StoresEachInputsConstraint(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	f.repo.On("Create", mock.Anything, mock.Anything).Return(returnsTemplate, nil)
	f.versions.On("Create", mock.Anything, mock.Anything).Return(returnsVersion, nil)

	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/", map[string]any{
		"name":        "Per Mile",
		"description": "Linehaul per loaded mile",
		"source":      boundedSource,
	})
	require.Equal(t, http.StatusCreated, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())

	var created map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&created))
	inputs := inputsOf(t, created)
	require.Len(t, inputs, 2)
	assert.Equal(t, rateBounds, inputs[0]["constraint"])
	assert.Equal(t, serviceChoices, inputs[1]["constraint"])
}

func TestGetVersion_AnswersTheDeclaredConstraintsTheRuleEditorBuildsFrom(t *testing.T) {
	t.Parallel()

	check, err := rules.CheckTariff(t.Context(), &rules.TariffCheckRequest{
		Source:  boundedSource,
		Kind:    rules.TariffKindTariff,
		Context: &rules.TariffContext{},
	})
	require.NoError(t, err)
	templateID := pulid.MustNew("ft_")
	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	f.versions.On("GetByTemplateAndVersion", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{
			ID:             pulid.MustNew("ftv_"),
			TemplateID:     templateID,
			OrganizationID: testutil.TestOrgID,
			BusinessUnitID: testutil.TestBuID,
			VersionNumber:  2,
			Source:         boundedSource,
			Inputs:         tarifftypes.InputDeclsFrom(check.Inputs),
		}, nil)

	ginCtx := f.serve(
		http.MethodGet,
		"/api/v1/formula-templates/"+templateID.String()+"/versions/2",
		nil,
	)
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())

	var version map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&version))
	inputs := inputsOf(t, version)
	require.Len(t, inputs, 2)
	assert.Equal(t, rateBounds, inputs[0]["constraint"])
	assert.Equal(t, serviceChoices, inputs[1]["constraint"])
}

func TestPreview_AnswersEachInputsConstraintAndTheChoices(t *testing.T) {
	t.Parallel()

	f := newFixture(t, fixtureOptions{granted: authorGrants()})
	f.rater.result = &tariffports.RateResult{
		Bill:    &tarifftypes.Bill{Total: decimal.NewFromInt(200), Lines: []tarifftypes.Line{}},
		Choices: []rules.TariffChoice{},
	}

	ginCtx := f.serve(http.MethodPost, "/api/v1/formula-templates/test", map[string]any{
		"source": boundedSource,
		"inputs": map[string]any{"rate": map[string]any{"$q": "2", "u": "$/mi"}},
	})
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), ginCtx.Recorder.Body.String())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	inputs := inputsOf(t, resp)
	require.Len(t, inputs, 2)
	assert.Equal(t, rateBounds, inputs[0]["constraint"])
	assert.Equal(t, []any{}, resp["choices"])
}
