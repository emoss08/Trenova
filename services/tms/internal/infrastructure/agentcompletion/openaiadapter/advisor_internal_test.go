package openaiadapter

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIAdapters_IgnoreTheAdvisor(t *testing.T) {
	t.Parallel()

	for _, adapter := range []modeladapter.Adapter{NewOpenAIChatAdapter(), NewOpenAIResponsesAdapter()} {
		server, captured := modeladaptertest.WireServer(t, "application/json", `{}`)
		request := modeladaptertest.AdvisorRequest()
		call := modeladaptertest.CallFor(adapter.Kind(), server.URL, &request)
		_, _ = adapter.Complete(t.Context(), call)

		capture := captured()
		require.NotEmpty(t, capture.Body, adapter.Kind())
		sent, err := modeladaptertest.WireJSON.MarshalToString(capture)
		require.NoError(t, err)
		assert.NotContains(t, strings.ToLower(sent), "advisor", adapter.Kind())
	}
}
