package retrievalservice_test

import (
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentevalgate"
	"github.com/emoss08/trenova/internal/testutil/retrievaltest"
	"github.com/emoss08/trenova/internal/testutil/ollamaembed"
	"github.com/stretchr/testify/require"
)

var record = flag.Bool("record", false,
	"re-record the retrieval embedding fixture from an Ollama nomic-embed-text endpoint")

const (
	serviceRoot   = "../../../../.."
	recordTimeout = 2 * time.Minute
)

func fromServiceRoot(path string) string {
	return filepath.Join(serviceRoot, path)
}

func loadRetrievalSuites(t *testing.T) (documents, inbox, memories *agentevalgate.RetrievalSuite) {
	t.Helper()

	var err error
	documents, err = agentevalgate.LoadRetrievalSuite(fromServiceRoot(retrievaltest.DocumentSuite))
	require.NoError(t, err)
	inbox, err = agentevalgate.LoadRetrievalSuite(fromServiceRoot(retrievaltest.InboxSuite))
	require.NoError(t, err)
	memories, err = agentevalgate.LoadRetrievalSuite(fromServiceRoot(retrievaltest.MemorySuite))
	require.NoError(t, err)

	return documents, inbox, memories
}

func TestRecordRetrievalEmbeddingFixture(t *testing.T) {
	if !*record {
		t.Skip("pass -record to re-record " + retrievaltest.FixturePath)
	}

	baseURL := ollamaembed.BaseURLFromEnv()
	dimensions := airetrieval.Dimensions768
	embed := ollamaembed.New(ollamaembed.Params{
		BaseURL:    baseURL,
		Model:      agentevalgate.EmbeddingFixtureModel,
		Dimensions: dimensions,
		Timeout:    recordTimeout,
	}).Embed

	documents, inbox, memories := loadRetrievalSuites(t)
	inputs := retrievaltest.EmbeddingInputs(documents, inbox, memories)

	fixture, err := agentevalgate.RecordEmbeddingFixture(t.Context(), embed, dimensions, inputs)
	require.NoError(
		t,
		err,
		"is Ollama serving %s at %s?",
		agentevalgate.EmbeddingFixtureModel,
		baseURL,
	)

	encoded, err := agentevalgate.MarshalJSON(fixture)
	require.NoError(t, err)
	require.NoError(t, agentevalgate.WriteFile(fromServiceRoot(retrievaltest.FixturePath), encoded))

	t.Logf("recorded %d chunk and %d query vectors; now set the floors with:\n  %s",
		len(fixture.Documents), len(fixture.Queries), retrievaltest.FloorsCommand)
}
