package pcmiler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/require"
)

func TestNewRequiresAPIKey(t *testing.T) {
	t.Parallel()

	_, err := New(Config{APIKey: "   "})
	require.Error(t, err)
}

func TestNewAppliesDefaultsAndTrimsBaseURL(t *testing.T) {
	t.Parallel()

	client, err := New(Config{APIKey: "key"})
	require.NoError(t, err)
	require.Equal(t, defaultBaseURL, client.baseURL)
	require.Equal(t, defaultTimeout, client.httpClient.Timeout)

	client, err = New(Config{APIKey: "key", BaseURL: " https://example.com/api/ ", Timeout: 30})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/api", client.baseURL)
	require.Equal(t, "30s", client.httpClient.Timeout.String())
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL})
	require.NoError(t, err)
	return client
}

func TestVersionsSendsAuthAndParsesResponse(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/pcmversion", r.URL.Path)
		require.Equal(t, "NA", r.URL.Query().Get("region"))
		require.Equal(t, "test-key", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Accept"))
		_, _ = w.Write([]byte(`{"pcmversions":["Current","PCM36"]}`))
	})

	versions, err := client.Versions(t.Context(), "NA")
	require.NoError(t, err)
	require.Equal(t, []Version{{Name: "Current"}, {Name: "PCM36"}}, versions)
}

func TestVersionsOmitsEmptyRegion(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.False(t, r.URL.Query().Has("region"))
		_, _ = w.Write([]byte(`{"pcmversions":[]}`))
	})

	versions, err := client.Versions(t.Context(), "  ")
	require.NoError(t, err)
	require.Empty(t, versions)
}

func TestMileageReturnsHTTPErrorWithTrimmedBody(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(strings.Repeat("x", 400)))
	})

	_, err := client.Mileage(t.Context(), []RouteRequest{{RouteID: "route-1"}})
	require.Error(t, err)

	var httpErr *HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusBadRequest, httpErr.StatusCode)
	require.Len(t, httpErr.Message, 256)
}

func TestHTTPErrorReportsEmptyResponse(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := client.Versions(t.Context(), "")
	require.Error(t, err)

	var httpErr *HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, "empty response", httpErr.Message)
}

func mileageHandler(t *testing.T, requestCount *int, batchSizes *[]int) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/route/routeReports", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload struct {
			ReportRoutes []struct {
				RouteID string `json:"RouteId"`
			} `json:"ReportRoutes"`
		}
		require.NoError(t, sonic.Unmarshal(body, &payload))

		*requestCount++
		*batchSizes = append(*batchSizes, len(payload.ReportRoutes))

		reports := make([]map[string]any, 0, len(payload.ReportRoutes))
		for _, route := range payload.ReportRoutes {
			reports = append(reports, map[string]any{
				"__type":  "MileageReport:http://pcmiler.alk.com/APIs/v1.0",
				"RouteID": route.RouteID,
				"ReportLines": []map[string]any{
					{"TMiles": "100.5"},
				},
			})
		}

		response, err := sonic.Marshal(reports)
		require.NoError(t, err)
		_, _ = w.Write(response)
	}
}

func TestMileageBatchesRequestsOfTwentyRoutes(t *testing.T) {
	t.Parallel()

	var requestCount int
	var batchSizes []int
	client := newTestClient(t, mileageHandler(t, &requestCount, &batchSizes))

	routes := make([]RouteRequest, 0, maxRoutesBatch+5)
	for idx := range maxRoutesBatch + 5 {
		routes = append(routes, RouteRequest{
			RouteID: fmt.Sprintf("route-%d", idx),
			Stops: []Stop{
				{City: "Philadelphia", State: "PA"},
				{City: "Pittsburgh", State: "PA"},
			},
		})
	}

	results, err := client.Mileage(t.Context(), routes)
	require.NoError(t, err)
	require.Len(t, results, maxRoutesBatch+5)
	require.Equal(t, 2, requestCount)
	require.Equal(t, []int{maxRoutesBatch, 5}, batchSizes)
	require.Equal(t, "route-0", results[0].RouteID)
	require.Equal(t, fmt.Sprintf("route-%d", maxRoutesBatch+4), results[len(results)-1].RouteID)
	require.InDelta(t, 100.5, results[0].Distance, 0.0001)
}

func TestMileageDefaultsDataVersionQuery(t *testing.T) {
	t.Parallel()

	var dataVersions []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		dataVersions = append(dataVersions, r.URL.Query().Get("dataVersion"))
		_, _ = w.Write([]byte(`[]`))
	})

	_, err := client.Mileage(t.Context(), []RouteRequest{{RouteID: "route-1"}})
	require.NoError(t, err)

	_, err = client.Mileage(t.Context(), []RouteRequest{
		{RouteID: "route-2", Options: RouteOptions{DataVersion: "PCM36"}},
	})
	require.NoError(t, err)

	require.Equal(t, []string{"Current", "PCM36"}, dataVersions)
}

func TestMileageWithNoRoutesMakesNoRequests(t *testing.T) {
	t.Parallel()

	var requestCount int
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
	})

	results, err := client.Mileage(t.Context(), nil)
	require.NoError(t, err)
	require.Empty(t, results)
	require.Zero(t, requestCount)
}
