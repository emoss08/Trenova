package aicorrectionretention_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/aicorrectionretention"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type retentionReader struct {
	settings *tenant.DataRetention
	err      error
}

func (r retentionReader) Get(
	context.Context,
	repositories.GetDataRetentionRequest,
) (*tenant.DataRetention, error) {
	return r.settings, r.err
}

const now = int64(1_800_000_000)

func TestSweepPurgesInBatchesUntilAShortOne(t *testing.T) {
	t.Parallel()

	sweep := aicorrectionretention.Sweep{
		Retention: retentionReader{err: errortypes.NewNotFoundError("none")},
		BatchSize: 10,
		Now:       func() int64 { return now },
	}
	batches := []int64{10, 10, 3}
	var befores []int64
	total, err := sweep.Run(
		t.Context(),
		services.PurgeExpiredAICorrectionsRequest{},
		func(_ context.Context, before int64, limit int) (int64, error) {
			assert.Equal(t, 10, limit)
			befores = append(befores, before)
			purged := batches[0]
			batches = batches[1:]
			return purged, nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, int64(23), total)
	require.Len(t, befores, 3)
	days := (&tenant.DataRetention{}).AICorrectionRetentionDays()
	assert.Equal(t, now-int64(days)*timeutils.SecondsPerDay, befores[0])
}

func TestSweepUsesTheRequestTimeAndStopsOnError(t *testing.T) {
	t.Parallel()

	sweep := aicorrectionretention.Sweep{
		Retention: retentionReader{settings: &tenant.DataRetention{}},
		BatchSize: 5,
		Now:       func() int64 { return 0 },
	}
	failed := errors.New("database unavailable")
	calls := 0
	total, err := sweep.Run(
		t.Context(),
		services.PurgeExpiredAICorrectionsRequest{Now: now},
		func(_ context.Context, before int64, _ int) (int64, error) {
			calls++
			assert.Less(t, before, now)
			if calls == 2 {
				return 0, failed
			}
			return 5, nil
		},
	)

	require.ErrorIs(t, err, failed)
	assert.Equal(t, int64(5), total)
}

func TestSweepReturnsARetentionReadFailure(t *testing.T) {
	t.Parallel()

	failed := errors.New("database unavailable")
	sweep := aicorrectionretention.Sweep{
		Retention: retentionReader{err: failed},
		BatchSize: 5,
		Now:       func() int64 { return now },
	}
	_, err := sweep.Run(
		t.Context(),
		services.PurgeExpiredAICorrectionsRequest{},
		func(context.Context, int64, int) (int64, error) {
			t.Fatal("nothing is purged without a cutoff")
			return 0, nil
		},
	)

	require.ErrorIs(t, err, failed)
}
