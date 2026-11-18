package supportaccessnotice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type created struct {
	items []*notification.Notification
}

func (c *created) Create(_ context.Context, n *notification.Notification) (*notification.Notification, error) {
	c.items = append(c.items, n)
	return n, nil
}

type holders struct {
	users []permissionports.PermittedUser
}

func (h *holders) ListUsersWithPermission(
	context.Context,
	permissionports.ListUsersWithPermissionRequest,
) ([]permissionports.PermittedUser, error) {
	return h.users, nil
}

func noticeFor(kind supportaccessports.NoticeKind, actor pulid.ID) *supportaccessports.Notice {
	revokedAt := int64(1_800_000_100)
	return &supportaccessports.Notice{
		Kind:       kind,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		ActorID:    actor,
		StaffName:  "Jane Doe",
		Grant: &supportaccess.Grant{
			ID:           pulid.MustNew(supportaccess.GrantIDPrefix),
			StaffEmail:   "jane.doe@trenova.app",
			Scope:        supportaccess.ScopeFull,
			ExpiresAt:    1_800_014_400,
			RevokedAt:    &revokedAt,
			RevokeReason: "Fixed",
		},
	}
}

func TestSupportAccessChanged_TellsEveryAdministratorButWhoActedAndNeverSupport(t *testing.T) {
	t.Parallel()

	actor, other := pulid.MustNew("usr_"), pulid.MustNew("usr_")
	sink := &created{}
	n := &Notifier{
		recipients: &holders{users: []permissionports.PermittedUser{
			{UserID: actor},
			{UserID: other},
			{UserID: pulid.MustNew(supportaccess.PrincipalIDPrefix)},
		}},
		notifications: sink,
	}

	require.NoError(t, n.SupportAccessChanged(t.Context(), noticeFor(supportaccessports.NoticeGranted, actor)))
	require.Len(t, sink.items, 1)
	got := sink.items[0]
	assert.Equal(t, other, *got.TargetUserID)
	assert.Equal(t, EventGranted, got.EventType)
	assert.Equal(t, "jane.doe@trenova.app has full access to this organization until Jan 15, 12:00 UTC.", got.Message)
	assert.Equal(t, SupportAccessURL, got.Data["link"])
}

func TestSupportAccessChanged_SaysWhatHappenedForEachKind(t *testing.T) {
	t.Parallel()

	for kind, want := range map[supportaccessports.NoticeKind]struct {
		event    string
		priority notification.Priority
		message  string
	}{
		supportaccessports.NoticeExtended: {EventExtended, notification.PriorityMedium,
			"jane.doe@trenova.app now has access until Jan 15, 12:00 UTC."},
		supportaccessports.NoticeFirstEntry: {EventFirstEntry, notification.PriorityHigh,
			"Jane Doe (jane.doe@trenova.app) is in this organization with full access. It ends by Jan 15, 12:00 UTC."},
		supportaccessports.NoticeRevoked: {EventEnded, notification.PriorityMedium,
			"Access for jane.doe@trenova.app was ended: Fixed"},
		supportaccessports.NoticeExpired: {EventEnded, notification.PriorityLow,
			"Access for jane.doe@trenova.app ended on its own at Jan 15, 12:00 UTC."},
	} {
		sink := &created{}
		n := &Notifier{
			recipients:    &holders{users: []permissionports.PermittedUser{{UserID: pulid.MustNew("usr_")}}},
			notifications: sink,
		}
		require.NoError(t, n.SupportAccessChanged(t.Context(), noticeFor(kind, pulid.Nil)), kind)
		require.Len(t, sink.items, 1, kind)
		assert.Equal(t, want.event, sink.items[0].EventType, kind)
		assert.Equal(t, want.priority, sink.items[0].Priority, kind)
		assert.Equal(t, want.message, sink.items[0].Message, kind)
		assert.NotContains(t, sink.items[0].Message, "sag_", "no record id reaches the reader")
	}
}
