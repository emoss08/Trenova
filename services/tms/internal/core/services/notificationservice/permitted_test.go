package notificationservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingNotificationRepo struct {
	repositories.NotificationRepository

	created []*notification.Notification
	recent  bool
	asked   *repositories.ExistsRecentNotificationRequest
}

func (r *recordingNotificationRepo) Create(
	_ context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	r.created = append(r.created, entity)
	return entity, nil
}

func (r *recordingNotificationRepo) ExistsRecent(
	_ context.Context,
	req repositories.ExistsRecentNotificationRequest,
) (bool, error) {
	r.asked = &req
	return r.recent, nil
}

type permittedRoles struct {
	repositories.RoleRepository

	users []repositories.PermittedUser
	asked *repositories.ListUsersWithPermissionRequest
}

func (r *permittedRoles) ListUsersWithPermission(
	_ context.Context,
	req repositories.ListUsersWithPermissionRequest,
) ([]repositories.PermittedUser, error) {
	r.asked = &req
	return r.users, nil
}

type silentRealtime struct {
	servicesport.RealtimeService
}

func (silentRealtime) PublishResourceInvalidation(
	context.Context,
	*servicesport.PublishResourceInvalidationRequest,
) error {
	return nil
}

func permittedService(repo *recordingNotificationRepo, roles *permittedRoles) *Service {
	return New(Params{Logger: zap.NewNop(), Repo: repo, Realtime: silentRealtime{}, Roles: roles})
}

func permittedRequest(tenant pagination.TenantInfo) NotifyPermittedRequest {
	return NotifyPermittedRequest{
		Tenant:    tenant,
		Resource:  permission.ResourceInboundMessage,
		Operation: permission.OpRead,
		Limit:     2,
		Now:       1_800_000_000,
		Notification: notification.Notification{
			EventType:     "inbound_message.needs_review",
			Title:         "A message needs review",
			Message:       "Load for Thursday",
			Priority:      notification.PriorityMedium,
			Source:        "inbox",
			CorrelationID: new("imsg_1"),
		},
	}
}

func TestNotifyPermitted_TellsEachPermittedUserOnceWithinTheLimit(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	roles := &permittedRoles{users: []repositories.PermittedUser{
		{UserID: pulid.MustNew("usr_")},
		{UserID: pulid.MustNew("usr_")},
		{UserID: pulid.MustNew("usr_")},
	}}
	repo := &recordingNotificationRepo{}

	sent, err := permittedService(
		repo,
		roles,
	).NotifyPermitted(t.Context(), permittedRequest(tenant))
	require.NoError(t, err)

	require.NotNil(t, roles.asked)
	assert.Equal(t, permission.ResourceInboundMessage, roles.asked.Resource)
	assert.Equal(t, tenant.OrgID, roles.asked.OrganizationID)
	assert.Equal(t, 2, sent)
	require.Len(t, repo.created, 2, "the limit bounds who is told")
	for i, created := range repo.created {
		require.NotNil(t, created.TargetUserID)
		assert.Equal(t, roles.users[i].UserID, *created.TargetUserID)
		assert.Equal(t, notification.ChannelUser, created.Channel)
		assert.Equal(t, tenant.OrgID, created.OrganizationID)
		assert.Equal(t, tenant.BuID, *created.BusinessUnitID)
		assert.Equal(t, "A message needs review", created.Title)
	}
	assert.NotSame(t, repo.created[0], repo.created[1], "each recipient gets their own row")
}

// A message reprocessed after a redelivery must not tell everybody twice.
func TestNotifyPermitted_SaysNothingWhenTheSameThingWasJustSaid(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	roles := &permittedRoles{users: []repositories.PermittedUser{{UserID: pulid.MustNew("usr_")}}}
	repo := &recordingNotificationRepo{recent: true}
	req := permittedRequest(tenant)
	req.DedupeSince = req.Now - 3600

	sent, err := permittedService(repo, roles).NotifyPermitted(t.Context(), req)
	require.NoError(t, err)

	assert.Zero(t, sent)
	assert.Empty(t, repo.created)
	require.NotNil(t, repo.asked)
	assert.Equal(t, "imsg_1", repo.asked.CorrelationID)
	assert.Nil(t, roles.asked, "nobody is looked up for a notice that is not going")
}
