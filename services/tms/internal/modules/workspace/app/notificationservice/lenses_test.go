package notificationservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type lensRepo struct {
	notificationports.NotificationRepository

	snoozed      *notificationports.SnoozeNotificationsRequest
	dismissedAll *notificationports.DismissAllNotificationsRequest
	listed       *notificationports.ListNotificationConnectionRequest
	counted      *notificationports.NotificationCountsRequest
	resolveAsked *notificationports.ResolveNotificationSubjectRequest
	resolved     int64
}

func (r *lensRepo) Snooze(
	_ context.Context,
	req notificationports.SnoozeNotificationsRequest,
) error {
	r.snoozed = &req
	return nil
}

func (r *lensRepo) DismissAll(
	_ context.Context,
	req notificationports.DismissAllNotificationsRequest,
) error {
	r.dismissedAll = &req
	return nil
}

func (r *lensRepo) Counts(
	_ context.Context,
	req notificationports.NotificationCountsRequest,
) (*notificationports.NotificationCounts, error) {
	r.counted = &req
	return &notificationports.NotificationCounts{ForYou: 3, ForYouUnread: 2, Team: 5}, nil
}

func (r *lensRepo) ResolveSubject(
	_ context.Context,
	req notificationports.ResolveNotificationSubjectRequest,
) (int64, error) {
	r.resolveAsked = &req
	return r.resolved, nil
}

type recordedRealtime struct {
	realtimeports.RealtimeService

	published []*realtimeports.PublishResourceInvalidationRequest
}

func (r *recordedRealtime) PublishResourceInvalidation(
	_ context.Context,
	req *realtimeports.PublishResourceInvalidationRequest,
) error {
	r.published = append(r.published, req)
	return nil
}

var lensNow = time.Date(2026, time.November, 13, 15, 0, 0, 0, time.UTC)

func lensService(repo *lensRepo, realtime realtimeports.RealtimeService) *Service {
	svc := New(Params{
		Logger:    zap.NewNop(),
		Repo:      repo,
		Realtime:  realtime,
		AppConfig: &appconfig.Config{},
		WebPush:   disabledWebPush(),
		Users:     &memberDirectory{members: map[pulid.ID]*tenant.User{}},
	})
	svc.now = func() time.Time { return lensNow }

	return svc
}

func lensTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func requireInvalid(t *testing.T, err error, field string) {
	t.Helper()
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, field, validation.Field())
	assert.Equal(t, errortypes.ErrInvalid, validation.Code)
}

func snoozeRequest(until time.Time) notificationports.SnoozeNotificationsRequest {
	return notificationports.SnoozeNotificationsRequest{
		NotificationActionRequest: notificationports.NotificationActionRequest{
			IDs:        []pulid.ID{pulid.MustNew("notif_")},
			TenantInfo: lensTenant(),
		},
		Until: until.Unix(),
	}
}

func TestSnoozeSetsANotificationAsideUntilATimeWithinThirtyDays(t *testing.T) {
	t.Parallel()

	repo := &lensRepo{}
	req := snoozeRequest(lensNow.Add(18 * time.Hour))

	require.NoError(t, lensService(repo, silentRealtime{}).Snooze(t.Context(), req))

	require.NotNil(t, repo.snoozed)
	assert.Equal(t, req, *repo.snoozed)
}

func TestSnoozeRefusesATimeThatHasPassedOrIsMoreThanThirtyDaysOff(t *testing.T) {
	t.Parallel()

	for name, until := range map[string]time.Time{
		"now":             lensNow,
		"an hour ago":     lensNow.Add(-time.Hour),
		"thirty-one days": lensNow.Add(31 * 24 * time.Hour),
	} {
		repo := &lensRepo{}

		err := lensService(repo, silentRealtime{}).Snooze(t.Context(), snoozeRequest(until))

		requireInvalid(t, err, "until")
		assert.Nil(t, repo.snoozed, name)
	}
}

func TestDismissAllMarksOnePartOfThePanelDoneAsOfNow(t *testing.T) {
	t.Parallel()

	repo := &lensRepo{}
	tenantInfo := lensTenant()

	err := lensService(repo, silentRealtime{}).DismissAll(
		t.Context(),
		notificationports.DismissAllNotificationsRequest{
			TenantInfo: tenantInfo,
			Lens:       notification.LensTeam,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, repo.dismissedAll)
	assert.Equal(t, notification.LensTeam, repo.dismissedAll.Lens)
	assert.Equal(t, tenantInfo, repo.dismissedAll.TenantInfo)
	assert.Equal(t, lensNow.Unix(), repo.dismissedAll.Now)
}

func TestDismissAllRefusesDoneAndAnythingThatIsNotAPartOfThePanel(t *testing.T) {
	t.Parallel()

	for _, lens := range []notification.Lens{notification.LensDone, "inbox", ""} {
		repo := &lensRepo{}

		err := lensService(repo, silentRealtime{}).DismissAll(
			t.Context(),
			notificationports.DismissAllNotificationsRequest{TenantInfo: lensTenant(), Lens: lens},
		)

		requireInvalid(t, err, "lens")
		assert.Nil(t, repo.dismissedAll, string(lens))
	}
}

func TestCountsAreTakenAsOfNowForThePersonAsking(t *testing.T) {
	t.Parallel()

	repo := &lensRepo{}
	tenantInfo := lensTenant()

	counts, err := lensService(repo, silentRealtime{}).Counts(t.Context(), tenantInfo, true)

	require.NoError(t, err)
	assert.Equal(
		t,
		&notificationports.NotificationCounts{ForYou: 3, ForYouUnread: 2, Team: 5},
		counts,
	)
	assert.Equal(
		t,
		notificationports.NotificationCountsRequest{
			TenantInfo:   tenantInfo,
			PersonalOnly: true,
			Now:          lensNow.Unix(),
		},
		*repo.counted,
	)
}

func TestResolveSubjectClearsItsOpenRowsAndTellsThePanelsOnce(t *testing.T) {
	t.Parallel()

	repo := &lensRepo{resolved: 4}
	realtime := &recordedRealtime{}
	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")

	resolved, err := lensService(repo, realtime).ResolveSubject(
		t.Context(),
		notificationports.ResolveNotificationSubjectRequest{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			SubjectKey:     "watchtower:item:wti_1",
			Resolution:     notification.ResolutionResolved,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, int64(4), resolved)
	assert.Equal(t, lensNow.Unix(), repo.resolveAsked.Now)
	require.Len(t, realtime.published, 1)
	assert.Equal(t, "notifications", realtime.published[0].Resource)
	assert.Equal(t, "resolved", realtime.published[0].Action)
	assert.Equal(t, orgID, realtime.published[0].OrganizationID)
	assert.Equal(t, buID, realtime.published[0].BusinessUnitID)
}

func TestResolveSubjectSaysNothingWhenNoRowWasOpenOrNoSubjectWasNamed(t *testing.T) {
	t.Parallel()

	quiet := &lensRepo{resolved: 0}
	realtime := &recordedRealtime{}
	svc := lensService(quiet, realtime)

	resolved, err := svc.ResolveSubject(
		t.Context(),
		notificationports.ResolveNotificationSubjectRequest{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			SubjectKey:     "watchtower:item:wti_1",
		},
	)
	require.NoError(t, err)
	assert.Zero(t, resolved)
	assert.Empty(t, realtime.published)

	unnamed := &lensRepo{resolved: 9}
	resolved, err = lensService(unnamed, realtime).ResolveSubject(
		t.Context(),
		notificationports.ResolveNotificationSubjectRequest{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
	)
	require.NoError(t, err)
	assert.Zero(t, resolved)
	assert.Nil(t, unnamed.resolveAsked, "a subject with no key matches every row, so it is never asked")
	assert.Empty(t, realtime.published)
}
