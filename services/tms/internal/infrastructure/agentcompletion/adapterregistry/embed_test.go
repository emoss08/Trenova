package adapterregistry

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_HasNoEmbedderForAnthropic(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()

	_, err := registry.Embedder(aiprovider.KindAnthropicMessages)
	require.Error(t, err)
	assert.ErrorIs(t, err, llmports.ErrNoProviderConfigured)

	for _, kind := range []aiprovider.Kind{
		aiprovider.KindOpenAIResponses,
		aiprovider.KindOpenAIChat,
	} {
		embedder, embedErr := registry.Embedder(kind)
		require.NoError(t, embedErr, kind)
		assert.NotNil(t, embedder, kind)
	}
}
