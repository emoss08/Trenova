package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCarrierIntelEvent_NamesTheListThatHandsOutItsID(t *testing.T) {
	t.Parallel()

	spec := carrierIntelEventGetSpec()
	tool, err := newGraphQLGetTool(&spec, &keyedReader{}, nil)
	require.NoError(t, err)
	assert.Contains(t, tool.Description(), "carrier_list_intelligence_events")
	assert.NotContains(t, tool.Description(), "No tool lists")

	properties, _ := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
	eventID, _ := properties["eventId"].(map[string]any)
	assert.Contains(t, eventID[toolschema.KeyDescription], "carrier_list_intelligence_events")
}

func invoiceMatchesAnswer(withheld ...string) *graphqlreadports.GraphQLReadResult {
	return &graphqlreadports.GraphQLReadResult{
		Data: map[string]any{"carrierInvoiceMatches": map[string]any{"items": []any{
			map[string]any{
				"id": "cim_1", "ediCarrierInvoiceId": "ecinv_1", "invoiceNumber": "INV-1",
				"status": "Variance", "matchedVia": "Pro", "currencyCode": "USD",
				"invoiceTotalMinor": 151500, "expectedTotalMinor": 150000, "varianceMinor": 1500,
			},
			map[string]any{
				"id": "cim_2", "documentAiExtractionId": "dax_1", "invoiceNumber": "INV-2",
				"status": "Matched", "matchedVia": "Bol", "currencyCode": "USD",
			},
			map[string]any{
				"id": "cim_3", "invoiceNumber": "INV-3", "status": "Matched",
				"matchedVia": "Manual", "currencyCode": "USD",
			},
		}}},
		Withheld: withheld,
	}
}

func queryInvoiceMatches(
	t *testing.T,
	answer *graphqlreadports.GraphQLReadResult,
) (any, []map[string]any) {
	t.Helper()

	reader := &keyedReader{answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskCarrierInvoiceMatches": answer,
	}}
	tool, err := newGraphQLUnpagedListTool(keyedSpec(t, "carrier_list_invoice_matches"), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{}))
	require.NoError(t, err)
	marked, ok := data.(*offsetPage)
	require.True(t, ok, "a page with an EDI invoice carries its mark")
	items, _ := marked.record[outcomeItems].([]map[string]any)
	require.Len(t, items, 3)

	return data, items
}

func TestListCarrierInvoiceMatches_MarksOnlyInvoicesACarrierSentOverEDI(t *testing.T) {
	t.Parallel()

	data, items := queryInvoiceMatches(t, invoiceMatchesAnswer(
		"carrierInvoiceMatches.items.invoiceTotalMinor",
		"carrierInvoiceMatches.items.expectedTotalMinor",
		"carrierInvoiceMatches.items.varianceMinor",
	))

	assert.Equal(t, matchSourceEDI, items[0][fieldMatchSource])
	assert.Equal(t, matchSourceDocument, items[1][fieldMatchSource])
	assert.Equal(t, matchSourceManual, items[2][fieldMatchSource])
	assert.Equal(t, []agentrecord.RecordRef{{
		EntityType: carrierInvoiceEntity,
		ID:         "ecinv_1",
	}}, data.(agent.TaintCarrier).TaintedRecords())

	assert.Equal(
		t,
		[]string{"invoiceTotal", "expectedTotal", "variance"},
		data.(*offsetPage).record[outcomeWithheld],
		"withheld amounts are named as a person reads them",
	)
}

func TestListCarrierInvoiceMatches_ShowsAmountsAsDecimals(t *testing.T) {
	t.Parallel()

	_, items := queryInvoiceMatches(t, invoiceMatchesAnswer())

	assert.Equal(t, "1515.00", items[0]["invoiceTotal"])
	assert.Equal(t, "1500.00", items[0]["expectedTotal"])
	assert.Equal(t, "15.00", items[0]["variance"])
	assert.NotContains(t, items[0], "invoiceTotalMinor")
	assert.NotContains(t, items[0], "varianceMinor")
}
