//go:build integration

package cloudsignuprepository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudsignup"
	"github.com/emoss08/trenova/internal/cloud/signup/signupport"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func pendingSignup(email, token string, expiresAt int64) *cloudsignup.CloudSignup {
	return &cloudsignup.CloudSignup{
		EmailAddress:    email,
		EmailNormalized: strings.ToLower(email),
		Name:            "Owner",
		CompanyName:     "Example Freight",
		PasswordHash:    "$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$aGFzaA",
		TokenHash:       token,
		Status:          cloudsignup.StatusPending,
		ExpiresAt:       expiresAt,
	}
}

func TestCloudSignupRepository_Lifecycle(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.NewTestConnection(db)
	repo := New(Params{DB: conn, Logger: zap.NewNop()})
	now := time.Now().Unix()
	token := strings.Repeat("ab", 32)

	created, err := repo.Create(ctx, pendingSignup("Owner@Example.com", token, now+3_600))
	require.NoError(t, err)

	_, err = repo.Create(ctx, pendingSignup("owner@example.com", strings.Repeat("cd", 32), now+3_600))
	require.True(t, dberror.IsUniqueConstraintViolation(err), "one pending signup per address")

	byEmail, err := repo.GetPendingByEmail(ctx, "owner@example.com")
	require.NoError(t, err)
	assert.Equal(t, created.ID, byEmail.ID)

	refreshedToken := strings.Repeat("ef", 32)
	refreshed, err := repo.Refresh(ctx, &signupport.RefreshCloudSignupRequest{
		ID:           created.ID,
		Name:         "Owner Renamed",
		EmailAddress: "Owner@Example.com",
		CompanyName:  "Example Freight",
		PasswordHash: created.PasswordHash,
		TokenHash:    refreshedToken,
		ExpiresAt:    now + 7_200,
		ClientIP:     "203.0.113.9",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, refreshed.Attempts)
	assert.Equal(t, "Owner Renamed", refreshed.Name)

	attempts, err := repo.IncrementAttempts(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, attempts)

	err = conn.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		locked, lockErr := repo.GetPendingByTokenHash(ctx, refreshedToken)
		require.NoError(t, lockErr)
		assert.Equal(t, created.ID, locked.ID)

		return repo.MarkProvisioned(ctx, &signupport.MarkCloudSignupProvisionedRequest{
			ID:             locked.ID,
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			UserID:         data.User.ID,
			VerifiedAt:     now,
		})
	})
	require.NoError(t, err)

	_, err = repo.GetPendingByTokenHash(ctx, refreshedToken)
	require.True(t, errortypes.IsNotFoundError(err), "a verified token is single use")

	count, err := repo.CountProvisionedSince(ctx, now-60)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	stale, err := repo.Create(ctx, pendingSignup("late@example.com", strings.Repeat("12", 32), now-1))
	require.NoError(t, err)
	expired, err := repo.Expire(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), expired)

	require.True(t, errortypes.IsNotFoundError(repo.Reject(ctx, &signupport.RejectCloudSignupRequest{
		ID:     stale.ID,
		Reason: "signups_paused",
	})), "only a pending signup can be rejected")
}
