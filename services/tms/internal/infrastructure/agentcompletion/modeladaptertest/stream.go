package modeladaptertest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/stretchr/testify/require"
)

// streamServer replies with a canned streaming body and records the request so
// the stream flag the adapter set can be asserted.
func StreamServer(
	t *testing.T,
	contentType string,
	body string,
) (*httptest.Server, *map[string]any) {
	t.Helper()

	captured := map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoded := map[string]any{}
		_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&decoded)
		for k, v := range decoded {
			captured[k] = v
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server, &captured
}

func SSE(events ...[2]string) string {
	var builder strings.Builder
	for _, event := range events {
		if event[0] != "" {
			builder.WriteString("event: " + event[0] + "\n")
		}
		builder.WriteString("data: " + event[1] + "\n\n")
	}

	return builder.String()
}

func CollectDeltas() (modeladapter.StreamSink, *[]string) {
	deltas := []string{}
	return func(delta string) { deltas = append(deltas, delta) }, &deltas
}

func StreamWith(
	t *testing.T,
	adapter modeladapter.Adapter,
	call *modeladapter.Call,
) (*modeladapter.Response, []string) {
	t.Helper()

	streamer, ok := adapter.(modeladapter.Streamer)
	require.True(t, ok, "adapter %s must stream", adapter.Kind())

	sink, deltas := CollectDeltas()
	resp, err := streamer.Stream(t.Context(), call, sink)
	require.NoError(t, err)

	return resp, *deltas
}
