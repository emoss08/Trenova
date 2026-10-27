//go:build integration

package iamrepository

import (
	"testing"

	"github.com/emoss08/keel"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConsumeRecoveryCodeMatchesEachHashVersionOnlyByItsOwnDigest(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := NewMFARepository(MFARepositoryParams{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	userID := data.User.ID
	ctx = data.Ctx(ctx)

	err := iamdb.MFARecoveryCodes.CreateMany(ctx, db, []*iam.MFARecoveryCode{
		{
			UserID:         userID,
			OrganizationID: data.Organization.ID,
			CodeHash:       "legacy-digest",
			HashVersion:    iam.RecoveryCodeHashSHA256,
		},
		{
			UserID:         userID,
			OrganizationID: data.Organization.ID,
			CodeHash:       "keyed-digest",
			HashVersion:    iam.RecoveryCodeHashHMAC,
		},
	})
	require.NoError(t, err)

	legacy, err := repo.CountLegacyRecoveryCodes(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 1, legacy)

	consumed, err := repo.ConsumeRecoveryCode(ctx, iamports.ConsumeRecoveryCodeRequest{
		UserID:         userID,
		CodeHash:       "legacy-digest",
		LegacyCodeHash: "keyed-digest",
		UsedAt:         1,
	})
	require.NoError(t, err)
	assert.False(t, consumed)

	consumed, err = repo.ConsumeRecoveryCode(ctx, iamports.ConsumeRecoveryCodeRequest{
		UserID:         userID,
		CodeHash:       "unrelated",
		LegacyCodeHash: "legacy-digest",
		UsedAt:         1,
	})
	require.NoError(t, err)
	assert.True(t, consumed)

	consumed, err = repo.ConsumeRecoveryCode(ctx, iamports.ConsumeRecoveryCodeRequest{
		UserID:         userID,
		CodeHash:       "keyed-digest",
		LegacyCodeHash: "unrelated",
		UsedAt:         2,
	})
	require.NoError(t, err)
	assert.True(t, consumed)

	consumed, err = repo.ConsumeRecoveryCode(ctx, iamports.ConsumeRecoveryCodeRequest{
		UserID:         userID,
		CodeHash:       "keyed-digest",
		LegacyCodeHash: "legacy-digest",
		UsedAt:         3,
	})
	require.NoError(t, err)
	assert.False(t, consumed)

	legacy, err = repo.CountLegacyRecoveryCodes(ctx, userID)
	require.NoError(t, err)
	assert.Zero(t, legacy)
}

func TestRecoveryCodeHashVersionDefaultsToKeyedForNewRows(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	_, err := keel.Exec(ctx, db,
		`INSERT INTO mfa_recovery_codes (id, user_id, organization_id, code_hash) VALUES ('mrc_default', $1, $2, 'digest')`,
		data.User.ID.String(), data.Organization.ID.String(),
	)
	require.NoError(t, err)

	version, err := keel.QueryValue[int16](ctx, db,
		`SELECT hash_version FROM mfa_recovery_codes WHERE id = 'mrc_default'`,
	)
	require.NoError(t, err)
	assert.Equal(t, iam.RecoveryCodeHashHMAC, version)
}
