package factoringquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/gettool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqllisttool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func factoringTool(
	t *testing.T,
	name string,
	reader graphqlreadports.GraphQLReader,
) agenttoolports.AgentQueryTool {
	t.Helper()

	engine := &mocks.DenyResourcesPermissionEngine{}
	for _, spec := range ReadSpecs() {
		if spec.Name == name {
			tool, err := graphqllisttool.NewRead(&spec, reader, engine)
			require.NoError(t, err)
			return tool
		}
	}
	for _, spec := range GetSpecs() {
		if spec.Name == name {
			tool, err := gettool.NewGraphQL(&spec, reader, engine)
			require.NoError(t, err)
			return tool
		}
	}
	require.FailNow(t, "no spec "+name)

	return nil
}

// The queue reads Ready invoices unless the call names a verdict, carries a
// NotReady invoice's failed checks and no other's, and reads the
// organization's totals beside every page.
func TestFactoringReadiness_ReadsReadyByDefaultWithTheQueueTotals(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskFactoringInvoiceStates": {Data: querytooltest.ConnectionPage("factoringInvoiceStates",
			map[string]any{"invoiceId": "inv_1", "verdict": "Ready", "checks": []any{
				map[string]any{"code": "Noa", "passed": true},
			}},
			map[string]any{"invoiceId": "inv_2", "verdict": "NotReady", "checks": []any{
				map[string]any{"code": "Noa", "passed": true},
				map[string]any{"code": "Agreement", "passed": false, "fix": "SetUpAgreement"},
			}},
		)},
		"FactoringSummary": {Data: map[string]any{"factoringSummary": map[string]any{
			"ready": map[string]any{"count": 1, "amountMinor": 120_000},
		}}},
	}}
	tool := factoringTool(t, toolListFactoringReadiness, reader)

	data, err := tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{}))
	require.NoError(t, err)

	sent := reader.Sent("DeskFactoringInvoiceStates")
	require.NotNil(t, sent)
	input, _ := sent.Variables["input"].(map[string]any)
	assert.Equal(t, []map[string]any{{"field": "verdict", "operator": "in",
		"value": []string{"Ready"}}}, input["fieldFilters"])
	summary := reader.Sent("FactoringSummary")
	require.NotNil(t, summary, "the totals are the organization's, read on every call")
	assert.Empty(t, summary.Variables)

	outcome := querytooltest.EncodedDocument(t, data)
	assert.Contains(t, outcome, keyFactoringSummary)
	items, _ := outcome["items"].([]any)
	require.Len(t, items, 2)
	assert.NotContains(t, items[0].(map[string]any), fieldFactoringChecks)
	checks, _ := items[1].(map[string]any)[fieldFactoringChecks].([]any)
	require.Len(t, checks, 1, "only the check that failed says what holds it back")
	assert.Equal(t, "Agreement", checks[0].(map[string]any)["code"])

	_, err = tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		paramFactoringVerdict: []any{"NotReady", "Submitted"},
	}))
	require.NoError(t, err)
	last := reader.Requests[len(reader.Requests)-2]
	input, _ = last.Variables["input"].(map[string]any)
	assert.Equal(t, []map[string]any{{"field": "verdict", "operator": "in",
		"value": []string{"NotReady", "Submitted"}}}, input["fieldFilters"])
}

// A notice binds the customer only while it is acknowledged and unexpired.
func TestFactoringNoas_SayWhetherEachNoticeIsEffective(t *testing.T) {
	t.Parallel()

	now := timeutils.NowUnix()
	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskFactoringNoas": {Data: querytooltest.ConnectionPage("factoringNoas",
			map[string]any{"id": "fnoa_1", "status": "Acknowledged"},
			map[string]any{"id": "fnoa_2", "status": "Acknowledged", fieldExpiresAt: now - 60},
			map[string]any{"id": "fnoa_3", "status": "Acknowledged", fieldExpiresAt: now + 3600},
			map[string]any{"id": "fnoa_4", "status": "Sent"},
		)},
	}}
	tool := factoringTool(t, toolListFactoringNoas, reader)
	customer := pulid.MustNew("cus_").String()

	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{querytool.ParamCustomerID: customer}),
	)
	require.NoError(t, err)

	sent := reader.Sent("DeskFactoringNoas")
	require.NotNil(t, sent)
	input, _ := sent.Variables["input"].(map[string]any)
	assert.Equal(t, []map[string]any{{"field": querytool.ParamCustomerID, "operator": "eq",
		"value": customer}}, input["fieldFilters"])
	outcome, ok := data.(graphqllisttool.Outcome)
	require.True(t, ok)
	require.Len(t, outcome.Items, 4)
	effective := func(idx int) any { return outcome.Items[idx]["effective"] }
	assert.Equal(t, true, effective(0))
	assert.Equal(t, false, effective(1), "an expired notice binds no one")
	assert.Equal(t, true, effective(2))
	assert.Equal(t, false, effective(3), "a notice not yet acknowledged binds no one")
}

// A factor's agreements are read whole by the factor, marking the one in
// force and writing the submission time as a clock.
func TestFactoringAgreements_MarkTheOneInForce(t *testing.T) {
	t.Parallel()

	now := timeutils.NowUnix()
	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskFactoringAgreements": {Data: map[string]any{"factoringAgreements": []any{
			map[string]any{
				"id":                         "fagr_1",
				"status":                     "Active",
				querytool.FieldEffectiveFrom: now - 60,
				fieldSubmissionTime:          930,
			},
			map[string]any{
				"id":                         "fagr_2",
				"status":                     "Ended",
				querytool.FieldEffectiveFrom: now - 600,
				querytool.FieldEffectiveTo:   now - 300,
			},
		}}},
	}}
	tool := factoringTool(t, toolListFactoringAgreement, reader)

	_, err := tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{}))
	require.ErrorContains(t, err, paramFactoringCompany)
	assert.Empty(t, reader.Requests)

	factor := pulid.MustNew("fco_").String()
	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{paramFactoringCompany: factor}),
	)
	require.NoError(t, err)
	sent := reader.Sent("DeskFactoringAgreements")
	require.NotNil(t, sent)
	assert.Equal(t, factor, sent.Variables[paramFactoringCompany])

	outcome, ok := data.(map[string]any)
	require.True(t, ok)
	items, _ := outcome["items"].([]map[string]any)
	require.Len(t, items, 2)
	assert.Equal(t, true, items[0]["inForce"])
	assert.Equal(t, "15:30", items[0][fieldSubmissionTime])
	assert.Equal(t, false, items[1]["inForce"])
}

// A remittance's lines are narrowed by the match statuses the call names,
// and only an unreadable line keeps the cells the factor printed.
func TestFactoringRemittance_ReadsItsLinesByMatchStatus(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskFactoringRemittance": {Data: map[string]any{"factoringRemittance": map[string]any{
			"id": "frem_1", "status": "Parsed", "lines": []any{
				map[string]any{"id": "freml_1", fieldMatchStatus: "Error",
					fieldRemittanceCells: []any{"A-1", "oops"}},
				map[string]any{"id": "freml_2", fieldMatchStatus: "Unmatched",
					fieldRemittanceCells: []any{"A-2", "100.00"}},
			},
		}}},
	}}
	tool := factoringTool(t, "factoring_get_remittance", reader)
	id := pulid.MustNew("frem_").String()

	data, err := tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		"remittanceId": id, "matchStatuses": []any{"Error", "Unmatched"},
	}))
	require.NoError(t, err)

	sent := reader.Sent("DeskFactoringRemittance")
	require.NotNil(t, sent)
	assert.Equal(t, id, sent.Variables["id"])
	assert.Equal(t, []any{"Error", "Unmatched"}, sent.Variables["matchStatuses"])

	record, ok := data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, factoringRemitsPath+"frem_1", record[keyFactoringPath])
	lines, _ := record["lines"].([]any)
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0].(map[string]any), fieldRemittanceCells)
	assert.NotContains(t, lines[1].(map[string]any), fieldRemittanceCells)
}

// A factor's balance is keyed by the factor, not by a record's id.
func TestFactoringReserveBalance_IsReadByTheFactor(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"FactoringReserveBalance": {Data: map[string]any{"factoringReserveBalance": map[string]any{
			"factoringCompanyId": "fco_1", "entryCount": 3,
		}}},
	}}
	tool := factoringTool(t, "factoring_get_reserve_balance", reader)

	_, err := tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{}))
	require.ErrorContains(t, err, paramFactoringCompany)

	factor := pulid.MustNew("fco_").String()
	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{paramFactoringCompany: factor}),
	)
	require.NoError(t, err)
	sent := reader.Sent("FactoringReserveBalance")
	require.NotNil(t, sent)
	assert.Equal(t, map[string]any{paramFactoringCompany: factor}, sent.Variables)
	record, ok := data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 3, record["entryCount"])
}

// Checking an invoice's readiness evaluates it now through the invoice it
// names.
func TestFactoringGetReadiness_EvaluatesTheInvoiceItNames(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskFactoringReadiness": {Data: map[string]any{"factoringReadiness": map[string]any{
			"invoiceId": "inv_1", "verdict": "NotReady",
		}}},
	}}
	tool := factoringTool(t, "factoring_get_readiness", reader)
	invoice := pulid.MustNew("inv_").String()

	data, err := tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		paramFactoringInvoice: invoice,
	}))
	require.NoError(t, err)
	sent := reader.Sent("DeskFactoringReadiness")
	require.NotNil(t, sent)
	assert.Equal(t, map[string]any{paramFactoringInvoice: invoice}, sent.Variables)
	record, ok := data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, factoringQueuePath, record[keyFactoringPath])
}

func TestFactoringGets_TypeTheirKeyByTheRecordItBelongsTo(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		param    string
		recordOf string
	}{
		"factoring_get_readiness": {param: paramFactoringInvoice, recordOf: "invoice"},
		"factoring_get_reserve_balance": {
			param:    paramFactoringCompany,
			recordOf: "factoring_company",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			schema := factoringTool(t, name, &querytooltest.KeyedReader{}).ParamSchema()
			properties, _ := schema[toolschema.KeyProperties].(map[string]any)
			property, _ := properties[tc.param].(map[string]any)
			require.NotNil(t, property, tc.param)
			assert.Equal(t, tc.recordOf, property[toolschema.KeyRecordOf])
			assert.Equal(t, []string{tc.param}, schema[toolschema.KeyRequired])
		})
	}
}
