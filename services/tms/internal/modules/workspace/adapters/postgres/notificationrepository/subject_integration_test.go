//go:build integration

package notificationrepository

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type subjectRig struct {
	t      *testing.T
	ctx    context.Context
	repo   notificationports.NotificationRepository
	tenant pagination.TenantInfo
	now    int64
}

func newSubjectRig(t *testing.T) *subjectRig {
	t.Helper()
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)

	return &subjectRig{
		t:    t,
		ctx:  data.Ctx(ctx),
		repo: New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()}),
		tenant: pagination.TenantInfo{
			OrgID:  data.Organization.ID,
			BuID:   data.BusinessUnit.ID,
			UserID: data.User.ID,
		},
		now: time.Now().Unix(),
	}
}

func (r *subjectRig) notice(subject, title string, reason notification.Reason) *notification.Notification {
	buID, userID := r.tenant.BuID, r.tenant.UserID

	return &notification.Notification{
		OrganizationID: r.tenant.OrgID,
		BusinessUnitID: &buID,
		TargetUserID:   &userID,
		SubjectKey:     subject,
		Reason:         reason,
		EventType:      "watchtower.assigned",
		Priority:       notification.PriorityHigh,
		Channel:        notification.ChannelUser,
		Title:          title,
		Message:        title + " needs a look",
		DeliveryStatus: notification.DeliveryStatusPending,
		Source:         "watchtower",
	}
}

func (r *subjectRig) create(n *notification.Notification) *notification.Notification {
	r.t.Helper()
	created, err := r.repo.Create(r.ctx, n)
	require.NoError(r.t, err)

	return created
}

func (r *subjectRig) action(ids ...pulid.ID) notificationports.NotificationActionRequest {
	return notificationports.NotificationActionRequest{IDs: ids, TenantInfo: r.tenant}
}

func (r *subjectRig) list(lens notification.Lens) []*notification.Notification {
	r.t.Helper()
	result, err := r.repo.ListConnection(r.ctx, &notificationports.ListNotificationConnectionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: r.tenant,
			Pagination: pagination.Info{Limit: 50},
		},
		Cursor: pagination.CursorInfo{Limit: 50},
		Lens:   lens,
		Now:    time.Now().Unix(),
	})
	require.NoError(r.t, err)

	return result.Items
}

func (r *subjectRig) titles(lens notification.Lens) []string {
	r.t.Helper()
	rows := r.list(lens)
	titles := make([]string, 0, len(rows))
	for _, row := range rows {
		titles = append(titles, row.Title)
	}

	return titles
}

func (r *subjectRig) counts() notificationports.NotificationCounts {
	r.t.Helper()
	counts, err := r.repo.Counts(r.ctx, notificationports.NotificationCountsRequest{
		TenantInfo: r.tenant,
		Now:        time.Now().Unix(),
	})
	require.NoError(r.t, err)

	return *counts
}

func TestCreate_ASecondNoticeAboutAnOpenSubjectJoinsItsRowUnreadAndCounted(t *testing.T) {
	rig := newSubjectRig(t)
	subject := notification.SubjectKey("watchtower", "item", "wti_1")

	first := rig.create(rig.notice(subject, "Missed pickup", notification.ReasonDirect).
		WithOccurrence(notification.Occurrence{Label: "Reno", At: rig.now}))
	require.NoError(t, rig.repo.MarkAsRead(rig.ctx, rig.action(first.ID)))

	second := rig.create(rig.notice(subject, "Missed pickup, now overdue", notification.ReasonDirect).
		WithOccurrence(notification.Occurrence{Label: "Reno, escalated", At: rig.now + 60}))

	assert.Equal(t, first.ID, second.ID, "the subject keeps its one open row")
	rows := rig.list(notification.LensForYou)
	require.Len(t, rows, 1)
	assert.Equal(t, "Missed pickup, now overdue", rows[0].Title)
	assert.Equal(t, 2, rows[0].OccurrenceCount)
	assert.Nil(t, rows[0].ReadAt, "coming up again makes it unread again")

	recent, ok := rows[0].Data[notification.DataKeyRecent].([]any)
	require.True(t, ok)
	require.Len(t, recent, 2)
	assert.Equal(t, "Reno, escalated", recent[0].(map[string]any)["label"])
	assert.Equal(t, "Reno", recent[1].(map[string]any)["label"])
}

func TestCreate_ANoticeAboutAFinishedSubjectStartsANewRow(t *testing.T) {
	rig := newSubjectRig(t)
	subject := notification.SubjectKey("watchtower", "item", "wti_1")

	first := rig.create(rig.notice(subject, "Missed pickup", notification.ReasonDirect))
	require.NoError(t, rig.repo.Dismiss(rig.ctx, rig.action(first.ID)))

	second := rig.create(rig.notice(subject, "Missed pickup", notification.ReasonDirect))

	assert.NotEqual(t, first.ID, second.ID)
	assert.Equal(t, 1, second.OccurrenceCount)
	assert.Len(t, rig.list(notification.LensForYou), 1)
	assert.Len(t, rig.list(notification.LensDone), 1)
}

func TestCreate_NoticesWithoutASubjectEachGetARow(t *testing.T) {
	rig := newSubjectRig(t)

	rig.create(rig.notice("", "Tender accepted", notification.ReasonDirect))
	rig.create(rig.notice("", "Tender accepted", notification.ReasonDirect))

	assert.Len(t, rig.list(notification.LensForYou), 2)
}

func TestLenses_EachRowIsInOnePartOfThePanelAndTheCountsAgree(t *testing.T) {
	rig := newSubjectRig(t)

	rig.create(rig.notice("", "Yours", notification.ReasonDirect))
	read := rig.create(rig.notice("", "Yours, read", notification.ReasonDirect))
	rig.create(rig.notice("", "The team's", notification.ReasonTeam))
	aside := rig.create(rig.notice("", "Set aside", notification.ReasonDirect))
	done := rig.create(rig.notice("", "Finished", notification.ReasonTeam))

	require.NoError(t, rig.repo.MarkAsRead(rig.ctx, rig.action(read.ID)))
	require.NoError(t, rig.repo.Snooze(rig.ctx, notificationports.SnoozeNotificationsRequest{
		NotificationActionRequest: rig.action(aside.ID),
		Until:                     rig.now + 3_600,
	}))
	require.NoError(t, rig.repo.Dismiss(rig.ctx, rig.action(done.ID)))

	assert.ElementsMatch(t, []string{"Yours", "Yours, read"}, rig.titles(notification.LensForYou))
	assert.Equal(t, []string{"The team's"}, rig.titles(notification.LensTeam))
	assert.Equal(t, []string{"Set aside"}, rig.titles(notification.LensLater))
	assert.Equal(t, []string{"Finished"}, rig.titles(notification.LensDone))
	assert.ElementsMatch(
		t,
		[]string{"Yours", "Yours, read", "The team's"},
		rig.titles(notification.LensOpen),
	)
	assert.Equal(
		t,
		notificationports.NotificationCounts{
			ForYou:       2,
			ForYouUnread: 1,
			Team:         1,
			TeamUnread:   1,
			Later:        1,
		},
		rig.counts(),
	)
}

func TestSnooze_SetsARowAsideUnreadAndWakeBringsItBack(t *testing.T) {
	rig := newSubjectRig(t)
	row := rig.create(rig.notice("", "Projected late", notification.ReasonDirect))
	require.NoError(t, rig.repo.MarkAsRead(rig.ctx, rig.action(row.ID)))

	require.NoError(t, rig.repo.Snooze(rig.ctx, notificationports.SnoozeNotificationsRequest{
		NotificationActionRequest: rig.action(row.ID),
		Until:                     rig.now + 7_200,
	}))

	later := rig.list(notification.LensLater)
	require.Len(t, later, 1)
	assert.Equal(t, rig.now+7_200, later[0].SurfacesAt)
	assert.Nil(t, later[0].ReadAt, "it comes back unread")
	assert.Empty(t, rig.list(notification.LensForYou))

	require.NoError(t, rig.repo.Wake(rig.ctx, rig.action(row.ID)))

	assert.Empty(t, rig.list(notification.LensLater))
	assert.Equal(t, []string{"Projected late"}, rig.titles(notification.LensForYou))
}

func TestCreate_ARowSetAsideAbsorbsRoutineNewsWithoutSurfacing(t *testing.T) {
	rig := newSubjectRig(t)
	subject := notification.SubjectKey("agent", "proposals", "agt_1")
	row := rig.create(rig.notice(subject, "Changes are waiting", notification.ReasonTeam))
	require.NoError(t, rig.repo.Snooze(rig.ctx, notificationports.SnoozeNotificationsRequest{
		NotificationActionRequest: rig.action(row.ID),
		Until:                     rig.now + 7_200,
	}))

	rig.create(rig.notice(subject, "More changes are waiting", notification.ReasonTeam))

	assert.Equal(t, []string{"More changes are waiting"}, rig.titles(notification.LensLater))
	assert.Empty(t, rig.list(notification.LensTeam))
}

func TestDismissAll_FinishesOnePartOfThePanelAndLeavesTheRest(t *testing.T) {
	rig := newSubjectRig(t)
	rig.create(rig.notice("", "Yours", notification.ReasonDirect))
	rig.create(rig.notice("", "The team's first", notification.ReasonTeam))
	rig.create(rig.notice("", "The team's second", notification.ReasonTeam))

	require.NoError(t, rig.repo.DismissAll(rig.ctx, notificationports.DismissAllNotificationsRequest{
		TenantInfo: rig.tenant,
		Lens:       notification.LensTeam,
		Now:        time.Now().Unix(),
	}))

	assert.Empty(t, rig.list(notification.LensTeam))
	assert.Equal(t, []string{"Yours"}, rig.titles(notification.LensForYou))
	assert.ElementsMatch(
		t,
		[]string{"The team's first", "The team's second"},
		rig.titles(notification.LensDone),
	)
	for _, row := range rig.list(notification.LensDone) {
		assert.NotNil(t, row.ReadAt, "a finished row is read")
		assert.Nil(t, row.ResolvedAt, "the person finished it; it did not clear itself")
	}
}

func TestResolveSubject_ClearsTheOpenRowsOfASubjectSayingHowAndRestoreForgetsIt(t *testing.T) {
	rig := newSubjectRig(t)
	subject := notification.SubjectKey("watchtower", "item", "wti_1")
	row := rig.create(rig.notice(subject, "Missed pickup", notification.ReasonDirect))
	rig.create(rig.notice(notification.SubjectKey("watchtower", "item", "wti_2"), "POD missing", notification.ReasonDirect))

	resolved, err := rig.repo.ResolveSubject(rig.ctx, notificationports.ResolveNotificationSubjectRequest{
		OrganizationID: rig.tenant.OrgID,
		BusinessUnitID: rig.tenant.BuID,
		SubjectKey:     subject,
		Resolution:     notification.ResolutionResolved,
		Now:            rig.now,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(1), resolved)
	assert.Equal(t, []string{"POD missing"}, rig.titles(notification.LensForYou))
	done := rig.list(notification.LensDone)
	require.Len(t, done, 1)
	assert.Equal(t, notification.ResolutionResolved, done[0].Resolution)
	require.NotNil(t, done[0].ResolvedAt)
	assert.Equal(t, rig.now, *done[0].ResolvedAt)
	assert.NotNil(t, done[0].ReadAt)

	again, err := rig.repo.ResolveSubject(rig.ctx, notificationports.ResolveNotificationSubjectRequest{
		OrganizationID: rig.tenant.OrgID,
		BusinessUnitID: rig.tenant.BuID,
		SubjectKey:     subject,
		Resolution:     notification.ResolutionResolved,
		Now:            rig.now,
	})
	require.NoError(t, err)
	assert.Zero(t, again, "a subject with no open row has nothing to clear")

	require.NoError(t, rig.repo.Restore(rig.ctx, rig.action(row.ID)))

	restored := rig.list(notification.LensForYou)
	require.Len(t, restored, 2)
	for _, entry := range restored {
		assert.Empty(t, entry.Resolution)
		assert.Nil(t, entry.ResolvedAt)
		assert.Nil(t, entry.DismissedAt)
	}
}

func TestListConnection_DoneReadsNewestFinishedFirst(t *testing.T) {
	rig := newSubjectRig(t)
	rig.create(rig.notice("", "Surfaced first, finished last", notification.ReasonTeam))
	rig.create(rig.notice("", "Surfaced last, finished first", notification.ReasonDirect))

	for _, finish := range []struct {
		lens notification.Lens
		at   int64
	}{
		{notification.LensForYou, rig.now + 10},
		{notification.LensTeam, rig.now + 20},
	} {
		require.NoError(t, rig.repo.DismissAll(rig.ctx, notificationports.DismissAllNotificationsRequest{
			TenantInfo: rig.tenant,
			Lens:       finish.lens,
			Now:        finish.at,
		}))
	}

	assert.Equal(
		t,
		[]string{"Surfaced first, finished last", "Surfaced last, finished first"},
		rig.titles(notification.LensDone),
	)
}
