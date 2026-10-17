package parallel_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/parallel"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newClient(t *testing.T, handler http.HandlerFunc) *parallel.Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := parallel.New("test-key",
		parallel.WithBaseURL(server.URL),
		parallel.WithRetry(restx.RetryConfig{}),
	)
	require.NoError(t, err)

	return client
}

func TestNewRequiresAnAPIKey(t *testing.T) {
	t.Parallel()

	_, err := parallel.New("   ")
	require.ErrorIs(t, err, parallel.ErrAPIKeyRequired)
}

func TestSearchSendsTheKeyAndDecodesResults(t *testing.T) {
	t.Parallel()

	var received map[string]any
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/search", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, sonic.Unmarshal(body, &received))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"search_id": "search_1",
			"session_id": "session_1",
			"results": [{
				"url": "https://www.fmcsa.dot.gov/hours-of-service",
				"title": "Summary of Hours of Service Regulations",
				"publish_date": "2024-03-01",
				"excerpts": ["Property-carrying drivers may drive 11 hours."]
			}],
			"usage": [{"name": "sku_search", "count": 1}]
		}`))
	})

	resp, err := client.Search(t.Context(), parallel.SearchRequest{
		Objective:     "hours of service driving limit",
		SearchQueries: []string{"  hours of service 11 hour rule  "},
		Mode:          parallel.ModeFast,
		AdvancedSettings: &parallel.SearchSettings{
			MaxResults:      5,
			ExcerptSettings: &parallel.ExcerptSettings{MaxCharsPerResult: 1200},
			SourcePolicy: &parallel.SourcePolicy{
				ExcludeDomains: []string{"example.com"},
				AfterDate:      "2025-09-23",
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, []any{"hours of service 11 hour rule"}, received["search_queries"])
	assert.Equal(t, "fast", received["mode"])
	settings, ok := received["advanced_settings"].(map[string]any)
	require.True(t, ok)
	assert.InDelta(t, 5, settings["max_results"], 0)
	policy, ok := settings["source_policy"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []any{"example.com"}, policy["exclude_domains"])
	assert.Equal(t, "2025-09-23", policy["after_date"])

	require.Len(t, resp.Results, 1)
	assert.Equal(t, "2024-03-01", resp.Results[0].PublishDate)
	assert.Equal(t, []string{"Property-carrying drivers may drive 11 hours."}, resp.Results[0].Excerpts)
	assert.True(t, decimal.RequireFromString("0.001").Equal(resp.TotalCost()))
}

func TestSearchCostFollowsTheModeAndExtraResults(t *testing.T) {
	t.Parallel()

	results := make([]string, 0, 12)
	for range 12 {
		results = append(results, `{"url": "https://example.gov/page", "excerpts": []}`)
	}
	payload := `{"search_id": "s", "session_id": "s", "results": [`
	for idx, result := range results {
		if idx > 0 {
			payload += ","
		}
		payload += result
	}
	payload += `]}`

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	})

	resp, err := client.Search(t.Context(), parallel.SearchRequest{
		SearchQueries: []string{"ifta deadlines"},
		Mode:          parallel.ModeAdvanced,
	})
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("0.007").Equal(resp.TotalCost()))
}

func TestSearchRefusesWhatTheAPIWouldReject(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("an invalid request must not be sent")
	})

	tests := []parallel.SearchRequest{
		{},
		{SearchQueries: []string{"   "}},
		{SearchQueries: []string{"ok"}, Mode: "deep"},
		{SearchQueries: []string{"ok"}, AdvancedSettings: &parallel.SearchSettings{MaxResults: 41}},
		{SearchQueries: []string{"ok"}, AdvancedSettings: &parallel.SearchSettings{
			SourcePolicy: &parallel.SourcePolicy{AfterDate: "23/09/2025"},
		}},
	}
	for _, req := range tests {
		_, err := client.Search(t.Context(), req)
		require.ErrorIs(t, err, parallel.ErrInvalidRequest)
	}
}

func TestExtractReturnsTheFullContentAndCountsEveryURL(t *testing.T) {
	t.Parallel()

	var received map[string]any
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/extract", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, sonic.Unmarshal(body, &received))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"extract_id": "extract_1",
			"session_id": "session_1",
			"results": [{
				"url": "https://www.ecfr.gov/current/title-49/section-395.8",
				"title": "49 CFR 395.8",
				"publish_date": null,
				"excerpts": [],
				"full_content": "# 395.8 Driver's record of duty status"
			}],
			"errors": [{"url": "https://blocked.example", "error_type": "fetch_failed", "http_status_code": 403}]
		}`))
	})

	maxAge := 86400
	resp, err := client.Extract(t.Context(), parallel.ExtractRequest{
		URLs: []string{"https://www.ecfr.gov/current/title-49/section-395.8", "https://blocked.example"},
		AdvancedSettings: &parallel.ExtractSettings{
			FullContent: &parallel.FullContentSettings{MaxCharsPerResult: 16001},
			FetchPolicy: &parallel.FetchPolicy{MaxAgeSeconds: &maxAge},
		},
	})
	require.NoError(t, err)

	settings, ok := received["advanced_settings"].(map[string]any)
	require.True(t, ok)
	full, ok := settings["full_content"].(map[string]any)
	require.True(t, ok)
	assert.InDelta(t, 16001, full["max_chars_per_result"], 0)

	page, found := resp.Page("https://www.ecfr.gov/current/title-49/section-395.8")
	require.True(t, found)
	assert.Equal(t, "# 395.8 Driver's record of duty status", page.FullContent)
	_, found = resp.Page("https://blocked.example")
	assert.False(t, found)
	assert.True(t, decimal.RequireFromString("0.002").Equal(resp.TotalCost()))
}

func TestExtractRefusesAFetchAgeBelowTheMinimum(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("an invalid request must not be sent")
	})

	maxAge := 60
	_, err := client.Extract(t.Context(), parallel.ExtractRequest{
		URLs:             []string{"https://example.gov"},
		AdvancedSettings: &parallel.ExtractSettings{FetchPolicy: &parallel.FetchPolicy{MaxAgeSeconds: &maxAge}},
	})
	require.ErrorIs(t, err, parallel.ErrInvalidRequest)
}

func TestErrorsCarryTheAPIMessage(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"type": "error", "error": {"ref_id": "r1", "message": "search_queries is too long"}}`))
	})

	_, err := client.Search(t.Context(), parallel.SearchRequest{SearchQueries: []string{"ifta"}})
	var apiErr *restx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.StatusCode)
	assert.Equal(t, "search_queries is too long", apiErr.Message)
}
