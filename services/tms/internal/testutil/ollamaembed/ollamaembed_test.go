package ollamaembed_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentevalgate"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/testutil/ollamaembed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions"`
}

func newServer(t *testing.T, status int, reply string, seen *capturedRequest) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/embed", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		if seen != nil {
			assert.NoError(t, sonic.Unmarshal(body, seen))
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)

	return server
}

func TestEmbedPrefixesByPurposeAndAsksForTheDimension(t *testing.T) {
	t.Parallel()

	cases := map[aiusageports.EmbeddingPurpose]string{
		aiusageports.EmbeddingPurposeQuery:    "search_query: ",
		aiusageports.EmbeddingPurposeDocument: "search_document: ",
	}
	for purpose, prefix := range cases {
		t.Run(string(purpose), func(t *testing.T) {
			t.Parallel()

			var seen capturedRequest
			server := newServer(t, http.StatusOK, `{"embeddings":[[0.5,0.25],[1,0]]}`, &seen)
			client := ollamaembed.New(ollamaembed.Params{
				BaseURL:    server.URL + "/",
				Model:      "nomic-embed-text",
				Dimensions: 2,
			})

			vectors, err := client.Embed(t.Context(), purpose, []string{"a", "b"})
			require.NoError(t, err)

			assert.Equal(t, [][]float32{{0.5, 0.25}, {1, 0}}, vectors)
			assert.Equal(t, "nomic-embed-text", seen.Model)
			assert.Equal(t, []string{prefix + "a", prefix + "b"}, seen.Input)
			assert.Equal(t, 2, seen.Dimensions)
		})
	}
}

func TestEmbedRefusesAReplyThatDoesNotMatchTheRequest(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"count":     `{"embeddings":[[0.5,0.25]]}`,
		"dimension": `{"embeddings":[[0.5],[1]]}`,
	}
	for name, reply := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := newServer(t, http.StatusOK, reply, nil)
			client := ollamaembed.New(ollamaembed.Params{
				BaseURL:    server.URL,
				Model:      "nomic-embed-text",
				Dimensions: 2,
			})

			_, err := client.Embed(t.Context(), aiusageports.EmbeddingPurposeQuery, []string{"a", "b"})
			require.Error(t, err)
		})
	}
}

func TestEmbedReportsTheServersError(t *testing.T) {
	t.Parallel()

	server := newServer(t, http.StatusNotFound, `{"error":"model 'nomic-embed-text' not found"}`, nil)
	client := ollamaembed.New(ollamaembed.Params{
		BaseURL:    server.URL,
		Model:      "nomic-embed-text",
		Dimensions: 2,
	})

	_, err := client.Embed(t.Context(), aiusageports.EmbeddingPurposeQuery, []string{"a"})
	require.ErrorContains(t, err, "model 'nomic-embed-text' not found")
}

func TestBaseURLFromEnvFallsBackToTheDefaultAddress(t *testing.T) {
	t.Setenv(agentevalgate.OllamaURLEnv, "")
	assert.Equal(t, agentevalgate.DefaultOllamaURL, ollamaembed.BaseURLFromEnv())

	t.Setenv(agentevalgate.OllamaURLEnv, " http://gpu-box:11434 ")
	assert.Equal(t, "http://gpu-box:11434", ollamaembed.BaseURLFromEnv())
}
