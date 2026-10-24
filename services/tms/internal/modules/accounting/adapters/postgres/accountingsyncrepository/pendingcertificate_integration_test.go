//go:build integration

package accountingsyncrepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newPendingCertificate(
	tenant pagination.TenantInfo,
	userID pulid.ID,
	key string,
	now int64,
) *accountingsync.AccountingPendingCertificate {
	return &accountingsync.AccountingPendingCertificate{
		ID:                   pulid.MustNew("acctpc_"),
		OrganizationID:       tenant.OrgID,
		BusinessUnitID:       tenant.BuID,
		IntegrationType:      integration.TypeNetSuite,
		PrivateKeyCiphertext: key,
		CertificatePEM:       "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n",
		ExpiresAt:            now + 2*365*24*60*60,
		PreparedByID:         userID,
		PreparedAt:           now,
	}
}

func TestPendingCertificateRepository_ReplaceKeepsOnePerTenantAndSystem(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	repo := NewPendingCertificateRepository(PendingCertificateParams{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	byType := accountingsyncports.GetAccountingConnectionRequest{
		TenantInfo:      tenant,
		IntegrationType: integration.TypeNetSuite,
	}
	now := timeutils.NowUnix()

	_, err := repo.GetByType(ctx, byType)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	first, err := repo.Replace(ctx, newPendingCertificate(tenant, data.User.ID, "sealed-1", now))
	require.NoError(t, err)
	second, err := repo.Replace(ctx, newPendingCertificate(tenant, data.User.ID, "sealed-2", now+5))
	require.NoError(t, err)
	require.NotEqual(t, first.ID, second.ID)

	read, err := repo.GetByType(ctx, byType)
	require.NoError(t, err)
	assert.Equal(t, second.ID, read.ID)
	assert.Equal(t, "sealed-2", read.PrivateKeyCiphertext)
	assert.Equal(t, now+5, read.PreparedAt)

	count, err := accountingsyncdb.AccountingPendingCertificates.Where(keel.True()).Count(ctx, db)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)

	require.NoError(t, repo.Delete(ctx, byType))
	_, err = repo.GetByType(ctx, byType)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestPendingCertificateRepository_IsTenantScoped(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "pc")
	repo := NewPendingCertificateRepository(PendingCertificateParams{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	otherTenant := pagination.TenantInfo{OrgID: other.Organization.ID, BuID: other.BusinessUnit.ID}
	now := timeutils.NowUnix()

	_, err := repo.Replace(ctx, newPendingCertificate(tenant, data.User.ID, "sealed-1", now))
	require.NoError(t, err)

	_, err = repo.GetByType(dbscope.WithTenant(ctx, otherTenant.DBTenant()), accountingsyncports.GetAccountingConnectionRequest{
		TenantInfo:      otherTenant,
		IntegrationType: integration.TypeNetSuite,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	require.NoError(t, repo.Delete(dbscope.WithTenant(ctx, otherTenant.DBTenant()), accountingsyncports.GetAccountingConnectionRequest{
		TenantInfo:      otherTenant,
		IntegrationType: integration.TypeNetSuite,
	}))
	_, err = repo.GetByType(ctx, accountingsyncports.GetAccountingConnectionRequest{
		TenantInfo:      tenant,
		IntegrationType: integration.TypeNetSuite,
	})
	require.NoError(t, err, "another tenant's delete must not reach this tenant's certificate")
}

func TestConnectionRepository_CertificateKeyIsATokenColumn(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	repo := NewConnectionRepository(ConnectionParams{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	now := timeutils.NowUnix()
	expires := now + 2*365*24*60*60

	conn := &accountingsync.AccountingConnection{
		OrganizationID:  tenant.OrgID,
		BusinessUnitID:  tenant.BuID,
		IntegrationType: integration.TypeNetSuite,
		ExternalRealmID: "1234567_SB1-s3",
	}
	conn.UseCertificate(accountingsync.ClientCertificate{
		KeyCiphertext: "sealed-key",
		CertificateID: "Zm9vYmFy",
		ExpiresAt:     expires,
	})
	conn.Connect(data.User.ID, accountingsync.TokenGrant{
		AccessTokenCiphertext: "cipher-access",
		AccessTokenExpiresAt:  now + 3600,
	}, now)
	conn.ApplyCompanyFacts(&accountingsync.CompanyFacts{CompanyName: "Peak", HomeCurrency: "USD"})
	created, err := repo.Create(ctx, conn)
	require.NoError(t, err)

	read, err := repo.GetByID(ctx, accountingsyncports.GetAccountingConnectionByIDRequest{
		TenantInfo: tenant,
		ID:         created.ID,
	})
	require.NoError(t, err)
	assert.Empty(t, read.ClientKeyCiphertext, "the key is never read outside the token lock")
	assert.Equal(t, accountingsync.AuthMethodClientCredentials, read.AuthMethod)
	assert.Equal(t, "Zm9vYmFy", read.ClientCertificateID)
	require.NotNil(t, read.ClientCertificateExpiresAt)
	assert.Equal(t, expires, *read.ClientCertificateExpiresAt)
	assert.Equal(t, expires, read.RefreshTokenAbsoluteExpiresAt)

	read.ExternalCompanyName = "Peak Renamed"
	_, err = repo.Update(ctx, read)
	require.NoError(t, err)

	locked, err := repo.LockWithTokens(ctx, accountingsyncports.GetAccountingConnectionByIDRequest{
		TenantInfo: tenant,
		ID:         created.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, "sealed-key", locked.ClientKeyCiphertext, "Update must not clear the key")

	require.NoError(t, repo.StoreTokens(ctx, accountingsyncports.StoreAccountingTokensRequest{
		TenantInfo:            tenant,
		ID:                    created.ID,
		AccessTokenCiphertext: "cipher-access-2",
		AccessTokenExpiresAt:  now + 7200,
		ClientKeyCiphertext:   "sealed-key-2",
		At:                    now + 1,
	}))
	locked, err = repo.LockWithTokens(ctx, accountingsyncports.GetAccountingConnectionByIDRequest{
		TenantInfo: tenant,
		ID:         created.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, "sealed-key-2", locked.ClientKeyCiphertext)
	assert.Empty(t, locked.RefreshTokenCiphertext)
}

func TestConnectionRepository_CertificateConnectionMustNameItsCertificate(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	repo := NewConnectionRepository(ConnectionParams{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	now := timeutils.NowUnix()

	conn := &accountingsync.AccountingConnection{
		OrganizationID:  tenant.OrgID,
		BusinessUnitID:  tenant.BuID,
		IntegrationType: integration.TypeNetSuite,
		ExternalRealmID: "1234567",
		AuthMethod:      accountingsync.AuthMethodClientCredentials,
	}
	conn.Connect(data.User.ID, accountingsync.TokenGrant{AccessTokenCiphertext: "a"}, now)
	_, err := repo.Create(ctx, conn)
	require.Error(t, err, "the database refuses a certificate connection without a certificate")
}
