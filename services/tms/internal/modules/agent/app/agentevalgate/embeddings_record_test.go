package agentevalgate_test

import (
	"flag"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentevalgate"
	"github.com/emoss08/trenova/internal/testutil/ollamaembed"
	"github.com/stretchr/testify/require"
)

var record = flag.Bool("record", false,
	"re-record the embedding fixture from an Ollama nomic-embed-text endpoint")

const recordTimeout = 2 * time.Minute

func TestRecordEmbeddingFixture(t *testing.T) {
	if !*record {
		t.Skip("pass -record to re-record " + agentevalgate.EmbeddingFixturePath)
	}

	baseURL := ollamaembed.BaseURLFromEnv()
	dimensions := airetrieval.Dimensions768
	embed := ollamaembed.New(ollamaembed.Params{
		BaseURL:    baseURL,
		Model:      agentevalgate.EmbeddingFixtureModel,
		Dimensions: dimensions,
		Timeout:    recordTimeout,
	}).Embed

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
