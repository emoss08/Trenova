package e2b_test

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/e2b"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// heldRequestLimit is the longest a handler holds a request its client should
// abandon.
const heldRequestLimit = 5 * time.Second

func newClient(t *testing.T, handler http.HandlerFunc) *e2b.Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := e2b.New("test-key",
		e2b.WithBaseURL(server.URL),
		e2b.WithSandboxURL(server.URL),
	)
	require.NoError(t, err)

	return client
}

func sandbox() *e2b.Sandbox {
	traffic := "traffic-token"
	return &e2b.Sandbox{
		SandboxID:          "sbx1",
		EnvdAccessToken:    "envd-token",
		TrafficAccessToken: &traffic,
	}
}

func TestNewRequiresAnAPIKey(t *testing.T) {
	t.Parallel()

	_, err := e2b.New(" ")
	require.ErrorIs(t, err, e2b.ErrAPIKeyRequired)
}

func TestCreateSandboxClosesTheNetworkAndSendsTheKey(t *testing.T) {
	t.Parallel()

	var raw string
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v2/sandboxes", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("X-API-Key"))
		body, _ := io.ReadAll(r.Body)
		raw = string(body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"templateID":"code-interpreter-v1","sandboxID":"sbx1",` +
			`"envdVersion":"0.4.0","envdAccessToken":"envd-token","trafficAccessToken":"tt",` +
			`"domain":"e2b.app"}`))
	})

	created, err := client.CreateSandbox(t.Context(), &e2b.CreateSandboxRequest{
		TemplateID:     e2b.DefaultTemplate,
		TimeoutSeconds: 120,
		Network:        &e2b.NetworkConfig{AllowPublicTraffic: false},
		Metadata:       map[string]string{"trenovaAnalysisRun": "aanr_1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "sbx1", created.SandboxID)
	assert.Equal(t, "tt", created.TrafficToken())
	assert.Equal(t, "e2b.app", created.Host())

	var body map[string]any
	require.NoError(t, sonic.UnmarshalString(raw, &body))
	assert.Equal(t, false, body["allow_internet_access"], "egress must be closed explicitly")
	assert.Equal(t, map[string]any{"allowPublicTraffic": false}, body["network"])
	assert.InDelta(t, 120, body["timeout"], 0)
	assert.Equal(t, "code-interpreter-v1", body["templateID"])
}

func TestCreateSandboxIsNeverRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	})

	_, err := client.CreateSandbox(t.Context(), &e2b.CreateSandboxRequest{
		TemplateID:     e2b.DefaultTemplate,
		TimeoutSeconds: 60,
	})
	require.Error(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func TestCreateSandboxRefusesAResponseWithoutAnAccessToken(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sandboxID":"sbx1"}`))
	})

	_, err := client.CreateSandbox(t.Context(), &e2b.CreateSandboxRequest{
		TemplateID:     e2b.DefaultTemplate,
		TimeoutSeconds: 60,
	})
	require.ErrorIs(t, err, e2b.ErrUnexpectedPayload)
}

func TestCreateSandboxValidatesBeforeSending(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newClient(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })

	for _, req := range []*e2b.CreateSandboxRequest{
		{TemplateID: " ", TimeoutSeconds: 60},
		{TemplateID: e2b.DefaultTemplate, TimeoutSeconds: 0},
		{TemplateID: e2b.DefaultTemplate, TimeoutSeconds: e2b.MaxTimeoutSeconds + 1},
	} {
		_, err := client.CreateSandbox(t.Context(), req)
		require.ErrorIs(t, err, e2b.ErrInvalidRequest)
	}
	assert.Zero(t, calls.Load())
}

func TestGetSandboxReadsItsSize(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sandboxes/sbx1", r.URL.Path)
		_, _ = w.Write([]byte(`{"sandboxID":"sbx1","state":"running","cpuCount":2,"memoryMB":512}`))
	})

	detail, err := client.GetSandbox(t.Context(), "sbx1")
	require.NoError(t, err)
	assert.Equal(t, 2, detail.CPUCount)
	assert.Equal(t, 512, detail.MemoryMB)
}

func TestKillSandboxTreatsOneAlreadyGoneAsStopped(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNotFound)
	})

	require.NoError(t, client.KillSandbox(t.Context(), "sbx1"))
}

func TestRunningSandboxesReadsTheTotalHeader(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/sandboxes", r.URL.Path)
		assert.Equal(t, "running", r.URL.Query().Get("state"))
		w.Header().Set("X-Total-Running", "3")
		_, _ = w.Write([]byte(`[{"sandboxID":"a"}]`))
	})

	running, err := client.RunningSandboxes(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 3, running)
}

func TestIDsThatWouldChangeThePathAreRefused(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newClient(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })

	for _, id := range []string{"", "../teams", "sbx?x=1", "sbx#a", "sbx%2F", "a.b"} {
		_, err := client.GetSandbox(t.Context(), id)
		require.ErrorIs(t, err, e2b.ErrInvalidRequest, id)
		require.ErrorIs(t, client.KillSandbox(t.Context(), id), e2b.ErrInvalidRequest, id)
	}
	assert.Zero(t, calls.Load())
}

func TestWriteFileUploadsWithTheSandboxTokens(t *testing.T) {
	t.Parallel()

	var got struct {
		path, user, access, traffic, sandbox, port, name, content string
	}
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/files", r.URL.Path)
		got.path = r.URL.Query().Get("path")
		got.user = r.URL.Query().Get("username")
		got.access = r.Header.Get("X-Access-Token")
		got.traffic = r.Header.Get("E2B-Traffic-Access-Token")
		got.sandbox = r.Header.Get("E2b-Sandbox-Id")
		got.port = r.Header.Get("E2b-Sandbox-Port")
		file, header, err := r.FormFile("file")
		if assert.NoError(t, err) {
			got.name = header.Filename
			data, _ := io.ReadAll(file)
			got.content = string(data)
		}
		_, _ = w.Write([]byte(`[]`))
	})

	session, err := client.Session(sandbox(), time.Minute)
	require.NoError(t, err)
	require.NoError(t, session.WriteFile(t.Context(), "/home/user/data/lanes.csv", []byte("a,b\n1,2\n")))

	assert.Equal(t, "/home/user/data/lanes.csv", got.path)
	assert.Equal(t, "user", got.user)
	assert.Equal(t, "envd-token", got.access)
	assert.Equal(t, "traffic-token", got.traffic)
	assert.Equal(t, "sbx1", got.sandbox)
	assert.Equal(t, "49983", got.port)
	assert.Equal(t, "lanes.csv", got.name)
	assert.Equal(t, "a,b\n1,2\n", got.content)
}

func TestWriteFileRefusesAPathThatLeavesWhereItPoints(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(http.ResponseWriter, *http.Request) {})
	session, err := client.Session(sandbox(), time.Minute)
	require.NoError(t, err)

	for _, path := range []string{"data/x.csv", "/home/user/../../etc/passwd"} {
		require.ErrorIs(t, session.WriteFile(t.Context(), path, []byte("x")), e2b.ErrInvalidRequest)
	}
}

func TestRunCodeReadsWhatTheCellPrintedDisplayedAndRaised(t *testing.T) {
	t.Parallel()

	png := base64.StdEncoding.EncodeToString([]byte("\x89PNG fake"))
	var raw string
	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/execute", r.URL.Path)
		assert.Equal(t, "49999", r.Header.Get("E2b-Sandbox-Port"))
		body, _ := io.ReadAll(r.Body)
		raw = string(body)
		_, _ = w.Write([]byte(strings.Join([]string{
			`{"type":"number_of_executions","execution_count":1}`,
			`{"type":"stdout","text":"rows: 42\n","timestamp":"2026-10-17T00:00:00Z"}`,
			`{"type":"result","png":"` + png + `","chart":{"type":"bar","title":"Margin by lane"},"is_main_result":false}`,
			`{"type":"result","text":"0.18","is_main_result":true}`,
			`{"type":"end_of_execution"}`,
		}, "\n")))
	})

	session, err := client.Session(sandbox(), time.Minute)
	require.NoError(t, err)
	execution, err := session.RunCode(t.Context(), "print('rows: 42')")
	require.NoError(t, err)

	assert.JSONEq(t, `{"code":"print('rows: 42')","language":"python"}`, raw)
	assert.Equal(t, "rows: 42\n", execution.Stdout)
	require.Len(t, execution.Results, 2)
	assert.Equal(t, []byte("\x89PNG fake"), execution.Results[0].PNG)
	assert.Equal(t, "Margin by lane", execution.Results[0].ChartTitle)
	assert.Equal(t, "0.18", execution.Results[1].Text)
	assert.True(t, execution.Results[1].IsMain)
	assert.Nil(t, execution.Error)
}

func TestParseExecutionKeepsTheExceptionTheCodeRaised(t *testing.T) {
	t.Parallel()

	execution, err := e2b.ParseExecution([]byte(
		`{"type":"error","name":"KeyError","value":"'margin'","traceback":"Traceback..."}` + "\n" +
			`{"type":"end_of_execution"}`,
	))
	require.NoError(t, err)
	require.NotNil(t, execution.Error)
	assert.Equal(t, "KeyError", execution.Error.Name)
	assert.Equal(t, "'margin'", execution.Error.Value)
}

func TestParseExecutionKeepsWhatTheCodeDisplayedAsJSON(t *testing.T) {
	t.Parallel()

	execution, err := e2b.ParseExecution([]byte(
		`{"type":"result","json":{"trenova":"figure","label":"Total open","value":6045662.16}}` +
			"\n" + `{"type":"end_of_execution"}`,
	))
	require.NoError(t, err)
	require.Len(t, execution.Results, 1)
	assert.Equal(t, map[string]any{
		"trenova": "figure",
		"label":   "Total open",
		"value":   6045662.16,
	}, execution.Results[0].JSON)
	assert.Empty(t, execution.Results[0].Text)
}

func TestParseExecutionSaysWhenTheCellNeverFinished(t *testing.T) {
	t.Parallel()

	execution, err := e2b.ParseExecution([]byte(`{"type":"stdout","text":"partial"}`))
	require.ErrorIs(t, err, e2b.ErrIncompleteRun)
	assert.Equal(t, "partial", execution.Stdout)
}

func TestRunCodeStopsAtItsTimeout(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		// The client gives up long before this; a client that never does
		// fails the test here instead of holding the server's Close, and the
		// package, until the test binary times out.
		select {
		case <-r.Context().Done():
		case <-time.After(heldRequestLimit):
		}
	})

	session, err := client.Session(sandbox(), 50*time.Millisecond)
	require.NoError(t, err)
	_, err = session.RunCode(t.Context(), "import time; time.sleep(10)")
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
