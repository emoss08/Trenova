package completionrouter

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chatProvider(name, baseURL string, priority int) *aiprovider.Provider {
	provider := openAIChatProvider(name, baseURL, priority)
	provider.Tasks = []aiprovider.Task{aiprovider.TaskAssistantChat}

	return provider
}

func chatRequest(preferred pulid.ID) *serviceports.ChatCompletionRequest {
	return &serviceports.ChatCompletionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		System:              "You are a test.",
		Messages:            serviceports.UserMessage("hello"),
		PreferredProviderID: preferred,
	}
}

// An agent may name the provider it wants. That provider goes first even when
// the routing order would have put another ahead of it.
func TestCompleteChat_HonoursThePreferredProvider(t *testing.T) {
	t.Parallel()

	primaryServer, primaryCalls := chatServer(t, 200, "from primary")
	preferredServer, preferredCalls := chatServer(t, 200, "from preferred")
	primary := chatProvider("primary", primaryServer.URL, 10)
	preferred := chatProvider("preferred", preferredServer.URL, 50)

	svc := newTestService(t, primary, preferred)
	result, err := svc.CompleteChat(t.Context(), chatRequest(preferred.ID))
	require.NoError(t, err)

	assert.Equal(t, "from preferred", result.Text)
	assert.Equal(t, preferred.ID, result.ProviderID)
	assert.Equal(t, int32(1), preferredCalls.Load())
	assert.Zero(t, primaryCalls.Load())
}

// A preference for a provider that no longer serves chat is ignored rather than
// stranding the agent: the usual order applies.
func TestCompleteChat_FallsBackToRoutingOrderWhenThePreferenceIsUnknown(t *testing.T) {
	t.Parallel()

	primaryServer, primaryCalls := chatServer(t, 200, "from primary")
	primary := chatProvider("primary", primaryServer.URL, 10)

	svc := newTestService(t, primary)
	result, err := svc.CompleteChat(t.Context(), chatRequest(pulid.MustNew("aiprv_")))
	require.NoError(t, err)

	assert.Equal(t, "from primary", result.Text)
	assert.Equal(t, int32(1), primaryCalls.Load())
}
