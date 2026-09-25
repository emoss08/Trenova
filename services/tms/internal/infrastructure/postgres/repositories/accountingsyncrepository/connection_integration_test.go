//go:build integration

package accountingsyncrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const realm = "9341452431742015"

func newConnection(
	tenant pagination.TenantInfo,
	userID pulid.ID,
	realmID string,
	now int64,
) *accountingsync.AccountingConnection {
	conn := &accountingsync.AccountingConnection{
		OrganizationID:  tenant.OrgID,
		BusinessUnitID:  tenant.BuID,
		IntegrationType: integration.TypeQuickBooksOnline,
		ExternalRealmID: realmID,
	}
	conn.Connect(userID, accountingsync.TokenGrant{
		AccessTokenCiphertext:  "cipher-access",
		AccessTokenExpiresAt:   now + 3600,
		RefreshTokenCiphertext: "cipher-refresh",
		RefreshTokenExpiresAt:  now + 8_640_000,
	}, now)
	conn.ApplyCompanyFacts(&accountingsync.CompanyFacts{CompanyName: "Acme", HomeCurrency: "USD"})
	return conn
}

func TestConnectionRepository_TokensAreReadOnlyUnderLockAndNeverOverwrittenByUpdate(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.NewTestConnection(db)
	repo := NewConnectionRepository(ConnectionParams{DB: conn, Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	now := timeutils.NowUnix()

	created, err := repo.Create(ctx, newConnection(tenant, data.User.ID, realm, now))
	require.NoError(t, err)

	read, err := repo.GetByType(ctx, repositories.GetAccountingConnectionRequest{
		TenantInfo:      tenant,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.NoError(t, err)
	assert.Empty(t, read.AccessTokenCiphertext)
	assert.Empty(t, read.RefreshTokenCiphertext)
	assert.Equal(t, "Acme", read.ExternalCompanyName)
	assert.Equal(t, now+3600, read.AccessTokenExpiresAt)

	read.ExternalCompanyName = "Acme Renamed"
	read.RecordFailure(accountingsync.ErrorCategoryTransient, "timeout", now+1)
	_, err = repo.Update(ctx, read)
	require.NoError(t, err)

	err = conn.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		locked, lockErr := repo.LockWithTokens(txCtx, repositories.GetAccountingConnectionByIDRequest{
			TenantInfo: tenant,
			ID:         created.ID,
		})
		require.NoError(t, lockErr)
		assert.Equal(t, "cipher-access", locked.AccessTokenCiphertext, "Update must not clear tokens")
		assert.Equal(t, "cipher-refresh", locked.RefreshTokenCiphertext)
		assert.Equal(t, "Acme Renamed", locked.ExternalCompanyName)
		assert.Equal(t, accountingsync.ConnectionStatusDegraded, locked.Status)
		return nil
	})
	require.NoError(t, err)

	refreshed := now + 10
	require.NoError(t, repo.StoreTokens(ctx, repositories.StoreAccountingTokensRequest{
		TenantInfo:             tenant,
		ID:                     created.ID,
		AccessTokenCiphertext:  "cipher-access-2",
		AccessTokenExpiresAt:   now + 7200,
		RefreshTokenCiphertext: "cipher-refresh-2",
		RefreshTokenExpiresAt:  now + 9_000_000,
		RefreshedAt:            &refreshed,
		At:                     refreshed,
	}))
	locked, err := repo.LockByTypeWithTokens(ctx, repositories.GetAccountingConnectionRequest{
		TenantInfo:      tenant,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.NoError(t, err)
	assert.Equal(t, "cipher-access-2", locked.AccessTokenCiphertext)
	assert.Equal(t, now+7200, locked.AccessTokenExpiresAt)
	require.NotNil(t, locked.LastRefreshedAt)
	assert.Equal(t, refreshed, *locked.LastRefreshedAt)

	require.NoError(t, repo.StoreTokens(ctx, repositories.StoreAccountingTokensRequest{
		TenantInfo: tenant,
		ID:         created.ID,
		At:         refreshed + 1,
	}))
	wiped, err := repo.LockWithTokens(ctx, repositories.GetAccountingConnectionByIDRequest{
		TenantInfo: tenant,
		ID:         created.ID,
	})
	require.NoError(t, err)
	assert.Empty(t, wiped.AccessTokenCiphertext)
	assert.Empty(t, wiped.RefreshTokenCiphertext)
}

func TestConnectionRepository_RealmIsHeldByOneTenantAtATime(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	first := seedtest.SeedFullTestData(t, ctx, db)
	second := seedtest.SeedAdditionalTenant(t, ctx, db, "QB")
	repo := NewConnectionRepository(ConnectionParams{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	firstTenant := pagination.TenantInfo{OrgID: first.Organization.ID, BuID: first.BusinessUnit.ID}
	secondTenant := pagination.TenantInfo{OrgID: second.Organization.ID, BuID: second.BusinessUnit.ID}
	now := timeutils.NowUnix()

	held, err := repo.Create(ctx, newConnection(firstTenant, first.User.ID, realm, now))
	require.NoError(t, err)

	holders, err := repo.ListHoldingRealm(ctx, repositories.ListAccountingConnectionsByRealmRequest{
		IntegrationType: integration.TypeQuickBooksOnline,
		RealmIDs:        []string{realm},
	})
	require.NoError(t, err)
	require.Len(t, holders, 1)
	assert.Equal(t, firstTenant.OrgID, holders[0].OrganizationID)
	assert.Empty(t, holders[0].AccessTokenCiphertext)

	_, err = repo.Create(ctx, newConnection(secondTenant, second.User.ID, realm, now))
	require.Error(t, err)
	assert.True(t, dberror.IsUniqueConstraintViolation(err))

	held.Disconnect(first.User.ID, now+5)
	_, err = repo.Update(ctx, held)
	require.NoError(t, err)

	_, err = repo.Create(ctx, newConnection(secondTenant, second.User.ID, realm, now+6))
	require.NoError(t, err, "a disconnected connection releases its company")

	_, err = repo.GetByID(ctx, repositories.GetAccountingConnectionByIDRequest{
		TenantInfo: secondTenant,
		ID:         held.ID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), "one tenant cannot read another's connection")

	firstOnly, err := repo.ListByTenant(ctx, firstTenant)
	require.NoError(t, err)
	require.Len(t, firstOnly, 1)
	assert.Equal(t, held.ID, firstOnly[0].ID)
}

func TestConnectionRepository_HealthQueueWebhooksAndVersioning(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := NewConnectionRepository(ConnectionParams{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	now := timeutils.NowUnix()

	created, err := repo.Create(ctx, newConnection(tenant, data.User.ID, realm, now))
	require.NoError(t, err)

	due, err := repo.ListDueForHealthCheck(ctx, repositories.ListDueAccountingConnectionsRequest{
		CheckedBefore: now - 60,
		Limit:         10,
	})
	require.NoError(t, err)
	assert.Empty(t, due, "checked just now")

	due, err = repo.ListDueForHealthCheck(ctx, repositories.ListDueAccountingConnectionsRequest{
		CheckedBefore: now + 60,
		Limit:         10,
	})
	require.NoError(t, err)
	require.Len(t, due, 1)
	assert.Empty(t, due[0].RefreshTokenCiphertext)

	updated, err := repo.MarkWebhookReceived(ctx, repositories.MarkAccountingWebhookRequest{
		IntegrationType: integration.TypeQuickBooksOnline,
		RealmIDs:        []string{realm, "someone-else"},
		ReceivedAt:      now + 3,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), updated)

	fresh, err := repo.GetByID(ctx, repositories.GetAccountingConnectionByIDRequest{TenantInfo: tenant, ID: created.ID})
	require.NoError(t, err)
	require.NotNil(t, fresh.LastWebhookAt)
	assert.Equal(t, now+3, *fresh.LastWebhookAt)

	stale := *fresh
	_, err = repo.Update(ctx, fresh)
	require.NoError(t, err)
	_, err = repo.Update(ctx, &stale)
	require.Error(t, err, "an update from a stale version is refused")

	fresh.Disconnect(data.User.ID, now+10)
	_, err = repo.Update(ctx, fresh)
	require.NoError(t, err)
	due, err = repo.ListDueForHealthCheck(ctx, repositories.ListDueAccountingConnectionsRequest{
		CheckedBefore: now + 60,
		Limit:         10,
	})
	require.NoError(t, err)
	assert.Empty(t, due, "disconnected connections are never checked")
}

func TestConnectionRepository_ReferenceRefreshRunsUntilItFinishesEitherWay(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := NewConnectionRepository(ConnectionParams{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	now := timeutils.NowUnix()

	created, err := repo.Create(ctx, newConnection(tenant, data.User.ID, realm, now))
	require.NoError(t, err)
	get := func() *accountingsync.AccountingConnection {
		conn, getErr := repo.GetByID(ctx, repositories.GetAccountingConnectionByIDRequest{TenantInfo: tenant, ID: created.ID})
		require.NoError(t, getErr)
		return conn
	}
	mark := func(req repositories.MarkAccountingReferenceRefreshRequest) {
		req.TenantInfo = tenant
		req.ID = created.ID
		require.NoError(t, repo.MarkReferenceRefresh(ctx, req))
	}

	started := now
	mark(repositories.MarkAccountingReferenceRefreshRequest{StartedAt: &started})
	require.NotNil(t, get().ReferenceRefreshStartedAt)

	mark(repositories.MarkAccountingReferenceRefreshRequest{Error: "QuickBooks did not answer"})
	failed := get()
	assert.Nil(t, failed.ReferenceRefreshStartedAt, "a failed refresh is no longer running")
	assert.Equal(t, "QuickBooks did not answer", failed.ReferenceRefreshError)
	assert.Nil(t, failed.ReferenceRefreshedAt)

	restarted := now + 10
	mark(repositories.MarkAccountingReferenceRefreshRequest{StartedAt: &restarted})
	running := get()
	require.NotNil(t, running.ReferenceRefreshStartedAt)
	assert.Empty(t, running.ReferenceRefreshError, "starting again clears the last failure")

	finished := now + 20
	mark(repositories.MarkAccountingReferenceRefreshRequest{RefreshedAt: &finished})
	running.SetupStep = accountingsync.SetupStepComplete
	_, err = repo.Update(ctx, running)
	require.NoError(t, err, "a copy read while the refresh ran still saves")
	done := get()
	assert.Equal(t, accountingsync.SetupStepComplete, done.SetupStep)
	assert.Nil(t, done.ReferenceRefreshStartedAt)
	require.NotNil(t, done.ReferenceRefreshedAt)
	assert.Equal(t, finished, *done.ReferenceRefreshedAt)
	assert.Empty(t, done.ReferenceRefreshError)
}
