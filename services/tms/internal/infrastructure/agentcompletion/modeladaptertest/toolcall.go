package modeladaptertest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
)

// captureServer records the request body an adapter sent and replies with a
// canned payload, so the wire translation can be asserted in both directions.
func CaptureServer(t *testing.T, reply any) (*httptest.Server, *map[string]any) {
	t.Helper()

	captured := map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&body)
		for k, v := range body {
			captured[k] = v
		}
		w.Header().Set("Content-Type", "application/json")
		encoded, _ := sonic.Marshal(reply)
		_, _ = w.Write(encoded)
	}))
	t.Cleanup(server.Close)

	return server, &captured
}

func CallFor(kind aiprovider.Kind, baseURL string, req *modeladapter.Request) *modeladapter.Call {
	return &modeladapter.Call{
		Provider: &aiprovider.Provider{
			Kind:                 kind,
			BaseURL:              baseURL,
			Model:                "test-model",
			AllowPrivateNetwork:  true,
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
		},
		APIKey: "k",
		Client: &http.Client{Timeout: 5 * time.Second},
		Request: &modeladapter.Request{
			System:         req.System,
			SystemStable:   req.SystemStable,
			CacheKey:       req.CacheKey,
			Messages:       req.Messages,
			Tools:          req.Tools,
			ToolChoiceNone: req.ToolChoiceNone,
			OutputSchema:   req.OutputSchema,
			SchemaName:     req.SchemaName,
			MaxTokens:      512,
			Reasoning:      req.Reasoning,
			Advisor:        req.Advisor,
		},
	}
}

func LookupTool() []modeladapter.ToolSpec {
	return []modeladapter.ToolSpec{{
		Name:        "lookup_shipment",
		Description: "Look up a shipment by number",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{"number": map[string]any{"type": "string"}},
			"required":   []string{"number"},
		},
	}}
}
