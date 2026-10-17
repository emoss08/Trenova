package browseruse_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/browseruse"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newClient(t *testing.T, handler http.HandlerFunc) *browseruse.Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := browseruse.New("test-key",
		browseruse.WithBaseURL(server.URL),
		browseruse.WithRetry(restx.RetryConfig{}),
	)
	require.NoError(t, err)

	return client
}

func validTask() *browseruse.CreateTaskRequest {
	country := "us"

	return &browseruse.CreateTaskRequest{
		Task:           "On the Acme DC portal, find out: which slots are open?",
		LLM:            "browser-use-2.0",
		StartURL:       "https://portal.acme.com/login",
		MaxSteps:       30,
		AllowedDomains: []string{"portal.acme.com"},
		Secrets:        map[string]string{"portal_password": "s3cret"},
		Metadata:       map[string]string{"trenovaPortalRun": "aprn_1"},
		SessionSettings: &browseruse.SessionSettings{
			ProxyCountryCode: &country,
			EnableRecording:  true,
		},
		SystemPromptExtension: "Stay on portal.acme.com.",
	}
}

func TestNewRequiresAnAPIKey(t *testing.T) {
	t.Parallel()

	_, err := browseruse.New("  ")
	require.ErrorIs(t, err, browseruse.ErrAPIKeyRequired)
}

func TestCreateTaskSendsTheAllowListAndSecrets(t *testing.T) {
	t.Parallel()

	var received map[string]any
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/tasks", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("X-Browser-Use-API-Key"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, sonic.Unmarshal(body, &received))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"id": "task-1", "sessionId": "session-1"}`))
	})

	created, err := client.CreateTask(t.Context(), validTask())
	require.NoError(t, err)
	assert.Equal(t, "task-1", created.ID)
	assert.Equal(t, "session-1", created.SessionID)

	assert.Equal(t, []any{"portal.acme.com"}, received["allowedDomains"])
	assert.Equal(t, map[string]any{"portal_password": "s3cret"}, received["secrets"])
	assert.Equal(t, "https://portal.acme.com/login", received["startUrl"])
	assert.Equal(t, false, received["vision"])
	assert.Equal(t, "Stay on portal.acme.com.", received["systemPromptExtension"])
	assert.Equal(t, map[string]any{"proxyCountryCode": "us", "enableRecording": true},
		received["sessionSettings"])
}

func TestCreateTaskSendsANullProxyWhenThereIsNone(t *testing.T) {
	t.Parallel()

	var raw string
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		raw = string(body)
		_, _ = w.Write([]byte(`{"id": "task-1"}`))
	})

	req := validTask()
	req.SessionSettings.ProxyCountryCode = nil
	_, err := client.CreateTask(t.Context(), req)
	require.NoError(t, err)
	assert.Contains(t, raw, `"proxyCountryCode":null`)
}

func TestCreateTaskRefusesWhatTheVendorWouldReject(t *testing.T) {
	t.Parallel()

	calls := 0
	client := newClient(t, func(http.ResponseWriter, *http.Request) { calls++ })

	tests := map[string]func(*browseruse.CreateTaskRequest){
		"no task":          func(r *browseruse.CreateTaskRequest) { r.Task = "  " },
		"too long":         func(r *browseruse.CreateTaskRequest) { r.Task = strings.Repeat("a", browseruse.MaxTaskLength+1) },
		"too many steps":   func(r *browseruse.CreateTaskRequest) { r.MaxSteps = browseruse.MaxSteps + 1 },
		"no allow-list":    func(r *browseruse.CreateTaskRequest) { r.AllowedDomains = nil },
		"blank domain":     func(r *browseruse.CreateTaskRequest) { r.AllowedDomains = []string{" "} },
		"plain http start": func(r *browseruse.CreateTaskRequest) { r.StartURL = "http://portal.acme.com" },
	}
	for name, mutate := range tests {
		req := validTask()
		mutate(req)
		_, err := client.CreateTask(t.Context(), req)
		require.ErrorIs(t, err, browseruse.ErrInvalidRequest, name)
	}
	assert.Zero(t, calls)
}

func TestCreateTaskRefusesAResponseWithoutATaskID(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})

	_, err := client.CreateTask(t.Context(), validTask())
	require.ErrorIs(t, err, browseruse.ErrUnexpectedPayload)
}

func TestGetTaskDecodesItsOutcome(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/tasks/task-1", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"id": "task-1",
			"sessionId": "session-1",
			"status": "finished",
			"output": "  Booked, confirmation 77812.  ",
			"isSuccess": true,
			"cost": "0.0425",
			"steps": [
				{"number": 1, "url": "https://portal.acme.com/login"},
				{"number": 2, "url": ""},
				{"number": 3, "url": "https://portal.acme.com/book"}
			]
		}`))
	})

	task, err := client.GetTask(t.Context(), "task-1")
	require.NoError(t, err)
	assert.True(t, task.Finished())
	assert.Equal(t, "Booked, confirmation 77812.", task.Answer())
	assert.True(t, decimal.RequireFromString("0.0425").Equal(task.CostUSD()))
	assert.Equal(t,
		[]string{"https://portal.acme.com/login", "https://portal.acme.com/book"},
		task.VisitedURLs())
	require.NotNil(t, task.IsSuccess)
	assert.True(t, *task.IsSuccess)
}

func TestTaskHelpersTolerateMissingFields(t *testing.T) {
	t.Parallel()

	bad := "not a number"
	task := &browseruse.Task{Status: browseruse.StatusStarted, Cost: &bad}
	assert.False(t, task.Finished())
	assert.Empty(t, task.Answer())
	assert.True(t, task.CostUSD().IsZero())
	assert.Empty(t, task.VisitedURLs())

	var none *browseruse.Task
	assert.False(t, none.Finished())
	assert.Nil(t, none.VisitedURLs())
}

func TestStopTaskStopsTheTaskAndItsSession(t *testing.T) {
	t.Parallel()

	var received map[string]any
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/tasks/task-1", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		assert.NoError(t, sonic.Unmarshal(body, &received))
		_, _ = w.Write([]byte(`{"id": "task-1", "status": "stopped"}`))
	})

	require.NoError(t, client.StopTask(t.Context(), "task-1"))
	assert.Equal(t, map[string]any{"action": "stop_task_and_session"}, received)
}

func TestIDsThatWouldChangeThePathAreRefused(t *testing.T) {
	t.Parallel()

	calls := 0
	client := newClient(t, func(http.ResponseWriter, *http.Request) { calls++ })

	for _, id := range []string{"", "  ", "../billing/account", "task?x=1", "task#a", "task%2F"} {
		_, err := client.GetTask(t.Context(), id)
		require.ErrorIs(t, err, browseruse.ErrInvalidRequest, id)
		require.ErrorIs(t, client.StopTask(t.Context(), id), browseruse.ErrInvalidRequest, id)
		_, err = client.GetSession(t.Context(), id)
		require.ErrorIs(t, err, browseruse.ErrInvalidRequest, id)
	}
	assert.Zero(t, calls)
}

func TestGetSessionReturnsItsRecording(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sessions/session-1", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"id": "session-1",
			"status": "stopped",
			"recordingUrl": "https://cdn.browser-use.com/rec.mp4",
			"publicShareUrl": null
		}`))
	})

	session, err := client.GetSession(t.Context(), "session-1")
	require.NoError(t, err)
	require.NotNil(t, session.RecordingURL)
	assert.Equal(t, "https://cdn.browser-use.com/rec.mp4", *session.RecordingURL)
	assert.Nil(t, session.PublicShareURL)
}

func TestGetAccountReadsTheBalance(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/billing/account", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"name": "Trenova",
			"totalCreditsBalanceUsd": 12.5,
			"concurrentSessionLimit": 3,
			"activeSessionCount": 1,
			"isFreeTier": false
		}`))
	})

	account, err := client.GetAccount(t.Context())
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("12.5").Equal(account.TotalCreditsBalanceUSD))
	assert.Equal(t, 3, account.ConcurrentSessionLimit)
}
