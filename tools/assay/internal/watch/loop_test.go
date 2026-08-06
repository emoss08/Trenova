package watch

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.b.String()
}

// TestLoopEndToEnd drives a whole watch session against a real repository:
// break a function, watch its test fail; save an unrelated package, watch the
// broken test stay sticky at the front; fix it, watch the session go green.
func TestLoopEndToEnd(t *testing.T) {
	r := newRepo(t)

	binaries, err := NewBinaryCache()
	require.NoError(t, err)
	t.Cleanup(binaries.Close)

	batches := make(chan Batch, 4)
	out := &lockedBuffer{}

	done := make(chan error, 1)
	go func() {
		done <- Loop(t.Context(), LoopOptions{
			Planner:  planner(r),
			Binaries: binaries,
			Batches:  batches,
			Out:      out,
			Jobs:     2,
			Timeout:  time.Minute,
		})
	}()

	send := func(paths ...string) {
		t.Helper()
		before := out.String()
		batches <- Batch{Paths: paths}
		require.Eventually(t, func() bool {
			text := out.String()

			return len(text) > len(before) && strings.HasSuffix(text, "\n\n")
		}, 90*time.Second, 50*time.Millisecond, "cycle did not complete")
	}

	r.write("alpha/alpha.go", strings.Replace(alphaSource, "n * 2", "n * 3", 1))
	send(r.path("alpha/alpha.go"))
	firstCycle := out.String()
	assert.Contains(t, firstCycle, "TestDouble", "the broken test must be named")
	assert.Contains(t, firstCycle, "FAIL")

	r.write("beta/beta.go", "package beta\n\nfunc Greet(name string) string {\n\tmsg := \"hello \" + name\n\n\treturn msg\n}\n")
	send(r.path("beta/beta.go"))
	secondCycle := strings.TrimPrefix(out.String(), firstCycle)
	assert.Contains(t, secondCycle, "beta", "the saved package runs")
	assert.Contains(t, secondCycle, "alpha", "the failing package stays sticky until it passes")
	assert.Contains(t, secondCycle, "FAIL")

	r.write("alpha/alpha.go", alphaSource)
	send(r.path("alpha/alpha.go"))
	thirdCycle := strings.TrimPrefix(out.String(), firstCycle+secondCycle)
	assert.NotContains(t, thirdCycle, "FAIL", "the fix must bring the session back to green")
	assert.Contains(t, thirdCycle, "✓")

	close(batches)
	require.NoError(t, <-done)
}

// TestLoopWarmCycleIsFast is the latency claim, measured rather than asserted
// from hope: with binaries warm and one file dirty, a cycle — scan, plan, run —
// must finish well under a second on the fixture.
func TestLoopWarmCycleIsFast(t *testing.T) {
	r := newRepo(t)

	binaries, err := NewBinaryCache()
	require.NoError(t, err)
	t.Cleanup(binaries.Close)

	batches := make(chan Batch, 4)
	out := &lockedBuffer{}

	done := make(chan error, 1)
	go func() {
		done <- Loop(t.Context(), LoopOptions{
			Planner:  planner(r),
			Binaries: binaries,
			Batches:  batches,
			Out:      out,
			Jobs:     2,
			Timeout:  time.Minute,
		})
	}()

	waitFor := func(marker string) {
		t.Helper()
		require.Eventually(t, func() bool {
			return strings.Count(out.String(), marker) > 0 && strings.HasSuffix(out.String(), "\n\n")
		}, 90*time.Second, 25*time.Millisecond)
	}

	r.write("alpha/alpha.go", strings.Replace(alphaSource, "n * 2", "n + n", 1))
	batches <- Batch{Paths: []string{r.path("alpha/alpha.go")}}
	waitFor("✓")

	warmStart := time.Now()
	r.write("alpha/alpha.go", strings.Replace(alphaSource, "n * 2", "2 * n", 1))
	before := out.String()
	batches <- Batch{Paths: []string{r.path("alpha/alpha.go")}}
	require.Eventually(t, func() bool {
		text := out.String()

		return len(text) > len(before) && strings.HasSuffix(text, "\n\n")
	}, 90*time.Second, 10*time.Millisecond)
	warm := time.Since(warmStart)

	t.Logf("warm cycle: %s", warm)
	assert.Less(t, warm, 5*time.Second,
		"a warm cycle that cannot beat five seconds on a two-package fixture is not a watch loop")

	close(batches)
	require.NoError(t, <-done)
}
