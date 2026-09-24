package agentevalgate_test

import (
	"context"
	"flag"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentevalgate"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/stretchr/testify/require"
)

var record = flag.Bool("record", false,
	"re-record the embedding fixture from an Ollama nomic-embed-text endpoint")

const recordTimeout = 2 * time.Minute

func TestRecordEmbeddingFixture(t *testing.T) {
	if !*record {
		t.Skip("pass -record to re-record " + agentevalgate.EmbeddingFixturePath)
	}

	baseURL := strings.TrimSpace(os.Getenv(agentevalgate.OllamaURLEnv))
	if baseURL == "" {
		baseURL = agentevalgate.DefaultOllamaURL
	}

	dimensions := airetrieval.Dimensions768
	provider := &aiprovider.Provider{
		Kind:                aiprovider.KindOllama,
		BaseURL:             baseURL,
		Model:               agentevalgate.EmbeddingFixtureModel,
		EmbeddingDimensions: &dimensions,
		EmbeddingInputStyle: aiprovider.EmbeddingInputStyleNomicPrefix,
	}
	embedder, err := modeladapter.NewRegistry().Embedder(aiprovider.KindOllama)
	require.NoError(t, err)
	client := &http.Client{Timeout: recordTimeout}

	embed := func(
		ctx context.Context,
		purpose serviceports.EmbeddingPurpose,
		inputs []string,
	) ([][]float32, error) {
		response, embedErr := embedder.Embed(ctx, &modeladapter.EmbedCall{
			Provider: provider,
			Client:   client,
			Purpose:  purpose,
			Inputs:   inputs,
		})
		if embedErr != nil {
			return nil, embedErr
		}

		return response.Vectors, nil
	}

	kit := newKit(t)
	suites := loadSuites(t, kit)
	inputs := suites.inputs(kit)

	fixture, err := agentevalgate.RecordEmbeddingFixture(t.Context(), embed, dimensions, inputs)
	require.NoError(t, err, "is Ollama serving %s at %s? (ollama pull %s)",
		agentevalgate.EmbeddingFixtureModel, baseURL, agentevalgate.EmbeddingFixtureModel)
	require.NoError(t, fixture.Stale(inputs))

	encoded, err := agentevalgate.MarshalJSON(fixture)
	require.NoError(t, err)
	require.NoError(t, agentevalgate.WriteFile(agentevalgate.EmbeddingFixturePath, encoded))

	t.Logf("recorded %d document and %d query vectors from %s into %s; now set the floors "+
		"with: %s", len(fixture.Documents), len(fixture.Queries), baseURL,
		agentevalgate.EmbeddingFixturePath, floorsCommand)
}
