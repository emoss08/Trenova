package aiprovider_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func builtInEmbedding() *aiprovider.Provider {
	return aiprovider.BuiltInEmbedding(
		pulid.MustNew("org_"),
		pulid.MustNew("bu_"),
		aiprovider.BuiltInEmbeddingService{
			URL:   "http://trenova-embedding:11434/",
			Model: "nomic-embed-text:v1.5",
		},
	)
}

func TestBuiltInEmbedding_ServesOnlyEmbedding(t *testing.T) {
	t.Parallel()

	provider := builtInEmbedding()

	ok, reason := provider.CanServeTask(aitask.TaskEmbedding)
	require.True(t, ok, reason)
	assert.True(t, provider.ServesOnlyEmbedding())
	assert.True(t, provider.BuiltIn)
	assert.Equal(t, aiprovider.BuiltInEmbeddingID, provider.ID)

	multiErr := errortypes.NewMultiError()
	provider.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), "the built-in provider is a valid embedding provider: %v", multiErr)
}

func TestBuiltInEmbedding_CallsOllamasOpenAIEndpointWithNomicPrefixes(t *testing.T) {
	t.Parallel()

	provider := builtInEmbedding()

	assert.Equal(t, "http://trenova-embedding:11434/v1", provider.ResolvedBaseURL())
	assert.Equal(t, aiprovider.KindOpenAIChat, provider.Kind)
	assert.Equal(t, aiprovider.EmbeddingInputStyleNomicPrefix, provider.ResolvedEmbeddingInputStyle())
	assert.False(t, provider.RequiresCredential())
	assert.False(t, provider.AllowPrivateNetwork,
		"the service is reached as instance configuration, never through an organization's private network setting")
}

func TestBuiltInEmbedding_RecordsItsModelUnderTheTunedFloor(t *testing.T) {
	t.Parallel()

	key := builtInEmbedding().EmbeddingModelKey()

	assert.Equal(t, "trenova-embedding:11434/nomic-embed-text:v1.5@768", key)
	dimensions, ok := airetrieval.ModelKeyDimensions(key)
	require.True(t, ok)
	assert.Equal(t, airetrieval.Dimensions768, dimensions)
	assert.Equal(t, airetrieval.TunedEmbeddingModel, airetrieval.ModelKeyModel(key))
	assert.Greater(t, airetrieval.SimilarityFloor(key), airetrieval.DefaultSimilarityFloor)
}

func TestBuiltInEmbedding_CostsNothing(t *testing.T) {
	t.Parallel()

	provider := builtInEmbedding()

	require.True(t, provider.Priced(), "a call is recorded at no cost, not as an unpriced call")
	assert.True(t, provider.InputCostPerMillion.IsZero())
	assert.True(t, provider.OutputCostPerMillion.IsZero())
}
