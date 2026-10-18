package exa_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/exa"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	webhookSecret = "whsec_test"
	deliveryBody  = `{"type":"monitor.run.completed","data":{"id":"run_1"}}`
	deliveryUnix  = int64(1792238400)
	deliveryMAC   = "5ec6770d775ad9847334c71355fe9a5c4f887413198b08fe288ab3b43a7e4823"
)

var deliveryTime = time.Unix(deliveryUnix, 0)

func signDelivery(secret string, timestamp int64, body string) string {
	stamp := strconv.FormatInt(timestamp, 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(stamp + "." + body))

	return "t=" + stamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func validMonitorRequest() *exa.CreateMonitorRequest {
	return &exa.CreateMonitorRequest{
		Search:  exa.MonitorSearch{Query: "FMCSA hours of service rule change"},
		Webhook: exa.MonitorWebhook{URL: "https://hooks.trenova.example/exa/watch_1"},
	}
}

func TestCreateMonitorSendsTheSearchScheduleAndWebhook(t *testing.T) {
	t.Parallel()

	var received map[string]any
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/monitors", r.URL.Path)
		assert.Empty(t, r.URL.RawQuery)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, sonic.Unmarshal(body, &received))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": "mon_123",
			"status": "active",
			"nextRunAt": "2026-10-17T18:00:00Z",
			"webhookSecret": "whsec_abc"
		}`))
	})

	monitor, err := client.CreateMonitor(t.Context(), &exa.CreateMonitorRequest{
		Name: "HOS rule watch",
		Search: exa.MonitorSearch{
			Query:          "  FMCSA hours of service rule change  ",
			NumResults:     10,
			IncludeDomains: []string{"fmcsa.dot.gov"},
			ExcludeDomains: []string{"spam.example.com"},
			Contents: &exa.MonitorContent{
				Highlights: &exa.HighlightOptions{
					Query:         "FMCSA hours of service rule change",
					MaxCharacters: 600,
				},
			},
		},
		Trigger:  &exa.MonitorTrigger{Type: exa.TriggerInterval, Period: "6h"},
		Metadata: map[string]string{"watchId": "aww_1", "organizationId": "org_1"},
		Webhook: exa.MonitorWebhook{
			URL:    "https://hooks.trenova.example/exa/watch_1",
			Events: []string{exa.EventMonitorRunCompleted},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"name": "HOS rule watch",
		"search": map[string]any{
			"query":          "FMCSA hours of service rule change",
			"numResults":     float64(10),
			"includeDomains": []any{"fmcsa.dot.gov"},
			"excludeDomains": []any{"spam.example.com"},
			"contents": map[string]any{
				"highlights": map[string]any{
					"query":         "FMCSA hours of service rule change",
					"maxCharacters": float64(600),
				},
			},
		},
		"trigger":  map[string]any{"type": "interval", "period": "6h"},
		"metadata": map[string]any{"watchId": "aww_1", "organizationId": "org_1"},
		"webhook": map[string]any{
			"url":    "https://hooks.trenova.example/exa/watch_1",
			"events": []any{"monitor.run.completed"},
		},
	}, received)

	assert.Equal(t, "mon_123", monitor.ID)
	assert.Equal(t, exa.MonitorStatusActive, monitor.Status)
	assert.Equal(t, "whsec_abc", monitor.WebhookSecret)
	require.NotNil(t, monitor.NextRunAt)
	assert.Equal(t, time.Date(2026, 10, 17, 18, 0, 0, 0, time.UTC).Unix(), monitor.NextRunAt.Unix())
}

func TestCreateMonitorLeavesUnsetOptionsOutOfTheBody(t *testing.T) {
	t.Parallel()

	var received map[string]any
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, sonic.Unmarshal(body, &received))

		_, _ = w.Write([]byte(`{"id": "mon_1", "status": "active", "nextRunAt": null,
			"webhookSecret": "whsec_1"}`))
	})

	monitor, err := client.CreateMonitor(t.Context(), validMonitorRequest())
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"search":  map[string]any{"query": "FMCSA hours of service rule change"},
		"webhook": map[string]any{"url": "https://hooks.trenova.example/exa/watch_1"},
	}, received)
	assert.Equal(t, "mon_1", monitor.ID)
	assert.Nil(t, monitor.NextRunAt)
}

func TestCreateMonitorRefusesAResponseWithoutAnIDOrSecret(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"no secret":  `{"id": "mon_1", "status": "active"}`,
		"no id":      `{"status": "active", "webhookSecret": "whsec_1"}`,
		"not json":   `<html>bad gateway</html>`,
		"wrong type": `{"id": 12, "webhookSecret": "whsec_1"}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(body))
			})

			monitor, err := client.CreateMonitor(t.Context(), validMonitorRequest())
			require.ErrorIs(t, err, exa.ErrUnexpectedPayload)
			assert.Nil(t, monitor)
		})
	}
}

func TestCreateMonitorRefusesAnInvalidRequestWithoutCalling(t *testing.T) {
	t.Parallel()

	tooManyDomains := make([]string, exa.MaxDomainFilters)
	for idx := range tooManyDomains {
		tooManyDomains[idx] = "site" + strconv.Itoa(idx) + ".example.com"
	}

	cases := map[string]func(*exa.CreateMonitorRequest){
		"blank query": func(r *exa.CreateMonitorRequest) { r.Search.Query = "   " },
		"query too long": func(r *exa.CreateMonitorRequest) {
			r.Search.Query = strings.Repeat("q", exa.MaxQueryLength+1)
		},
		"name too long": func(r *exa.CreateMonitorRequest) {
			r.Name = strings.Repeat("n", exa.MaxMonitorNameLength+1)
		},
		"negative results": func(r *exa.CreateMonitorRequest) { r.Search.NumResults = -1 },
		"too many results": func(r *exa.CreateMonitorRequest) {
			r.Search.NumResults = exa.MaxNumResults + 1
		},
		"too many domains": func(r *exa.CreateMonitorRequest) {
			r.Search.IncludeDomains = tooManyDomains
			r.Search.ExcludeDomains = []string{"one-more.example.com"}
		},
		"cron trigger": func(r *exa.CreateMonitorRequest) {
			r.Trigger = &exa.MonitorTrigger{Type: "cron", Period: "1d"}
		},
		"unsupported period": func(r *exa.CreateMonitorRequest) {
			r.Trigger = &exa.MonitorTrigger{Type: exa.TriggerInterval, Period: "2h"}
		},
		"empty period": func(r *exa.CreateMonitorRequest) {
			r.Trigger = &exa.MonitorTrigger{Type: exa.TriggerInterval}
		},
		"plain http webhook": func(r *exa.CreateMonitorRequest) {
			r.Webhook.URL = "http://hooks.trenova.example/exa/watch_1"
		},
		"webhook without host": func(r *exa.CreateMonitorRequest) {
			r.Webhook.URL = "https:///exa/watch_1"
		},
		"missing webhook": func(r *exa.CreateMonitorRequest) { r.Webhook.URL = "" },
		"unparseable webhook": func(r *exa.CreateMonitorRequest) {
			r.Webhook.URL = "https://hooks.trenova.example/%zz"
		},
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := newClient(t, func(http.ResponseWriter, *http.Request) {
				t.Error("an invalid monitor must not reach Exa")
			})

			req := validMonitorRequest()
			mutate(req)
			monitor, err := client.CreateMonitor(t.Context(), req)
			require.ErrorIs(t, err, exa.ErrInvalidRequest)
			assert.True(t, exa.IsInvalidRequest(err))
			assert.Nil(t, monitor)
		})
	}
}

func TestCreateMonitorAcceptsTheLimitsExactly(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"id": "mon_1", "webhookSecret": "whsec_1"}`))
	})

	domains := make([]string, exa.MaxDomainFilters)
	for idx := range domains {
		domains[idx] = "site" + strconv.Itoa(idx) + ".example.com"
	}
	req := validMonitorRequest()
	req.Name = strings.Repeat("n", exa.MaxMonitorNameLength)
	req.Search.Query = strings.Repeat("q", exa.MaxQueryLength)
	req.Search.NumResults = exa.MaxNumResults
	req.Search.IncludeDomains = domains
	req.Trigger = &exa.MonitorTrigger{Type: exa.TriggerInterval, Period: "7d"}

	_, err := client.CreateMonitor(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func TestCreateMonitorCarriesExasErrorTag(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error": "You are out of credits", "tag": "NO_MORE_CREDITS"}`))
	})

	monitor, err := client.CreateMonitor(t.Context(), validMonitorRequest())
	require.Error(t, err)
	assert.Nil(t, monitor)
	assert.True(t, exa.IsOutOfCredits(err))

	var apiErr *restx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusPaymentRequired, apiErr.StatusCode)
	assert.Equal(t, "NO_MORE_CREDITS", apiErr.Code)
	assert.Equal(t, "You are out of credits", apiErr.Message)
	assert.NotContains(t, err.Error(), "test-key")
}

func TestGetMonitorReadsItsStatus(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/monitors/mon_123", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Empty(t, r.Header.Get("Content-Type"))

		_, _ = w.Write([]byte(`{"id": "mon_123", "status": "disabled", "nextRunAt": null}`))
	})

	monitor, err := client.GetMonitor(t.Context(), "  mon_123  ")
	require.NoError(t, err)

	assert.Equal(t, &exa.Monitor{ID: "mon_123", Status: exa.MonitorStatusDisabled}, monitor)
}

func TestGetMonitorReportsAMissingMonitor(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Monitor not found", "tag": "NOT_FOUND"}`))
	})

	monitor, err := client.GetMonitor(t.Context(), "mon_gone")
	require.Error(t, err)
	assert.Nil(t, monitor)
	assert.True(t, restx.IsStatus(err, http.StatusNotFound))

	var apiErr *restx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "NOT_FOUND", apiErr.Code)
}

func TestGetMonitorRefusesAnUnreadableBody(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	})

	_, err := client.GetMonitor(t.Context(), "mon_1")
	require.ErrorIs(t, err, exa.ErrUnexpectedPayload)
}

func TestDeleteMonitorSendsADelete(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/monitors/mon_123", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Empty(t, body)

		_, _ = w.Write([]byte(`{"id": "mon_123", "status": "disabled"}`))
	})

	require.NoError(t, client.DeleteMonitor(t.Context(), "mon_123"))
	assert.Equal(t, int32(1), calls.Load())
}

func TestDeleteMonitorTreatsOneAlreadyGoneAsDeleted(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Monitor not found"}`))
	})

	require.NoError(t, client.DeleteMonitor(t.Context(), "mon_gone"))
}

func TestDeleteMonitorReportsAnyOtherFailure(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "Invalid API key", "tag": "INVALID_API_KEY"}`))
	})

	err := client.DeleteMonitor(t.Context(), "mon_123")
	require.Error(t, err)
	assert.True(t, exa.IsUnauthorized(err))
}

func TestListMonitorRunsAsksForOnePageNewestFirst(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/monitors/mon_123/runs", r.URL.Path)
		assert.Equal(t, "limit=20", r.URL.RawQuery)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))

		_, _ = w.Write([]byte(`{
			"data": [
				{
					"id": "run_2",
					"monitorId": "mon_123",
					"status": "running",
					"output": null,
					"failReason": null,
					"completedAt": null,
					"createdAt": "2026-10-17T12:00:00Z"
				},
				{
					"id": "run_1",
					"monitorId": "mon_123",
					"status": "failed",
					"output": null,
					"failReason": "  The search timed out  ",
					"completedAt": "2026-10-17T06:01:30Z",
					"createdAt": "2026-10-17T06:00:00Z"
				}
			],
			"hasMore": true,
			"nextCursor": "cur_2"
		}`))
	})

	page, err := client.ListMonitorRuns(t.Context(), "mon_123", 20)
	require.NoError(t, err)

	assert.True(t, page.HasMore)
	require.NotNil(t, page.NextCursor)
	assert.Equal(t, "cur_2", *page.NextCursor)
	require.Len(t, page.Data, 2)

	running := page.Data[0]
	assert.Equal(t, "run_2", running.ID)
	assert.Equal(t, "mon_123", running.MonitorID)
	assert.Equal(t, exa.RunStatusRunning, running.Status)
	assert.Nil(t, running.Output)
	assert.Nil(t, running.FailReason)
	assert.Nil(t, running.CompletedAt)
	assert.Empty(t, running.FailureReason())
	assert.Equal(t, time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC).Unix(), running.CreatedAt.Unix())

	failed := page.Data[1]
	assert.Equal(t, exa.RunStatusFailed, failed.Status)
	assert.Equal(t, "The search timed out", failed.FailureReason())
	require.NotNil(t, failed.CompletedAt)
	assert.Equal(t, time.Date(2026, 10, 17, 6, 1, 30, 0, time.UTC).Unix(),
		failed.CompletedAt.Unix())
}

func TestListMonitorRunsReadsTheLastPage(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "limit=100", r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"data": [], "hasMore": false, "nextCursor": null}`))
	})

	page, err := client.ListMonitorRuns(t.Context(), "mon_123", exa.MaxMonitorRunsPage)
	require.NoError(t, err)
	assert.Empty(t, page.Data)
	assert.False(t, page.HasMore)
	assert.Nil(t, page.NextCursor)
}

func TestListMonitorRunsRefusesALimitOutOfRange(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("a limit out of range must not reach Exa")
	})

	for _, limit := range []int{-1, 0, exa.MaxMonitorRunsPage + 1} {
		page, err := client.ListMonitorRuns(t.Context(), "mon_123", limit)
		require.ErrorIs(t, err, exa.ErrInvalidRequest, "limit %d", limit)
		assert.Nil(t, page)
	}
}

func TestGetMonitorRunReadsTheFullOutput(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/monitors/mon_123/runs/run_1", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))

		_, _ = w.Write([]byte(`{
			"id": "run_1",
			"monitorId": "mon_123",
			"status": "completed",
			"output": {
				"results": [{
					"id": "https://www.fmcsa.dot.gov/hos-pilot",
					"url": "https://www.fmcsa.dot.gov/hos-pilot",
					"title": "Split duty period pilot program",
					"publishedDate": "2026-10-16T00:00:00.000Z",
					"author": null,
					"highlights": ["FMCSA proposes a pilot.", "Comments close in 60 days."]
				}],
				"content": "  FMCSA proposed a split duty period pilot.  "
			},
			"failReason": null,
			"completedAt": "2026-10-17T12:02:00Z",
			"createdAt": "2026-10-17T12:00:00Z"
		}`))
	})

	run, err := client.GetMonitorRun(t.Context(), "mon_123", "run_1")
	require.NoError(t, err)

	assert.Equal(t, "run_1", run.ID)
	assert.Equal(t, exa.RunStatusCompleted, run.Status)
	require.NotNil(t, run.CompletedAt)
	assert.Equal(t, time.Date(2026, 10, 17, 12, 2, 0, 0, time.UTC).Unix(), run.CompletedAt.Unix())
	require.NotNil(t, run.Output)
	assert.Equal(t, []exa.MonitorRunResult{{
		ID:            "https://www.fmcsa.dot.gov/hos-pilot",
		URL:           "https://www.fmcsa.dot.gov/hos-pilot",
		Title:         "Split duty period pilot program",
		PublishedDate: "2026-10-16T00:00:00.000Z",
		Highlights:    []string{"FMCSA proposes a pilot.", "Comments close in 60 days."},
	}}, run.Output.Results)
	assert.Equal(t, "FMCSA proposed a split duty period pilot.", run.Output.Summary())
}

func TestGetMonitorRunRefusesAnUnreadableBody(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id": "run_1", "createdAt": "yesterday"}`))
	})

	_, err := client.GetMonitorRun(t.Context(), "mon_123", "run_1")
	require.ErrorIs(t, err, exa.ErrUnexpectedPayload)
}

func TestMonitorCallsRefuseAnIDThatWouldChangeThePath(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("an unsafe id must not reach Exa")
	})

	for _, id := range []string{
		"", "   ", ".", "..", "mon/1", "mon?1", "mon#1", "mon%2F1", `mon\1`, "mon 1",
	} {
		_, err := client.GetMonitor(t.Context(), id)
		require.ErrorIs(t, err, exa.ErrInvalidRequest, "get %q", id)

		require.ErrorIs(t, client.DeleteMonitor(t.Context(), id), exa.ErrInvalidRequest,
			"delete %q", id)

		_, err = client.ListMonitorRuns(t.Context(), id, 20)
		require.ErrorIs(t, err, exa.ErrInvalidRequest, "list %q", id)

		_, err = client.GetMonitorRun(t.Context(), id, "run_1")
		require.ErrorIs(t, err, exa.ErrInvalidRequest, "get run of %q", id)

		_, err = client.GetMonitorRun(t.Context(), "mon_1", id)
		require.ErrorIs(t, err, exa.ErrInvalidRequest, "get run %q", id)
	}
}

func TestMonitorRunOutputSummaryRendersAnyContentAsText(t *testing.T) {
	t.Parallel()

	var missing *exa.MonitorRunOutput
	assert.Empty(t, missing.Summary())
	assert.Empty(t, (&exa.MonitorRunOutput{}).Summary())
	assert.Equal(t, "New rule proposed.",
		(&exa.MonitorRunOutput{Content: "\n New rule proposed. \n"}).Summary())
	assert.Equal(t, `{"headline":"New rule proposed."}`,
		(&exa.MonitorRunOutput{
			Content: map[string]any{"headline": "New rule proposed."},
		}).Summary())
	assert.Equal(t, `["a","b"]`, (&exa.MonitorRunOutput{Content: []any{"a", "b"}}).Summary())
}

func TestMonitorRunFailureReasonIsEmptyUnlessExaGaveOne(t *testing.T) {
	t.Parallel()

	var missing *exa.MonitorRun
	assert.Empty(t, missing.FailureReason())
	assert.Empty(t, (&exa.MonitorRun{}).FailureReason())

	reason := "  Webhook unreachable \n"
	assert.Equal(t, "Webhook unreachable", (&exa.MonitorRun{FailReason: &reason}).FailureReason())
}

func TestValidMonitorPeriodAcceptsOnlyExasIntervals(t *testing.T) {
	t.Parallel()

	for _, period := range []string{"1h", "6h", "12h", "1d", "7d"} {
		assert.True(t, exa.ValidMonitorPeriod(period), period)
	}
	for _, period := range []string{"", "2h", "1D", "24h", "1w", " 1d"} {
		assert.False(t, exa.ValidMonitorPeriod(period), period)
	}
}

func TestVerifySignatureAcceptsTheKnownVector(t *testing.T) {
	t.Parallel()

	header := "t=" + strconv.FormatInt(deliveryUnix, 10) + ",v1=" + deliveryMAC
	assert.Equal(t, header, signDelivery(webhookSecret, deliveryUnix, deliveryBody))

	require.NoError(t, exa.VerifySignature(header, []byte(deliveryBody), webhookSecret,
		deliveryTime))
}

func TestVerifySignatureAcceptsALooselyFormattedHeader(t *testing.T) {
	t.Parallel()

	stamp := strconv.FormatInt(deliveryUnix, 10)
	for name, header := range map[string]string{
		"spaces":           " t=" + stamp + " , v1=" + deliveryMAC + " ",
		"v1 first":         "v1=" + deliveryMAC + ",t=" + stamp,
		"uppercase hex":    "t=" + stamp + ",v1=" + strings.ToUpper(deliveryMAC),
		"unknown parts":    "t=" + stamp + ",v0=deadbeef,v1=" + deliveryMAC + ",x",
		"later v1 ignored": "t=" + stamp + ",v1=" + deliveryMAC + ",v1=deadbeef",
	} {
		assert.NoError(t,
			exa.VerifySignature(header, []byte(deliveryBody), webhookSecret, deliveryTime),
			name)
	}
}

func TestVerifySignatureRefusesADeliveryItCannotTrust(t *testing.T) {
	t.Parallel()

	stamp := strconv.FormatInt(deliveryUnix, 10)
	valid := "t=" + stamp + ",v1=" + deliveryMAC

	cases := map[string]struct {
		header string
		body   string
		secret string
	}{
		"wrong secret": {
			header: signDelivery("whsec_other", deliveryUnix, deliveryBody),
			body:   deliveryBody,
			secret: webhookSecret,
		},
		"tampered body": {
			header: valid,
			body:   strings.Replace(deliveryBody, "run_1", "run_2", 1),
			secret: webhookSecret,
		},
		"body with trailing newline": {
			header: valid,
			body:   deliveryBody + "\n",
			secret: webhookSecret,
		},
		"timestamp swapped after signing": {
			header: "t=" + strconv.FormatInt(deliveryUnix+1, 10) + ",v1=" + deliveryMAC,
			body:   deliveryBody,
			secret: webhookSecret,
		},
		"blank secret":   {header: valid, body: deliveryBody, secret: "  "},
		"missing header": {header: "", body: deliveryBody, secret: webhookSecret},
		"no timestamp": {
			header: "v1=" + deliveryMAC,
			body:   deliveryBody,
			secret: webhookSecret,
		},
		"no signature": {header: "t=" + stamp, body: deliveryBody, secret: webhookSecret},
		"empty signature": {
			header: "t=" + stamp + ",v1=",
			body:   deliveryBody,
			secret: webhookSecret,
		},
		"bare signature": {header: deliveryMAC, body: deliveryBody, secret: webhookSecret},
		"semicolons": {
			header: "t=" + stamp + ";v1=" + deliveryMAC,
			body:   deliveryBody,
			secret: webhookSecret,
		},
		"timestamp not unix": {
			header: "t=2026-10-17,v1=" + deliveryMAC,
			body:   deliveryBody,
			secret: webhookSecret,
		},
		"truncated signature": {
			header: "t=" + stamp + ",v1=" + deliveryMAC[:32],
			body:   deliveryBody,
			secret: webhookSecret,
		},
	}

	for name, tc := range cases {
		err := exa.VerifySignature(tc.header, []byte(tc.body), tc.secret, deliveryTime)
		assert.ErrorIs(t, err, exa.ErrSignatureInvalid, name)
	}
}

func TestVerifySignatureRefusesATimestampOutsideTheTolerance(t *testing.T) {
	t.Parallel()

	header := "t=" + strconv.FormatInt(deliveryUnix, 10) + ",v1=" + deliveryMAC
	body := []byte(deliveryBody)

	require.NoError(t, exa.VerifySignature(header, body, webhookSecret,
		deliveryTime.Add(exa.SignatureTolerance)))
	require.NoError(t, exa.VerifySignature(header, body, webhookSecret,
		deliveryTime.Add(-exa.SignatureTolerance)))

	require.ErrorIs(t, exa.VerifySignature(header, body, webhookSecret,
		deliveryTime.Add(exa.SignatureTolerance+time.Second)), exa.ErrSignatureInvalid)
	require.ErrorIs(t, exa.VerifySignature(header, body, webhookSecret,
		deliveryTime.Add(-exa.SignatureTolerance-time.Second)), exa.ErrSignatureInvalid)
	require.ErrorIs(t, exa.VerifySignature(header, body, webhookSecret,
		deliveryTime.Add(24*time.Hour)), exa.ErrSignatureInvalid)
}

func TestDeleteMonitorAcceptsAnEmptyAnswer(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusAccepted, http.StatusNoContent} {
		client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		})

		require.NoError(t, client.DeleteMonitor(t.Context(), "mon_123"), status)
	}
}
