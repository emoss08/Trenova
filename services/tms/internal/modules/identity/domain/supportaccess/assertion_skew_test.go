package supportaccess_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/stretchr/testify/require"
)

func TestVerifyAssertion_ForgivesAClockAtMostFifteenSecondsApart(t *testing.T) {
	t.Parallel()

	key := newSigningKey(t, testKeyID)
	now := time.Unix(1_800_000_000, 0)

	ahead := claimsAt(now.Add(10 * time.Second))
	_, err := supportaccess.VerifyAssertion(sign(t, key, ahead), keySetOf(key), now)
	require.NoError(t, err, "a control plane clock a little ahead is forgiven")

	behind := claimsAt(now.Add(-70 * time.Second))
	_, err = supportaccess.VerifyAssertion(sign(t, key, behind), keySetOf(key), now)
	require.NoError(t, err, "an assertion expired less than fifteen seconds ago is forgiven")

	tooFarAhead := claimsAt(now.Add(20 * time.Second))
	_, err = supportaccess.VerifyAssertion(sign(t, key, tooFarAhead), keySetOf(key), now)
	requireRefused(t, err, supportaccess.RefusalNotYetValid)

	tooOld := claimsAt(now.Add(-80 * time.Second))
	_, err = supportaccess.VerifyAssertion(sign(t, key, tooOld), keySetOf(key), now)
	requireRefused(t, err, supportaccess.RefusalExpired)
}
