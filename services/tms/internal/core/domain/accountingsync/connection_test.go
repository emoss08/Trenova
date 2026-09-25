package accountingsync_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const now = int64(1_790_000_000)

func connected(t *testing.T) *accountingsync.AccountingConnection {
	t.Helper()
	conn := &accountingsync.AccountingConnection{
		IntegrationType: integration.TypeQuickBooksOnline,
		ExternalRealmID: "9341452431742015",
	}
	conn.BindApp(accountingsync.AppIdentity{
		Source:      accountingsync.AppSourceInstance,
		Environment: accountingsync.AppEnvironmentSandbox,
		Fingerprint: accountingsync.AppFingerprint(accountingsync.AppEnvironmentSandbox, "client"),
	})
	conn.Connect(pulid.MustNew("usr_"), accountingsync.TokenGrant{
		AccessTokenCiphertext:  "access",
		AccessTokenExpiresAt:   now + 3600,
		RefreshTokenCiphertext: "refresh",
		RefreshTokenExpiresAt:  now + 100*24*3600,
	}, now)
	return conn
}

func TestConnectSetsHealthyStateAndAbsoluteExpiry(t *testing.T) {
	t.Parallel()

	conn := connected(t)
	assert.Equal(t, accountingsync.ConnectionStatusConnected, conn.Status)
	assert.True(t, conn.IsActive())
	assert.True(t, conn.HasTokens())
	assert.Equal(t, now, conn.ConnectedAt)
	require.NotNil(t, conn.LastSuccessAt)
	assert.Equal(t, now+accountingsync.RefreshTokenAbsoluteLifetime, conn.RefreshTokenAbsoluteExpiresAt)

	multiErr := errortypes.NewMultiError()
	conn.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestTransientFailuresDegradeThenFail(t *testing.T) {
	t.Parallel()

	conn := connected(t)
	conn.RecordFailure(accountingsync.ErrorCategoryTransient, "timeout", now+1)
	assert.Equal(t, accountingsync.ConnectionStatusDegraded, conn.Status)
	conn.RecordFailure(accountingsync.ErrorCategoryTransient, "timeout", now+2)
	assert.Equal(t, accountingsync.ConnectionStatusDegraded, conn.Status)
	conn.RecordFailure(accountingsync.ErrorCategoryRateLimited, "429", now+3)
	assert.Equal(t, accountingsync.ConnectionStatusFailing, conn.Status)
	assert.Equal(t, 3, conn.ConsecutiveFailures)
	assert.True(t, conn.HasTokens())

	conn.RecordSuccess(now + 4)
	assert.Equal(t, accountingsync.ConnectionStatusConnected, conn.Status)
	assert.Zero(t, conn.ConsecutiveFailures)
	assert.Empty(t, conn.LastErrorMessage)
	assert.Empty(t, conn.LastErrorCategory)
}

func TestRevocationWipesTokensAndStaysRevoked(t *testing.T) {
	t.Parallel()

	conn := connected(t)
	conn.RecordFailure(accountingsync.ErrorCategoryRevoked, "invalid_grant", now+1)
	assert.Equal(t, accountingsync.ConnectionStatusRevoked, conn.Status)
	assert.False(t, conn.IsActive())
	assert.False(t, conn.HasTokens())
	assert.Zero(t, conn.AccessTokenExpiresAt)
}

func TestDisconnectIsTerminalForHealthUpdates(t *testing.T) {
	t.Parallel()

	conn := connected(t)
	userID := pulid.MustNew("usr_")
	conn.Disconnect(userID, now+10)
	assert.Equal(t, accountingsync.ConnectionStatusDisconnected, conn.Status)
	assert.Equal(t, userID, conn.DisconnectedByID)
	assert.False(t, conn.HasTokens())

	conn.RecordFailure(accountingsync.ErrorCategoryTransient, "late failure", now+11)
	assert.Equal(t, accountingsync.ConnectionStatusDisconnected, conn.Status)
	conn.RecordSuccess(now + 12)
	assert.Equal(t, accountingsync.ConnectionStatusDisconnected, conn.Status)
}

func TestReconnectAfterDisconnectClearsDisconnectFields(t *testing.T) {
	t.Parallel()

	conn := connected(t)
	conn.Disconnect(pulid.MustNew("usr_"), now+10)
	conn.Connect(pulid.MustNew("usr_"), accountingsync.TokenGrant{
		AccessTokenCiphertext:  "a2",
		AccessTokenExpiresAt:   now + 7200,
		RefreshTokenCiphertext: "r2",
		RefreshTokenExpiresAt:  now + 200,
	}, now+100)
	assert.Equal(t, accountingsync.ConnectionStatusConnected, conn.Status)
	assert.True(t, conn.DisconnectedByID.IsNil())
	assert.Nil(t, conn.DisconnectedAt)
	assert.Equal(t, now+100+accountingsync.RefreshTokenAbsoluteLifetime, conn.RefreshTokenAbsoluteExpiresAt)
}

func TestExpiryWindows(t *testing.T) {
	t.Parallel()

	conn := connected(t)
	assert.False(t, conn.AccessTokenExpiresWithin(now, 60))
	assert.True(t, conn.AccessTokenExpiresWithin(now, 3600))
	assert.False(t, conn.RefreshTokenNearAbsoluteExpiry(now))
	assert.True(t, conn.RefreshTokenNearAbsoluteExpiry(conn.RefreshTokenAbsoluteExpiresAt-60))
}

func TestErrorMessageIsBounded(t *testing.T) {
	t.Parallel()

	conn := connected(t)
	conn.RecordFailure(accountingsync.ErrorCategoryUnknown, strings.Repeat("é", 5000), now+1)
	assert.Equal(t, 2000, len([]rune(conn.LastErrorMessage)))
}

func TestValidateRejectsNonAccountingType(t *testing.T) {
	t.Parallel()

	conn := connected(t)
	conn.IntegrationType = integration.TypeSamsara
	multiErr := errortypes.NewMultiError()
	conn.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
}

func TestSyncingNeedsSetupCompleteAndAStartDate(t *testing.T) {
	t.Parallel()

	conn := connected(t)
	conn.SetupStep = accountingsync.SetupStepStartDate
	assert.False(t, conn.IsSyncing())
	assert.False(t, conn.CanDispatch())

	start := now - 30*24*3600
	conn.EnableSync(start, true, now)
	assert.Equal(t, accountingsync.SetupStepComplete, conn.SetupStep)
	require.NotNil(t, conn.SyncEnabledAt)
	assert.Equal(t, now, *conn.SyncEnabledAt)
	assert.True(t, conn.AutoSync)
	assert.True(t, conn.IsSyncing())
	assert.True(t, conn.CanDispatch())

	conn.EnableSync(start+10, false, now+500)
	assert.Equal(t, now, *conn.SyncEnabledAt, "the first enable time bounds the backfill")
	assert.Equal(t, start+10, *conn.SyncStartDate)
	assert.False(t, conn.AutoSync)

	conn.RecordFailure(accountingsync.ErrorCategoryRevoked, "revoked", now+600)
	assert.False(t, conn.IsSyncing())
}

func TestPauseStopsDispatchUntilResumed(t *testing.T) {
	t.Parallel()

	conn := connected(t)
	conn.EnableSync(now, true, now)
	user := pulid.MustNew("usr_")

	conn.Pause(user, "  Month-end close  ", now+1)
	assert.True(t, conn.IsPaused())
	assert.True(t, conn.IsSyncing())
	assert.False(t, conn.CanDispatch())
	assert.Equal(t, user, conn.PausedByID)
	assert.Equal(t, "Month-end close", conn.PausedReason)

	conn.Resume()
	assert.False(t, conn.IsPaused())
	assert.True(t, conn.CanDispatch())
	assert.Empty(t, conn.PausedReason)
	assert.True(t, conn.PausedByID.IsNil())
}

func TestCoversAndBooksClosedOn(t *testing.T) {
	t.Parallel()

	conn := connected(t)
	assert.False(t, conn.Covers(now))
	conn.EnableSync(now, true, now)
	assert.True(t, conn.Covers(now))
	assert.False(t, conn.Covers(now-1))

	assert.False(t, conn.BooksClosedOn(now))
	closed := now + 100
	conn.ExternalBooksClosedThrough = &closed
	assert.True(t, conn.BooksClosedOn(closed))
	assert.False(t, conn.BooksClosedOn(closed+1))
}

func TestValidateRequiresAStartDateOnceComplete(t *testing.T) {
	t.Parallel()

	conn := connected(t)
	conn.SetupStep = accountingsync.SetupStepComplete
	multiErr := errortypes.NewMultiError()
	conn.Validate(multiErr)
	require.True(t, multiErr.HasErrors())

	conn.EnableSync(now, true, now)
	multiErr = errortypes.NewMultiError()
	conn.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}
