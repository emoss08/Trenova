//go:build integration

package pushsubscriptionrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSaveNeverReassignsAnotherUsersSubscription(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.NewUser(data.Organization.ID, data.BusinessUnit.ID).
		WithUsername("push_other_user").
		WithEmail("push-other@example.com").
		Build(t, ctx, db)
	ctx = data.Ctx(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	owner := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}
	const endpoint = "https://fcm.googleapis.com/fcm/send/owner-device"
	first, err := repo.Save(ctx, &notificationports.SavePushSubscriptionRequest{
		TenantInfo: owner,
		Endpoint:   endpoint,
		P256dh:     "owner-key",
		Auth:       "owner-auth",
	})
	require.NoError(t, err)

	refreshed, err := repo.Save(ctx, &notificationports.SavePushSubscriptionRequest{
		TenantInfo: owner,
		Endpoint:   endpoint,
		P256dh:     "owner-key-rotated",
		Auth:       "owner-auth-rotated",
	})
	require.NoError(t, err)
	require.Equal(t, first.ID, refreshed.ID)

	_, err = repo.Save(ctx, &notificationports.SavePushSubscriptionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  data.Organization.ID,
			BuID:   data.BusinessUnit.ID,
			UserID: other.ID,
		},
		Endpoint: endpoint,
		P256dh:   "attacker-key",
		Auth:     "attacker-auth",
	})
	var conflict *errortypes.ConflictError
	require.ErrorAs(t, err, &conflict)

	ownerSubscriptions, err := repo.ListByUser(ctx, data.User.ID)
	require.NoError(t, err)
	require.Len(t, ownerSubscriptions, 1)
	require.Equal(t, "owner-key-rotated", ownerSubscriptions[0].P256dh)

	otherSubscriptions, err := repo.ListByUser(ctx, other.ID)
	require.NoError(t, err)
	require.Empty(t, otherSubscriptions)
}
