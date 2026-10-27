package scriptvm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testRuntime = sync.OnceValue(func() *Runtime { return New(Options{}) })

type call struct {
	Function Function
	Payload  string
}

type fakeHost struct {
	mu    sync.Mutex
	calls []call
	fail  error
}

func (h *fakeHost) Call(_ context.Context, function Function, payload []byte) ([]byte, error) {
	h.mu.Lock()
	h.calls = append(h.calls, call{Function: function, Payload: string(payload)})
	h.mu.Unlock()
	if h.fail != nil {
		return nil, h.fail
	}
	switch function {
	case FunctionRead:
		return []byte(`{"status":"ok","data":[{"id":"s1","customerId":"c1"},{"id":"s2","customerId":"c2"}]}`), nil
	case FunctionGather:
		var calls []map[string]any
		if err := sonic.Unmarshal(payload, &calls); err != nil {
			return nil, err
		}
		replies := make([]map[string]any, 0, len(calls))
		for _, c := range calls {
			args, _ := c["args"].(map[string]any)
			replies = append(replies, map[string]any{
				"status": "ok",
				"data":   map[string]any{"name": "Customer " + args["id"].(string)},
			})
		}
		return sonic.Marshal(replies)
	default:
		return []byte(`{"status":"ok"}`), nil
	}
}

func program(code string) *Program {
	return &Program{
		Code: code,
		Index: []byte(`[{"name":"shipment_list","description":"List shipments."},
			{"name":"customer_get","description":"Get a customer."}]`),
		Inputs: []byte(`{"days":60}`),
		Clock: Clock{
			Now:      time.Date(2026, 10, 23, 15, 4, 0, 0, time.UTC),
			Location: time.FixedZone("CDT", -5*60*60),
		},
	}
}

func TestTheCommittedModuleMatchesItsChecksum(t *testing.T) {
	t.Parallel()

	sidecar, err := os.ReadFile("scripts.wasm.sha256")
	require.NoError(t, err)
	sum := sha256.Sum256(moduleBytes)
	assert.Equal(t, hex.EncodeToString(sum[:])+"  scripts.wasm\n", string(sidecar))
}

func TestAScriptReadsGathersPrintsAndReturnsAResult(t *testing.T) {
	t.Parallel()

	host := &fakeHost{}
	outcome, err := testRuntime().Run(t.Context(), program(`
loads = trenova.shipment.list(status="Delayed")
names = trenova.gather(*[trenova.customer.get.defer(id=l["customerId"]) for l in loads])
for load, customer in zip(loads, names):
    print(load["id"], customer["name"])
import datetime
print(datetime.date.today(), inputs["days"])
trenova.result({"loads": len(loads)})
`), host)
	require.NoError(t, err)
	require.Equal(t, StatusOK, outcome.Status, "%+v", outcome.Error)
	assert.Equal(t, "s1 Customer c1\ns2 Customer c2\n2026-10-23 60\n", outcome.Output)
	require.Len(t, host.calls, 3)
	assert.Equal(t, FunctionRead, host.calls[0].Function)
	assert.JSONEq(t, `{"tool":"shipment_list","args":{"status":"Delayed"}}`, host.calls[0].Payload)
	assert.Equal(t, FunctionGather, host.calls[1].Function)
	assert.Equal(t, call{Function: FunctionResult, Payload: `{"loads": 2}`}, host.calls[2])
	assert.Positive(t, outcome.Duration)
}

func TestAnExceptionComesBackWithItsTraceback(t *testing.T) {
	t.Parallel()

	outcome, err := testRuntime().Run(t.Context(), program("x = {}\nx['missing']\n"), &fakeHost{})
	require.NoError(t, err)
	assert.Equal(t, StatusRaised, outcome.Status)
	require.NotNil(t, outcome.Error)
	assert.Equal(t, "KeyError", outcome.Error.Name)
	assert.Contains(t, outcome.Error.Traceback, `File "script.py", line 2`)
}

func TestTheWallLimitStopsAScriptMidLoop(t *testing.T) {
	t.Parallel()

	rt := New(Options{MaxConcurrent: 1})
	t.Cleanup(func() { _ = rt.Close(context.Background()) })
	require.NoError(t, rt.Warm())
	prog := program("while True:\n    pass\n")
	prog.Limits.CPU = time.Minute
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	started := time.Now()
	outcome, err := rt.Run(ctx, prog, &fakeHost{})
	require.NoError(t, err)
	assert.Equal(t, StatusLimit, outcome.Status)
	assert.Equal(t, LimitWall, outcome.Limit)
	assert.Less(t, time.Since(started), 5*time.Second)
}

func TestTheCPULimitEndsAScriptThatSpins(t *testing.T) {
	t.Parallel()

	prog := program("while True:\n    pass\n")
	prog.Limits.CPU = 100 * time.Millisecond
	outcome, err := testRuntime().Run(t.Context(), prog, &fakeHost{})
	require.NoError(t, err)
	assert.Equal(t, StatusLimit, outcome.Status)
	assert.Equal(t, LimitCPU, outcome.Limit)
}

func TestTheMemoryLimitEndsAScriptThatHoardsAndLeavesTheNextOneWhole(t *testing.T) {
	t.Parallel()

	prog := program("rows = []\nwhile True:\n    rows.append('x' * 100000)\n")
	prog.Limits.MemoryBytes = 32 << 20
	outcome, err := testRuntime().Run(t.Context(), prog, &fakeHost{})
	require.NoError(t, err)
	assert.Equal(t, StatusLimit, outcome.Status)
	assert.Equal(t, LimitMemory, outcome.Limit)

	outcome, err = testRuntime().Run(t.Context(), program("print('fine')"), &fakeHost{})
	require.NoError(t, err)
	assert.Equal(t, StatusOK, outcome.Status)
	assert.Equal(t, "fine\n", outcome.Output)
}

func TestAHostThatCannotAnswerFailsTheRun(t *testing.T) {
	t.Parallel()

	broken := errors.New("broker down")
	_, err := testRuntime().Run(t.Context(), program("trenova.shipment.list()"), &fakeHost{fail: broken})
	require.ErrorIs(t, err, ErrHost)
	require.ErrorIs(t, err, broken)
}

func TestScriptsRunSideBySide(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := range 8 {
		wg.Go(func() {
			code := strings.Repeat("x = 1\n", i) + "print(sum(range(1000)))"
			outcome, err := testRuntime().Run(t.Context(), program(code), &fakeHost{})
			switch {
			case err != nil:
				errs <- err
			case outcome.Output != "499500\n":
				errs <- errors.New(outcome.Output)
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}

type blockingHost struct {
	entered chan struct{}
	release chan struct{}
}

func (h *blockingHost) Call(ctx context.Context, _ Function, _ []byte) ([]byte, error) {
	h.entered <- struct{}{}
	select {
	case <-h.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return []byte(`{"status":"ok","data":[]}`), nil
}

func TestAScriptPastTheWorkersSlotsWaitsThenIsRefused(t *testing.T) {
	t.Parallel()

	rt := New(Options{MaxConcurrent: 1, QueueWait: 50 * time.Millisecond})
	t.Cleanup(func() { _ = rt.Close(context.Background()) })
	host := &blockingHost{entered: make(chan struct{}, 1), release: make(chan struct{})}

	first := make(chan error, 1)
	go func() {
		_, err := rt.Run(t.Context(), program("trenova.shipment.list()"), host)
		first <- err
	}()
	<-host.entered

	started := time.Now()
	_, err := rt.Run(t.Context(), program("print(1)"), &fakeHost{})
	require.ErrorIs(t, err, ErrBusy)
	assert.GreaterOrEqual(t, time.Since(started), 50*time.Millisecond, "it waited for a slot first")

	close(host.release)
	require.NoError(t, <-first)
	outcome, err := rt.Run(t.Context(), program("print(1)"), &fakeHost{})
	require.NoError(t, err)
	assert.Equal(t, "1\n", outcome.Output, "a freed slot runs the next script")
}

func TestAScriptWaitingForASlotGivesUpWithItsContext(t *testing.T) {
	t.Parallel()

	rt := New(Options{MaxConcurrent: 1, QueueWait: time.Minute})
	t.Cleanup(func() { _ = rt.Close(context.Background()) })
	host := &blockingHost{entered: make(chan struct{}, 1), release: make(chan struct{})}
	t.Cleanup(func() { close(host.release) })
	go func() { _, _ = rt.Run(t.Context(), program("trenova.shipment.list()"), host) }()
	<-host.entered

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	_, err := rt.Run(ctx, program("print(1)"), &fakeHost{})
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestClosingARuntimeThatRanNothingCompilesNothing(t *testing.T) {
	t.Parallel()

	rt := New(Options{})
	require.NoError(t, rt.Close(t.Context()))
	assert.Nil(t, rt.loaded.Load(), "Close loaded the module")
}

func TestTheClockAScriptSeesMovesInWholeMilliseconds(t *testing.T) {
	t.Parallel()

	nanotime := coarseNanotime(time.Now().Add(-1234567 * time.Nanosecond))
	previous := int64(0)
	for range 1000 {
		now := nanotime()
		assert.Zero(t, now%int64(time.Millisecond), "nanotime %d", now)
		assert.GreaterOrEqual(t, now, previous)
		previous = now
	}
	_, nanos := coarseWalltime()
	assert.Zero(t, int64(nanos)%int64(time.Millisecond), "walltime nanos %d", nanos)
}

func BenchmarkStartAndPrint(b *testing.B) {
	rt := testRuntime()
	require.NoError(b, rt.Warm())
	prog := program("print('ok')")
	for b.Loop() {
		if _, err := rt.Run(b.Context(), prog, &fakeHost{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFanOutOverFiveThousandRows(b *testing.B) {
	rt := testRuntime()
	require.NoError(b, rt.Warm())
	prog := program(`
rows = [{"id": i, "customer": "C" + str(i % 50), "amount": (i * 37) % 1000, "open": i % 3 != 0} for i in range(5000)]
totals = {}
for r in rows:
    if r["open"]:
        totals[r["customer"]] = totals.get(r["customer"], 0) + r["amount"]
for name, amount in sorted(totals.items(), key=lambda kv: kv[1], reverse=True)[:10]:
    print(name, amount)
`)
	for b.Loop() {
		if _, err := rt.Run(b.Context(), prog, &fakeHost{}); err != nil {
			b.Fatal(err)
		}
	}
}
