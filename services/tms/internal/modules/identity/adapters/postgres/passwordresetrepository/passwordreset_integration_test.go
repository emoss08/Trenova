//go:build integration

package passwordresetrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestInviteTokensKeepTheirPurposeAndStayOutOfTheResetAllowance(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	now := timeutils.NowUnix()

	invite := &tenant.PasswordResetToken{
		UserID:         data.User.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		OrganizationID: data.Organization.ID,
		TokenHash:      tokenutils.Hash("invite-token"),
		Purpose:        tenant.PasswordResetPurposeInvite,
		ExpiresAt:      now + 3_600,
	}
	require.NoError(t, repo.Create(ctx, invite))

	reset := &tenant.PasswordResetToken{
		UserID:         data.User.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		OrganizationID: data.Organization.ID,
		TokenHash:      tokenutils.Hash("reset-token"),
		ExpiresAt:      now + 3_600,
	}
	require.NoError(t, repo.Create(ctx, reset))

	found, err := repo.FindRedeemableByHash(ctx, tokenutils.Hash("invite-token"), now)
	require.NoError(t, err)
	assert.Equal(t, tenant.PasswordResetPurposeInvite, found.Purpose)
	require.NotNil(t, found.User)
	assert.Equal(t, data.User.ID, found.User.ID)

	found, err = repo.FindRedeemableByHash(ctx, tokenutils.Hash("reset-token"), now)
	require.NoError(t, err)
	assert.Equal(t, tenant.PasswordResetPurposeReset, found.Purpose)

	issued, err := repo.CountSince(ctx, data.User.ID, now-60)
	require.NoError(t, err)
	assert.Equal(t, 1, issued, "only the reset link counts toward the hourly allowance")
}
