package modeladaptertest

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
)

var WireJSON = sonic.Config{SortMapKeys: true}.Froze()

type WireCapture struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Query   string            `json:"query"`
	Headers map[string]string `json:"headers"`
	Body    map[string]any    `json:"body"`
}

var wireHeaders = []string{
	"Content-Type",
	"X-Api-Key",
	"Anthropic-Version",
	"Anthropic-Beta",
	"Authorization",
}

func WireServer(t *testing.T, contentType, reply string) (*httptest.Server, func() WireCapture) {
	t.Helper()

	var (
		mu       sync.Mutex
		captured WireCapture
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&body)
		headers := make(map[string]string, len(wireHeaders))
		for _, name := range wireHeaders {
			if value := r.Header.Get(name); value != "" {
				headers[name] = value
			}
		}
		mu.Lock()
		captured = WireCapture{
			Method:  r.Method,
			Path:    r.URL.Path,
			Query:   r.URL.RawQuery,
			Headers: headers,
			Body:    body,
		}
		mu.Unlock()
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)

	return server, func() WireCapture {
		mu.Lock()
		defer mu.Unlock()
		return captured
	}
}

func WireTools() []modeladapter.ToolSpec {
	return []modeladapter.ToolSpec{
		{
			Name:        "get_shipment",
			Description: "Look up a shipment by its number",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"number": map[string]any{
						"type":        "string",
						"description": "Pro number",
						"maxLength":   20,
					},
					"include": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
				"required":             []any{"number"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "list_workers",
			Description: "List workers",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		},
	}
}

func DeferredWireRequest() modeladapter.Request {
	tools := append(WireTools(), modeladapter.ToolSpec{
		Name:        "find_tools",
		Description: "Load more tools.",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{"need": map[string]any{"type": "string"}},
			"required":   []any{"need"},
		},
		Searches: true,
	}, modeladapter.ToolSpec{
		Name:        "list_trailers",
		Description: "List trailers",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		Deferred:    true,
	})
	tools[1].Deferred = true

	return modeladapter.Request{
		System:       "Rules that never change.\n\nToday is Friday.",
		SystemStable: len("Rules that never change."),
		Messages: []modeladapter.Message{
			{Role: modeladapter.RoleUser, Content: "Which trailers are due?"},
			{
				Role: modeladapter.RoleAssistant,
				ToolCalls: []modeladapter.ToolCall{{
					ID: "call_find", Name: "find_tools",
					Arguments: map[string]any{"need": "trailers due inspection"},
				}},
			},
			{
				Role: modeladapter.RoleTool, ToolCallID: "call_find", ToolName: "find_tools",
				Content:        "These tools are now callable:\n- list_trailers: List trailers",
				ToolReferences: []string{"list_trailers", "get_shipment", "not_declared"},
			},
		},
		Tools: tools,
	}
}
