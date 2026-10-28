package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rateSpec(t *testing.T, name string) *graphQLListSpec {
	t.Helper()

	for _, spec := range rateReadSpecs() {
		if spec.name == name {
			return &spec
		}
	}
	require.FailNow(t, "no spec "+name)

	return nil
}

func rateTool(
	t *testing.T,
	name string,
	reader graphqlreadports.GraphQLReader,
) agenttoolports.AgentQueryTool {
	t.Helper()

	tool, err := newGraphQLReadTool(
		rateSpec(t, name),
		reader,
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)

	return tool
}

func TestFuelIndexPrices_SendsTheIndexTheWindowAndAPage(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{}
	tool := rateTool(t, toolRateListFuelIndexPrices, reader)
	index := pulid.MustNew("fidx_").String()

	_, err := tool.Query(t.Context(), keyedCall(map[string]any{
		paramFuelIndexID: index, "from": "2026-01-05", "to": "2026-03-30",
	}))
	require.NoError(t, err)

	sent := reader.sent("DeskFuelIndexPrices")
	require.NotNil(t, sent)
	assert.Equal(t, map[string]any{
		paramFuelIndexID: index, "from": "2026-01-05", "to": "2026-03-30",
		paramLimit: graphQLListLimit,
	}, sent.Variables, "a call that names no limit reads a page, not the resolver's two years")
}

func TestFuelIndexPrices_RefusesADayThatIsNotADayAndACallWithoutAnIndex(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{}
	tool := rateTool(t, toolRateListFuelIndexPrices, reader)

	_, err := tool.Query(t.Context(), keyedCall(map[string]any{
		paramFuelIndexID: pulid.MustNew("fidx_").String(), "from": "last spring",
	}))
	require.ErrorContains(t, err, "YYYY-MM-DD")
	_, err = tool.Query(t.Context(), keyedCall(map[string]any{"from": "2026-01-05"}))
	require.ErrorContains(t, err, paramFuelIndexID)
	assert.Empty(t, reader.requests, "nothing is read for a call refused")
}

func fuelRatesAnswer(withheld ...string) map[string]*graphqlreadports.GraphQLReadResult {
	program := func(code, name string) map[string]any {
		return map[string]any{"id": "fsp_" + code, "code": code, "name": name}
	}

	return map[string]*graphqlreadports.GraphQLReadResult{"DeskFuelProgramCurrentRates": {
		Data: map[string]any{"fuelProgramCurrentRates": []any{
			map[string]any{
				"program": program("DOE-STD", "DOE standard"), "usedFallback": false,
				"price": map[string]any{
					"id":        "fip_1",
					"priceDate": "2026-10-19",
					"price":     "3.71",
				},
				"ratePerMile": "0.42",
			},
			map[string]any{"program": program("REEFER", "Reefer table"), "usedFallback": false},
			map[string]any{
				"program": program("FLAT", "Flat table"), "usedFallback": true,
				"price": map[string]any{"id": "fip_2", "priceDate": "2026-10-12", "price": "9.99"},
			},
		}},
		Withheld: withheld,
	}}
}

func TestFuelSurchargePrograms_MatchesTheQueryInTheRowsItRead(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: fuelRatesAnswer()}
	tool := rateTool(t, toolRateListFuelSurcharge, reader)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{"query": "doe"}))
	require.NoError(t, err)

	outcome, ok := data.(map[string]any)
	require.True(t, ok)
	items, ok := outcome[outcomeItems].([]map[string]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	assert.Equal(t, "DOE-STD", items[0]["program"].(map[string]any)["code"])
	assert.Empty(t, reader.sent("DeskFuelProgramCurrentRates").Variables,
		"the root takes no search; the query never reaches it")
}

func TestFuelSurchargePrograms_SaysWhyAProgramHasNoSurcharge(t *testing.T) {
	t.Parallel()

	tool := rateTool(t, toolRateListFuelSurcharge, &keyedReader{answers: fuelRatesAnswer()})

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{}))
	require.NoError(t, err)

	items := data.(map[string]any)[outcomeItems].([]map[string]any)
	require.Len(t, items, 3)
	assert.NotContains(t, items[0], outcomeNote)
	assert.Equal(t, noteNoFuelPrice, items[1][outcomeNote])
	assert.Equal(t, noteOutsideFuelTable, items[2][outcomeNote])
}

// A row whose price or rate access withheld is not reported as having none.
func TestFuelSurchargePrograms_SaysNothingOfWhatAccessWithheld(t *testing.T) {
	t.Parallel()

	tool := rateTool(t, toolRateListFuelSurcharge, &keyedReader{answers: fuelRatesAnswer(
		"fuelProgramCurrentRates.price", "fuelProgramCurrentRates.ratePerMile",
	)})

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{}))
	require.NoError(t, err)

	items := data.(map[string]any)[outcomeItems].([]map[string]any)
	for _, item := range items {
		assert.NotContains(t, item, outcomeNote)
	}
	assert.Equal(t, []string{"price", "ratePerMile"}, data.(map[string]any)[outcomeWithheld])
}

func TestRateAgreementGet_ReadsTheCardWithItsLanes(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("rag_").String()
	reader := &keyedReader{answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskRateAgreementCard": {
			Data: map[string]any{"rateAgreement": map[string]any{
				"id": id, "code": "ACME-2026", "laneCount": 1, "inputs": nil,
				"lanes": []any{map[string]any{
					"id": "ragr_1", "originCity": "Dallas", "inputs": nil, "label": "",
				}},
			}},
			Withheld: []string{"rateAgreement.inputs", "rateAgreement.lanes.inputs"},
		},
	}}
	spec := rateAgreementGetSpec()
	tool, err := newGraphQLGetTool(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{"rateAgreementId": id}))
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"id": id}, reader.sent("DeskRateAgreementCard").Variables)
	record := data.(map[string]any)
	assert.Equal(t, 1, record["laneCount"])
	lanes := record["lanes"].([]any)
	require.Len(t, lanes, 1)
	assert.NotContains(t, lanes[0], "inputs")
	assert.Equal(t, []string{"inputs", "lanes.inputs"}, record[outcomeWithheld])
}

// What the agreement's summary promises, its card selects.
func TestRateAgreementGet_TheCardSelectsWhatTheSummaryPromises(t *testing.T) {
	t.Parallel()

	read := graphQLReads[toolRateGetAgreement]
	selected := make(map[string]bool, len(read.Fields))
	for _, field := range read.Fields {
		selected[field.Name] = field.Selected
	}
	for _, field := range []string{
		"lanes", "laneCount", "accessorials", "fuelBinding", "currentVersion",
	} {
		assert.True(t, selected[field], field)
	}
}
