//go:build integration

package subscriptionrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSubscriptionRepository_Lifecycle(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	now := time.Now().Unix()

	_, err := repo.GetByOrganization(ctx, repositories.GetSubscriptionRequest{TenantInfo: tenant})
	require.True(t, errortypes.IsNotFoundError(err))

	created, err := repo.Create(ctx, &subscription.Subscription{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		PlanKey:        "free_demo",
		Status:         subscription.StatusTrialing,
		TrialEndsAt:    now - 10,
		ReadOnlyUntil:  now + 3_600,
	})
	require.NoError(t, err)
	require.False(t, created.ID.IsNil())

	read, err := repo.GetByOrganization(ctx, repositories.GetSubscriptionRequest{TenantInfo: tenant})
	require.NoError(t, err)
	assert.Equal(t, created.ID, read.ID)

	due, err := repo.ListDue(ctx, &repositories.ListDueSubscriptionsRequest{Now: now})
	require.NoError(t, err)
	require.Len(t, due, 1)

	count, err := repo.CountByStatus(ctx, &repositories.CountSubscriptionsByStatusRequest{
		Statuses: []subscription.Status{subscription.StatusTrialing, subscription.StatusReadOnly},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	updated, err := repo.UpdateStatus(ctx, &repositories.UpdateSubscriptionStatusRequest{
		TenantInfo: tenant,
		ID:         read.ID,
		Version:    read.Version,
		Status:     subscription.StatusReadOnly,
	})
	require.NoError(t, err)
	assert.Equal(t, subscription.StatusReadOnly, updated.Status)
	assert.Equal(t, read.Version+1, updated.Version)

	_, err = repo.UpdateStatus(ctx, &repositories.UpdateSubscriptionStatusRequest{
		TenantInfo: tenant,
		ID:         read.ID,
		Version:    read.Version,
		Status:     subscription.StatusExpired,
	})
	require.True(t, errortypes.IsVersionMismatchError(err))

	due, err = repo.ListDue(ctx, &repositories.ListDueSubscriptionsRequest{Now: now})
	require.NoError(t, err)
	assert.Empty(t, due, "a read-only organization is not due until its grace ends")

	_, err = repo.Create(ctx, &subscription.Subscription{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		PlanKey:        "free_demo",
		Status:         subscription.StatusTrialing,
		TrialEndsAt:    now,
		ReadOnlyUntil:  now,
	})
	require.Error(t, err, "an organization holds one subscription")
}
