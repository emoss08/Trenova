package accountingsync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAProviderRetryDelayPostponesTheNextAttempt(t *testing.T) {
	t.Parallel()

	record := &AccountingSyncRecord{AttemptCount: 1}
	outcome := record.MarkFailed(&SyncError{
		Category:   SyncErrorRateLimited,
		RetryAfter: 3 * time.Hour,
	}, 1000)
	assert.Equal(t, SyncAttemptRetrying, outcome)
	require.NotNil(t, record.NextAttemptAt)
	assert.Equal(t, int64(1000+3*60*60), *record.NextAttemptAt)

	capped := &AccountingSyncRecord{AttemptCount: 1}
	capped.MarkFailed(&SyncError{Category: SyncErrorTransient, RetryAfter: 48 * time.Hour}, 1000)
	assert.Equal(t, int64(1000)+int64(SyncRetryCeiling/time.Second), *capped.NextAttemptAt)

	short := &AccountingSyncRecord{AttemptCount: 1}
	short.MarkFailed(&SyncError{Category: SyncErrorTransient, RetryAfter: time.Second}, 1000)
	assert.Equal(t, int64(1000)+int64(SyncRetryDelay(1)/time.Second), *short.NextAttemptAt)
}
