package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/agentrunquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqllisttool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/masterdataquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/shipmentquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/workerquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func workersAnswer() *graphqlreadports.GraphQLReadResult {
	edge := func(id, name, city, cursor string) map[string]any {
		return map[string]any{"cursor": cursor, "node": map[string]any{
			"id": id, "wholeName": name, "firstName": name, "city": city, "status": "Active",
			"canBeAssigned": true, "availableForDispatch": true,
			"profile": map[string]any{"complianceStatus": "Compliant", "cdlClass": "A"},
		}}
	}

	return &graphqlreadports.GraphQLReadResult{Data: map[string]any{"workers": map[string]any{
		"edges": []any{
			edge("wrk_1", "Ann", "Dallas", "c1"),
			edge("wrk_2", "Bo", "Tulsa", "c2"),
			edge("wrk_3", "Cy", "Waco", "c3"),
		},
		"pageInfo": map[string]any{"hasNextPage": false, "endCursor": "c3"},
	}}}
}

func workerList(t *testing.T) (*graphqllisttool.Tool, *querytooltest.KeyedReader) {
	t.Helper()

	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkers": workersAnswer(),
	}}
	tool, err := graphqllisttool.New(keyedSpec(t, querytool.ToolWorkerList), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	return tool, reader
}

// A page cut short resumes at the edge after the last row it kept, under the
// call's own cursor scope, so the next page reads the rows the cut left out.
func TestGraphQLListOutcome_ACutPageResumesAfterItsLastRow(t *testing.T) {
	t.Parallel()

	tool, _ := workerList(t)
	call := querytooltest.KeyedCall(map[string]any{})
	data, err := tool.Query(t.Context(), call)
	require.NoError(t, err)
	page, ok := data.(graphqllisttool.Outcome)
	require.True(t, ok)
	require.Equal(t, 3, page.PageRows())
	assert.False(t, page.HasMore)

	cut, ok := page.CutPage(2)
	require.True(t, ok)
	short, _ := cut.(graphqllisttool.Outcome)
	assert.Equal(t, 2, short.Count)
	assert.Len(t, short.Items, 2)
	assert.True(t, short.HasMore, "a cut page always has more")

	scope, err := querytool.CursorScope(querytool.ToolWorkerList, call.Params)
	require.NoError(t, err)
	resumed, err := pagination.DecodeKeyCursor(scope, short.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, "c2", resumed, "the next page starts after the second row, not the third")

	_, ok = page.CutPage(3)
	assert.False(t, ok, "a cut keeps fewer rows than the page")
}

// Rows ranked closest name first are no longer in the list's order, so no
// cut could say where the rest begins.
func TestGraphQLListOutcome_RankedRowsAreNotCut(t *testing.T) {
	t.Parallel()

	tool, _ := workerList(t)
	data, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{graphqlread.ParamQuery: "bo"}),
	)
	require.NoError(t, err)
	page, ok := data.(graphqllisttool.Outcome)
	require.True(t, ok)

	_, ok = page.CutPage(1)
	assert.False(t, ok)
}

// fields is offered as every column the read selects but the id, and the
// list returns every column whatever it names: the runtime narrows the
// model's copy, and the person's table, drawn from the list, keeps them all.
func TestGraphQLList_FieldsLeaveTheListWhole(t *testing.T) {
	t.Parallel()

	tool, _ := workerList(t)
	properties, _ := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
	fields, ok := properties[agenttoolports.ListFieldsParam].(map[string]any)
	require.True(t, ok, "a list with columns takes fields")
	items, _ := fields[toolschema.KeyItems].(map[string]any)
	assert.Contains(t, items[toolschema.KeyEnum], "wholeName")
	assert.Contains(t, items[toolschema.KeyEnum], "profile")
	assert.Contains(t, items[toolschema.KeyEnum], workerquerytools.DispatchableField,
		"what the list works out for every row may be named too")
	assert.NotContains(t, items[toolschema.KeyEnum], querytool.RecordIDField)

	data, err := tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		agenttoolports.ListFieldsParam: []any{"wholeName"},
	}))
	require.NoError(t, err)
	page, _ := data.(graphqllisttool.Outcome)
	require.Len(t, page.Items, 3)
	assert.Equal(t, "Dallas", page.Items[0]["city"], "the table keeps every column")
	assert.Contains(t, page.Columns, "city")

	plain, err := querytool.CursorScope(querytool.ToolWorkerList, map[string]any{})
	require.NoError(t, err)
	narrowed, err := querytool.CursorScope(
		querytool.ToolWorkerList,
		map[string]any{agenttoolports.ListFieldsParam: []any{"wholeName"}},
	)
	require.NoError(t, err)
	assert.Equal(t, plain, narrowed, "the columns read never change which rows a page holds")
}

// withoutCredential is a root argument, sent beside the table input.
func TestGraphQLList_WithoutCredentialReachesTheRoot(t *testing.T) {
	t.Parallel()

	tool, reader := workerList(t)
	properties, _ := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
	require.Contains(t, properties, "withoutCredential")
	_, err := tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		"withoutCredential": "ROAD_TEST",
	}))
	require.NoError(t, err)
	sent := reader.Sent("DeskWorkers")
	require.NotNil(t, sent)
	assert.Equal(t, "ROAD_TEST", sent.Variables["withoutCredential"])
}

func TestSearchOutcome_ACutResumesByCursorOrOffset(t *testing.T) {
	t.Parallel()

	rows := []shipmentquerytools.Row{{ID: "shp_1"}, {ID: "shp_2"}, {ID: "shp_3"}}
	byCursor := querytool.SearchOutcome{
		Count: 3, Items: rows,
		Resume: &querytool.PageResume{Scope: "scope", Cursors: []string{"k1", "k2", "k3"}},
	}
	cut, ok := byCursor.CutPage(1)
	require.True(t, ok)
	short, _ := cut.(querytool.SearchOutcome)
	assert.Equal(t, 1, short.Count)
	assert.True(t, short.HasMore)
	key, err := pagination.DecodeKeyCursor("scope", short.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, "k1", key)

	byOffset := querytool.SearchOutcome{
		Count:  3,
		Items:  rows,
		Offset: 25,
		Resume: &querytool.PageResume{ByOffset: true},
	}
	cut, ok = byOffset.CutPage(2)
	require.True(t, ok)
	short, _ = cut.(querytool.SearchOutcome)
	require.NotNil(t, short.NextOffset)
	assert.Equal(t, 27, *short.NextOffset)

	_, ok = querytool.SearchOutcome{Count: 3, Items: rows}.CutPage(1)
	assert.False(t, ok, "rows not in the list's order cannot be cut")

	gated := fieldgate.Result(&byCursor, nil)
	gated.Withheld = []string{"totalCharge"}
	cut, ok = gated.CutPage(2)
	require.True(t, ok)
	kept, _ := cut.(*fieldgate.Outcome)
	require.NotNil(t, kept)
	assert.Equal(t, []string{"totalCharge"}, kept.Withheld, "a gated page keeps what it withheld")
	assert.Equal(t, 2, kept.Count)
}

// A list paged by offset resumes at the row after the last one kept, counted
// among the rows read, which a query matched only some of.
func TestOffsetPage_ACutResumesAtTheRowAfter(t *testing.T) {
	t.Parallel()

	items := []map[string]any{{"id": "a"}, {"id": "c"}, {"id": "d"}}
	page := &graphqllisttool.OffsetPage{
		Record:    map[string]any{graphqlread.OutcomeItems: items, graphqlread.OutcomeCount: 3},
		Outside:   []agentrecord.RecordRef{{EntityType: "x", ID: "d"}},
		Positions: []int{0, 2, 3},
		Window:    querytool.OffsetWindow{Scope: "scope", Offset: 10, Limit: 4},
	}

	cut, ok := page.CutPage(2)
	require.True(t, ok)
	short, _ := cut.(*graphqllisttool.OffsetPage)
	assert.Equal(t, 2, short.Record[graphqlread.OutcomeCount])
	assert.Equal(t, true, short.Record[graphqllisttool.OutcomeHasMore])
	key, err := pagination.DecodeKeyCursor(
		"scope",
		short.Record[graphqllisttool.OutcomeNextCursor].(string),
	)
	require.NoError(t, err)
	assert.Equal(t, "13", key)
	assert.Empty(t, short.TaintedRecords(), "a marked row the cut left out is not in the page")
	assert.Len(t, page.Items(), 3, "the page cut from is unchanged")
}

// A page read beside other records cuts as its page does and keeps them.
func TestBesidePage_ACutKeepsWhatWasReadBeside(t *testing.T) {
	t.Parallel()

	tool, _ := workerList(t)
	data, err := tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{}))
	require.NoError(t, err)
	page, _ := data.(graphqllisttool.Outcome)
	beside, err := graphqllisttool.WithBeside(&page, map[string]any{"catalog": []any{"x"}})
	require.NoError(t, err)

	cut, ok := beside.(agenttoolports.PageCutter).CutPage(1)
	require.True(t, ok)
	record, ok := graphqllisttool.PageRecord(cut)
	require.True(t, ok)
	assert.Equal(t, []any{"x"}, record["catalog"])
	assert.Len(t, record[graphqlread.OutcomeItems], 1)
}

// A repository list offers fields as its row's columns but the id, so what
// the model reads narrows as a GraphQL list's does; a list that declares no
// row takes none.
func TestRepositoryList_FieldsAreItsRowsColumns(t *testing.T) {
	t.Parallel()

	properties, _ := masterdataquerytools.NewListFleetCodesTool(nil).ParamSchema()[toolschema.KeyProperties].(map[string]any)
	fields, ok := properties[agenttoolports.ListFieldsParam].(map[string]any)
	require.True(t, ok, "a list that declares its row takes fields")
	items, _ := fields[toolschema.KeyItems].(map[string]any)
	assert.Equal(t, []string{"code", "name", "description", "status"}, items[toolschema.KeyEnum])

	spec := agentrunquerytools.AgentRunListSpec()
	properties, _ = listtool.Build(&spec).ParamSchema()[toolschema.KeyProperties].(map[string]any)
	assert.NotContains(t, properties, agenttoolports.ListFieldsParam)
}
