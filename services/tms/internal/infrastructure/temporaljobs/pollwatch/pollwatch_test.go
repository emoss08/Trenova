package pollwatch

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

const pollers = 70

var errUnavailable = errors.New("rpc error: code = Unavailable desc = connection refused")

func failAll(w *Watch) {
	var wg sync.WaitGroup
	for range pollers {
		wg.Go(func() { w.Failed(errUnavailable) })
	}
	wg.Wait()
}

func TestWatch_OneOutageWritesOneLineThenAMinuteLineThenRecovery(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		core, logs := observer.New(zap.InfoLevel)
		w := newWatch(zap.New(core))
		var up atomic.Bool
		w.Bind(func(context.Context) error {
			if up.Load() {
				return nil
			}
			return errUnavailable
		})

		failAll(w)
		require.Equal(t, 1, logs.Len(), "seventy failed polls are one line")
		first := logs.All()[0]
		assert.Equal(t, zapcore.WarnLevel, first.Level)
		assert.Contains(t, first.Message, "Cannot reach the workflow engine")
		assert.Equal(t, errUnavailable.Error(), first.ContextMap()["error"])

		time.Sleep(30 * time.Second)
		failAll(w)
		assert.Equal(t, 1, logs.Len(), "nothing more inside the first minute")

		time.Sleep(31 * time.Second)
		synctest.Wait()
		still := logs.FilterMessageSnippet("still unreachable").All()
		require.Len(t, still, 1)
		assert.Equal(t, time.Minute, still[0].ContextMap()["away"])
		assert.EqualValues(t, 2*pollers, still[0].ContextMap()["failedPolls"])

		up.Store(true)
		time.Sleep(probeEvery)
		synctest.Wait()
		back := logs.FilterMessageSnippet("reachable again").All()
		require.Len(t, back, 1)
		assert.Equal(t, zapcore.InfoLevel, back[0].Level)
		assert.EqualValues(t, 2*pollers, back[0].ContextMap()["failedPolls"])
		assert.Equal(t, 65*time.Second, back[0].ContextMap()["away"])
		assert.Equal(t, 3, logs.Len())

		require.NoError(t, w.stop(t.Context()))
	})
}

func TestWatch_ASecondOutageIsItsOwnLine(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		core, logs := observer.New(zap.InfoLevel)
		w := newWatch(zap.New(core))
		w.Bind(func(context.Context) error { return nil })

		failAll(w)
		time.Sleep(probeEvery)
		synctest.Wait()
		failAll(w)
		time.Sleep(probeEvery)
		synctest.Wait()

		assert.Equal(t, 2, logs.FilterMessageSnippet("Cannot reach the workflow engine").Len())
		assert.Equal(t, 2, logs.FilterMessageSnippet("reachable again").Len())
		assert.Equal(t, 4, logs.Len())

		require.NoError(t, w.stop(t.Context()))
	})
}

func TestWatch_StopEndsTheWatchAndIgnoresLaterFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		core, logs := observer.New(zap.InfoLevel)
		w := newWatch(zap.New(core))
		w.Bind(func(context.Context) error { return errUnavailable })

		failAll(w)
		require.NoError(t, w.stop(t.Context()))
		failAll(w)

		assert.Equal(t, 1, logs.Len())
	})
}
