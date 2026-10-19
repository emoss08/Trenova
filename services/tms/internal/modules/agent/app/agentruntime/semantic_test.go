package agentruntime

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findContent(
	t *testing.T,
	service *Service,
	set *toolSet,
	arguments map[string]any,
) string {
	t.Helper()

	return service.resolveFind(t.Context(), set, testActor(), arguments).Content
}

type fixedVectorizer struct {
	vector agentports.QueryVector
	calls  []agentports.QueryVectorRequest
}

func (v *fixedVectorizer) Vectorize(
	_ context.Context,
	req *agentports.QueryVectorRequest,
) (agentports.QueryVector, error) {
	v.calls = append(v.calls, *req)

	return v.vector, nil
}

func (v *fixedVectorizer) Availability(
	context.Context,
	pagination.TenantInfo,
) (airetrieval.Availability, error) {
	return airetrieval.Availability{Available: v.vector.Available}, nil
}

type fixedSimilarities struct {
	byKey    map[string]float64
	requests []agentports.CatalogSimilarityRequest
}

func (f *fixedSimilarities) Similarities(
	_ context.Context,
	req *agentports.CatalogSimilarityRequest,
) (agentports.CatalogSimilarities, error) {
	f.requests = append(f.requests, *req)

	return agentports.CatalogSimilarities{Available: true, ByKey: f.byKey}, nil
}

func usableQuery() agentports.QueryVector {
	vector := make([]float32, airetrieval.Dimensions768)
	vector[0] = 1
	vector[767] = 0.5

	return agentports.QueryVector{
		Available:  true,
		Vector:     vector,
		ModelKey:   "localhost/nomic-embed-text@768",
		Dimensions: airetrieval.Dimensions768,
	}
}

func semanticRuntime(
	t *testing.T,
	query agentports.QueryVector,
	similarity map[string]float64,
) (*Service, []string, *fixedVectorizer, *fixedSimilarities) {
	t.Helper()

	service, names := wideRuntime(t)
	vectorizer := &fixedVectorizer{vector: query}
	similarities := &fixedSimilarities{byKey: similarity}
	service.vectorizer = vectorizer
	service.vectors = similarities

	return service, names, vectorizer, similarities
}

func findToolsHistory(result conversation.Message) []conversation.Message {
	return []conversation.Message{
		{Role: conversation.RoleUser, Content: "say hello"},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{{
			ID:        "c1",
			Name:      findToolsName,
			Arguments: map[string]any{"need": "trailer inspection"},
		}}},
		result,
		{Role: conversation.RoleAssistant, Content: "Which one?"},
	}
}

func TestCarryOver_ReloadsTheToolsASearchRecorded(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	history := findToolsHistory(conversation.Message{
		Role:       conversation.RoleTool,
		ToolCallID: "c1",
		ToolName:   findToolsName,
		Content:    "These tools are now callable",
		FoundTools: []string{"worker_list"},
	})

	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: testDefinition(names...),
		actor:      testActor(),
		input:      "yes, do that",
		history:    history,
	})

	require.True(t, set.disclosed)
	sent := specNames(set.specs)
	assert.Contains(t, sent, "worker_list", "the recorded result is reloaded as it was")
	assert.NotContains(t, sent, "equipment_list_trailers",
		"the search is not run again when its result named what it found")
}

func TestCarryOver_SearchesAgainForAResultRecordedBeforeNamesWereKept(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	history := findToolsHistory(conversation.Message{
		Role:       conversation.RoleTool,
		ToolCallID: "c1",
		ToolName:   findToolsName,
		Content:    "These tools are now callable",
	})

	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: testDefinition(names...),
		actor:      testActor(),
		input:      "yes, do that",
		history:    history,
	})

	require.True(t, set.disclosed)
	sent := specNames(set.specs)
	assert.Contains(t, sent, "equipment_list_trailers", "an older result is answered by searching again")
	assert.NotContains(t, sent, "worker_list")
}

func TestCarryOver_NeverWidensPastWhatTheAgentHolds(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	held := make([]string, 0, len(names))
	for _, name := range names {
		if name != "worker_list" {
			held = append(held, name)
		}
	}
	history := findToolsHistory(conversation.Message{
		Role:       conversation.RoleTool,
		ToolCallID: "c1",
		ToolName:   findToolsName,
		FoundTools: []string{"worker_list"},
	})

	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: testDefinition(held...),
		actor:      testActor(),
		input:      "yes, do that",
		history:    history,
	})

	require.True(t, set.disclosed)
	assert.False(t, set.offers("worker_list"))
}

func TestRun_RecordsWhatFindToolsFoundOnItsResult(t *testing.T) {
	t.Parallel()

	service, _, names := wideRun(t,
		toolTurn(findToolsName, map[string]any{"need": "driver medical card expiry"}),
		textTurn("Two cards are due."),
	)

	result, err := service.Run(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(names...),
		Actor:      testActor(),
		Input:      "say hello",
	})
	require.NoError(t, err)

	require.Len(t, result.Messages, 4)
	assert.Equal(t, findToolsName, result.Messages[2].ToolName)
	assert.Contains(t, result.Messages[2].FoundTools, "worker_list_expiring_credentials")
}

func TestNewToolSet_KeywordRankingIsUnchangedWithoutAVector(t *testing.T) {
	t.Parallel()

	keyword, names := wideRuntime(t)
	unavailable, _, vectorizer, similarities := semanticRuntime(
		t,
		agentports.UnavailableQueryVector(airetrieval.UnavailableReasonNotIndexed),
		map[string]float64{"carrier_list": 0.99},
	)

	request := toolSetRequest{
		definition: testDefinition(names...),
		actor:      testActor(),
		input:      "which trailers are due inspection",
		query: agentports.QueryVectorRequest{
			TenantInfo: testActor().TenantInfo(),
			Text:       "which trailers are due inspection",
		},
	}
	want := keyword.newToolSet(t.Context(), request)
	got := unavailable.newToolSet(t.Context(), request)

	assert.Equal(t, specNames(want.specs), specNames(got.specs))
	assert.Len(t, vectorizer.calls, 1)
	assert.Empty(t, similarities.requests, "an unavailable vector is never compared")
	assert.Nil(t, got.state().Query)
}

func TestNewToolSet_PreselectsByMeaningWhenAVectorIsAvailable(t *testing.T) {
	t.Parallel()

	query := usableQuery()
	service, names, vectorizer, similarities := semanticRuntime(t, query, map[string]float64{
		"worker_list_expiring_credentials": 0.83,
		"worker_list":                      0.61,
		"carrier_list":                     0.12,
	})

	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: testDefinition(names...),
		actor:      testActor(),
		input:      "whose paperwork is going stale",
		query: agentports.QueryVectorRequest{
			TenantInfo: testActor().TenantInfo(),
			Text:       "whose paperwork is going stale",
		},
	})

	require.True(t, set.disclosed)
	require.Len(t, vectorizer.calls, 1, "one query vector per turn")
	require.Len(t, similarities.requests, 1)
	assert.Equal(t, airetrieval.CatalogCorpusTools, similarities.requests[0].Corpus)
	require.NotEmpty(t, set.preselected)
	assert.Equal(t, "worker_list_expiring_credentials", set.preselected[0],
		"the closest tool by meaning leads when no word matches")
	assert.NotContains(t, set.preselected[:3], "carrier_list",
		"a tool below the similarity floor gains nothing from the vector")

	state := set.state()
	require.NotNil(t, state.Query, "the vector rides on the turn for find_tools")
	restored := restoreToolSet(state)
	assert.Equal(t, query, restored.query)
}

func TestFindFor_SearchesByMeaningWithTheTurnsVector(t *testing.T) {
	t.Parallel()

	_, names := wideRuntime(t)
	similarity := make(map[string]float64, len(names))
	for idx, name := range names {
		similarity[name] = 0.9 - 0.02*float64(idx)
	}
	service, _, vectorizer, similarities := semanticRuntime(t, usableQuery(), similarity)

	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: testDefinition(names...),
		actor:      testActor(),
		input:      "say hello",
		query: agentports.QueryVectorRequest{
			TenantInfo: testActor().TenantInfo(),
			Text:       "say hello",
		},
	})
	require.True(t, set.disclosed)
	preselected := specNames(set.specs)
	for _, name := range names[:preselectedTools] {
		require.Contains(t, preselected, name, "preselection spends the vector first")
	}
	callsAtOpen := len(vectorizer.calls)

	found := service.FindFor(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(names...),
		Actor:      testActor(),
	}, set.state(), map[string]any{"need": "who is off this week"})

	assert.Equal(t, callsAtOpen, len(vectorizer.calls), "find_tools reuses the turn's vector")
	assert.Len(t, similarities.requests, 2)
	require.NotEmpty(t, found.Found)
	assert.Equal(t, names[preselectedTools], found.Found[0],
		"the next closest tool the turn does not carry comes first")
	for _, name := range found.Found {
		assert.NotContains(t, names[:preselectedTools], name,
			"a tool the turn already carries takes no slot")
	}
	assert.Contains(t, found.Loaded, names[preselectedTools])
}

func TestFindFor_ReturnsNothingForNonsenseBelowTheFloor(t *testing.T) {
	t.Parallel()

	service, names, _, _ := semanticRuntime(t, usableQuery(), map[string]float64{
		"carrier_list": 0.21,
		"worker_list":  0.18,
	})

	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: testDefinition(names...),
		actor:      testActor(),
		input:      "say hello",
		query: agentports.QueryVectorRequest{
			TenantInfo: testActor().TenantInfo(),
			Text:       "say hello",
		},
	})

	found := service.FindFor(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(names...),
		Actor:      testActor(),
	}, set.state(), map[string]any{"need": "zzzz qqqq"})

	assert.Empty(t, found.Found)
	assert.Empty(t, found.Loaded)
}

func TestQueryVectorState_RefusesATornVector(t *testing.T) {
	t.Parallel()

	state := packQueryVector(usableQuery())
	require.NotNil(t, state)

	state.Vector = state.Vector[:len(state.Vector)-4]
	assert.False(t, state.unpack().Usable())

	var missing *QueryVectorState
	assert.False(t, missing.unpack().Usable())
}
