package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findContent(
	t *testing.T,
	service *Service,
	set *toolaccess.ToolSet,
	arguments map[string]any,
) string {
	t.Helper()

	return service.access.ResolveFind(t.Context(), set, runtimetest.Actor(), arguments).Content
}

func semanticRuntime(
	t *testing.T,
	query retrievalports.QueryVector,
	similarity map[string]float64,
) (*Service, []string, *runtimetest.FixedVectorizer, *runtimetest.FixedSimilarities) {
	t.Helper()

	service, names := wideRuntime(t)
	vectorizer := &runtimetest.FixedVectorizer{Vector: query}
	similarities := &runtimetest.FixedSimilarities{ByKey: similarity}
	rewire(service, func(p *Params) { p.Vectorizer = vectorizer })
	rewire(service, func(p *Params) { p.CatalogVectors = similarities })

	return service, names, vectorizer, similarities
}

func findToolsHistory(result conversation.Message) []conversation.Message {
	return []conversation.Message{
		{Role: conversation.RoleUser, Content: "say hello"},
		{Role: conversation.RoleAssistant, ToolCalls: []conversation.ToolCallRecord{{
			ID:        "c1",
			Name:      runtimetools.FindToolsName,
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
		ToolName:   runtimetools.FindToolsName,
		Content:    "These tools are now callable",
		FoundTools: []string{"worker_list"},
	})

	set := service.access.NewToolSet(t.Context(), &toolaccess.ToolSetRequest{
		Definition: runtimetest.Definition(names...),
		Actor:      runtimetest.Actor(),
		Input:      "yes, do that",
		History:    history,
	})

	require.True(t, set.Disclosed)
	sent := specNames(set.Specs)
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
		ToolName:   runtimetools.FindToolsName,
		Content:    "These tools are now callable",
	})

	set := service.access.NewToolSet(t.Context(), &toolaccess.ToolSetRequest{
		Definition: runtimetest.Definition(names...),
		Actor:      runtimetest.Actor(),
		Input:      "yes, do that",
		History:    history,
	})

	require.True(t, set.Disclosed)
	sent := specNames(set.Specs)
	assert.Contains(
		t,
		sent,
		"equipment_list_trailers",
		"an older result is answered by searching again",
	)
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
		ToolName:   runtimetools.FindToolsName,
		FoundTools: []string{"worker_list"},
	})

	set := service.access.NewToolSet(t.Context(), &toolaccess.ToolSetRequest{
		Definition: runtimetest.Definition(held...),
		Actor:      runtimetest.Actor(),
		Input:      "yes, do that",
		History:    history,
	})

	require.True(t, set.Disclosed)
	assert.False(t, set.Offers("worker_list"))
}

func TestRun_RecordsWhatFindToolsFoundOnItsResult(t *testing.T) {
	t.Parallel()

	service, _, names := wideRun(
		t,
		runtimetest.ToolTurn(
			runtimetools.FindToolsName,
			map[string]any{"need": "driver medical card expiry"},
		),
		runtimetest.TextTurn("Two cards are due."),
	)

	result, err := service.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(names...),
		Actor:      runtimetest.Actor(),
		Input:      "say hello",
	})
	require.NoError(t, err)

	require.Len(t, result.Messages, 4)
	assert.Equal(t, runtimetools.FindToolsName, result.Messages[2].ToolName)
	assert.Contains(t, result.Messages[2].FoundTools, "worker_list_credentials")
}

func TestNewToolSet_KeywordRankingIsUnchangedWithoutAVector(t *testing.T) {
	t.Parallel()

	keyword, names := wideRuntime(t)
	unavailable, _, vectorizer, similarities := semanticRuntime(
		t,
		retrievalports.UnavailableQueryVector(airetrieval.UnavailableReasonNotIndexed),
		map[string]float64{"carrier_list": 0.99},
	)

	request := toolaccess.ToolSetRequest{
		Definition: runtimetest.Definition(names...),
		Actor:      runtimetest.Actor(),
		Input:      "which trailers are due inspection",
		Query: retrievalports.QueryVectorRequest{
			TenantInfo: runtimetest.Actor().TenantInfo(),
			Text:       "which trailers are due inspection",
		},
	}
	want := keyword.access.NewToolSet(t.Context(), &request)
	got := unavailable.access.NewToolSet(t.Context(), &request)

	assert.Equal(t, specNames(want.Specs), specNames(got.Specs))
	assert.Len(t, vectorizer.Calls, 1)
	assert.Empty(t, similarities.Requests, "an unavailable vector is never compared")
	assert.Nil(t, got.State().Query)
}

func TestNewToolSet_PreselectsByMeaningWhenAVectorIsAvailable(t *testing.T) {
	t.Parallel()

	query := runtimetest.UsableQuery()
	service, names, vectorizer, similarities := semanticRuntime(t, query, map[string]float64{
		"worker_list_credentials": 0.83,
		"worker_list":             0.61,
		"carrier_list":            0.12,
	})

	set := service.access.NewToolSet(t.Context(), &toolaccess.ToolSetRequest{
		Definition: runtimetest.Definition(names...),
		Actor:      runtimetest.Actor(),
		Input:      "whose paperwork is going stale",
		Query: retrievalports.QueryVectorRequest{
			TenantInfo: runtimetest.Actor().TenantInfo(),
			Text:       "whose paperwork is going stale",
		},
	})

	require.True(t, set.Disclosed)
	require.Len(t, vectorizer.Calls, 1, "one query vector per turn")
	require.Len(t, similarities.Requests, 1)
	assert.Equal(t, airetrieval.CatalogCorpusTools, similarities.Requests[0].Corpus)
	require.NotEmpty(t, set.Preselected)
	assert.Equal(t, "worker_list_credentials", set.Preselected[0],
		"the closest tool by meaning leads when no word matches")
	assert.NotContains(t, set.Preselected[:3], "carrier_list",
		"a tool below the similarity floor gains nothing from the vector")

	state := set.State()
	require.NotNil(t, state.Query, "the vector rides on the turn for find_tools")
	restored := toolaccess.RestoreToolSet(&state)
	assert.Equal(t, query, restored.Query)
}

func TestFindFor_SearchesByMeaningWithTheTurnsVector(t *testing.T) {
	t.Parallel()

	_, names := wideRuntime(t)
	similarity := make(map[string]float64, len(names))
	for idx, name := range names {
		similarity[name] = 0.9 - 0.02*float64(idx)
	}
	service, _, vectorizer, similarities := semanticRuntime(
		t,
		runtimetest.UsableQuery(),
		similarity,
	)

	set := service.access.NewToolSet(t.Context(), &toolaccess.ToolSetRequest{
		Definition: runtimetest.Definition(names...),
		Actor:      runtimetest.Actor(),
		Input:      "say hello",
		Query: retrievalports.QueryVectorRequest{
			TenantInfo: runtimetest.Actor().TenantInfo(),
			Text:       "say hello",
		},
	})
	require.True(t, set.Disclosed)
	preselected := specNames(set.Specs)
	for _, name := range names[:toolaccess.PreselectedTools] {
		require.Contains(t, preselected, name, "preselection spends the vector first")
	}
	callsAtOpen := len(vectorizer.Calls)

	found := service.FindFor(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(names...),
		Actor:      runtimetest.Actor(),
	}, new(set.State()), map[string]any{"need": "who is off this week"})

	assert.Equal(t, callsAtOpen, len(vectorizer.Calls), "find_tools reuses the turn's vector")
	assert.Len(t, similarities.Requests, 2)
	require.NotEmpty(t, found.Found)
	assert.Equal(t, names[toolaccess.PreselectedTools], found.Found[0],
		"the next closest tool the turn does not carry comes first")
	for _, name := range found.Found {
		assert.NotContains(t, names[:toolaccess.PreselectedTools], name,
			"a tool the turn already carries takes no slot")
	}
	assert.Contains(t, found.Loaded, names[toolaccess.PreselectedTools])
}

func TestFindFor_ReturnsNothingForNonsenseBelowTheFloor(t *testing.T) {
	t.Parallel()

	service, names, _, _ := semanticRuntime(t, runtimetest.UsableQuery(), map[string]float64{
		"carrier_list": 0.21,
		"worker_list":  0.18,
	})

	set := service.access.NewToolSet(t.Context(), &toolaccess.ToolSetRequest{
		Definition: runtimetest.Definition(names...),
		Actor:      runtimetest.Actor(),
		Input:      "say hello",
		Query: retrievalports.QueryVectorRequest{
			TenantInfo: runtimetest.Actor().TenantInfo(),
			Text:       "say hello",
		},
	})

	found := service.FindFor(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(names...),
		Actor:      runtimetest.Actor(),
	}, new(set.State()), map[string]any{"need": "zzzz qqqq"})

	assert.Empty(t, found.Found)
	assert.Empty(t, found.Loaded)
}
