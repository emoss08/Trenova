package wasmmodule

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingObserver struct {
	mu        sync.Mutex
	calls     map[string]int
	inUse     int
	maxInUse  int
	acquired  int
	discarded map[string]int
}

func newRecordingObserver() *recordingObserver {
	return &recordingObserver{calls: map[string]int{}, discarded: map[string]int{}}
}

func (o *recordingObserver) CallFinished(op, outcome string, _ time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.calls[op+" "+outcome]++
}

func (o *recordingObserver) InstanceAcquired() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.acquired++
	o.inUse++
	o.maxInUse = max(o.maxInUse, o.inUse)
}

func (o *recordingObserver) InstanceReleased() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.inUse--
}

func (o *recordingObserver) InstanceDiscarded(reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.discarded[reason]++
}

func (o *recordingObserver) snapshot() recordingObserver {
	o.mu.Lock()
	defer o.mu.Unlock()
	calls := make(map[string]int, len(o.calls))
	for k, v := range o.calls {
		calls[k] = v
	}
	discarded := make(map[string]int, len(o.discarded))
	for k, v := range o.discarded {
		discarded[k] = v
	}
	return recordingObserver{
		calls:     calls,
		inUse:     o.inUse,
		maxInUse:  o.maxInUse,
		acquired:  o.acquired,
		discarded: discarded,
	}
}

const testVersionOp = "rules.version"

func testSpec(t *testing.T) Spec {
	t.Helper()
	module, err := os.ReadFile(filepath.Join("..", "rules", "rules.wasm"))
	require.NoError(t, err)
	return Spec{
		Name:         "rules",
		Bytes:        module,
		VersionOp:    testVersionOp,
		CacheDirEnv:  "TRENOVA_WASMMODULE_TEST_CACHE_DIR",
		CacheDirName: "trenova-wasmmodule-test",
	}
}

func newTestRuntime(t *testing.T, opts Options) (*Runtime, *recordingObserver) {
	t.Helper()
	observer := newRecordingObserver()
	r := New(testSpec(t), opts)
	r.Instrument(zap.NewNop(), observer)
	t.Cleanup(func() {
		assert.NoError(t, r.Close(context.WithoutCancel(t.Context())))
	})
	return r, observer
}

func TestUnknownOperationIsARequestError(t *testing.T) {
	t.Parallel()

	r, observer := newTestRuntime(t, Options{PoolSize: 1})
	err := r.Call(t.Context(), "nope.never", map[string]int{"a": 1}, nil)

	var requestErr *RequestError
	require.ErrorAs(t, err, &requestErr)
	assert.ErrorIs(t, err, ErrRequest)
	assert.Equal(t, RequestError{
		Module: "rules",
		Op:     "nope.never",
		Code:   CodeUnknownOp,
		Detail: `no operation is named "nope.never"`,
	}, *requestErr)
	assert.Equal(t, map[string]int{"unknown request_error": 1}, observer.snapshot().calls)
	assert.Empty(t, observer.snapshot().discarded)
}

func TestMalformedInputIsARequestError(t *testing.T) {
	t.Parallel()

	r, _ := newTestRuntime(t, Options{PoolSize: 1})
	for name, input := range map[string][]byte{
		"truncated JSON": []byte(`{"a":`),
		"not UTF-8":      {'"', 0xff, '"'},
	} {
		_, err := r.CallRaw(t.Context(), testVersionOp, input)
		var requestErr *RequestError
		require.ErrorAs(t, err, &requestErr, name)
		assert.Equal(t, CodeMalformed, requestErr.Code, name)
		assert.ErrorIs(t, err, ErrRequest, name)
	}

	info, err := r.Version(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, info.Version)
}

func TestAnOversizedRequestIsRefusedBeforeItRuns(t *testing.T) {
	t.Parallel()

	r, observer := newTestRuntime(t, Options{PoolSize: 1})
	_, err := r.CallRaw(t.Context(), testVersionOp, bytes.Repeat([]byte(" "), MaxRequestBytes))

	var requestErr *RequestError
	require.ErrorAs(t, err, &requestErr)
	assert.Equal(t, CodeTooLarge, requestErr.Code)
	assert.Zero(t, observer.snapshot().acquired)
}

func TestConcurrentCallsShareASmallPool(t *testing.T) {
	t.Parallel()

	r, observer := newTestRuntime(t, Options{PoolSize: 2})
	const workers, callsEach = 16, 25

	var wg sync.WaitGroup
	errs := make(chan error, workers*callsEach)
	for range workers {
		wg.Go(func() {
			for range callsEach {
				info, err := r.Version(t.Context())
				if err == nil && info.Version == "" {
					err = errors.New("empty version")
				}
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	snapshot := observer.snapshot()
	assert.Equal(t, map[string]int{"rules.version ok": workers * callsEach}, snapshot.calls)
	assert.LessOrEqual(t, snapshot.maxInUse, 2)
	assert.Zero(t, snapshot.inUse)
	assert.Empty(t, snapshot.discarded)
}

func TestACancelledContextFailsWithoutRunning(t *testing.T) {
	t.Parallel()

	r, observer := newTestRuntime(t, Options{PoolSize: 1})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := r.Version(ctx)
	require.ErrorIs(t, err, ErrContextDone)
	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, r.started.Load(), "a cancelled call compiled the module")
	assert.Zero(t, observer.snapshot().acquired)
	assert.Equal(t, map[string]int{"rules.version context_done": 1}, observer.snapshot().calls)
}

func TestWaitingForAnInstanceRespectsTheContext(t *testing.T) {
	t.Parallel()

	r, _ := newTestRuntime(t, Options{PoolSize: 1})
	held := make(chan struct{})
	done := make(chan struct{})
	go func() {
		_ = r.withInstance(t.Context(), func(*instance) error {
			close(held)
			<-done
			return nil
		})
	}()
	<-held

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	_, err := r.Version(ctx)
	close(done)

	require.ErrorIs(t, err, ErrContextDone)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	info, err := r.Version(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, info.Version)
}

func TestABrokenInstanceIsDiscardedAndReplaced(t *testing.T) {
	t.Parallel()

	r, observer := newTestRuntime(t, Options{PoolSize: 1})
	var first, second *instance
	require.NoError(t, r.withInstance(t.Context(), func(in *instance) error {
		first = in
		in.broken = DiscardTrap
		return nil
	}))
	require.NoError(t, r.withInstance(t.Context(), func(in *instance) error {
		second = in
		return nil
	}))

	assert.NotSame(t, first, second)
	assert.Equal(t, map[string]int{DiscardTrap: 1}, observer.snapshot().discarded)

	var third *instance
	require.NoError(t, r.withInstance(t.Context(), func(in *instance) error {
		third = in
		return nil
	}))
	assert.Same(t, second, third, "a healthy instance was not reused")
}

func TestATrapIsReportedAndTheInstanceReplaced(t *testing.T) {
	t.Parallel()

	r, observer := newTestRuntime(t, Options{PoolSize: 1, MemoryLimitBytes: 4 * 1024 * 1024})
	input := []byte(`"` + strings.Repeat("x", 6*1024*1024) + `"`)

	_, err := r.CallRaw(t.Context(), "nope.never", input)
	require.ErrorIs(t, err, ErrTrap)
	assert.NotErrorIs(t, err, ErrRequest)
	assert.Equal(t, map[string]int{DiscardTrap: 1}, observer.snapshot().discarded)

	info, err := r.Version(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, info.Version)
	assert.Equal(t, map[string]int{
		"nope.never trap": 1,
		"rules.version ok":  1,
	}, observer.snapshot().calls)
}

func TestAnInstanceThatGrewPastTheThresholdIsRecycled(t *testing.T) {
	t.Parallel()

	r, observer := newTestRuntime(t, Options{PoolSize: 1, RecycleMemoryBytes: 4 * 1024 * 1024})

	var small *instance
	require.NoError(t, r.withInstance(t.Context(), func(in *instance) error {
		small = in
		assert.Less(t, in.memoryBytes(), uint64(4*1024*1024))
		return nil
	}))
	_, err := r.Version(t.Context())
	require.NoError(t, err)
	assert.Empty(t, observer.snapshot().discarded)

	input := []byte(`"` + strings.Repeat("x", 6*1024*1024) + `"`)
	_, err = r.CallRaw(t.Context(), "nope.never", input)
	var requestErr *RequestError
	require.ErrorAs(t, err, &requestErr)
	assert.Equal(t, CodeUnknownOp, requestErr.Code)
	assert.Equal(t, map[string]int{DiscardMemory: 1}, observer.snapshot().discarded)

	require.NoError(t, r.withInstance(t.Context(), func(in *instance) error {
		assert.NotSame(t, small, in)
		assert.Less(t, in.memoryBytes(), uint64(4*1024*1024))
		return nil
	}))
}

func TestAClosedRuntimeIsUnavailable(t *testing.T) {
	t.Parallel()

	r := New(testSpec(t), Options{PoolSize: 1})
	_, err := r.Version(t.Context())
	require.NoError(t, err)
	require.NoError(t, r.Close(t.Context()))

	_, err = r.Version(t.Context())
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestTheCompilationCacheIsUnderTheModulesSum(t *testing.T) {
	base := t.TempDir()
	spec := testSpec(t)
	t.Setenv(spec.CacheDirEnv, base)

	r := New(spec, Options{PoolSize: 1})
	_, err := r.Version(t.Context())
	require.NoError(t, err)
	require.NoError(t, r.Close(t.Context()))

	sum := sha256.Sum256(spec.Bytes)
	entries, err := os.ReadDir(filepath.Join(base, hex.EncodeToString(sum[:])))
	require.NoError(t, err)
	assert.NotEmpty(t, entries, "the compiled module was not cached under its sum")
}

func TestAModuleWithoutTheNamedExportsIsUnavailable(t *testing.T) {
	t.Parallel()

	spec := testSpec(t)
	spec.Name = "billing"
	r := New(spec, Options{PoolSize: 1})
	_, err := r.Version(t.Context())
	require.ErrorIs(t, err, ErrUnavailable)
	assert.ErrorContains(t, err, "no export billing_")
}

func TestOperationLabelsAreBounded(t *testing.T) {
	t.Parallel()

	for op, want := range map[string]string{
		"tariff.check":                 "tariff.check",
		"allocation.chargeTotal":       "allocation.chargeTotal",
		"":                             unknownOpLabel,
		"tariff":                       unknownOpLabel,
		".check":                       unknownOpLabel,
		"tariff.check.extra":           unknownOpLabel,
		"tariff.check; drop":           unknownOpLabel,
		strings.Repeat("a", 70):        unknownOpLabel,
		"a." + strings.Repeat("b", 70): unknownOpLabel,
	} {
		assert.Equal(t, want, opLabel(op), op)
	}
}
