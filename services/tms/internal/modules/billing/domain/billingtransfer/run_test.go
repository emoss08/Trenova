package billingtransfer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTallyFailureCountsOrdersMostCommonFirst(t *testing.T) {
	counts := TallyFailureCounts(map[FailureCode]int{
		FailureRateValidation:     2,
		FailureRequirementsUnmet:  5,
		FailureAlreadyTransferred: 2,
		FailureUnexpected:         0,
	})

	assert.Equal(t, []FailureCount{
		{Code: FailureRequirementsUnmet, Count: 5},
		{Code: FailureAlreadyTransferred, Count: 2},
		{Code: FailureRateValidation, Count: 2},
	}, counts)
}

func TestTallyFailureCountsOfNothingIsAnEmptyList(t *testing.T) {
	assert.Equal(t, []FailureCount{}, TallyFailureCounts(nil))
}

func TestFailureCountRetryableFollowsItsReason(t *testing.T) {
	assert.True(t, FailureCount{Code: FailureRequirementsUnmet, Count: 1}.Retryable())
	assert.False(t, FailureCount{Code: FailureAlreadyTransferred, Count: 1}.Retryable())
	assert.False(t, FailureCount{Code: FailureNotFound, Count: 1}.Retryable())
}
