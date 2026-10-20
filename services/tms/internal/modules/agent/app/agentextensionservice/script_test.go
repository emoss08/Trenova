package agentextensionservice

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeBroker struct {
	mu       sync.Mutex
	index    []codemode.Read
	limits   codemode.Limits
	answers  map[string]codemode.Reply
	calls    []codemode.Call
	replied  map[int]int64
	active   atomic.Int32
	peak     atomic.Int32
	hold     time.Duration
	maxCalls int
}

func newFakeBroker() *fakeBroker {
	return &fakeBroker{
		index: []codemode.Read{{Name: "shipment_list", Description: "List shipments."}},
		answers: map[string]codemode.Reply{
			"shipment_list": {Status: codemode.CallOK, Data: []byte(`{"items":[{"pro":"P1"}]}`)},
		},
		replied: map[int]int64{},
	}
}

func (b *fakeBroker) Index() []codemode.Read { return b.index }

func (b *fakeBroker) Bound(limits codemode.Limits) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.limits = limits
}

func (b *fakeBroker) Read(_ context.Context, call codemode.Call) codemode.Reply {
	now := b.active.Add(1)
	defer b.active.Add(-1)
	for {
		peak := b.peak.Load()
		if now <= peak || b.peak.CompareAndSwap(peak, now) {
			break
		}
	}
	if b.hold > 0 {
		time.Sleep(b.hold)
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	b.calls = append(b.calls, call)
	if b.maxCalls > 0 && len(b.calls) > b.maxCalls {
		return codemode.Reply{Status: codemode.CallLimit, Limit: codemode.LimitCalls}
	}
	if reply, ok := b.answers[call.Tool]; ok {
		return reply
	}

	return codemode.Reply{Status: codemode.CallRefused, Message: "not a read this script may call"}
}

func (b *fakeBroker) NoteReply(seq int, ms int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.replied[seq] = ms
}

func (b *fakeBroker) Summary() codemode.Summary {
	b.mu.Lock()
	defer b.mu.Unlock()

	usage := codemode.Usage{Calls: len(b.calls)}
	if b.maxCalls > 0 && len(b.calls) > b.maxCalls {
		usage.Hit = codemode.LimitCalls
	}

	return codemode.Summary{Usage: usage}
}

func (b *fakeBroker) seen() []codemode.Call {
	b.mu.Lock()
	defer b.mu.Unlock()

	return append([]codemode.Call(nil), b.calls...)
}

type frameScript func(s *scriptedSandbox, emit func(text string))

type scriptedSandbox struct {
	*fakeSandbox
	nonce   string
	play    frameScript
	replies sync.Map
	timeout bool
	stderr  string
}

func newScriptedSandbox(play frameScript) *scriptedSandbox {
	return &scriptedSandbox{fakeSandbox: newFakeSandbox(), play: play}
}

func (s *scriptedSandbox) WriteFiles(
	ctx context.Context,
	sandbox *agentextensionports.Sandbox,
	files []agentextensionports.SandboxFile,
) error {
	for _, file := range files {
		if file.Path == codemode.SandboxConfigPath {
			var config codemode.SandboxConfig
			if err := sonic.Unmarshal(file.Content, &config); err == nil {
				s.nonce = config.Nonce
			}
		}
		if strings.HasPrefix(file.Path, codemode.SandboxRPCDir+"/") {
			s.replies.Store(file.Path, file.Content)
		}
	}

	return s.fakeSandbox.WriteFiles(ctx, sandbox, files)
}

func (s *scriptedSandbox) RunCode(
	_ context.Context,
	req *agentextensionports.RunCodeRequest,
) (*agentextensionports.SandboxExecution, error) {
	s.fakeSandbox.mu.Lock()
	s.fakeSandbox.ran = append(s.fakeSandbox.ran, req)
	s.fakeSandbox.mu.Unlock()
	emit := func(text string) {
		if req.Output != nil {
			_, _ = req.Output("stdout", text)
		}
	}
	s.play(s, emit)
	if s.timeout {
		return &agentextensionports.SandboxExecution{TimedOut: true}, nil
	}

	return &agentextensionports.SandboxExecution{Stderr: s.stderr}, nil
}

func (s *scriptedSandbox) frame(kind string, payload any) string {
	body, _ := sonic.Marshal(payload)
	return "\x1etrenova-" + kind + ":" + s.nonce + ":" + string(body) + "\n"
}

func (s *scriptedSandbox) request(seq int, tool string, args map[string]any) string {
	return s.frame(codemode.FrameRequest, codemode.Call{Seq: seq, Tool: tool, Arguments: args})
}

func (s *scriptedSandbox) waitReply(t *testing.T, seq int) codemode.Reply {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if raw, ok := s.replies.Load(codemode.ReplyPath(seq)); ok {
			body := string(raw.([]byte))
			body = body[:strings.Index(body, "\n\x1e")]
			var reply codemode.Reply
			require.NoError(t, sonic.Unmarshal([]byte(body), &reply))
			return reply
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("no reply to read %d", seq)

	return codemode.Reply{}
}

func newScriptFixture(t *testing.T, sandbox agentextensionports.ExtensionConnector) *fixture {
	t.Helper()

	f := &fixture{
		repo:     newMemoryRepo(),
		usage:    &memoryUsage{cost: decimal.Zero},
		web:      &fakeWeb{typ: agentextension.TypeExa},
		parallel: &fakeWeb{typ: agentextension.TypeParallel},
		browser:  &fakeBrowser{},
		sandbox:  newFakeSandbox(),
		sites:    newMemorySites(),
		runs:     newMemoryRuns(),
		analyses: newMemoryAnalyses(),
		audit:    &recordingAudit{},
		tenant: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
	}
	f.svc = New(Params{
		Logger:       zap.NewNop(),
		Repo:         f.repo,
		Usage:        f.usage,
		Encryption:   testEncryptionService(),
		AuditService: f.audit,
		Connectors: []agentextensionports.ExtensionConnector{
			f.web, f.parallel, f.browser, sandbox,
		},
		Sites:    f.sites,
		Runs:     f.runs,
		Analyses: f.analyses,
	})
	f.svc.now = func() int64 { return testNow }

	return f
}

func (f *fixture) enableScripts(t *testing.T, extra map[string]string) {
	t.Helper()

	settings := map[string]string{
		agentextension.ConfigKeyAPIKey:   testSandboxKey,
		agentextension.ConfigKeyCodeMode: "true",
	}
	for key, value := range extra {
		settings[key] = value
	}
	f.enableType(t, agentextension.TypeE2B, settings)
}

func (f *fixture) scriptRequest(broker codemode.Broker) *agentextensionports.ScriptRequest {
	return &agentextensionports.ScriptRequest{
		TenantInfo: f.tenant,
		Title:      "Late loads",
		Purpose:    "Every late load with its customer",
		Code:       "print(trenova.shipment.list())",
		Inputs:     map[string]any{"day": "2026-10-20"},
		Broker:     broker,
		StepKey:    "step-key",
	}
}

func TestRunScriptRelaysEachReadToTheBrokerAndKeepsOnlyOutput(t *testing.T) {
	t.Parallel()

	sandbox := newScriptedSandbox(nil)
	sandbox.play = func(s *scriptedSandbox, emit func(string)) {
		emit("hello\n")
		emit(s.request(1, "shipment_list", map[string]any{"status": []any{"Delayed"}}))
		reply := s.waitReply(t, 1)
		assert.Equal(t, codemode.CallOK, reply.Status)
		emit(s.frame(codemode.FrameResult, map[string]any{"value": map[string]any{"late": 1}}))
		emit(s.frame(codemode.FrameEnd, codemode.End{Status: codemode.EndOK, RoundTripsMs: []int64{12}}))
	}
	f := newScriptFixture(t, sandbox)
	f.enableScripts(t, nil)
	broker := newFakeBroker()

	outcome, err := f.svc.RunScript(t.Context(), f.scriptRequest(broker))
	require.NoError(t, err)

	assert.Equal(t, agentextensionports.ScriptSucceeded, outcome.Status)
	assert.Equal(t, "hello\n", outcome.Output, "frames never reach the output")
	assert.JSONEq(t, `{"late":1}`, string(outcome.Result))
	assert.Equal(t, []int64{12}, outcome.RoundTripsMs)
	require.Len(t, broker.seen(), 1)
	assert.Equal(t, "shipment_list", broker.seen()[0].Tool)
	assert.Contains(t, broker.replied, 1, "the reply's latency is noted")
	assert.Equal(t, 60*time.Second, broker.limits.Wall, "the broker is bound to the settings")
	assert.Equal(t, 60, broker.limits.Calls)

	require.Len(t, sandbox.started, 1)
	assert.Equal(t, "step-key", sandbox.started[0].Metadata[scriptMetadataStep])
	assert.Equal(t, 60*time.Second+scriptLifeMargin, sandbox.started[0].Lifetime)
	paths := make([]string, 0, len(sandbox.files))
	for _, file := range sandbox.files {
		paths = append(paths, file.Path)
	}
	assert.Subset(t, paths, []string{
		codemode.SandboxConfigPath, codemode.SandboxIndexPath, codemode.SandboxInputsPath,
		codemode.SandboxScriptPath, codemode.SandboxShimPath, codemode.SandboxMainPath,
	})
	assert.Equal(t, []string{"sbx1"}, sandbox.stopped, "the sandbox is stopped")
	assert.Equal(t, string(codemode.RunnerSource), sandbox.ran[0].Code)
	assert.Equal(t, 60*time.Second+scriptRunMargin, sandbox.ran[0].Timeout)
}

func TestRunScriptIgnoresFramesWithAnotherNonce(t *testing.T) {
	t.Parallel()

	sandbox := newScriptedSandbox(nil)
	sandbox.play = func(s *scriptedSandbox, emit func(string)) {
		emit("\x1etrenova-rpc:forged:{\"seq\":1,\"tool\":\"invoice_void\",\"args\":{}}\n")
		emit(s.frame(codemode.FrameEnd, codemode.End{Status: codemode.EndOK}))
	}
	f := newScriptFixture(t, sandbox)
	f.enableScripts(t, nil)
	broker := newFakeBroker()

	outcome, err := f.svc.RunScript(t.Context(), f.scriptRequest(broker))
	require.NoError(t, err)
	assert.Empty(t, broker.seen(), "a frame without the run's nonce is only output")
	assert.Contains(t, outcome.Output, "trenova-rpc:forged")
}

func TestRunScriptRefusesAnOversizedFrameAndKeepsGoing(t *testing.T) {
	t.Parallel()

	sandbox := newScriptedSandbox(nil)
	sandbox.play = func(s *scriptedSandbox, emit func(string)) {
		huge := strings.Repeat("x", codemode.DefaultFrameBytes*2)
		emit(s.request(1, "shipment_list", map[string]any{"query": huge}))
		emit(s.request(2, "shipment_list", nil))
		s.waitReply(t, 2)
		emit(s.frame(codemode.FrameEnd, codemode.End{Status: codemode.EndOK}))
	}
	f := newScriptFixture(t, sandbox)
	f.enableScripts(t, nil)
	broker := newFakeBroker()

	outcome, err := f.svc.RunScript(t.Context(), f.scriptRequest(broker))
	require.NoError(t, err)
	assert.Equal(t, codemode.LimitFrame, outcome.Limit)
	require.Len(t, broker.seen(), 1)
	assert.Equal(t, 2, broker.seen()[0].Seq)
}

func TestRunScriptHoldsReadsToEightAtOnce(t *testing.T) {
	t.Parallel()

	sandbox := newScriptedSandbox(nil)
	sandbox.play = func(s *scriptedSandbox, emit func(string)) {
		for seq := 1; seq <= 24; seq++ {
			emit(s.request(seq, "shipment_list", map[string]any{"page": seq}))
		}
		for seq := 1; seq <= 24; seq++ {
			s.waitReply(t, seq)
		}
		emit(s.frame(codemode.FrameEnd, codemode.End{Status: codemode.EndOK}))
	}
	f := newScriptFixture(t, sandbox)
	f.enableScripts(t, map[string]string{agentextension.ConfigKeyScriptReads: "200"})
	broker := newFakeBroker()
	broker.hold = 20 * time.Millisecond

	_, err := f.svc.RunScript(t.Context(), f.scriptRequest(broker))
	require.NoError(t, err)
	assert.Len(t, broker.seen(), 24)
	assert.LessOrEqual(t, broker.peak.Load(), int32(codemode.MaxConcurrentReads))
	assert.Greater(t, broker.peak.Load(), int32(1), "reads a script gathers run side by side")
}

func TestRunScriptAnswersARepeatedSequenceOnce(t *testing.T) {
	t.Parallel()

	sandbox := newScriptedSandbox(nil)
	sandbox.play = func(s *scriptedSandbox, emit func(string)) {
		emit(s.request(1, "shipment_list", nil))
		emit(s.request(1, "shipment_list", nil))
		s.waitReply(t, 1)
		emit(s.frame(codemode.FrameEnd, codemode.End{Status: codemode.EndOK}))
	}
	f := newScriptFixture(t, sandbox)
	f.enableScripts(t, nil)
	broker := newFakeBroker()

	_, err := f.svc.RunScript(t.Context(), f.scriptRequest(broker))
	require.NoError(t, err)
	assert.Len(t, broker.seen(), 1)
}

func TestRunScriptKeepsTheEndOfLongOutput(t *testing.T) {
	t.Parallel()

	sandbox := newScriptedSandbox(nil)
	sandbox.play = func(s *scriptedSandbox, emit func(string)) {
		for line := range 3000 {
			emit(fmt.Sprintf("line %04d\n", line))
		}
		emit(s.frame(codemode.FrameEnd, codemode.End{Status: codemode.EndOK}))
	}
	f := newScriptFixture(t, sandbox)
	f.enableScripts(t, nil)

	outcome, err := f.svc.RunScript(t.Context(), f.scriptRequest(newFakeBroker()))
	require.NoError(t, err)
	assert.True(t, outcome.OutputTruncated)
	assert.Equal(t, codemode.LimitOutput, outcome.Limit)
	assert.LessOrEqual(t, len(outcome.Output), codemode.DefaultOutput)
	assert.True(t, strings.HasSuffix(outcome.Output, "line 2999\n"), "the end is kept")
}

func TestRunScriptRefusesAResultPastItsLimit(t *testing.T) {
	t.Parallel()

	sandbox := newScriptedSandbox(nil)
	sandbox.play = func(s *scriptedSandbox, emit func(string)) {
		emit(s.frame(codemode.FrameResult, map[string]any{
			"value": strings.Repeat("y", codemode.DefaultResult+1),
		}))
		emit(s.frame(codemode.FrameEnd, codemode.End{Status: codemode.EndOK}))
	}
	f := newScriptFixture(t, sandbox)
	f.enableScripts(t, nil)

	outcome, err := f.svc.RunScript(t.Context(), f.scriptRequest(newFakeBroker()))
	require.NoError(t, err)
	assert.Empty(t, outcome.Result)
	assert.Contains(t, []codemode.Limit{codemode.LimitResult, codemode.LimitFrame}, outcome.Limit)
}

func TestRunScriptReportsHowItEnded(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		end     *codemode.End
		timeout bool
		status  agentextensionports.ScriptStatus
		limit   codemode.Limit
	}{
		"raised": {
			end: &codemode.End{Status: codemode.EndRaised, Error: &codemode.ScriptError{
				Name: "KeyError", Value: "'pro'",
			}},
			status: agentextensionports.ScriptErrored,
		},
		"memory":  {end: &codemode.End{Status: codemode.EndLimit, Limit: codemode.LimitMemory}, status: agentextensionports.ScriptStopped, limit: codemode.LimitMemory},
		"wall":    {end: &codemode.End{Status: codemode.EndKilled, Limit: codemode.LimitWall}, status: agentextensionports.ScriptTimedOut, limit: codemode.LimitWall},
		"cpu":     {end: &codemode.End{Status: codemode.EndKilled, Limit: codemode.LimitCPU}, status: agentextensionports.ScriptTimedOut, limit: codemode.LimitCPU},
		"timeout": {timeout: true, status: agentextensionports.ScriptTimedOut, limit: codemode.LimitWall},
		"vanished": {
			status: agentextensionports.ScriptFailed,
		},
	}
	for label, tc := range cases {
		t.Run(label, func(t *testing.T) {
			t.Parallel()

			sandbox := newScriptedSandbox(nil)
			sandbox.timeout = tc.timeout
			sandbox.play = func(s *scriptedSandbox, emit func(string)) {
				if tc.end != nil {
					emit(s.frame(codemode.FrameEnd, tc.end))
				}
			}
			f := newScriptFixture(t, sandbox)
			f.enableScripts(t, nil)

			outcome, err := f.svc.RunScript(t.Context(), f.scriptRequest(newFakeBroker()))
			require.NoError(t, err)
			assert.Equal(t, tc.status, outcome.Status)
			assert.Equal(t, tc.limit, outcome.Limit)
			assert.Equal(t, []string{"sbx1"}, sandbox.stopped)
		})
	}
}

func TestRunScriptNamesTheBrokersLimit(t *testing.T) {
	t.Parallel()

	sandbox := newScriptedSandbox(nil)
	sandbox.play = func(s *scriptedSandbox, emit func(string)) {
		emit(s.request(1, "shipment_list", nil))
		emit(s.request(2, "shipment_list", nil))
		s.waitReply(t, 2)
		emit(s.frame(codemode.FrameEnd, codemode.End{Status: codemode.EndOK}))
	}
	f := newScriptFixture(t, sandbox)
	f.enableScripts(t, nil)
	broker := newFakeBroker()
	broker.maxCalls = 1

	outcome, err := f.svc.RunScript(t.Context(), f.scriptRequest(broker))
	require.NoError(t, err)
	assert.Equal(t, codemode.LimitCalls, outcome.Limit)
}

func TestRunScriptIsRefusedWhileScriptsAreOff(t *testing.T) {
	t.Parallel()

	sandbox := newScriptedSandbox(func(*scriptedSandbox, func(string)) {})
	f := newScriptFixture(t, sandbox)
	f.enableType(t, agentextension.TypeE2B, map[string]string{
		agentextension.ConfigKeyAPIKey: testSandboxKey,
	})

	_, err := f.svc.RunScript(t.Context(), f.scriptRequest(newFakeBroker()))
	require.Error(t, err)
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "turned off")
	assert.Empty(t, sandbox.started, "no sandbox starts while scripts are off")
}

func TestRunScriptRefusesWhatItCannotRun(t *testing.T) {
	t.Parallel()

	sandbox := newScriptedSandbox(func(*scriptedSandbox, func(string)) {})
	f := newScriptFixture(t, sandbox)
	f.enableScripts(t, nil)

	empty := f.scriptRequest(newFakeBroker())
	empty.Code = "  "
	_, err := f.svc.RunScript(t.Context(), empty)
	require.True(t, IsToolError(err))

	long := f.scriptRequest(newFakeBroker())
	long.Code = strings.Repeat("x", codemode.MaxCodeChars+1)
	_, err = f.svc.RunScript(t.Context(), long)
	require.True(t, IsToolError(err))

	big := f.scriptRequest(newFakeBroker())
	big.Inputs = map[string]any{"ids": strings.Repeat("a", codemode.MaxInputBytes)}
	_, err = f.svc.RunScript(t.Context(), big)
	require.True(t, IsToolError(err))

	assert.Empty(t, sandbox.started)
}
