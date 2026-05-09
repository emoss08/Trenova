//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package shipmentimportchat

import "github.com/bytedance/sonic"

type HistoryAction struct {
	Type     string         `json:"type"`
	FieldKey string         `json:"fieldKey"`
	Value    string         `json:"value"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type TurnPayload struct {
	Context     map[string]any
	Suggestions []HistorySuggestion
	ToolCalls   []HistoryToolCall
	Actions     []HistoryAction
}

type EncodedTurnPayload struct {
	ContextJSON     rawJSON
	SuggestionsJSON rawJSON
	ToolCallsJSON   rawJSON
	ActionsJSON     rawJSON
}

func (p TurnPayload) Encode() EncodedTurnPayload {
	return EncodedTurnPayload{
		ContextJSON:     encodeJSON(p.Context, []byte("{}")),
		SuggestionsJSON: encodeJSON(p.Suggestions, []byte("[]")),
		ToolCallsJSON:   encodeJSON(p.ToolCalls, []byte("[]")),
		ActionsJSON:     encodeJSON(p.Actions, []byte("[]")),
	}
}

func DecodeTurnPayload(turn *Turn) TurnPayload {
	payload := TurnPayload{
		Context:     map[string]any{},
		Suggestions: []HistorySuggestion{},
		ToolCalls:   []HistoryToolCall{},
		Actions:     []HistoryAction{},
	}

	if turn == nil {
		return payload
	}

	_ = decodeJSON(turn.ContextJSON, &payload.Context)
	_ = decodeJSON(turn.SuggestionsJSON, &payload.Suggestions)
	_ = decodeJSON(turn.ToolCallsJSON, &payload.ToolCalls)
	_ = decodeJSON(turn.ActionsJSON, &payload.Actions)

	return payload
}

func encodeJSON(value any, fallback rawJSON) rawJSON {
	data, err := sonic.Marshal(value)
	if err != nil {
		return fallback
	}

	return data
}

func decodeJSON(raw rawJSON, target any) error {
	if len(raw) == 0 {
		return nil
	}

	return sonic.Unmarshal(raw, target)
}
