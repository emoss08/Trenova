package modeladapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func transferTool() []ToolSpec {
	return []ToolSpec{{
		Name: "transfer_to_billing",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"shipmentIds": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "string"},
				},
				"billType":                    map[string]any{"type": "string"},
				"markCompletedReadyToInvoice": map[string]any{"type": "boolean"},
			},
		},
	}}
}

func TestLiftInlineToolCalls_ReadsACompleteGLMCall(t *testing.T) {
	t.Parallel()

	text := "Transferring them now.\n<tool_call>transfer_to_billing\n" +
		"<arg_key>markCompletedReadyToInvoice</arg_key>\n<arg_value>true</arg_value>\n" +
		"<arg_key>billType</arg_key>\n<arg_value>Invoice</arg_value>\n" +
		"<arg_key>shipmentIds</arg_key>\n<arg_value>[\"shp_1\", \"shp_2\"]</arg_value>\n" +
		"</tool_call>"

	lifted := liftInlineToolCalls(text, transferTool(), 0)

	assert.Equal(t, "Transferring them now.", lifted.text)
	assert.Nil(t, lifted.cutOff)
	require.Len(t, lifted.calls, 1)
	call := lifted.calls[0]
	assert.Equal(t, "transfer_to_billing", call.Name)
	assert.Equal(t, "call_0", call.ID)
	assert.True(t, call.SynthesizedID)
	assert.Empty(t, call.ArgumentsError)
	assert.Equal(t, map[string]any{
		"markCompletedReadyToInvoice": true,
		"billType":                    "Invoice",
		"shipmentIds":                 []any{"shp_1", "shp_2"},
	}, call.Arguments)
}

func TestLiftInlineToolCalls_KeepsAStringParameterAsWrittenEvenWhenItLooksLikeJSON(t *testing.T) {
	t.Parallel()

	text := "<tool_call>transfer_to_billing<arg_key>billType</arg_key>" +
		"<arg_value>123</arg_value></tool_call>"

	lifted := liftInlineToolCalls(text, transferTool(), 0)

	require.Len(t, lifted.calls, 1)
	assert.Equal(t, "123", lifted.calls[0].Arguments["billType"])
}

func TestLiftInlineToolCalls_ReadsHermesJSONCalls(t *testing.T) {
	t.Parallel()

	text := "<tool_call>\n{\"name\": \"transfer_to_billing\", \"arguments\": " +
		"{\"shipmentIds\": [\"shp_1\"]}}\n</tool_call>\n" +
		"<tool_call>{\"name\": \"lookup_shipment\", \"arguments\": \"{\\\"number\\\": \\\"42\\\"}\"}" +
		"</tool_call>"

	lifted := liftInlineToolCalls(text, append(transferTool(), lookupTool()...), 2)

	assert.Empty(t, lifted.text)
	require.Len(t, lifted.calls, 2)
	assert.Equal(t, "transfer_to_billing", lifted.calls[0].Name)
	assert.Equal(t, "call_2", lifted.calls[0].ID)
	assert.Equal(t, map[string]any{"shipmentIds": []any{"shp_1"}}, lifted.calls[0].Arguments)
	assert.Equal(t, "lookup_shipment", lifted.calls[1].Name)
	assert.Equal(t, "call_3", lifted.calls[1].ID)
	assert.Equal(t, map[string]any{"number": "42"}, lifted.calls[1].Arguments)
}

func TestLiftInlineToolCalls_StripsAnUnfinishedCallAndNamesIt(t *testing.T) {
	t.Parallel()

	text := "I'll transfer all 21.\n<tool_call>transfer_to_billing<arg_key>" +
		"markCompletedReadyToInvoice</arg_key><arg_value>true</arg_value><arg_key>shipmentIds" +
		"</arg_key><arg_value>[\"shp_01K1\", \"shp_01K2"

	lifted := liftInlineToolCalls(text, transferTool(), 0)

	assert.Equal(t, "I'll transfer all 21.", lifted.text)
	assert.Empty(t, lifted.calls)
	require.NotNil(t, lifted.cutOff)
	assert.Equal(t, "transfer_to_billing", lifted.cutOff.Name)
}

func TestLiftInlineToolCalls_StripsAnUnfinishedHermesCall(t *testing.T) {
	t.Parallel()

	lifted := liftInlineToolCalls(
		"<tool_call>{\"name\": \"transfer_to_billing\", \"arguments\": {\"shipmentIds\": [\"shp_",
		transferTool(), 0,
	)

	assert.Empty(t, lifted.text)
	require.NotNil(t, lifted.cutOff)
	assert.Equal(t, "transfer_to_billing", lifted.cutOff.Name)
}

func TestLiftInlineToolCalls_StripsAnOpeningTagCutPartway(t *testing.T) {
	t.Parallel()

	lifted := liftInlineToolCalls("Here goes.\n<tool_ca", transferTool(), 0)

	assert.Equal(t, "Here goes.", lifted.text)
	require.NotNil(t, lifted.cutOff)
	assert.Empty(t, lifted.cutOff.Name)
}

func TestLiftInlineToolCalls_LeavesMarkupTheModelIsQuoting(t *testing.T) {
	t.Parallel()

	fenced := "GLM writes a call like this:\n```\n<tool_call>transfer_to_billing" +
		"<arg_key>billType</arg_key><arg_value>Invoice</arg_value></tool_call>\n```\nThat is all."
	inline := "The marker is `<tool_call>` and it closes with `</tool_call>`."

	for _, text := range []string{fenced, inline} {
		lifted := liftInlineToolCalls(text, transferTool(), 0)

		assert.Equal(t, text, lifted.text)
		assert.Empty(t, lifted.calls)
		assert.Nil(t, lifted.cutOff)
	}
}

func TestLiftInlineToolCalls_LeavesTextWithoutMarkupAlone(t *testing.T) {
	t.Parallel()

	text := "Totals: 21 shipments, $48,210.00. Is x < y? Yes."
	lifted := liftInlineToolCalls(text, transferTool(), 0)

	assert.Equal(t, text, lifted.text)
	assert.Empty(t, lifted.calls)
	assert.Nil(t, lifted.cutOff)
}

func TestLiftInlineToolCalls_NamesACompleteCallWhoseArgumentsDoNotParse(t *testing.T) {
	t.Parallel()

	lifted := liftInlineToolCalls(
		"<tool_call>{\"name\": \"transfer_to_billing\", \"arguments\": {\"shipmentIds\": [}</tool_call>",
		transferTool(), 0,
	)

	require.Len(t, lifted.calls, 1)
	assert.Equal(t, "transfer_to_billing", lifted.calls[0].Name)
	assert.NotEmpty(t, lifted.calls[0].ArgumentsError)
}

func TestOpenAIChatAdapter_StreamTurnsInlineMarkupIntoToolCalls(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "text/event-stream", sse(
		[2]string{
			"",
			`{"model":"glm","choices":[{"index":0,"delta":{"content":"On it.\n<tool_call>transfer_to_billing<arg_key>shipment"},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"glm","choices":[{"index":0,"delta":{"content":"Ids</arg_key><arg_value>[\"shp_1\"]</arg_value></tool_call>"},"finish_reason":"stop"}]}`,
		},
		[2]string{"", "[DONE]"},
	))

	resp, _ := streamWith(t, NewOpenAIChatAdapter(), callFor(
		aiprovider.KindOpenAIChat, server.URL,
		&Request{Messages: UserMessage("transfer"), Tools: transferTool()},
	))

	assert.Equal(t, "On it.", resp.Text)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "transfer_to_billing", resp.ToolCalls[0].Name)
	assert.Equal(t, map[string]any{"shipmentIds": []any{"shp_1"}}, resp.ToolCalls[0].Arguments)
	assert.Nil(t, resp.CutOffCall)
}

func TestOpenAIChatAdapter_StreamStripsACallTheOutputLimitCut(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "text/event-stream", sse(
		[2]string{
			"",
			`{"model":"glm","choices":[{"index":0,"delta":{"content":"<tool_call>transfer_to_billing<arg_key>shipmentIds</arg_key><arg_value>[\"shp_1\", \"shp"},"finish_reason":null}]}`,
		},
		[2]string{"", `{"model":"glm","choices":[{"index":0,"delta":{},"finish_reason":"length"}]}`},
		[2]string{"", "[DONE]"},
	))

	resp, _ := streamWith(t, NewOpenAIChatAdapter(), callFor(
		aiprovider.KindOpenAIChat, server.URL,
		&Request{Messages: UserMessage("transfer"), Tools: transferTool()},
	))

	assert.True(t, resp.Truncated)
	assert.Empty(t, resp.Text)
	assert.Empty(t, resp.ToolCalls)
	require.NotNil(t, resp.CutOffCall)
	assert.Equal(t, "transfer_to_billing", resp.CutOffCall.Name)
}

func TestOpenAIChatAdapter_CompleteTurnsInlineMarkupIntoToolCalls(t *testing.T) {
	t.Parallel()

	server, _ := captureServer(t, map[string]any{
		"model": "glm",
		"choices": []any{map[string]any{
			"finish_reason": "stop",
			"message": map[string]any{
				"role": "assistant",
				"content": "<tool_call>transfer_to_billing<arg_key>shipmentIds</arg_key>" +
					"<arg_value>[\"shp_1\"]</arg_value></tool_call>",
			},
		}},
	})

	resp, err := NewOpenAIChatAdapter().Complete(t.Context(), callFor(
		aiprovider.KindOpenAIChat, server.URL,
		&Request{Messages: UserMessage("transfer"), Tools: transferTool()},
	))

	require.NoError(t, err)
	assert.Empty(t, resp.Text)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "transfer_to_billing", resp.ToolCalls[0].Name)
}
