package startwait

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

var errRefused = errors.New("connection refused")

func TestWait_StartsWhenTheDependencyAnswersLate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		core, logs := observer.New(zap.InfoLevel)
		start := time.Now()
		checks := 0

		err := Wait(t.Context(), Params{
			Logger:     zap.New(core),
			Dependency: "Redis",
			Deadline:   start.Add(5 * time.Minute),
			Check: func(context.Context) error {
				checks++
				if time.Since(start) < 90*time.Second {
					return errRefused
				}
				return nil
			},
		})

		require.NoError(t, err)
		assert.Greater(t, checks, 2)
		assert.Less(t, time.Since(start), 90*time.Second+maxInterval*2, "it notices within a retry")

		messages := make([]string, 0, logs.Len())
		for _, entry := range logs.All() {
			messages = append(messages, entry.Message)
		}
		assert.Equal(t, []string{
			"Waiting for Redis to answer before starting",
			"Still waiting for Redis to answer",
			"Redis answered; starting",
		}, messages, "one line to begin, one a minute, one at the end")
	})
}

func TestWait_GivesUpWhenTheBudgetIsSpent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		core, logs := observer.New(zap.InfoLevel)
		start := time.Now()

		err := Wait(t.Context(), Params{
			Logger:     zap.New(core),
			Dependency: "Meilisearch",
			Deadline:   start.Add(2 * time.Minute),
			Check:      func(context.Context) error { return errRefused },
		})

		require.ErrorIs(t, err, errRefused)
		assert.Contains(t, err.Error(), "Meilisearch did not answer within the start-up wait")
		assert.Contains(t, err.Error(), "app.startupWait")
		assert.InDelta(t, 2*time.Minute, time.Since(start), float64(time.Second))
		assert.Equal(
			t,
			1,
			logs.FilterMessage("Waiting for Meilisearch to answer before starting").Len(),
		)
		assert.Equal(t, 1, logs.FilterMessage("Still waiting for Meilisearch to answer").Len())
	})
}

func TestWait_AnAnsweringDependencyWritesNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		core, logs := observer.New(zap.DebugLevel)

		err := Wait(t.Context(), Params{
			Logger:     zap.New(core),
			Dependency: "Temporal",
			Deadline:   time.Now().Add(time.Minute),
			Check:      func(context.Context) error { return nil },
		})

		require.NoError(t, err)
		assert.Zero(t, logs.Len())
	})
}

func TestWait_ASpentBudgetStillTriesOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		checks := 0

		err := Wait(t.Context(), Params{
			Logger:     zap.NewNop(),
			Dependency: "Redis",
			Deadline:   time.Now().Add(-time.Second),
			Check: func(ctx context.Context) error {
				checks++
				return ctx.Err()
			},
		})

		require.NoError(t, err, "the attempt is not cut short by the spent budget")
		assert.Equal(t, 1, checks)
	})
}

func TestWait_EachAttemptIsBounded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()

		err := Wait(t.Context(), Params{
			Logger:     zap.NewNop(),
			Dependency: "Redis",
			Deadline:   start.Add(time.Second),
			Check: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
		})

		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Equal(t, attemptTimeout, time.Since(start))
	})
}
