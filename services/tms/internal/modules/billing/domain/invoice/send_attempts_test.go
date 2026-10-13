package invoice

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestLatestSendAttemptsKeepsOnlyTheMostRecentSend(t *testing.T) {
	t.Parallel()

	firstSend := &EmailAttempt{ID: pulid.MustNew("inea_"), AttemptNumber: 1, CreatedAt: 100, Status: SendStatusFailed}
	resendPart1 := &EmailAttempt{ID: pulid.MustNew("inea_"), AttemptNumber: 1, CreatedAt: 200, Status: SendStatusSent}
	resendPart2 := &EmailAttempt{ID: pulid.MustNew("inea_"), AttemptNumber: 2, CreatedAt: 200, Status: SendStatusSent}

	latest := LatestSendAttempts([]*EmailAttempt{resendPart2, firstSend, nil, resendPart1})

	require.Equal(t, []*EmailAttempt{resendPart1, resendPart2}, latest)
	require.Empty(t, LatestSendAttempts(nil))
}
