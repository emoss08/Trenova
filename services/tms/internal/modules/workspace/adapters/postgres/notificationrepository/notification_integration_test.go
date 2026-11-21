//go:build integration

package notificationrepository

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNotifiedUsers_NamesOnlyThePeopleToldOfThatEventSinceItHappened(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})

	buID := data.BusinessUnit.ID
	userID := data.User.ID
	since := time.Now().Add(-time.Minute).Unix()
	create := func(eventType, correlation string, target *pulid.ID) {
		t.Helper()
		channel := notification.ChannelUser
		if target == nil {
			channel = notification.ChannelGlobal
		}
		_, err := repo.Create(ctx, &notification.Notification{
			OrganizationID: data.Organization.ID,
			BusinessUnitID: &buID,
			TargetUserID:   target,
			EventType:      eventType,
			Priority:       notification.PriorityHigh,
			Channel:        channel,
			Title:          "Role changed",
			Message:        "A role was assigned",
			DeliveryStatus: notification.DeliveryStatusPending,
			Source:         "auditalertservice",
			CorrelationID:  &correlation,
		})
		require.NoError(t, err)
	}
	create("audit.alert.RoleChanged", "ae_1", &userID)
	create("audit.alert.RoleChanged", "ae_2", &userID)
	create("audit.alert.APIKeyChanged", "ae_1", &userID)
	create("audit.alert.RoleChanged", "ae_1", nil)

	notified, err := repo.NotifiedUsers(ctx, notificationports.ExistsRecentNotificationRequest{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: buID,
		EventType:      "audit.alert.RoleChanged",
		CorrelationID:  "ae_1",
		Since:          since,
	})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{userID}, notified)

	later, err := repo.NotifiedUsers(ctx, notificationports.ExistsRecentNotificationRequest{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: buID,
		EventType:      "audit.alert.RoleChanged",
		CorrelationID:  "ae_1",
		Since:          time.Now().Add(time.Hour).Unix(),
	})
	require.NoError(t, err)
	assert.Empty(t, later, "a notification sent before the event does not count")
}

func TestARecipientIsCheckedOnceAndAnotherTenantsPersonIsNeverWritten(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "NR")
	ctx = data.Ctx(ctx)
	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()}).(*repository)

	buID := data.BusinessUnit.ID
	notice := func(target pulid.ID) *notification.Notification {
		return &notification.Notification{
			OrganizationID: data.Organization.ID,
			BusinessUnitID: &buID,
			TargetUserID:   &target,
			EventType:      "audit.alert.RoleChanged",
			Priority:       notification.PriorityHigh,
			Channel:        notification.ChannelUser,
			Title:          "Role changed",
			Message:        "A role was assigned",
			DeliveryStatus: notification.DeliveryStatusPending,
			Source:         "auditalertservice",
		}
	}

	_, err := repo.Create(ctx, notice(other.User.ID))
	require.True(t, errors.Is(err, errRecipientOutside), "%v", err)
	_, cached := repo.recipients.GetIfPresent(
		recipientKey{user: other.User.ID, organization: data.Organization.ID})
	assert.False(t, cached, "a refusal is never kept")

	_, err = repo.Create(ctx, notice(data.User.ID))
	require.NoError(t, err)
	_, cached = repo.recipients.GetIfPresent(
		recipientKey{user: data.User.ID, organization: data.Organization.ID})
	assert.True(t, cached, "a person's business unit never changes, so the answer is kept")
	_, err = repo.Create(ctx, notice(data.User.ID))
	require.NoError(t, err)
}
