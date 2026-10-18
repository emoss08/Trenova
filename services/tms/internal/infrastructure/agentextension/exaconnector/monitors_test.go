package exaconnector

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/exa"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const watchSecret = "whsec_watch"

var exaSettings = map[string]string{agentextension.ConfigKeyAPIKey: "exa-key"}

func serveMonitors(t *testing.T, handler http.HandlerFunc) *Connector {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &Connector{httpClient: server.Client(), baseURL: server.URL}
}

func watchRequest() *agentextensionports.WebWatchCreateRequest {
	return &agentextensionports.WebWatchCreateRequest{
		Settings:       exaSettings,
		Name:           "HOS rule watch",
		Query:          "FMCSA hours of service rule change",
		IncludeDomains: []string{"fmcsa.dot.gov", "federalregister.gov"},
		ExcludeDomains: []string{"spam.example.com"},
		Cadence:        agentextension.WebWatchCadenceDaily,
		WebhookURL:     "https://hooks.trenova.example/agent-extensions/web-watches/aww_1",
		Metadata:       map[string]string{"watchId": "aww_1", "organizationId": "org_1"},
	}
}

func signed(secret string, at time.Time, body string) string {
	stamp := strconv.FormatInt(at.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(stamp + "." + body))

	return "t=" + stamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestCreateWatchSendsTheWatchAsAMonitor(t *testing.T) {
	t.Parallel()

	var received map[string]any
	connector := serveMonitors(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/monitors", r.URL.Path)
		assert.Equal(t, "exa-key", r.Header.Get("x-api-key"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, sonic.Unmarshal(body, &received))

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": "mon_123", "status": "active",
			"nextRunAt": "2026-10-18T12:00:00Z", "webhookSecret": "whsec_abc"}`))
	})

	created, err := connector.CreateWatch(t.Context(), watchRequest())
	require.NoError(t, err)

	assert.Equal(t, &agentextensionports.WebWatchCreated{
		VendorID: "mon_123",
		Secret:   "whsec_abc",
	}, created)
	assert.Equal(t, map[string]any{
		"name": "HOS rule watch",
		"search": map[string]any{
			"query":          "FMCSA hours of service rule change",
			"numResults":     float64(agentextension.WebWatchResultsPerRun),
			"includeDomains": []any{"fmcsa.dot.gov", "federalregister.gov"},
			"excludeDomains": []any{"spam.example.com"},
			"contents": map[string]any{
				"highlights": map[string]any{
					"query":         "FMCSA hours of service rule change",
					"maxCharacters": float64(600),
				},
			},
		},
		"trigger":  map[string]any{"type": "interval", "period": "1d"},
		"metadata": map[string]any{"watchId": "aww_1", "organizationId": "org_1"},
		"webhook": map[string]any{
			"url":    "https://hooks.trenova.example/agent-extensions/web-watches/aww_1",
			"events": []any{"monitor.run.completed"},
		},
	}, received)
}

func TestCreateWatchMapsEachCadenceToAnExaPeriod(t *testing.T) {
	t.Parallel()

	for cadence, period := range map[agentextension.WebWatchCadence]string{
		agentextension.WebWatchCadenceSixHours: "6h",
		agentextension.WebWatchCadenceDaily:    "1d",
		agentextension.WebWatchCadenceWeekly:   "7d",
	} {
		t.Run(string(cadence), func(t *testing.T) {
			t.Parallel()

			var received struct {
				Trigger exa.MonitorTrigger `json:"trigger"`
			}
			connector := serveMonitors(t, func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.NoError(t, sonic.Unmarshal(body, &received))
				_, _ = w.Write([]byte(`{"id": "mon_1", "webhookSecret": "whsec_1"}`))
			})

			req := watchRequest()
			req.Cadence = cadence
			_, err := connector.CreateWatch(t.Context(), req)
			require.NoError(t, err)

			assert.Equal(t, exa.MonitorTrigger{Type: "interval", Period: period}, received.Trigger)
		})
	}
}

func TestCreateWatchLeavesOutDomainsAndMetadataItWasNotGiven(t *testing.T) {
	t.Parallel()

	var received map[string]any
	connector := serveMonitors(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, sonic.Unmarshal(body, &received))
		_, _ = w.Write([]byte(`{"id": "mon_1", "webhookSecret": "whsec_1"}`))
	})

	req := watchRequest()
	req.Name = ""
	req.IncludeDomains = nil
	req.ExcludeDomains = nil
	req.Metadata = nil
	_, err := connector.CreateWatch(t.Context(), req)
	require.NoError(t, err)

	assert.NotContains(t, received, "name")
	assert.NotContains(t, received, "metadata")
	search, ok := received["search"].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, search, "includeDomains")
	assert.NotContains(t, search, "excludeDomains")
}

func TestCreateWatchRefusesAnUnknownCadenceWithoutCalling(t *testing.T) {
	t.Parallel()

	connector := serveMonitors(t, func(http.ResponseWriter, *http.Request) {
		t.Error("a watch without an Exa period must not reach Exa")
	})

	req := watchRequest()
	req.Cadence = agentextension.WebWatchCadence("Hourly")
	created, err := connector.CreateWatch(t.Context(), req)
	require.ErrorIs(t, err, exa.ErrInvalidRequest)
	assert.Nil(t, created)
}

func TestCreateWatchRefusesSettingsItCannotUse(t *testing.T) {
	t.Parallel()

	connector := serveMonitors(t, func(http.ResponseWriter, *http.Request) {
		t.Error("unusable settings must not reach Exa")
	})

	req := watchRequest()
	req.Settings = map[string]string{}
	created, err := connector.CreateWatch(t.Context(), req)
	require.ErrorIs(t, err, exa.ErrAPIKeyRequired)
	assert.Nil(t, created)

	req.Settings = map[string]string{
		agentextension.ConfigKeyAPIKey:     "exa-key",
		agentextension.ConfigKeySearchType: "bogus",
	}
	created, err = connector.CreateWatch(t.Context(), req)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Nil(t, created)
}

func TestCreateWatchIsNeverRetried(t *testing.T) {
	t.Parallel()

	for _, status := range []int{
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusTooManyRequests,
	} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32
			connector := serveMonitors(t, func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error": "try again"}`))
			})

			created, err := connector.CreateWatch(t.Context(), watchRequest())
			require.Error(t, err)
			assert.True(t, restx.IsStatus(err, status))
			assert.Nil(t, created)
			assert.Equal(t, int32(1), calls.Load())
		})
	}
}

func TestDeleteWatchDeletesTheMonitor(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	connector := serveMonitors(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/monitors/mon_123", r.URL.Path)
		assert.Equal(t, "exa-key", r.Header.Get("x-api-key"))
		_, _ = w.Write([]byte(`{"id": "mon_123"}`))
	})

	require.NoError(t, connector.DeleteWatch(t.Context(), exaSettings, "mon_123"))
	assert.Equal(t, int32(1), calls.Load())
}

func TestDeleteWatchTreatsAMonitorAlreadyGoneAsDeleted(t *testing.T) {
	t.Parallel()

	connector := serveMonitors(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Monitor not found"}`))
	})

	require.NoError(t, connector.DeleteWatch(t.Context(), exaSettings, "mon_gone"))
}

func TestDeleteWatchReportsARefusedKey(t *testing.T) {
	t.Parallel()

	connector := serveMonitors(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "Invalid API key", "tag": "INVALID_API_KEY"}`))
	})

	err := connector.DeleteWatch(t.Context(), exaSettings, "mon_123")
	require.Error(t, err)
	assert.True(t, exa.IsUnauthorized(err))
}

func TestWatchStateReportsOnlyADisabledMonitorAsDisabled(t *testing.T) {
	t.Parallel()

	for status, disabled := range map[string]bool{
		exa.MonitorStatusActive:   false,
		exa.MonitorStatusPaused:   false,
		exa.MonitorStatusDisabled: true,
	} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()

			connector := serveMonitors(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/monitors/mon_123", r.URL.Path)
				assert.Equal(t, "exa-key", r.Header.Get("x-api-key"))
				_, _ = w.Write([]byte(`{"id": "mon_123", "status": "` + status + `"}`))
			})

			state, err := connector.WatchState(t.Context(), exaSettings, "mon_123")
			require.NoError(t, err)
			assert.Equal(t, &agentextensionports.WebWatchVendorState{Disabled: disabled}, state)
		})
	}
}

func TestWatchStateReportsAFailedLookup(t *testing.T) {
	t.Parallel()

	connector := serveMonitors(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Monitor not found"}`))
	})

	state, err := connector.WatchState(t.Context(), exaSettings, "mon_gone")
	require.Error(t, err)
	assert.True(t, restx.IsStatus(err, http.StatusNotFound))
	assert.Nil(t, state)
}

func TestWatchRunsMapsEveryRunStatus(t *testing.T) {
	t.Parallel()

	connector := serveMonitors(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/monitors/mon_123/runs", r.URL.Path)
		assert.Equal(t, "limit=20", r.URL.RawQuery)
		assert.Equal(t, "exa-key", r.Header.Get("x-api-key"))

		_, _ = w.Write([]byte(`{"data": [
			{"id": "run_pending", "status": "pending", "output": null,
				"createdAt": "2026-10-17T12:00:00Z"},
			{"id": "run_running", "status": "running", "output": null,
				"createdAt": "2026-10-17T11:00:00Z"},
			{"id": "run_completed", "status": "completed",
				"output": {"results": [], "content": null},
				"completedAt": "2026-10-17T10:02:00Z", "createdAt": "2026-10-17T10:00:00Z"},
			{"id": "run_failed", "status": "failed", "output": null,
				"failReason": " Search timed out ",
				"completedAt": "2026-10-17T09:01:00Z", "createdAt": "2026-10-17T09:00:00Z"},
			{"id": "run_cancelled", "status": "cancelled", "output": null,
				"createdAt": "2026-10-17T08:00:00Z"},
			{"id": "run_unknown", "status": "queued", "output": null,
				"createdAt": "2026-10-17T07:00:00Z"}
		], "hasMore": false, "nextCursor": null}`))
	})

	runs, err := connector.WatchRuns(t.Context(), &agentextensionports.WebWatchRunsRequest{
		Settings: exaSettings,
		VendorID: "mon_123",
	})
	require.NoError(t, err)

	at := func(hour, minute int) int64 {
		return time.Date(2026, 10, 17, hour, minute, 0, 0, time.UTC).Unix()
	}
	expected := []struct {
		id        string
		phase     agentextensionports.WebWatchRunPhase
		failure   string
		createdAt int64
		settledAt int64
		cost      string
	}{
		{"run_pending", agentextensionports.WebWatchRunPending, "", at(12, 0), at(12, 0), "0"},
		{"run_running", agentextensionports.WebWatchRunPending, "", at(11, 0), at(11, 0), "0"},
		{
			"run_completed", agentextensionports.WebWatchRunCompleted, "",
			at(10, 0), at(10, 2), "0.015",
		},
		{
			"run_failed", agentextensionports.WebWatchRunFailed, "Search timed out",
			at(9, 0), at(9, 1), "0",
		},
		{"run_cancelled", agentextensionports.WebWatchRunCancelled, "", at(8, 0), at(8, 0), "0"},
		{"run_unknown", agentextensionports.WebWatchRunPending, "", at(7, 0), at(7, 0), "0"},
	}

	require.Len(t, runs, len(expected))
	for idx, want := range expected {
		run := runs[idx]
		assert.Equal(t, want.id, run.ID)
		assert.Equal(t, want.phase, run.Phase, want.id)
		assert.Equal(t, want.failure, run.Failure, want.id)
		assert.Equal(t, want.createdAt, run.CreatedAt, want.id)
		assert.Equal(t, want.settledAt, run.SettledAt, want.id)
		assert.Equal(t, want.cost, run.CostUSD.String(), want.id)
		assert.Empty(t, run.Summary, want.id)
		assert.Empty(t, run.Results, want.id)
	}
	assert.True(t, agentextension.ExaMonitorRunCostUSD.Equal(runs[2].CostUSD))
}

func TestWatchRunsReadsACompletedRunInFull(t *testing.T) {
	t.Parallel()

	var fetched atomic.Int32
	connector := serveMonitors(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/monitors/mon_123/runs":
			_, _ = w.Write([]byte(`{"data": [
				{"id": "run_2", "status": "completed", "output": null,
					"completedAt": "2026-10-17T12:03:00Z", "createdAt": "2026-10-17T12:00:00Z"},
				{"id": "run_1", "status": "failed", "output": null,
					"createdAt": "2026-10-16T12:00:00Z"}
			], "hasMore": false}`))
		case "/monitors/mon_123/runs/run_2":
			fetched.Add(1)
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "exa-key", r.Header.Get("x-api-key"))
			_, _ = w.Write([]byte(`{
				"id": "run_2",
				"monitorId": "mon_123",
				"status": "completed",
				"output": {
					"results": [
						{
							"url": " https://www.fmcsa.dot.gov/hos-pilot ",
							"title": " Split duty period pilot ",
							"publishedDate": " 2026-10-16T00:00:00.000Z ",
							"highlights": ["FMCSA proposes a pilot.", "Comments close in 60 days."]
						},
						{"url": "https://www.federalregister.gov/d/2026-1", "title": "Notice",
							"publishedDate": "", "highlights": []}
					],
					"content": "  FMCSA proposed a split duty period pilot.  "
				},
				"completedAt": "2026-10-17T12:03:00Z",
				"createdAt": "2026-10-17T12:00:00Z"
			}`))
		default:
			t.Errorf("unexpected request to %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	runs, err := connector.WatchRuns(t.Context(), &agentextensionports.WebWatchRunsRequest{
		Settings: exaSettings,
		VendorID: "mon_123",
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), fetched.Load())

	require.Len(t, runs, 2)
	completed := runs[0]
	assert.Equal(t, "run_2", completed.ID)
	assert.Equal(t, agentextensionports.WebWatchRunCompleted, completed.Phase)
	assert.Equal(t, "FMCSA proposed a split duty period pilot.", completed.Summary)
	assert.Equal(t, []agentextension.WebWatchResult{
		{
			Title:         "Split duty period pilot",
			URL:           "https://www.fmcsa.dot.gov/hos-pilot",
			PublishedDate: "2026-10-16T00:00:00.000Z",
			Excerpt:       "FMCSA proposes a pilot. … Comments close in 60 days.",
		},
		{
			Title: "Notice",
			URL:   "https://www.federalregister.gov/d/2026-1",
		},
	}, completed.Results)
	assert.Equal(t, time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC).Unix(), completed.CreatedAt)
	assert.Equal(t, time.Date(2026, 10, 17, 12, 3, 0, 0, time.UTC).Unix(), completed.SettledAt)
	assert.Equal(t, "0.015", completed.CostUSD.String())

	failed := runs[1]
	assert.Equal(t, "run_1", failed.ID)
	assert.Equal(t, agentextensionports.WebWatchRunFailed, failed.Phase)
	assert.Nil(t, failed.Results)
	assert.Equal(t, "0", failed.CostUSD.String())
}

func TestWatchRunsReportsARunItCouldNotRead(t *testing.T) {
	t.Parallel()

	connector := serveMonitors(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/monitors/mon_123/runs" {
			_, _ = w.Write([]byte(`{"data": [{"id": "run_1", "status": "completed",
				"output": null, "createdAt": "2026-10-17T12:00:00Z"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Run not found"}`))
	})

	runs, err := connector.WatchRuns(t.Context(), &agentextensionports.WebWatchRunsRequest{
		Settings: exaSettings,
		VendorID: "mon_123",
	})
	require.Error(t, err)
	assert.True(t, restx.IsStatus(err, http.StatusNotFound))
	assert.Nil(t, runs)
}

func TestWatchRunsReportsAFailedList(t *testing.T) {
	t.Parallel()

	connector := serveMonitors(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error": "You are out of credits", "tag": "NO_MORE_CREDITS"}`))
	})

	runs, err := connector.WatchRuns(t.Context(), &agentextensionports.WebWatchRunsRequest{
		Settings: exaSettings,
		VendorID: "mon_123",
	})
	require.Error(t, err)
	assert.True(t, exa.IsOutOfCredits(err))
	assert.Nil(t, runs)
}

func TestWatchRunsIsEmptyForAMonitorThatHasNotRun(t *testing.T) {
	t.Parallel()

	connector := serveMonitors(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data": [], "hasMore": false, "nextCursor": null}`))
	})

	runs, err := connector.WatchRuns(t.Context(), &agentextensionports.WebWatchRunsRequest{
		Settings: exaSettings,
		VendorID: "mon_123",
	})
	require.NoError(t, err)
	assert.NotNil(t, runs)
	assert.Empty(t, runs)
}

func TestVerifyDeliveryChecksExasSignature(t *testing.T) {
	t.Parallel()

	connector := New()
	body := `{"type":"monitor.run.completed","data":{"id":"run_1","monitorId":"mon_123"}}`
	arrived := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

	delivery := func(header string, at time.Time) *agentextensionports.WebWatchDelivery {
		headers := http.Header{}
		if header != "" {
			headers.Set("exa-signature", header)
		}
		return &agentextensionports.WebWatchDelivery{Header: headers, Body: []byte(body), At: at}
	}

	require.NoError(t, connector.VerifyDelivery(
		delivery(signed(watchSecret, arrived, body), arrived), watchSecret))
	require.NoError(t, connector.VerifyDelivery(
		delivery(signed(watchSecret, arrived, body), arrived.Add(exa.SignatureTolerance)),
		watchSecret))

	assert.ErrorIs(t, connector.VerifyDelivery(
		delivery(signed("whsec_other", arrived, body), arrived), watchSecret),
		exa.ErrSignatureInvalid)
	assert.ErrorIs(t, connector.VerifyDelivery(
		delivery(signed(watchSecret, arrived, body+" "), arrived), watchSecret),
		exa.ErrSignatureInvalid)
	assert.ErrorIs(t, connector.VerifyDelivery(delivery("", arrived), watchSecret),
		exa.ErrSignatureInvalid)
	assert.ErrorIs(t, connector.VerifyDelivery(
		delivery(signed(watchSecret, arrived, body), arrived), ""),
		exa.ErrSignatureInvalid)
	assert.ErrorIs(t, connector.VerifyDelivery(
		delivery(signed(watchSecret, arrived, body),
			arrived.Add(exa.SignatureTolerance+time.Second)), watchSecret),
		exa.ErrSignatureInvalid)
}

func TestVerifyDeliveryJudgesAnUntimedDeliveryAgainstNow(t *testing.T) {
	t.Parallel()

	connector := New()
	body := `{"type":"monitor.run.completed"}`
	delivery := func(header string) *agentextensionports.WebWatchDelivery {
		headers := http.Header{}
		headers.Set(exa.SignatureHeader, header)
		return &agentextensionports.WebWatchDelivery{Header: headers, Body: []byte(body)}
	}

	require.NoError(t, connector.VerifyDelivery(
		delivery(signed(watchSecret, time.Now(), body)), watchSecret))
	assert.ErrorIs(t, connector.VerifyDelivery(
		delivery(signed(watchSecret, time.Now().Add(-time.Hour), body)), watchSecret),
		exa.ErrSignatureInvalid)
}

func TestWatchRunsDoesNotReadARunAlreadyCountedAgain(t *testing.T) {
	t.Parallel()

	var fetched atomic.Int32
	connector := serveMonitors(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/monitors/mon_123/runs":
			_, _ = w.Write([]byte(`{"data": [
				{"id": "run_2", "status": "completed", "output": null,
					"completedAt": "2026-10-17T12:03:00Z", "createdAt": "2026-10-17T12:00:00Z"}
			], "hasMore": false}`))
		default:
			fetched.Add(1)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	counted := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC).Unix()
	runs, err := connector.WatchRuns(t.Context(), &agentextensionports.WebWatchRunsRequest{
		Settings: exaSettings,
		VendorID: "mon_123",
		After:    counted,
	})
	require.NoError(t, err)

	assert.Zero(t, fetched.Load(), "the watch already kept what that run found")
	require.Len(t, runs, 1)
	assert.Equal(t, counted, runs[0].CreatedAt)
}
