package carrierok_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/carrierok"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	liveKey    = "sk_live_0123456789abcdefSECRET"
	sandboxKey = "sk_test_0123456789abcdefSECRET"
)

type recordingLimiter struct {
	mu      sync.Mutex
	buckets []restx.Bucket
}

func (l *recordingLimiter) Acquire(_ context.Context, bucket restx.Bucket) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buckets = append(l.buckets, bucket)
	return nil
}

func (l *recordingLimiter) recorded() []restx.Bucket {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]restx.Bucket(nil), l.buckets...)
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return data
}

func newTestClient(
	t *testing.T,
	key string,
	handler http.HandlerFunc,
	opts ...carrierok.Option,
) *carrierok.Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	options := append([]carrierok.Option{
		carrierok.WithBaseURL(server.URL),
		carrierok.WithTimeout(5 * time.Second),
		carrierok.WithRetry(restx.RetryConfig{}),
	}, opts...)

	client, err := carrierok.New(key, options...)
	require.NoError(t, err)
	return client
}

func writeJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func assertNoKey(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), liveKey)
	assert.NotContains(t, err.Error(), sandboxKey)
	assert.NotContains(t, err.Error(), "0123456789abcdefSECRET")
}

func TestNewRequiresAPIKey(t *testing.T) {
	t.Parallel()

	_, err := carrierok.New("   ")
	require.ErrorIs(t, err, carrierok.ErrAPIKeyRequired)
}

func TestIsSandboxKey(t *testing.T) {
	t.Parallel()

	assert.True(t, carrierok.IsSandboxKey(sandboxKey))
	assert.True(t, carrierok.IsSandboxKey("  sk_test_x"))
	assert.False(t, carrierok.IsSandboxKey(liveKey))
	assert.False(t, carrierok.IsSandboxKey(""))
}

func TestProfileSendsAuthAndExactQuery(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, liveKey, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v2/profile", r.URL.Path)
		assert.Equal(t, "dot_number=265752", r.URL.RawQuery)
		assert.Equal(t, "Bearer "+liveKey, r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		writeJSON(w, http.StatusOK, fixture(t, "profile_265752.json"))
	})

	profile, err := client.Profile(t.Context(), carrierok.ProfileQuery{DOTNumber: " 265752 "})
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "265752", profile.Identity.DOTNumber.Value())
	assert.Equal(t, "265752-MC179059", profile.ProfileID())
	assert.NotEmpty(t, profile.Raw)
}

func TestProfileQueryParameterNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query carrierok.ProfileQuery
		want  string
	}{
		{
			name:  "docket",
			query: carrierok.ProfileQuery{DocketNumber: "MC277621"},
			want:  "docket_number=MC277621",
		},
		{name: "company", query: carrierok.ProfileQuery{Company: "Acme"}, want: "company=Acme"},
		{name: "ein", query: carrierok.ProfileQuery{EIN: "364123456"}, want: "ein=364123456"},
		{name: "email", query: carrierok.ProfileQuery{Email: "a@b.co"}, want: "email=a%40b.co"},
		{
			name:  "phone",
			query: carrierok.ProfileQuery{Phone: "3125550100"},
			want:  "phone=3125550100",
		},
		{
			name:  "address",
			query: carrierok.ProfileQuery{Address: "100 Main"},
			want:  "address=100+Main",
		},
		{name: "vin", query: carrierok.ProfileQuery{VIN: "1XKY"}, want: "vin=1XKY"},
		{
			name:  "plate",
			query: carrierok.ProfileQuery{PlateNumber: "P1", PlateState: "il"},
			want:  "plate_number=P1&plate_state=IL",
		},
		{
			name:  "unit",
			query: carrierok.ProfileQuery{UnitNumber: "101", UnitType: "trailer"},
			want:  "unit_number=101&unit_type=TRAILER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := newTestClient(t, liveKey, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v2/profile-lite", r.URL.Path)
				assert.Equal(t, tt.want, r.URL.RawQuery)
				writeJSON(w, http.StatusOK, fixture(t, "profile_265752.json"))
			})

			_, err := client.ProfileLite(t.Context(), tt.query)
			require.NoError(t, err)
		})
	}
}

func TestProfileValidationSkipsNetwork(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newTestClient(t, liveKey, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	})

	_, err := client.Profile(t.Context(), carrierok.ProfileQuery{})
	require.ErrorIs(t, err, carrierok.ErrInvalidQuery)
	_, err = client.Profile(t.Context(), carrierok.ProfileQuery{DOTNumber: "1", VIN: "x"})
	require.ErrorIs(t, err, carrierok.ErrInvalidQuery)
	_, err = client.ProfileFMCSA(t.Context(), carrierok.FMCSAQuery{})
	require.ErrorIs(t, err, carrierok.ErrInvalidQuery)
	assert.Equal(t, int32(0), calls.Load())
}

func TestProfileNotFound(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, liveKey, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, fixture(t, "not_found.json"))
	})

	profile, err := client.Profile(t.Context(), carrierok.ProfileQuery{DOTNumber: "1"})
	assert.Nil(t, profile)
	require.ErrorIs(t, err, carrierok.ErrNotFound)
	assert.True(t, carrierok.IsNotFound(err))
	assertNoKey(t, err)
}

func TestProfileNotFoundStatus(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, liveKey, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, fixture(t, "error_404.json"))
	})

	_, err := client.Profile(t.Context(), carrierok.ProfileQuery{DOTNumber: "1"})
	assert.True(t, carrierok.IsNotFound(err))
	assert.False(t, carrierok.IsPaymentRequired(err))
	assertNoKey(t, err)
}

func TestProfilePaymentRequired(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, liveKey, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusPaymentRequired, fixture(t, "error_402.json"))
	})

	_, err := client.Profile(t.Context(), carrierok.ProfileQuery{DOTNumber: "818175"})
	assert.True(t, carrierok.IsPaymentRequired(err))
	assert.False(t, carrierok.IsNotFound(err))

	var apiErr *restx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "payment_failed", apiErr.Code)
	assert.Equal(t, "Payment failed for this account", apiErr.Message)
	assert.Equal(t, "Update the billing method in the CarrierOk dashboard", apiErr.Hint)
	assertNoKey(t, err)
}

func TestProfileUnauthorized(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, liveKey, func(w http.ResponseWriter, r *http.Request) {
		body, _ := sonic.Marshal(map[string]string{
			"error": "Invalid API key " + r.Header.Get("Authorization"),
		})
		writeJSON(w, http.StatusUnauthorized, body)
	})

	_, err := client.Profile(t.Context(), carrierok.ProfileQuery{DOTNumber: "818175"})
	assert.True(t, carrierok.IsUnauthorized(err))
	assertNoKey(t, err)
}

func TestProfileRateLimited(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newTestClient(t, liveKey, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		writeJSON(w, http.StatusTooManyRequests, fixture(t, "error_429.json"))
	})

	_, err := client.Profile(t.Context(), carrierok.ProfileQuery{DOTNumber: "818175"})
	assert.True(t, carrierok.IsRateLimited(err))
	assert.Equal(t, int32(1), calls.Load())
	assertNoKey(t, err)
}

func TestProfileRateLimitedThenRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newTestClient(t, liveKey, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			writeJSON(w, http.StatusTooManyRequests, fixture(t, "error_429.json"))
			return
		}
		writeJSON(w, http.StatusOK, fixture(t, "profile_265752.json"))
	}, carrierok.WithRetry(restx.RetryConfig{
		Enabled:        true,
		MaxAttempts:    2,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	}))

	profile, err := client.Profile(t.Context(), carrierok.ProfileQuery{DOTNumber: "818175"})
	require.NoError(t, err)
	assert.Equal(t, "FEDEX GROUND PACKAGE SYSTEM INC", profile.Identity.LegalName.Value())
	assert.Equal(t, int32(2), calls.Load())
}

func TestLimiterBucketsForLiveKey(t *testing.T) {
	t.Parallel()

	limiter := &recordingLimiter{}
	client := newTestClient(t, liveKey, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, fixture(t, "profile_265752.json"))
	}, carrierok.WithLimiter(limiter, "carrierok:org_1"))

	_, err := client.Profile(t.Context(), carrierok.ProfileQuery{DOTNumber: "818175"})
	require.NoError(t, err)

	buckets := limiter.recorded()
	require.Len(t, buckets, 1)
	assert.Equal(t, "carrierok:org_1:profile", buckets[0].Key)
	assert.Equal(t, 5, buckets[0].Limit)
	assert.Equal(t, time.Minute, buckets[0].Period)
	assert.Equal(t, 1, buckets[0].Burst)
}

func TestSandboxKeySkipsLimiter(t *testing.T) {
	t.Parallel()

	limiter := &recordingLimiter{}
	client := newTestClient(t, sandboxKey, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer "+sandboxKey, r.Header.Get("Authorization"))
		writeJSON(w, http.StatusOK, fixture(t, "profile_265752.json"))
	}, carrierok.WithLimiter(limiter, "carrierok:org_1"))

	assert.True(t, client.Sandbox())
	_, err := client.Profile(t.Context(), carrierok.ProfileQuery{DOTNumber: "818175"})
	require.NoError(t, err)
	assert.Empty(t, limiter.recorded())
}

func TestDefaultPolicies(t *testing.T) {
	t.Parallel()

	policies := carrierok.DefaultPolicies()
	require.Len(t, policies, len(carrierok.Endpoints()))

	expected := map[carrierok.Endpoint]struct {
		limit int
		burst int
	}{
		carrierok.EndpointProfile:          {limit: 5, burst: 1},
		carrierok.EndpointProfileLite:      {limit: 30, burst: 1},
		carrierok.EndpointProfileFMCSA:     {limit: 150, burst: 3},
		carrierok.EndpointAutocomplete:     {limit: 2000, burst: 34},
		carrierok.EndpointSearch:           {limit: 30, burst: 1},
		carrierok.EndpointMonitoringAdd:    {limit: 60, burst: 1},
		carrierok.EndpointMonitoringRemove: {limit: 60, burst: 1},
		carrierok.EndpointMonitoringList:   {limit: 120, burst: 2},
	}
	for endpoint, want := range expected {
		bucket, ok := policies[endpoint]
		require.True(t, ok, endpoint)
		assert.Empty(t, bucket.Key)
		assert.Equal(t, want.limit, bucket.Limit, endpoint)
		assert.Equal(t, want.burst, bucket.Burst, endpoint)
		assert.Equal(t, time.Minute, bucket.Period, endpoint)
	}

	assert.Equal(t, carrierok.Endpoint("monitoring-add"), carrierok.EndpointMonitoringAdd)
	assert.Equal(t, carrierok.Endpoint("profile-fmcsa"), carrierok.EndpointProfileFMCSA)
}

func TestProfileFMCSAAcceptsBareObject(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, liveKey, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/profile-fmcsa", r.URL.Path)
		assert.Equal(t, "docket_number=MC277621", r.URL.RawQuery)
		writeJSON(w, http.StatusOK, []byte(`{"dot_number":"818175","legal_name":"SANDBOX"}`))
	})

	profile, err := client.ProfileFMCSA(
		t.Context(),
		carrierok.FMCSAQuery{DocketNumber: "MC277621"},
	)
	require.NoError(t, err)
	assert.Equal(t, "SANDBOX", profile.Identity.LegalName.Value())
}

func TestSearch(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, liveKey, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/search", r.URL.Path)
		query := r.URL.Query()
		assert.Equal(t, "sandbox", query.Get("query"))
		assert.Equal(t, "Sandbox Freight", query.Get("company_name"))
		assert.Equal(t, "IL", query.Get("state"))
		assert.Equal(t, "25", query.Get("limit"))
		assert.Equal(t, "50", query.Get("offset"))
		assert.Len(t, query, 5)
		writeJSON(w, http.StatusOK, fixture(t, "search.json"))
	})

	result, err := client.Search(t.Context(), carrierok.SearchParams{
		Query:       "sandbox",
		CompanyName: "Sandbox Freight",
		State:       "il",
		Limit:       25,
		Offset:      50,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), result.TotalCount)
	require.Len(t, result.Items, 2)
	assert.Equal(t, "2233445", result.Items[1].Identity.DOTNumber.Value())

	_, err = client.Search(t.Context(), carrierok.SearchParams{})
	require.ErrorIs(t, err, carrierok.ErrInvalidQuery)
}

func TestAutocomplete(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, liveKey, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/autocomplete", r.URL.Path)
		assert.Equal(t, "limit=10&q=sand", r.URL.RawQuery)
		writeJSON(w, http.StatusOK, fixture(t, "autocomplete.json"))
	})

	suggestions, err := client.Autocomplete(t.Context(), "sand", 10)
	require.NoError(t, err)
	require.Len(t, suggestions, 2)
	assert.Equal(t, carrierok.Suggestion{
		DOTNumber: "818175",
		LegalName: "SANDBOX FREIGHT LINES INC",
		DBAName:   "SANDBOX FREIGHT",
		City:      "CHICAGO",
		State:     "IL",
	}, suggestions[0])
	assert.Empty(t, suggestions[1].DBAName)

	_, err = client.Autocomplete(t.Context(), "s", 5)
	require.ErrorIs(t, err, carrierok.ErrInvalidQuery)
	_, err = client.Autocomplete(t.Context(), "sand", 16)
	require.ErrorIs(t, err, carrierok.ErrInvalidQuery)
}

func TestAutocompleteAcceptsItemsEnvelope(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, liveKey, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, []byte(`{"items":[{"dot_number":1,"legal_name":"A"}]}`))
	})

	suggestions, err := client.Autocomplete(t.Context(), "ab", 0)
	require.NoError(t, err)
	require.Len(t, suggestions, 1)
	assert.Equal(t, "1", suggestions[0].DOTNumber)
}

func TestAddToMonitoringPartial(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, liveKey, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v2/monitoring/add", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.JSONEq(
			t,
			`{"profile_ids":["818175-MC277621","999999-MC000000"]}`,
			string(body),
		)
		writeJSON(w, http.StatusMultiStatus, fixture(t, "monitoring_add_207.json"))
	})

	result, err := client.AddToMonitoring(
		t.Context(),
		[]string{"818175-MC277621", " 999999-MC000000 ", "818175-MC277621", ""},
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusMultiStatus, result.StatusCode)
	assert.True(t, result.Partial)
	assert.Equal(t, int64(1), result.Added)
	assert.Equal(t, "partial", result.Status)
	require.Len(t, result.Failed, 1)
	assert.Equal(t, carrierok.MonitoringFailure{
		ProfileID: "999999-MC000000",
		Error:     "Profile not found",
	}, result.Failed[0])

	_, err = client.AddToMonitoring(t.Context(), []string{" "})
	require.ErrorIs(t, err, carrierok.ErrInvalidQuery)
}

func TestRemoveFromMonitoring(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, liveKey, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/monitoring/remove", r.URL.Path)
		writeJSON(w, http.StatusOK, []byte(`{"status":"ok","removed":1}`))
	})

	result, err := client.RemoveFromMonitoring(t.Context(), []string{"818175-MC277621"})
	require.NoError(t, err)
	assert.False(t, result.Partial)
	assert.Equal(t, int64(1), result.Removed)
	assert.Empty(t, result.Failed)
}

func TestListMonitoring(t *testing.T) {
	t.Parallel()

	yes := true
	no := false
	client := newTestClient(t, liveKey, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/monitoring/list", r.URL.Path)
		query := r.URL.Query()
		assert.Equal(t, "2", query.Get("page"))
		assert.Equal(t, "500", query.Get("pageSize"))
		assert.Equal(t, "true", query.Get("view_changes"))
		assert.Equal(t, "false", query.Get("view_changes_insurance"))
		assert.Equal(t, "true", query.Get("view_changes_safety"))
		assert.Equal(t, "last_changed_date", query.Get("date_type"))
		assert.Equal(t, "20260901", query.Get("date_min"))
		assert.Equal(t, "20260915", query.Get("date_max"))
		assert.Equal(t, "last_changed_date", query.Get("sortBy"))
		assert.Equal(t, "desc", query.Get("sortOrder"))
		assert.Equal(t, "818175-MC277621", query.Get("profile_id"))
		assert.False(t, query.Has("view_changes_fleet"))
		writeJSON(w, http.StatusOK, fixture(t, "monitoring_list.json"))
	})

	result, err := client.ListMonitoring(t.Context(), carrierok.MonitoringListParams{
		Page:                 2,
		PageSize:             500,
		ViewChanges:          &yes,
		ViewChangesInsurance: &no,
		ViewChangesSafety:    &yes,
		DateType:             "last_changed_date",
		DateMin:              time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		DateMax:              time.Date(2026, time.September, 15, 0, 0, 0, 0, time.UTC),
		SortBy:               "last_changed_date",
		SortOrder:            "DESC",
		ProfileID:            "818175-MC277621",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.TotalCount)
	require.Len(t, result.Profiles, 2)

	assert.Equal(t, "568253-MC277622", result.Profiles[0].ProfileID)
	assert.Equal(t, "818175-MC277621", result.Profiles[1].ProfileID)

	_, err = client.ListMonitoring(t.Context(), carrierok.MonitoringListParams{PageSize: 501})
	require.ErrorIs(t, err, carrierok.ErrInvalidQuery)
}
