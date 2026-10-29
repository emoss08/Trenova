package adapterregistry

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/anthropicadapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/openaiadapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A stream that fails before the first byte is an ordinary transport failure
// and must carry the provider's message and retryability like a blocking call.
func TestStream_NonSuccessStatusIsATransportError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"model is loading"}}`))
	}))
	t.Cleanup(server.Close)

	for _, adapter := range []modeladapter.Adapter{
		openaiadapter.NewOpenAIChatAdapter(), anthropicadapter.NewAnthropicAdapter(), openaiadapter.NewOpenAIResponsesAdapter(),
	} {
		streamer := adapter.(modeladapter.Streamer)
		_, err := streamer.Stream(t.Context(), modeladaptertest.CallFor(
			adapter.Kind(),
			server.URL,
			&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
		), func(string) {})

		var transport *modeladapter.TransportError
		require.ErrorAs(t, err, &transport, "adapter %s", adapter.Kind())
		assert.Equal(t, http.StatusServiceUnavailable, transport.StatusCode)
		assert.True(t, transport.Retryable)
		assert.Contains(t, transport.Message, "model is loading")
	}
}
