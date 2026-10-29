package dotclearinghouse_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/dotclearinghouse"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrInt64(v int64) *int64 { return &v }

func TestClearinghouseQuery_Validate(t *testing.T) {
	t.Parallel()

	t.Run("a full query needs consent", func(t *testing.T) {
		t.Parallel()
		q := &dotclearinghouse.WorkerClearinghouseQuery{
			WorkerID:    pulid.MustNew("wrk_"),
			QueryType:   dotclearinghouse.ClearinghouseQueryPreEmploymentFull,
			Result:      dotclearinghouse.ClearinghouseResultNoViolations,
			RequestedAt: 1_700_000_000,
			CompletedAt: ptrInt64(1_700_100_000),
		}

		multiErr := errortypes.NewMultiError()
		q.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "electronic consent")
	})

	t.Run("violations found needs a count", func(t *testing.T) {
		t.Parallel()
		q := &dotclearinghouse.WorkerClearinghouseQuery{
			WorkerID:    pulid.MustNew("wrk_"),
			QueryType:   dotclearinghouse.ClearinghouseQueryAnnualLimited,
			Result:      dotclearinghouse.ClearinghouseResultViolationsFound,
			RequestedAt: 1_700_000_000,
			CompletedAt: ptrInt64(1_700_100_000),
		}

		multiErr := errortypes.NewMultiError()
		q.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "how many violations")
	})
}

func TestClearinghouseQuery_NextDueAt(t *testing.T) {
	t.Parallel()

	completed := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC).Unix()
	q := &dotclearinghouse.WorkerClearinghouseQuery{
		QueryType:   dotclearinghouse.ClearinghouseQueryAnnualLimited,
		Result:      dotclearinghouse.ClearinghouseResultNoViolations,
		RequestedAt: completed,
		CompletedAt: &completed,
	}

	due := q.NextDueAt()
	require.NotNil(t, due)
	assert.Equal(t, time.Date(2027, time.March, 15, 0, 0, 0, 0, time.UTC).Unix(), *due)

	// A query with no answer yet starts no clock: the year runs from the
	// answer, not from the asking.
	pending := &dotclearinghouse.WorkerClearinghouseQuery{
		QueryType:   dotclearinghouse.ClearinghouseQueryAnnualLimited,
		Result:      dotclearinghouse.ClearinghouseResultPending,
		RequestedAt: completed,
	}
	assert.Nil(t, pending.NextDueAt())
}
