package parallelconnector

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/shared/parallel"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var settings = map[string]string{
	agentextension.ConfigKeyAPIKey:     "parallel-key",
	agentextension.ConfigKeySearchType: agentextension.ParallelModeAdvanced,
}

func serve(t *testing.T, handler http.HandlerFunc) *Connector {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return newConnector(server.Client(), server.URL)
}

func TestSearchSendsTheOrganizationsChoices(t *testing.T) {
	t.Parallel()

	var received map[string]any
	connector := serve(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "parallel-key", r.Header.Get("x-api-key"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, sonic.Unmarshal(body, &received))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"search_id": "s", "session_id": "s", "results": [
			{"url": " https://www.fmcsa.dot.gov/hos ", "title": " HOS ", "publish_date": "2024-03-01", "excerpts": ["11 hours"]},
			{"url": "", "excerpts": []}
		]}`))
	})

	resp, err := connector.Search(t.Context(), &agentextensionports.WebSearchProviderRequest{
		Settings:           settings,
		Query:              "hours of service driving limit",
		NumResults:         5,
		ExcludeDomains:     []string{"spam.example.com", "*.blog.example.org"},
		StartPublishedDate: "2025-09-23T00:00:00Z",
		ExcerptCharacters:  1500,
	})
	require.NoError(t, err)

	assert.Equal(t, "advanced", received["mode"])
	assert.Equal(t, []any{"hours of service driving limit"}, received["search_queries"])
	advanced, ok := received["advanced_settings"].(map[string]any)
	require.True(t, ok)
	assert.InDelta(t, 5, advanced["max_results"], 0)
	policy, ok := advanced["source_policy"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{"spam.example.com", "blog.example.org"}, policy["exclude_domains"])
	assert.Equal(t, "2025-09-23", policy["after_date"])

	require.Len(t, resp.Results, 1)
	assert.Equal(t, "https://www.fmcsa.dot.gov/hos", resp.Results[0].URL)
	assert.Equal(t, "HOS", resp.Results[0].Title)
	assert.Equal(t, "2024-03-01", resp.Results[0].PublishedDate)
}

func TestReadReturnsTheFullPage(t *testing.T) {
	t.Parallel()

	connector := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"extract_id": "e", "session_id": "s", "errors": [], "results": [
			{"url": "https://www.ecfr.gov/395.8", "title": "395.8", "excerpts": [], "full_content": "Record of duty status"}
		]}`))
	})

	page, err := connector.Read(t.Context(), agentextensionports.WebPageProviderRequest{
		Settings:      settings,
		URL:           "https://www.ecfr.gov/395.8",
		MaxCharacters: 8001,
	})
	require.NoError(t, err)
	assert.Equal(t, "Record of duty status", page.Text)
	assert.Equal(t, "395.8", page.Title)
}

func TestReadOfAPageThatFailedIsNotFound(t *testing.T) {
	t.Parallel()

	connector := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"extract_id": "e", "session_id": "s", "results": [], "errors": [
			{"url": "https://blocked.example", "error_type": "fetch_failed", "http_status_code": 403}
		]}`))
	})

	page, err := connector.Read(t.Context(), agentextensionports.WebPageProviderRequest{
		Settings:      settings,
		URL:           "https://blocked.example",
		MaxCharacters: 8001,
	})
	require.ErrorIs(t, err, parallel.ErrContentNotFound)
	require.NotNil(t, page)
	assert.Equal(t,
		agentextensionports.ProviderFailureNotFound,
		connector.Classify(err).Kind,
	)
}

func TestClassifyNamesTheFailure(t *testing.T) {
	t.Parallel()

	connector := New()
	tests := []struct {
		err  error
		want agentextensionports.ProviderFailureKind
	}{
		{&restx.APIError{StatusCode: http.StatusUnauthorized}, agentextensionports.ProviderFailureUnauthorized},
		{&restx.APIError{StatusCode: http.StatusPaymentRequired}, agentextensionports.ProviderFailureOutOfCredits},
		{&restx.APIError{StatusCode: http.StatusTooManyRequests}, agentextensionports.ProviderFailureRateLimited},
		{&restx.APIError{StatusCode: http.StatusUnprocessableEntity, Message: "bad"}, agentextensionports.ProviderFailureInvalidRequest},
		{parallel.ErrInvalidRequest, agentextensionports.ProviderFailureInvalidRequest},
		{&restx.TransportError{}, agentextensionports.ProviderFailureUnknown},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, connector.Classify(tt.err).Kind, tt.err)
	}
	assert.Equal(t, "bad", connector.Classify(&restx.APIError{
		StatusCode: http.StatusUnprocessableEntity,
		Message:    "bad",
	}).Detail)
}

func TestConnectionTestRunsOneFastSearch(t *testing.T) {
	t.Parallel()

	var received map[string]any
	connector := serve(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, sonic.Unmarshal(body, &received))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"search_id": "s", "session_id": "s", "results": []}`))
	})

	result, err := connector.Test(t.Context(), settings)
	require.NoError(t, err)
	assert.Equal(t, "fast", received["mode"])
	assert.Equal(t, emptyTestNote, result.Note)
}
