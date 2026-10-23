package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func reportTool(
	t *testing.T,
	name string,
	reader agentports.GraphQLReader,
) agentports.AgentQueryTool {
	t.Helper()

	for _, spec := range reportReadSpecs() {
		if spec.name == name {
			tool, err := newGraphQLReadTool(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
			require.NoError(t, err)
			return tool
		}
	}
	require.FailNow(t, "no spec "+name)

	return nil
}

func runNode(id string, status report.RunStatus, stored bool) map[string]any {
	return map[string]any{"id": id, fieldRunStatus: string(status), fieldRunRowsStored: stored}
}

// A run can be compared only when it succeeded and its rows were stored, and
// each row says which, and why not. The statuses are the wire values the
// resolver sends (report.RunStatus), so a stored succeeded run is comparable.
func TestReportRuns_SayWhichRunsCanBeCompared(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
		"DeskReportRuns": {Data: connectionPage("reportRuns",
			runNode("rrun_1", report.RunStatusSucceeded, true),
			runNode("rrun_2", report.RunStatusSucceeded, false),
			runNode("rrun_3", report.RunStatusFailed, false),
			runNode("rrun_4", report.RunStatusRunning, false),
		)},
	}}
	tool := reportTool(t, "report_list_runs", reader)
	definition := pulid.MustNew("rdef_").String()

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{
		paramReportDefinition: definition,
	}))
	require.NoError(t, err)

	sent := reader.sent("DeskReportRuns")
	require.NotNil(t, sent)
	input, _ := sent.Variables["input"].(map[string]any)
	assert.Equal(t, []map[string]any{{"field": paramReportDefinition, "operator": "eq",
		"value": definition}}, input["fieldFilters"])
	outcome, ok := data.(graphQLListOutcome)
	require.True(t, ok)
	require.Len(t, outcome.Items, 4)
	assert.Equal(t, true, outcome.Items[0][fieldRunComparable])
	assert.NotContains(t, outcome.Items[0], fieldRunReason)
	assert.Equal(t, false, outcome.Items[1][fieldRunComparable])
	assert.Contains(t, outcome.Items[1][fieldRunReason], "before its rows were stored")
	assert.Equal(t, false, outcome.Items[2][fieldRunComparable])
	assert.Contains(t, outcome.Items[2][fieldRunReason], "ended as failed")
	assert.Equal(t, false, outcome.Items[3][fieldRunComparable])
	assert.Contains(t, outcome.Items[3][fieldRunReason], "still running")
}

// A run whose status or storage the reader may not see is not called
// incomparable: the row says nothing it cannot know.
func TestReportRuns_LeaveAWithheldRunUnjudged(t *testing.T) {
	t.Parallel()

	for _, field := range []string{fieldRunStatus, fieldRunRowsStored} {
		row := runNode("rrun_1", report.RunStatusSucceeded, true)
		delete(row, field)
		runComparable(row, map[string]struct{}{field: {}}, clock{})
		assert.NotContains(t, row, fieldRunComparable, field)
		assert.NotContains(t, row, fieldRunReason, field)
	}
}

// The schedules are read by the arguments the call names, and say which the
// person set up.
func TestReportSchedules_SendTheCallsArguments(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
		"DeskReportSchedules": {Data: map[string]any{"reportSchedules": []any{
			map[string]any{"id": "rsch_1", "mine": true, "enabled": true},
		}}},
	}}
	tool := reportTool(t, "report_list_schedules", reader)
	definition := pulid.MustNew("rdef_").String()

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{
		paramReportDefinition: definition, "enabledOnly": true, paramLimit: 10,
	}))
	require.NoError(t, err)

	sent := reader.sent("DeskReportSchedules")
	require.NotNil(t, sent)
	assert.Equal(t, map[string]any{
		paramReportDefinition: definition, "enabledOnly": true, paramLimit: 10,
	}, sent.Variables)
	outcome, ok := data.(map[string]any)
	require.True(t, ok)
	items, _ := outcome[outcomeItems].([]map[string]any)
	require.Len(t, items, 1)
	assert.Equal(t, true, items[0]["mine"])
}

// An alert says how many conditions it checks rather than carrying each,
// and says nothing about them when they were withheld.
func TestTableChangeAlerts_CountOnlyTheConditionsTheyShow(t *testing.T) {
	t.Parallel()

	shown := map[string]any{"id": "tcas_1", fieldAlertConditions: []any{
		map[string]any{"field": "status"}, map[string]any{"field": "total"},
	}}
	alertConditionCount(shown, nil, clock{})
	assert.Equal(t, 2, shown["conditionCount"])
	assert.NotContains(t, shown, fieldAlertConditions)

	none := map[string]any{"id": "tcas_2"}
	alertConditionCount(none, nil, clock{})
	assert.Equal(t, 0, none["conditionCount"], "a read leaves an empty list out")

	hidden := map[string]any{"id": "tcas_3"}
	alertConditionCount(hidden, map[string]struct{}{fieldAlertConditions: {}}, clock{})
	assert.NotContains(t, hidden, "conditionCount")
}

// The free text matches an alert's name and table, which stay Restricted,
// so a query is refused at Internal before anything is read, and sent as the
// table input's query when the reader's access reaches them.
func TestTableChangeAlerts_RefuseAQueryOverHiddenNames(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{answers: map[string]*agentports.GraphQLReadResult{
		"DeskTableChangeAlerts": {Data: connectionPage("tcaSubscriptions",
			map[string]any{"id": "tcas_1", "status": "Active"},
		)},
	}}
	tool := reportTool(t, "report_list_table_change_alerts", reader)

	internal := keyedCall(map[string]any{paramQuery: "invoice"})
	internal.DataAccessCeiling = permission.SensitivityInternal
	_, err := tool.Query(t.Context(), internal)
	require.ErrorContains(t, err, "cannot search by query")
	assert.Nil(t, reader.sent("DeskTableChangeAlerts"), "nothing is read")

	named := keyedCall(map[string]any{listFieldName: "invoice"})
	named.DataAccessCeiling = permission.SensitivityInternal
	_, err = tool.Query(t.Context(), named)
	require.ErrorContains(t, err, "hidden at your access")

	spec := tableChangeAlertsListSpec()
	reached, err := newGraphQLReadTool(&spec, reader, &restrictedRole{})
	require.NoError(t, err)
	restricted := keyedCall(map[string]any{paramQuery: "invoice", paramStatus: "Paused"})
	restricted.DataAccessCeiling = permission.SensitivityRestricted
	_, err = reached.Query(t.Context(), restricted)
	require.NoError(t, err)
	sent := reader.sent("DeskTableChangeAlerts")
	require.NotNil(t, sent)
	input, _ := sent.Variables["input"].(map[string]any)
	assert.Equal(t, "invoice", input[paramQuery])
	assert.Equal(t, []map[string]any{{"field": paramStatus, "operator": "eq",
		"value": "Paused"}}, input["fieldFilters"], "the status is an enum, matched whole")
}

// A searched field the record lacks would leave a query unguarded, so the
// tool is not built.
func TestGraphQLListTool_RefusesASearchedFieldTheRecordLacks(t *testing.T) {
	t.Parallel()

	spec := tableChangeAlertsListSpec()
	spec.searches = []string{"nickname"}
	_, err := newGraphQLReadTool(&spec, &keyedReader{}, &mocks.DenyResourcesPermissionEngine{})
	require.ErrorContains(t, err, `searches "nickname"`)
}
