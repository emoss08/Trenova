package carrierokconnector

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/carrierok"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	liveKey    = "sk_live_0123456789abcdefSECRET"
	sandboxKey = "sk_test_0123456789abcdefSECRET"
)

type callLog struct {
	mu    sync.Mutex
	calls []services.CarrierIntelCall
}

func (l *callLog) record(_ context.Context, call services.CarrierIntelCall) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, call)
}

func (l *callLog) all() []services.CarrierIntelCall {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]services.CarrierIntelCall(nil), l.calls...)
}

type denyingLimiter struct {
	calls atomic.Int32
}

func (l *denyingLimiter) Acquire(_ context.Context, bucket restx.Bucket) error {
	l.calls.Add(1)
	return &restx.RateLimitedError{RetryAfter: 7 * time.Second, Key: bucket.Key}
}

type keyRecordingLimiter struct {
	mu   sync.Mutex
	keys []string
}

func (l *keyRecordingLimiter) Acquire(_ context.Context, bucket restx.Bucket) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.keys = append(l.keys, bucket.Key)
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func testConnector(cfg *config.Config) *Connector {
	if cfg == nil {
		cfg = &config.Config{}
	}
	connector := New(cfg)
	connector.extraOptions = []carrierok.Option{carrierok.WithRetry(restx.RetryConfig{})}
	return connector
}

type boundClient struct {
	client *Client
	log    *callLog
}

func bindTestClient(
	t *testing.T,
	key string,
	limiter restx.Limiter,
	handler http.HandlerFunc,
) boundClient {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	log := &callLog{}
	bound, err := testConnector(nil).Bind(&services.CarrierIntelBindParams{
		Config: map[string]string{
			integration.ConfigKeyCarrierIntelAPIKey:  key,
			integration.ConfigKeyCarrierIntelBaseURL: server.URL + "/",
		},
		Limiter:  limiter,
		Recorder: log.record,
	})
	require.NoError(t, err)

	client, ok := bound.(*Client)
	require.True(t, ok)
	return boundClient{client: client, log: log}
}

func assertNoSecret(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "0123456789abcdefSECRET")
}

func TestConnector_DescriptorAndPriceBook(t *testing.T) {
	t.Parallel()

	connector := New(&config.Config{})
	assert.Equal(t, integration.TypeCarrierOK, connector.IntegrationType())

	descriptor := connector.Descriptor()
	assert.True(t, descriptor.Capabilities.Has(carrierintel.CapabilityNativeMonitoring))
	assert.True(t, descriptor.Capabilities.Has(carrierintel.CapabilityEquipmentLookup))
	assert.False(t, descriptor.Capabilities.Has(carrierintel.CapabilitySnapshotMonitoring))
	assert.Equal(t, carrierintel.AllSections(), descriptor.Sections)

	prices := connector.PriceBook()
	assert.Equal(
		t,
		carrierintel.BillingModelPerDOTMonth,
		prices.Price(carrierintel.EndpointProfileFull).Model,
	)
	assert.Equal(t, "3", prices.Price(carrierintel.EndpointProfileFull).UnitCost.String())
	assert.Equal(t, "0.5", prices.Price(carrierintel.EndpointProfileLite).UnitCost.String())
	assert.Equal(
		t,
		carrierintel.BillingModelPerMatch,
		prices.Price(carrierintel.EndpointProfileFMCSA).Model,
	)
	assert.Equal(t, "0.003", prices.Price(carrierintel.EndpointSearch).UnitCost.String())
	assert.Equal(
		t,
		carrierintel.BillingModelFree,
		prices.Price(carrierintel.EndpointMonitorRemove).Model,
	)
	assert.Equal(t, "3", prices.Price(carrierintel.EndpointEquipment).UnitCost.String())
	assert.Equal(t, "5", prices.MonitoringMonthlyCost(10).String())
}

func TestConnector_BindRejectsSandboxKeyWhenDisallowed(t *testing.T) {
	t.Parallel()

	disallowed := false
	cfg := &config.Config{}
	cfg.CarrierIntelligence.SandboxAllowed = &disallowed

	_, err := New(cfg).Bind(&services.CarrierIntelBindParams{
		Config: map[string]string{integration.ConfigKeyCarrierIntelAPIKey: sandboxKey},
	})
	require.Error(t, err)
	assert.Equal(t, "Sandbox API keys are not allowed in this environment", err.Error())

	err = New(cfg).TestConnection(t.Context(), map[string]string{
		integration.ConfigKeyCarrierIntelAPIKey: sandboxKey,
	})
	require.Error(t, err)
	assert.Equal(t, "Sandbox API keys are not allowed in this environment", err.Error())
}

func TestConnector_BindRejectsUnapprovedHost(t *testing.T) {
	t.Parallel()

	_, err := New(&config.Config{}).Bind(&services.CarrierIntelBindParams{
		Config: map[string]string{
			integration.ConfigKeyCarrierIntelAPIKey:  liveKey,
			integration.ConfigKeyCarrierIntelBaseURL: "https://attacker.example.com",
		},
	})
	require.Error(t, err)
	assert.Equal(t, "Base URL host is not an approved carrier intelligence host", err.Error())
	assertNoSecret(t, err)
}

func TestConnector_BindRequiresAPIKey(t *testing.T) {
	t.Parallel()

	_, err := New(&config.Config{}).Bind(&services.CarrierIntelBindParams{
		Config: map[string]string{},
	})
	require.Error(t, err)
	assert.Equal(t, "CarrierOk API key is required", err.Error())
}

func TestConnector_TestConnectionMapsErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		status  int
		body    []byte
		message string
	}{
		{name: "success", status: http.StatusOK, body: fixture(t, "autocomplete.json")},
		{
			name:    "unauthorized",
			status:  http.StatusUnauthorized,
			body:    []byte(`{"error":"bad key"}`),
			message: "CarrierOk rejected the API key",
		},
		{
			name:    "payment",
			status:  http.StatusPaymentRequired,
			body:    fixture(t, "error_402.json"),
			message: "CarrierOk reports the account's payment failed",
		},
		{
			name:    "server",
			status:  http.StatusBadGateway,
			body:    []byte(`{}`),
			message: "Could not reach CarrierOk",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/v2/autocomplete", r.URL.Path)
					assert.Equal(t, "Bearer "+liveKey, r.Header.Get("Authorization"))
					writeJSON(w, tc.status, tc.body)
				}),
			)
			t.Cleanup(server.Close)

			err := testConnector(nil).TestConnection(t.Context(), map[string]string{
				integration.ConfigKeyCarrierIntelAPIKey:  liveKey,
				integration.ConfigKeyCarrierIntelBaseURL: server.URL,
			})
			if tc.message == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tc.message, err.Error())
		})
	}
}

func TestClient_LookupFullRecordsOnce(t *testing.T) {
	t.Parallel()

	limiter := &keyRecordingLimiter{}
	bound := bindTestClient(t, liveKey, limiter, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/profile", r.URL.Path)
		assert.Equal(t, "dot_number=818175", r.URL.RawQuery)
		writeJSON(w, http.StatusOK, fixture(t, "profile_818175.json"))
	})

	assert.Equal(t, integration.TypeCarrierOK, bound.client.Provider())
	assert.False(t, bound.client.IsSandbox())

	result, err := bound.client.Lookup(t.Context(), &services.CarrierIntelLookupRequest{
		Identifier: services.CarrierIntelIdentifier{DOTNumber: " 818175 "},
		Depth:      carrierintel.LookupDepthFull,
	})
	require.NoError(t, err)
	assert.Equal(t, carrierintel.EndpointProfileFull, result.Endpoint)
	assert.Equal(t, carrierintel.LookupDepthFull, result.Depth)
	assert.Equal(t, "818175-MC277621", result.ProviderRef)
	assert.Equal(t, "818175", result.Profile.DOTNumber())
	assert.NotEmpty(t, result.Raw)
	require.NotNil(t, result.SourceAsOf)
	assert.Equal(t, unixDate(t, "2026-09-10"), *result.SourceAsOf)

	calls := bound.log.all()
	require.Len(t, calls, 1)
	assert.Equal(t, carrierintel.EndpointProfileFull, calls[0].Endpoint)
	assert.Equal(t, "818175", calls[0].DOTNumber)
	assert.Equal(t, http.StatusOK, calls[0].StatusCode)
	assert.True(t, calls[0].Found)
	assert.Equal(t, 1, calls[0].Units)
	require.NoError(t, calls[0].Err)

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	require.Len(t, limiter.keys, 1)
	assert.True(t, strings.HasPrefix(limiter.keys[0], "carrierok:"))
	assert.True(t, strings.HasSuffix(limiter.keys[0], ":profile"))
	assert.NotContains(t, limiter.keys[0], "SECRET")
	assert.Len(t, strings.Split(limiter.keys[0], ":")[1], 16)
}

func TestClient_LookupDepthSelectsEndpoint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		depth    carrierintel.LookupDepth
		id       services.CarrierIntelIdentifier
		path     string
		query    string
		endpoint carrierintel.Endpoint
	}{
		{
			name:     "lite by company",
			depth:    carrierintel.LookupDepthLite,
			id:       services.CarrierIntelIdentifier{Company: "Sandbox Freight"},
			path:     "/v2/profile-lite",
			query:    "company=Sandbox+Freight",
			endpoint: carrierintel.EndpointProfileLite,
		},
		{
			name:     "fmcsa by docket",
			depth:    carrierintel.LookupDepthFMCSA,
			id:       services.CarrierIntelIdentifier{DocketNumber: "MC277621"},
			path:     "/v2/profile-fmcsa",
			query:    "docket_number=MC277621",
			endpoint: carrierintel.EndpointProfileFMCSA,
		},
		{
			name:     "full prefers dot over docket",
			depth:    carrierintel.LookupDepthFull,
			id:       services.CarrierIntelIdentifier{DOTNumber: "818175", DocketNumber: "MC1"},
			path:     "/v2/profile",
			query:    "dot_number=818175",
			endpoint: carrierintel.EndpointProfileFull,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bound := bindTestClient(t, liveKey, nil, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.path, r.URL.Path)
				assert.Equal(t, tc.query, r.URL.RawQuery)
				writeJSON(w, http.StatusOK, fixture(t, "profile_818175.json"))
			})

			result, err := bound.client.Lookup(t.Context(), &services.CarrierIntelLookupRequest{
				Identifier: tc.id,
				Depth:      tc.depth,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.endpoint, result.Endpoint)
			assert.Equal(t, tc.depth, result.Depth)

			calls := bound.log.all()
			require.Len(t, calls, 1)
			assert.Equal(t, tc.endpoint, calls[0].Endpoint)
			assert.Equal(t, "818175", calls[0].DOTNumber)
		})
	}
}

func TestClient_LookupFMCSARequiresDOTOrDocket(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	bound := bindTestClient(t, liveKey, nil, func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	})

	_, err := bound.client.Lookup(t.Context(), &services.CarrierIntelLookupRequest{
		Identifier: services.CarrierIntelIdentifier{Company: "Acme"},
		Depth:      carrierintel.LookupDepthFMCSA,
	})
	assert.True(t, services.IsCarrierIntelErrorKind(err, services.CarrierIntelErrorInvalidRequest))
	assert.Equal(t, int32(0), hits.Load())
	assert.Empty(t, bound.log.all())
}

func TestClient_ErrorMapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		status  int
		body    []byte
		headers map[string]string
		kind    services.CarrierIntelErrorKind
		retry   time.Duration
	}{
		{
			name:   "unauthorized",
			status: http.StatusUnauthorized,
			body:   []byte(`{"error":"invalid key"}`),
			kind:   services.CarrierIntelErrorUnauthorized,
		},
		{
			name:   "forbidden",
			status: http.StatusForbidden,
			body:   []byte(`{"error":"no"}`),
			kind:   services.CarrierIntelErrorUnauthorized,
		},
		{
			name:   "payment",
			status: http.StatusPaymentRequired,
			body:   fixture(t, "error_402.json"),
			kind:   services.CarrierIntelErrorPaymentRequired,
		},
		{
			name:   "not found status",
			status: http.StatusNotFound,
			body:   []byte(`{"error":"nope"}`),
			kind:   services.CarrierIntelErrorNotFound,
		},
		{
			name:   "not found empty items",
			status: http.StatusOK,
			body:   fixture(t, "not_found.json"),
			kind:   services.CarrierIntelErrorNotFound,
		},
		{
			name:    "rate limited",
			status:  http.StatusTooManyRequests,
			body:    fixture(t, "error_429.json"),
			headers: map[string]string{"Retry-After": "12"},
			kind:    services.CarrierIntelErrorRateLimited,
			retry:   12 * time.Second,
		},
		{
			name:   "bad request",
			status: http.StatusUnprocessableEntity,
			body:   []byte(`{"error":"bad"}`),
			kind:   services.CarrierIntelErrorInvalidRequest,
		},
		{
			name:   "server error",
			status: http.StatusServiceUnavailable,
			body:   []byte(`{}`),
			kind:   services.CarrierIntelErrorUnavailable,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bound := bindTestClient(t, liveKey, nil, func(w http.ResponseWriter, _ *http.Request) {
				for key, value := range tc.headers {
					w.Header().Set(key, value)
				}
				writeJSON(w, tc.status, tc.body)
			})

			result, err := bound.client.Lookup(t.Context(), &services.CarrierIntelLookupRequest{
				Identifier: services.CarrierIntelIdentifier{DOTNumber: "818175"},
				Depth:      carrierintel.LookupDepthFull,
			})
			assert.Nil(t, result)
			assertNoSecret(t, err)

			var providerErr *services.CarrierIntelProviderError
			require.ErrorAs(t, err, &providerErr)
			assert.Equal(t, tc.kind, providerErr.Kind)
			assert.Equal(t, integration.TypeCarrierOK, providerErr.Provider)
			assert.Equal(t, tc.retry, providerErr.RetryAfter)

			calls := bound.log.all()
			require.Len(t, calls, 1)
			assert.False(t, calls[0].Found)
			assert.Equal(t, tc.status, calls[0].StatusCode)
			assert.Equal(t, err, calls[0].Err)
		})
	}
}

func TestClient_LimiterDenialPassesThrough(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	limiter := &denyingLimiter{}
	bound := bindTestClient(t, liveKey, limiter, func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		writeJSON(w, http.StatusOK, fixture(t, "profile_818175.json"))
	})

	_, err := bound.client.Lookup(t.Context(), &services.CarrierIntelLookupRequest{
		Identifier: services.CarrierIntelIdentifier{DOTNumber: "818175"},
		Depth:      carrierintel.LookupDepthFull,
	})

	var limited *restx.RateLimitedError
	require.ErrorAs(t, err, &limited)
	assert.Equal(t, 7*time.Second, limited.RetryAfter)
	_, isProviderErr := services.CarrierIntelErrorKindOf(err)
	assert.False(t, isProviderErr)
	assert.Equal(t, int32(0), hits.Load())
	assert.Equal(t, int32(1), limiter.calls.Load())

	calls := bound.log.all()
	require.Len(t, calls, 1)
	assert.Equal(t, 0, calls[0].StatusCode)
	assert.ErrorAs(t, calls[0].Err, &limited)
}

func TestClient_SandboxSkipsLimiter(t *testing.T) {
	t.Parallel()

	limiter := &denyingLimiter{}
	bound := bindTestClient(t, sandboxKey, limiter, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, fixture(t, "profile_818175.json"))
	})
	assert.True(t, bound.client.IsSandbox())

	_, err := bound.client.Lookup(t.Context(), &services.CarrierIntelLookupRequest{
		Identifier: services.CarrierIntelIdentifier{DOTNumber: "818175"},
	})
	require.NoError(t, err)
	assert.Equal(t, int32(0), limiter.calls.Load())
}

func TestClient_SearchAndAutocomplete(t *testing.T) {
	t.Parallel()

	bound := bindTestClient(t, liveKey, nil, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/search":
			assert.Equal(t, "Sandbox", r.URL.Query().Get("query"))
			assert.Equal(t, "IL", r.URL.Query().Get("state"))
			assert.Equal(t, "10", r.URL.Query().Get("limit"))
			writeJSON(w, http.StatusOK, fixture(t, "search.json"))
		case "/v2/autocomplete":
			assert.Equal(t, "15", r.URL.Query().Get("limit"))
			writeJSON(w, http.StatusOK, fixture(t, "autocomplete.json"))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})

	search, err := bound.client.Search(t.Context(), &services.CarrierIntelSearchRequest{
		Query: "Sandbox",
		State: "il",
		Limit: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 42, search.Total)
	require.Len(t, search.Items, 2)
	assert.Equal(t, "818175", search.Items[0].ProviderRef)
	assert.Equal(t, "SANDBOX FREIGHT LINES INC", search.Items[0].Profile.Identity.LegalName)
	assert.Equal(t, "2233445", search.Items[1].Profile.DOTNumber())

	suggestions, err := bound.client.Autocomplete(t.Context(), "sandbox", 50)
	require.NoError(t, err)
	require.Len(t, suggestions, 2)
	assert.Equal(t, "818175", suggestions[0].DOTNumber)
	assert.Equal(t, "CHICAGO", suggestions[0].City)

	calls := bound.log.all()
	require.Len(t, calls, 2)
	assert.Equal(t, carrierintel.EndpointSearch, calls[0].Endpoint)
	assert.Equal(t, carrierintel.EndpointAutocomplete, calls[1].Endpoint)

	_, err = bound.client.Autocomplete(t.Context(), "a", 5)
	assert.True(t, services.IsCarrierIntelErrorKind(err, services.CarrierIntelErrorInvalidRequest))
}

func TestClient_MonitoringRef(t *testing.T) {
	t.Parallel()

	client := &Client{}
	profile := &carrierintel.Profile{Identity: &carrierintel.Identity{
		DOTNumber:    "818175",
		DocketPrefix: "MC",
		DocketNumber: "277621",
	}}

	assert.Equal(t, "818175-MC277621", client.MonitoringRef(profile, "", ""))
	assert.Equal(t, "818175-MC277621", client.MonitoringRef(nil, "818175", "277621"))
	assert.Equal(t, "818175-FF12", client.MonitoringRef(nil, "818175", "FF12"))
	assert.Equal(t, "818175", client.MonitoringRef(nil, "818175", ""))
	assert.Equal(t, "42", client.MonitoringRef(
		&carrierintel.Profile{Identity: &carrierintel.Identity{DOTNumber: "42"}}, "", "",
	))
	assert.Empty(t, client.MonitoringRef(nil, "", "MC1"))
}

func TestClient_EnrollSplitsFailures(t *testing.T) {
	t.Parallel()

	bound := bindTestClient(t, liveKey, nil, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v2/monitoring/add", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		var payload struct {
			ProfileIDs []string `json:"profile_ids"`
		}
		assert.NoError(t, sonic.Unmarshal(body, &payload))
		assert.Equal(t, []string{"818175-MC277621", "999999-MC000000"}, payload.ProfileIDs)
		writeJSON(w, http.StatusMultiStatus, fixture(t, "monitoring_add_207.json"))
	})

	result, err := bound.client.Enroll(t.Context(), []string{
		"818175-MC277621", " 999999-MC000000 ", "818175-MC277621", "",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"818175-MC277621"}, result.Succeeded)
	require.Len(t, result.Failed, 1)
	assert.Equal(t, "999999-MC000000", result.Failed[0].ProviderRef)
	assert.Equal(t, "Profile not found", result.Failed[0].Message)

	calls := bound.log.all()
	require.Len(t, calls, 1)
	assert.Equal(t, carrierintel.EndpointMonitorAdd, calls[0].Endpoint)
	assert.Empty(t, calls[0].DOTNumber)
	assert.Equal(t, 1, calls[0].Units)
	assert.True(t, calls[0].Found)
	assert.Equal(t, http.StatusMultiStatus, calls[0].StatusCode)

	empty, err := bound.client.Unenroll(t.Context(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty.Succeeded)
	assert.Len(t, bound.log.all(), 1)
}

func TestClient_ListChanges(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC).Unix()
	until := time.Date(2026, 9, 12, 18, 0, 0, 0, time.UTC).Unix()

	bound := bindTestClient(t, liveKey, nil, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/monitoring/list", r.URL.Path)
		query := r.URL.Query()
		assert.Equal(t, "true", query.Get("view_changes"))
		assert.Equal(t, "last_changed_date", query.Get("date_type"))
		assert.Equal(t, "20260908", query.Get("date_min"))
		assert.Equal(t, "20260912", query.Get("date_max"))
		assert.Equal(t, "1", query.Get("page"))
		assert.Equal(t, "500", query.Get("pageSize"))
		writeJSON(w, http.StatusOK, fixture(t, "monitoring_list.json"))
	})

	page, err := bound.client.ListChanges(t.Context(), &services.CarrierIntelChangeFeedRequest{
		Since:    since,
		Until:    until,
		PageSize: 900,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, page.Total)
	assert.Equal(t, 1, page.Page)
	assert.Equal(t, 500, page.PageSize)
	assert.False(t, page.HasMore)
	require.Len(t, page.Items, 2)

	first := page.Items[0]
	assert.Equal(t, "568253-MC277622", first.ProviderRef)
	assert.Equal(t, "568253", first.DOTNumber)
	require.Len(t, first.Changes, 1)
	assert.Equal(t, "authority_common", first.Changes[0].VendorField)
	assert.Equal(t, "authority.common.status", first.Changes[0].Path)
	assert.Equal(t, carrierintel.SectionAuthority, first.Changes[0].Section)
	assert.Equal(t, "ACTIVE", first.Changes[0].Prior)
	assert.Equal(t, "INACTIVE", first.Changes[0].Current)
	require.NotNil(t, first.ChangedAt)
	assert.Equal(t, unixDate(t, "2026-09-10"), *first.ChangedAt)

	second := page.Items[1]
	assert.Equal(t, "818175-MC277621", second.ProviderRef)
	paths := make([]string, 0, len(second.Changes))
	for _, change := range second.Changes {
		paths = append(paths, change.Path)
	}
	assert.ElementsMatch(t, []string{"insurance.bipdOnFile", "safety.rating"}, paths)
	require.NotNil(t, second.ChangedAt)
	assert.Equal(t, unixDate(t, "2026-09-12"), *second.ChangedAt)

	calls := bound.log.all()
	require.Len(t, calls, 1)
	assert.Equal(t, carrierintel.EndpointMonitorList, calls[0].Endpoint)
}

func TestClient_ListWatchlist(t *testing.T) {
	t.Parallel()

	bound := bindTestClient(t, liveKey, nil, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("view_changes"))
		assert.Equal(t, "1", r.URL.Query().Get("pageSize"))
		writeJSON(w, http.StatusOK, fixture(t, "monitoring_list.json"))
	})

	page, err := bound.client.ListWatchlist(t.Context(), 1, 1)
	require.NoError(t, err)
	assert.Equal(t, []string{"568253-MC277622", "818175-MC277621"}, page.ProviderRefs)
	assert.Equal(t, 2, page.Total)
	assert.True(t, page.HasMore)
}

func TestClient_FindByEquipment(t *testing.T) {
	t.Parallel()

	bound := bindTestClient(t, liveKey, nil, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/profile", r.URL.Path)
		switch r.URL.Query().Get("unit_number") {
		case "101":
			assert.Equal(t, "TRUCK", r.URL.Query().Get("unit_type"))
			writeJSON(w, http.StatusOK, fixture(t, "profile_818175.json"))
		default:
			writeJSON(w, http.StatusOK, fixture(t, "not_found.json"))
		}
	})

	result, err := bound.client.FindByEquipment(t.Context(), &services.CarrierIntelEquipmentRequest{
		UnitNumber: "101",
		UnitType:   carrierintel.UnitTypeTractor,
	})
	require.NoError(t, err)
	require.Len(t, result.Matches, 1)
	match := result.Matches[0]
	assert.Equal(t, "818175", match.DOTNumber)
	assert.Equal(t, "SANDBOX FREIGHT LINES INC", match.LegalName)
	require.NotNil(t, match.Unit)
	assert.Equal(t, "1XKYD49X0JJ123456", match.Unit.VIN)
	assert.True(t, strings.HasPrefix(string(result.Raw), "[{"))

	var raws []map[string]any
	require.NoError(t, sonic.Unmarshal(result.Raw, &raws))
	assert.Len(t, raws, 1)

	_, err = bound.client.FindByEquipment(t.Context(), &services.CarrierIntelEquipmentRequest{
		UnitNumber: "999",
	})
	assert.True(t, services.IsCarrierIntelErrorKind(err, services.CarrierIntelErrorNotFound))

	calls := bound.log.all()
	require.Len(t, calls, 2)
	assert.Equal(t, carrierintel.EndpointEquipment, calls[0].Endpoint)
	assert.Equal(t, "818175", calls[0].DOTNumber)
	assert.True(t, calls[0].Found)
	assert.False(t, calls[1].Found)

	_, err = bound.client.FindByEquipment(t.Context(), &services.CarrierIntelEquipmentRequest{})
	assert.True(t, services.IsCarrierIntelErrorKind(err, services.CarrierIntelErrorInvalidRequest))
}

func TestMatchingUnit_ByVINAndPlate(t *testing.T) {
	t.Parallel()

	equipment := normalizeProfile(fixtureProfile(t)).Equipment

	byVIN := matchingUnit(
		equipment,
		&services.CarrierIntelEquipmentRequest{VIN: "1grAA0621kb700001"},
	)
	require.NotNil(t, byVIN)
	assert.Equal(t, carrierintel.UnitTypeTrailer, byVIN.UnitType)

	byPlate := matchingUnit(equipment, &services.CarrierIntelEquipmentRequest{
		PlateNumber: "p-123456",
		PlateState:  "il",
	})
	require.NotNil(t, byPlate)
	assert.Equal(t, "101", byPlate.UnitNumber)

	assert.Nil(t, matchingUnit(equipment, &services.CarrierIntelEquipmentRequest{
		PlateNumber: "P123456",
		PlateState:  "TX",
	}))
}
