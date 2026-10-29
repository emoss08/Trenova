package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pageRecord is a list's outcome as one object, whichever page carries it:
// a page read beside other records, or one that pages by offset.
func pageRecord(data any) (map[string]any, bool) {
	switch outcome := data.(type) {
	case map[string]any:
		return outcome, true
	case *besidePage:
		record, err := outcome.record()
		return record, err == nil
	case *offsetPage:
		return outcome.record, true
	}

	return nil, false
}

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

func workerList(t *testing.T) (*graphQLListTool, *keyedReader) {
	t.Helper()

	reader := &keyedReader{answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskWorkers": workersAnswer(),
	}}
	tool, err := newGraphQLListTool(keyedSpec(t, toolWorkerList), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	return tool, reader
}

// A page cut short resumes at the edge after the last row it kept, under the
// call's own cursor scope, so the next page reads the rows the cut left out.
func TestGraphQLListOutcome_ACutPageResumesAfterItsLastRow(t *testing.T) {
	t.Parallel()

	tool, _ := workerList(t)
	call := keyedCall(map[string]any{})
	data, err := tool.Query(t.Context(), call)
	require.NoError(t, err)
	page, ok := data.(graphQLListOutcome)
	require.True(t, ok)
	require.Equal(t, 3, page.PageRows())
	assert.False(t, page.HasMore)

	cut, ok := page.CutPage(2)
	require.True(t, ok)
	short, _ := cut.(graphQLListOutcome)
	assert.Equal(t, 2, short.Count)
	assert.Len(t, short.Items, 2)
	assert.True(t, short.HasMore, "a cut page always has more")

	scope, err := cursorScope(toolWorkerList, call.Params)
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
	data, err := tool.Query(t.Context(), keyedCall(map[string]any{graphQLParamQuery: "bo"}))
	require.NoError(t, err)
	page, ok := data.(graphQLListOutcome)
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
	fields, ok := properties[paramFields].(map[string]any)
	require.True(t, ok, "a list with columns takes fields")
	items, _ := fields[toolschema.KeyItems].(map[string]any)
	assert.Contains(t, items[toolschema.KeyEnum], "wholeName")
	assert.Contains(t, items[toolschema.KeyEnum], "profile")
	assert.Contains(t, items[toolschema.KeyEnum], workerDispatchable,
		"what the list works out for every row may be named too")
	assert.NotContains(t, items[toolschema.KeyEnum], recordIDField)

	data, err := tool.Query(t.Context(), keyedCall(map[string]any{
		paramFields: []any{"wholeName"},
	}))
	require.NoError(t, err)
	page, _ := data.(graphQLListOutcome)
	require.Len(t, page.Items, 3)
	assert.Equal(t, "Dallas", page.Items[0]["city"], "the table keeps every column")
	assert.Contains(t, page.Columns, "city")

	plain, err := cursorScope(toolWorkerList, map[string]any{})
	require.NoError(t, err)
	narrowed, err := cursorScope(toolWorkerList, map[string]any{paramFields: []any{"wholeName"}})
	require.NoError(t, err)
	assert.Equal(t, plain, narrowed, "the columns read never change which rows a page holds")
}

// withoutCredential is a root argument, sent beside the table input.
func TestGraphQLList_WithoutCredentialReachesTheRoot(t *testing.T) {
	t.Parallel()

	tool, reader := workerList(t)
	properties, _ := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
	require.Contains(t, properties, "withoutCredential")
	_, err := tool.Query(t.Context(), keyedCall(map[string]any{
		"withoutCredential": "ROAD_TEST",
	}))
	require.NoError(t, err)
	sent := reader.sent("DeskWorkers")
	require.NotNil(t, sent)
	assert.Equal(t, "ROAD_TEST", sent.Variables["withoutCredential"])
}

func TestSearchOutcome_ACutResumesByCursorOrOffset(t *testing.T) {
	t.Parallel()

	rows := []shipmentRow{{ID: "shp_1"}, {ID: "shp_2"}, {ID: "shp_3"}}
	byCursor := searchOutcome{
		Count: 3, Items: rows,
		resume: &pageResume{scope: "scope", cursors: []string{"k1", "k2", "k3"}},
	}
	cut, ok := byCursor.CutPage(1)
	require.True(t, ok)
	short, _ := cut.(searchOutcome)
	assert.Equal(t, 1, short.Count)
	assert.True(t, short.HasMore)
	key, err := pagination.DecodeKeyCursor("scope", short.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, "k1", key)

	byOffset := searchOutcome{
		Count:  3,
		Items:  rows,
		Offset: 25,
		resume: &pageResume{byOffset: true},
	}
	cut, ok = byOffset.CutPage(2)
	require.True(t, ok)
	short, _ = cut.(searchOutcome)
	require.NotNil(t, short.NextOffset)
	assert.Equal(t, 27, *short.NextOffset)

	_, ok = searchOutcome{Count: 3, Items: rows}.CutPage(1)
	assert.False(t, ok, "rows not in the list's order cannot be cut")

	gated := gatedResult(&byCursor, nil)
	gated.Withheld = []string{"totalCharge"}
	cut, ok = gated.CutPage(2)
	require.True(t, ok)
	kept, _ := cut.(*gatedOutcome)
	require.NotNil(t, kept)
	assert.Equal(t, []string{"totalCharge"}, kept.Withheld, "a gated page keeps what it withheld")
	assert.Equal(t, 2, kept.Count)
}

// A list paged by offset resumes at the row after the last one kept, counted
// among the rows read, which a query matched only some of.
func TestOffsetPage_ACutResumesAtTheRowAfter(t *testing.T) {
	t.Parallel()

	items := []map[string]any{{"id": "a"}, {"id": "c"}, {"id": "d"}}
	page := &offsetPage{
		record:    map[string]any{outcomeItems: items, outcomeCount: 3},
		outside:   []agentrecord.RecordRef{{EntityType: "x", ID: "d"}},
		positions: []int{0, 2, 3},
		window:    offsetWindow{scope: "scope", offset: 10, limit: 4},
	}

	cut, ok := page.CutPage(2)
	require.True(t, ok)
	short, _ := cut.(*offsetPage)
	assert.Equal(t, 2, short.record[outcomeCount])
	assert.Equal(t, true, short.record[outcomeHasMore])
	key, err := pagination.DecodeKeyCursor("scope", short.record[outcomeNextCursor].(string))
	require.NoError(t, err)
	assert.Equal(t, "13", key)
	assert.Empty(t, short.TaintedRecords(), "a marked row the cut left out is not in the page")
	assert.Len(t, page.items(), 3, "the page cut from is unchanged")
}

// A page read beside other records cuts as its page does and keeps them.
func TestBesidePage_ACutKeepsWhatWasReadBeside(t *testing.T) {
	t.Parallel()

	tool, _ := workerList(t)
	data, err := tool.Query(t.Context(), keyedCall(map[string]any{}))
	require.NoError(t, err)
	page, _ := data.(graphQLListOutcome)
	beside, err := withBeside(&page, map[string]any{"catalog": []any{"x"}})
	require.NoError(t, err)

	cut, ok := beside.(agenttoolports.PageCutter).CutPage(1)
	require.True(t, ok)
	record, ok := pageRecord(cut)
	require.True(t, ok)
	assert.Equal(t, []any{"x"}, record["catalog"])
	assert.Len(t, record[outcomeItems], 1)
}

// A repository list offers fields as its row's columns but the id, so what
// the model reads narrows as a GraphQL list's does; a list that declares no
// row takes none.
func TestRepositoryList_FieldsAreItsRowsColumns(t *testing.T) {
	t.Parallel()

	properties, _ := newListFleetCodesTool(nil).ParamSchema()[toolschema.KeyProperties].(map[string]any)
	fields, ok := properties[paramFields].(map[string]any)
	require.True(t, ok, "a list that declares its row takes fields")
	items, _ := fields[toolschema.KeyItems].(map[string]any)
	assert.Equal(t, []string{"code", "name", "description", "status"}, items[toolschema.KeyEnum])

	spec := agentRunListSpec()
	properties, _ = buildListTool(&spec).ParamSchema()[toolschema.KeyProperties].(map[string]any)
	assert.NotContains(t, properties, paramFields)
}
