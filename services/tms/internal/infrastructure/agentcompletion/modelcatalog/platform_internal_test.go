package modelcatalog

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlatform_ListsModelsOnlyWhereThePlatformServesThem(t *testing.T) {
	t.Parallel()

	for _, platform := range []aiplatform.Platform{
		aiplatform.PlatformBedrock, aiplatform.PlatformVertexAI, aiplatform.PlatformFoundry,
	} {
		_, err := ListModels(t.Context(), &ModelsCall{
			Provider: &aiprovider.Provider{
				Kind: aiprovider.KindAnthropicMessages, Platform: platform,
				PlatformSettings: &aiplatform.PlatformSettings{
					Region:    "us-east-1",
					ProjectID: "p",
					Resource:  "r",
				},
			},
			APIKey: "k",
			Client: http.DefaultClient,
		})
		require.ErrorIs(t, err, ErrModelsNotListed, platform)
	}

	server := serveModels(t, map[string]string{
		"/v1/models?limit=1000": `{"data":[{"id":"claude-opus-5-5","display_name":"Claude Opus 5.5",` +
			`"max_input_tokens":1000000}],"has_more":false}`,
	})
	models, err := ListModels(t.Context(), &ModelsCall{
		Provider: &aiprovider.Provider{
			Kind: aiprovider.KindAnthropicMessages, Platform: aiplatform.PlatformClaudeAWS,
			BaseURL: server.URL,
			PlatformSettings: &aiplatform.PlatformSettings{
				Auth: aiplatform.PlatformAuthAPIKey, Region: "us-east-1", WorkspaceID: "wrkspc_01ABC",
			},
		},
		APIKey: "sk-ant-aws",
		Client: server.Client(),
	})
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(
		t,
		1_000_000,
		models[0].ContextWindow,
		"the window the model reports is the one it has",
	)
}
