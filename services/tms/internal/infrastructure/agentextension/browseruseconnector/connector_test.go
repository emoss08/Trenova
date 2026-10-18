package browseruseconnector

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serve(t *testing.T, handler http.HandlerFunc) *Connector {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return newConnector(server.Client(), server.URL)
}

func taskRequest(settings map[string]string) *agentextensionports.PortalTaskRequest {
	return &agentextensionports.PortalTaskRequest{
		Settings:       settings,
		Instruction:    "On the Acme DC portal (dock appointments), find out: which slots are open?",
		Guidance:       "Stay on portal.acme.com; never open another site.",
		StartURL:       "https://portal.acme.com/login",
		AllowedDomains: []string{"portal.acme.com"},
		Secrets: map[string]string{
			agentextension.PortalSecretUsername: "dispatch@trenova.test",
			agentextension.PortalSecretPassword: "s3cret-pass",
		},
		ProfileID: "profile-1",
		Metadata:  map[string]string{"trenovaPortalRun": "aprn_1"},
	}
}

// vendor answers as Browser Use does: a session opened with POST /sessions,
// a task created with POST /tasks, a session stopped with PATCH /sessions/{id}.
type vendor struct {
	mu        sync.Mutex
	sessions  []map[string]any
	tasks     []map[string]any
	stopped   []string
	taskFails bool
}

func (v *vendor) handle(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "bu-key", r.Header.Get("X-Browser-Use-API-Key"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		var received map[string]any
		assert.NoError(t, sonic.Unmarshal(body, &received))

		v.mu.Lock()
		defer v.mu.Unlock()
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/sessions":
			v.sessions = append(v.sessions, received)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id": "session-1", "status": "active"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			v.tasks = append(v.tasks, received)
			if v.taskFails {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			_, _ = w.Write([]byte(`{"id": "task-1", "sessionId": "session-1"}`))
		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/sessions/"):
			assert.Equal(t, map[string]any{"action": "stop"}, received)
			v.stopped = append(v.stopped, strings.TrimPrefix(r.URL.Path, "/sessions/"))
			_, _ = w.Write([]byte(`{"status": "stopped"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func TestStartTaskOpensABrowserThatOutlivesTheTask(t *testing.T) {
	t.Parallel()

	v := &vendor{}
	connector := serve(t, v.handle(t))

	started, err := connector.StartTask(t.Context(), taskRequest(map[string]string{
		agentextension.ConfigKeyAPIKey:           "bu-key",
		agentextension.ConfigKeyBrowserModel:     agentextension.BrowserUseModelSonnet,
		agentextension.ConfigKeyProxyCountry:     agentextension.ProxyCountryCA,
		agentextension.ConfigKeyMaxSteps:         "40",
		agentextension.ConfigKeyAllowScreenshots: "true",
		agentextension.ConfigKeyRecordSessions:   "false",
	}))
	require.NoError(t, err)
	assert.Equal(t, "task-1", started.TaskID)
	assert.Equal(t, "session-1", started.SessionID)

	require.Len(t, v.sessions, 1)
	assert.Equal(t, map[string]any{
		"profileId":        "profile-1",
		"proxyCountryCode": "ca",
		"enableRecording":  false,
		"keepAlive":        true,
		"persistMemory":    true,
	}, v.sessions[0])

	require.Len(t, v.tasks, 1)
	task := v.tasks[0]
	assert.Equal(t, "session-1", task["sessionId"])
	assert.NotContains(t, task, "sessionSettings")
	assert.Equal(t, agentextension.BrowserUseModelSonnet, task["llm"])
	assert.InDelta(t, 40, task["maxSteps"], 0)
	assert.Equal(t, true, task["vision"])
	assert.Equal(t, []any{"portal.acme.com"}, task["allowedDomains"])
	assert.Equal(t, "https://portal.acme.com/login", task["startUrl"])
	assert.Equal(t, "Stay on portal.acme.com; never open another site.",
		task["systemPromptExtension"])
	assert.Equal(t, map[string]any{
		"portal_username": "dispatch@trenova.test",
		"portal_password": "s3cret-pass",
	}, task["secrets"])
	assert.Empty(t, v.stopped)
}

func TestStartTaskDefaultsKeepScreenshotsOffAndRecordTheSession(t *testing.T) {
	t.Parallel()

	v := &vendor{}
	connector := serve(t, v.handle(t))

	_, err := connector.StartTask(t.Context(), taskRequest(map[string]string{
		agentextension.ConfigKeyAPIKey:       "bu-key",
		agentextension.ConfigKeyProxyCountry: agentextension.ProxyCountryNone,
	}))
	require.NoError(t, err)

	require.Len(t, v.tasks, 1)
	assert.Equal(t, agentextension.BrowserUseModelDefault, v.tasks[0]["llm"])
	assert.Equal(t, false, v.tasks[0]["vision"])
	require.Len(t, v.sessions, 1)
	assert.Equal(t, true, v.sessions[0]["enableRecording"])
	assert.Contains(t, v.sessions[0], "proxyCountryCode")
	assert.Nil(t, v.sessions[0]["proxyCountryCode"])
}

func TestAFollowUpTaskCarriesOnInTheBrowserItNames(t *testing.T) {
	t.Parallel()

	v := &vendor{}
	connector := serve(t, v.handle(t))

	req := taskRequest(map[string]string{agentextension.ConfigKeyAPIKey: "bu-key"})
	req.SessionID = "session-7"
	req.StartURL = ""
	started, err := connector.StartTask(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, "session-7", started.SessionID)

	assert.Empty(t, v.sessions, "no second browser is opened")
	require.Len(t, v.tasks, 1)
	assert.Equal(t, "session-7", v.tasks[0]["sessionId"])
	assert.NotContains(t, v.tasks[0], "startUrl")
}

func TestStartTaskIsNeverRetriedAndClosesTheBrowserItOpened(t *testing.T) {
	t.Parallel()

	v := &vendor{taskFails: true}
	connector := serve(t, v.handle(t))

	_, err := connector.StartTask(t.Context(), taskRequest(map[string]string{
		agentextension.ConfigKeyAPIKey: "bu-key",
	}))
	require.Error(t, err)
	assert.Len(t, v.tasks, 1,
		"a create that failed after the vendor began it must not start a second browser")
	assert.Equal(t, []string{"session-1"}, v.stopped)
}

func TestAFollowUpThatFailsLeavesItsBrowserToTheRun(t *testing.T) {
	t.Parallel()

	v := &vendor{taskFails: true}
	connector := serve(t, v.handle(t))

	req := taskRequest(map[string]string{agentextension.ConfigKeyAPIKey: "bu-key"})
	req.SessionID = "session-7"
	_, err := connector.StartTask(t.Context(), req)
	require.Error(t, err)
	assert.Empty(t, v.stopped, "the run that owns the browser closes it")
}

func TestStopSessionClosesTheBrowser(t *testing.T) {
	t.Parallel()

	v := &vendor{}
	connector := serve(t, v.handle(t))

	require.NoError(t, connector.StopSession(t.Context(),
		map[string]string{agentextension.ConfigKeyAPIKey: "bu-key"}, "session-1"))
	assert.Equal(t, []string{"session-1"}, v.stopped)
}

func TestTaskStateIsRetriedAndMapsTheVendorStatus(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	connector := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{
			"id": "task-1",
			"sessionId": " session-1 ",
			"status": "finished",
			"output": "Open slots: 8:00 AM.",
			"isSuccess": false,
			"cost": "0.03",
			"steps": [{"number": 1, "url": "https://portal.acme.com/login"}]
		}`))
	})

	state, err := connector.TaskState(t.Context(),
		map[string]string{agentextension.ConfigKeyAPIKey: "bu-key"}, "task-1")
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
	assert.Equal(t, agentextensionports.PortalTaskFinished, state.Phase)
	assert.Equal(t, "Open slots: 8:00 AM.", state.Answer)
	require.NotNil(t, state.Succeeded)
	assert.False(t, *state.Succeeded)
	assert.Equal(t, []string{"https://portal.acme.com/login"}, state.VisitedURLs)
	assert.Equal(t, 1, state.Steps)
	assert.True(t, decimal.RequireFromString("0.03").Equal(state.CostUSD))
	assert.Equal(t, "session-1", state.SessionID)
}

func TestPhaseOfCoversEveryVendorStatus(t *testing.T) {
	t.Parallel()

	assert.Equal(t, agentextensionports.PortalTaskRunning, phaseOf("created"))
	assert.Equal(t, agentextensionports.PortalTaskRunning, phaseOf("started"))
	assert.Equal(t, agentextensionports.PortalTaskFinished, phaseOf("finished"))
	assert.Equal(t, agentextensionports.PortalTaskFailed, phaseOf("failed"))
	assert.Equal(t, agentextensionports.PortalTaskStopped, phaseOf("stopped"))
	assert.Equal(t, agentextensionports.PortalTaskRunning, phaseOf("paused"))
}

func TestRecordingTrimsWhatTheVendorReturns(t *testing.T) {
	t.Parallel()

	connector := serve(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sessions/session-1", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"id": "session-1",
			"recordingUrl": " https://cdn.browser-use.com/rec.mp4 ",
			"publicShareUrl": null
		}`))
	})

	recording, err := connector.Recording(t.Context(),
		map[string]string{agentextension.ConfigKeyAPIKey: "bu-key"}, "session-1")
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.browser-use.com/rec.mp4", recording.RecordingURL)
	assert.Empty(t, recording.ShareURL)
}

func TestTestReportsTheAccountsCredit(t *testing.T) {
	t.Parallel()

	connector := serve(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/billing/account", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"totalCreditsBalanceUsd": 12.5,
			"concurrentSessionLimit": 3,
			"isFreeTier": true
		}`))
	})

	result, err := connector.Test(t.Context(),
		map[string]string{agentextension.ConfigKeyAPIKey: "bu-key"})
	require.NoError(t, err)
	assert.Equal(t,
		"The account has $12.50 of credit and runs up to 3 browsers at once. It is on the "+
			"free tier, which limits how long a task may run.",
		result.Note)
}

func TestClassifyTellsABusyAccountFromABadKey(t *testing.T) {
	t.Parallel()

	connector := New()
	busy := connector.Classify(&restx.APIError{StatusCode: http.StatusTooManyRequests})
	assert.Equal(t, agentextensionports.ProviderFailureRateLimited, busy.Kind)

	refused := connector.Classify(&restx.APIError{StatusCode: http.StatusUnauthorized})
	assert.NotEqual(t, agentextensionports.ProviderFailureRateLimited, refused.Kind)
}

func TestLiveViewIsTheOpenBrowsersAddress(t *testing.T) {
	t.Parallel()

	connector := serve(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sessions/session-1", r.URL.Path)
		_, _ = w.Write([]byte(`{"id": "session-1", "liveUrl": " https://live.browser-use.com/s/1 "}`))
	})

	live, err := connector.LiveView(t.Context(),
		map[string]string{agentextension.ConfigKeyAPIKey: "bu-key"}, "session-1")
	require.NoError(t, err)
	assert.Equal(t, "https://live.browser-use.com/s/1", live)
}

func TestLiveViewOfAClosedBrowserIsEmpty(t *testing.T) {
	t.Parallel()

	connector := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id": "session-1", "liveUrl": null}`))
	})

	live, err := connector.LiveView(t.Context(),
		map[string]string{agentextension.ConfigKeyAPIKey: "bu-key"}, "session-1")
	require.NoError(t, err)
	assert.Empty(t, live)
}
