package modeladapter

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadedTools_SendsWhatIsLoadedInTheOrderDeclared(t *testing.T) {
	t.Parallel()

	tools := deferredWireRequest().Tools
	messages := deferredWireRequest().Messages

	assert.Equal(t,
		[]string{"get_shipment", "find_tools", "list_trailers"},
		toolNamesOf(loadedTools(tools, messages)),
		"a deferred tool a result loaded is sent, one nothing loaded is not",
	)
	assert.Equal(t,
		[]string{"get_shipment", "find_tools"},
		toolNamesOf(loadedTools(tools, messages[:1])),
	)
	for _, tool := range loadedTools(tools, messages) {
		assert.False(t, tool.Deferred)
	}
	plain := wireTools()
	assert.Equal(t, plain, loadedTools(plain, messages), "nothing deferred is sent as declared")
}

func TestOpenAIChat_IsSentWhatIsLoaded(t *testing.T) {
	t.Parallel()

	request := deferredWireRequest()

	assert.Len(t, toChatTools(loadedTools(request.Tools, request.Messages)), 3)
}

func TestToolSearchModel_IsGPT54AndLater(t *testing.T) {
	t.Parallel()

	for model, want := range map[string]bool{
		"gpt-5.4":        true,
		"gpt-5.4-mini":   true,
		"gpt-5.5":        true,
		"gpt-6-luna":     true,
		"gpt-6.1-sol":    true,
		"GPT-6":          true,
		"gpt-5.2":        false,
		"gpt-5-mini":     false,
		"gpt-4.1":        false,
		"o4-mini":        false,
		"claude-haiku-5": false,
	} {
		assert.Equal(t, want, toolSearchModel(model), model)
	}
}

func responsesCall(baseURL, model string, request *Request) *Call {
	call := callFor(aiprovider.KindOpenAIResponses, baseURL, request)
	call.Provider.BaseURL = baseURL
	call.Provider.Model = model

	return call
}

func encodedItems(t *testing.T, items []responsesItem) []map[string]any {
	t.Helper()

	encoded, err := sonic.Marshal(items)
	require.NoError(t, err)
	var decoded []map[string]any
	require.NoError(t, sonic.Unmarshal(encoded, &decoded))

	return decoded
}

func TestOpenAIResponses_RunsFindToolsAsAClientToolSearchOnOpenAI(t *testing.T) {
	t.Parallel()

	request := deferredWireRequest()
	body := openAIResponsesAdapter{}.streamRequestFor(
		responsesCall("https://api.openai.com", "gpt-6-luna", &request),
	)

	types := make([]string, 0, len(body.Tools))
	for _, tool := range body.Tools {
		types = append(types, tool.Type+":"+tool.Name)
		assert.False(t, tool.DeferLoading)
	}
	assert.ElementsMatch(t, []string{"function:get_shipment", "tool_search:"}, types,
		"deferred tools are not declared; the search stands in for find_tools")

	items := encodedItems(t, body.Input)
	require.Len(t, items, 5)
	assert.Equal(t, "tool_search_call", items[2]["type"])
	assert.Equal(t, "client", items[2]["execution"])
	assert.Equal(t, "call_find", items[2]["call_id"])
	assert.Equal(t, map[string]any{"need": "trailers due inspection"}, items[2]["arguments"],
		"a search's arguments are an object, not encoded text")
	assert.Equal(t, "tool_search_output", items[3]["type"])
	assert.Equal(t, "call_find", items[3]["call_id"])
	loaded := items[3]["tools"].([]any)
	require.Len(t, loaded, 1, "only a declared deferred tool is handed back")
	assert.Equal(t, "list_trailers", loaded[0].(map[string]any)["name"])
	assert.Equal(t, true, loaded[0].(map[string]any)["defer_loading"])
	assert.Equal(t, "developer", items[4]["role"], "what the search said follows it")
}

func TestOpenAIResponses_SendsWhatIsLoadedWhereToolSearchIsNotTaken(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ baseURL, model string }{
		{"https://api.openai.com", "gpt-5.2"},
		{"https://bedrock-mantle.us-east-1.api.aws", "gpt-6-luna"},
	} {
		request := deferredWireRequest()
		body := openAIResponsesAdapter{}.streamRequestFor(
			responsesCall(tc.baseURL, tc.model, &request),
		)

		names := make([]string, 0, len(body.Tools))
		for _, tool := range body.Tools {
			assert.Equal(t, "function", tool.Type, tc.model)
			names = append(names, tool.Name)
		}
		assert.Equal(t, []string{"get_shipment", "find_tools", "list_trailers"}, names, tc.model)
		items := encodedItems(t, body.Input)
		assert.Equal(t, "function_call", items[2]["type"], tc.model)
		assert.Equal(t, "function_call_output", items[3]["type"], tc.model)
	}
}

func TestSplitResponsesOutput_ReadsAToolSearchCallAsTheSearchTool(t *testing.T) {
	t.Parallel()

	envelope := &responsesEnvelope{Output: []responsesItem{
		{
			Type:      "tool_search_call",
			CallID:    "call_s",
			Execution: "client",
			Arguments: map[string]any{"need": "trailers"},
		},
		{
			Type:      "function_call",
			CallID:    "call_f",
			Name:      "get_shipment",
			Arguments: `{"number":"S1"}`,
		},
	}}

	_, calls, _ := splitResponsesOutput(envelope, "find_tools")

	require.Len(t, calls, 2)
	assert.Equal(t, ToolCall{ID: "call_s", Name: "find_tools",
		Arguments: map[string]any{"need": "trailers"}}, calls[0])
	assert.Equal(t, map[string]any{"number": "S1"}, calls[1].Arguments)

	_, unsearched, _ := splitResponsesOutput(envelope, "")
	assert.Len(t, unsearched, 1, "a search call is not read where no search was declared")
}

func toolNamesOf(tools []ToolSpec) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}

	return names
}
