package exa_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/exa"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newClient(t *testing.T, handler http.HandlerFunc) *exa.Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := exa.New("test-key",
		exa.WithBaseURL(server.URL),
		exa.WithRetry(restx.RetryConfig{}),
	)
	require.NoError(t, err)

	return client
}

func TestNewRequiresAnAPIKey(t *testing.T) {
	t.Parallel()

	_, err := exa.New("   ")
	require.ErrorIs(t, err, exa.ErrAPIKeyRequired)
}

func TestSearchSendsTheKeyAndDecodesResults(t *testing.T) {
	t.Parallel()

	var received map[string]any
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/search", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, sonic.Unmarshal(body, &received))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"requestId": "req-1",
			"results": [{
				"id": "https://www.fmcsa.dot.gov/hours-of-service",
				"title": "Summary of Hours of Service Regulations",
				"url": "https://www.fmcsa.dot.gov/hours-of-service",
				"publishedDate": "2024-03-01T00:00:00.000Z",
				"author": null,
				"highlights": ["Property-carrying drivers may drive 11 hours."]
			}],
			"costDollars": {"total": 0.007}
		}`))
	})

	resp, err := client.Search(t.Context(), exa.SearchRequest{
		Query:          "  hours of service 11 hour rule  ",
		Type:           "auto",
		NumResults:     5,
		ExcludeDomains: []string{"example.com"},
		Contents: &exa.ContentOptions{
			Highlights: &exa.HighlightOptions{MaxCharacters: 1200},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "hours of service 11 hour rule", received["query"])
	assert.Equal(t, "auto", received["type"])
	assert.InDelta(t, 5, received["numResults"], 0)
	assert.Equal(t, []any{"example.com"}, received["excludeDomains"])
	assert.NotContains(t, received, "includeDomains")

	require.Len(t, resp.Results, 1)
	assert.Equal(t, "Summary of Hours of Service Regulations", resp.Results[0].Title)
	assert.Empty(t, resp.Results[0].Author)
	assert.Equal(t, []string{"Property-carrying drivers may drive 11 hours."}, resp.Results[0].Highlights)
	assert.True(t, decimal.RequireFromString("0.007").Equal(resp.TotalCost()))
}

func TestSearchRejectsABlankQueryWithoutCalling(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("a blank query must not reach Exa")
	})

	_, err := client.Search(t.Context(), exa.SearchRequest{Query: "  "})
	require.ErrorIs(t, err, exa.ErrInvalidRequest)
	assert.True(t, exa.IsInvalidRequest(err))
}

func TestErrorsCarryExasTag(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"requestId":"r","error":"You are out of credits","tag":"NO_MORE_CREDITS"}`))
	})

	_, err := client.Search(t.Context(), exa.SearchRequest{Query: "ifta"})
	require.Error(t, err)
	assert.True(t, exa.IsOutOfCredits(err))

	var apiErr *restx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "NO_MORE_CREDITS", apiErr.Code)
	assert.Equal(t, "You are out of credits", apiErr.Message)
	assert.NotContains(t, err.Error(), "test-key")
}

func TestUnauthorizedIsRecognized(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Invalid API key","tag":"INVALID_API_KEY"}`))
	})

	_, err := client.Search(t.Context(), exa.SearchRequest{Query: "eld mandate"})
	assert.True(t, exa.IsUnauthorized(err))
}

func TestContentsReportsAPageThatCouldNotBeRead(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/contents", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"requestId": "req-2",
			"results": [],
			"statuses": [{"id": "https://example.gov/x", "status": "error", "error": {"tag": "CRAWL_NOT_FOUND", "httpStatusCode": 404}}],
			"costDollars": {"total": 0.001}
		}`))
	})

	resp, err := client.Contents(t.Context(), exa.ContentsRequest{
		URLs: []string{"https://example.gov/x"},
		Text: &exa.TextOptions{MaxCharacters: 10000},
	})
	require.NoError(t, err)

	status, failed := resp.Failure("https://example.gov/x")
	require.True(t, failed)
	assert.Equal(t, "CRAWL_NOT_FOUND", status.Error.Tag)
	assert.Equal(t, http.StatusNotFound, status.Error.HTTPStatusCode)
}

func TestContentsRequiresAURL(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("an empty request must not reach Exa")
	})

	_, err := client.Contents(t.Context(), exa.ContentsRequest{})
	require.ErrorIs(t, err, exa.ErrInvalidRequest)
}
