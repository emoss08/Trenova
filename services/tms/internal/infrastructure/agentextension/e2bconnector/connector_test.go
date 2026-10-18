package e2bconnector

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var settings = map[string]string{agentextension.ConfigKeyAPIKey: "e2b-key"}

func serve(t *testing.T, handler http.HandlerFunc) *Connector {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return newConnector(server.Client(), server.URL, server.URL)
}

func started() *agentextensionports.Sandbox {
	return &agentextensionports.Sandbox{
		ID:           "sbx1",
		Domain:       "e2b.app",
		AccessToken:  "envd-token",
		TrafficToken: "traffic-token",
		CPUCount:     2,
		MemoryMB:     512,
	}
}

func TestStartSandboxClosesTheNetworkAndReadsItsSize(t *testing.T) {
	t.Parallel()

	var created map[string]any
	connector := serve(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "e2b-key", r.Header.Get("X-API-Key"))
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes":
			body, _ := io.ReadAll(r.Body)
			assert.NoError(t, sonic.Unmarshal(body, &created))
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"sandboxID":"sbx1","envdAccessToken":"envd-token",` +
				`"trafficAccessToken":"traffic-token","domain":"e2b.app"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/sandboxes/sbx1":
			_, _ = w.Write([]byte(`{"sandboxID":"sbx1","cpuCount":4,"memoryMB":2048}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})

	sandbox, err := connector.StartSandbox(t.Context(), &agentextensionports.SandboxRequest{
		Settings: map[string]string{
			agentextension.ConfigKeyAPIKey:          "e2b-key",
			agentextension.ConfigKeyRunSeconds:      "120",
			agentextension.ConfigKeySandboxTemplate: "lanes-v2",
		},
		Metadata: map[string]string{"trenovaAnalysisRun": "aanr_1"},
	})
	require.NoError(t, err)

	assert.Equal(t, false, created["allow_internet_access"])
	assert.Equal(t, map[string]any{"allowPublicTraffic": false}, created["network"])
	assert.Equal(t, "lanes-v2", created["templateID"])
	assert.InDelta(t, 240, created["timeout"], 0, "the run's limit plus two minutes")
	assert.Equal(t, map[string]any{"trenovaAnalysisRun": "aanr_1"}, created["metadata"])

	assert.Equal(t, &agentextensionports.Sandbox{
		ID:           "sbx1",
		Domain:       "e2b.app",
		AccessToken:  "envd-token",
		TrafficToken: "traffic-token",
		CPUCount:     4,
		MemoryMB:     2048,
	}, sandbox)
}

func TestStartSandboxIsNeverRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	connector := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	_, err := connector.StartSandbox(t.Context(), &agentextensionports.SandboxRequest{Settings: settings})
	require.Error(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func TestStartSandboxKeepsDefaultSizesWhenTheDetailIsUnavailable(t *testing.T) {
	t.Parallel()

	connector := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"sandboxID":"sbx1","envdAccessToken":"t"}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	})

	sandbox, err := connector.StartSandbox(t.Context(), &agentextensionports.SandboxRequest{Settings: settings})
	require.NoError(t, err)
	assert.Equal(t, 2, sandbox.CPUCount)
	assert.Equal(t, 512, sandbox.MemoryMB)
	assert.Equal(t, "e2b.app", sandbox.Domain)
}

func TestWriteFilesWritesEachFile(t *testing.T) {
	t.Parallel()

	var paths []string
	connector := serve(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "envd-token", r.Header.Get("X-Access-Token"))
		paths = append(paths, r.URL.Query().Get("path"))
		_, _ = w.Write([]byte(`[]`))
	})

	require.NoError(t, connector.WriteFiles(t.Context(), started(), []agentextensionports.SandboxFile{
		{Path: "/home/user/data/lanes.csv", Content: []byte("a\n1\n")},
		{Path: "/home/user/data/invoices.xlsx", Content: []byte("x")},
	}))
	assert.Equal(t, []string{"/home/user/data/lanes.csv", "/home/user/data/invoices.xlsx"}, paths)
}

func TestRunCodeReportsOutputChartsAndTheException(t *testing.T) {
	t.Parallel()

	png := base64.StdEncoding.EncodeToString([]byte("\x89PNG"))
	connector := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Join([]string{
			`{"type":"stdout","text":"42 lanes\n"}`,
			`{"type":"result","png":"` + png + `","chart":{"title":"Margin by lane"}}`,
			`{"type":"error","name":"ZeroDivisionError","value":"division by zero","traceback":"tb"}`,
			`{"type":"end_of_execution"}`,
		}, "\n")))
	})

	execution, err := connector.RunCode(t.Context(), &agentextensionports.RunCodeRequest{
		Sandbox: started(),
		Code:    "print(1/0)",
		Timeout: time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, "42 lanes\n", execution.Stdout)
	require.Len(t, execution.Results, 1)
	assert.Equal(t, []byte("\x89PNG"), execution.Results[0].PNG)
	assert.Equal(t, "Margin by lane", execution.Results[0].Title)
	assert.Equal(t, &agentextension.AnalysisError{
		Name:      "ZeroDivisionError",
		Value:     "division by zero",
		Traceback: "tb",
	}, execution.Error)
	assert.False(t, execution.TimedOut)
}

func TestRunCodeSaysWhenTheCodeRanOutOfTime(t *testing.T) {
	t.Parallel()

	connector := serve(t, func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	})

	execution, err := connector.RunCode(t.Context(), &agentextensionports.RunCodeRequest{
		Sandbox: started(),
		Code:    "import time; time.sleep(60)",
		Timeout: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	assert.True(t, execution.TimedOut)
}

func TestRunCodeSaysWhenTheCellNeverFinished(t *testing.T) {
	t.Parallel()

	connector := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"type":"stdout","text":"loading"}`))
	})

	execution, err := connector.RunCode(t.Context(), &agentextensionports.RunCodeRequest{
		Sandbox: started(),
		Code:    "big = [0] * 10**12",
		Timeout: time.Minute,
	})
	require.NoError(t, err)
	assert.True(t, execution.Incomplete)
	assert.Equal(t, "loading", execution.Stdout)
}

func TestCostOfPricesTheTimeAtWhatTheSandboxHeld(t *testing.T) {
	t.Parallel()

	cost := New().CostOf(started(), 10*time.Second)
	assert.True(t, decimal.RequireFromString("0.000303").Equal(cost), cost.String())
	assert.True(t, New().CostOf(started(), 0).IsZero())
	assert.True(t, New().CostOf(nil, time.Second).IsZero())
}

func TestTestSaysHowManySandboxesAreRunning(t *testing.T) {
	t.Parallel()

	connector := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Total-Running", "2")
		_, _ = w.Write([]byte(`[{}]`))
	})

	result, err := connector.Test(t.Context(), settings)
	require.NoError(t, err)
	assert.Equal(t, "The key works. 2 sandboxes are running on this account now.", result.Note)
}

func TestClassifyTellsABusyAccountFromABadKey(t *testing.T) {
	t.Parallel()

	busy := New().Classify(&restx.APIError{StatusCode: http.StatusTooManyRequests})
	assert.Equal(t, agentextensionports.ProviderFailureRateLimited, busy.Kind)
	refused := New().Classify(&restx.APIError{StatusCode: http.StatusUnauthorized})
	assert.Equal(t, agentextensionports.ProviderFailureUnauthorized, refused.Kind)
}
