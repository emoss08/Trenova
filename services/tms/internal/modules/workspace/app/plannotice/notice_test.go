package plannotice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const noticeNow = int64(1_794_650_400)

type recordingNotifier struct {
	sent []notificationservice.NotifyPermittedRequest
	err  error
}

func (r *recordingNotifier) NotifyPermitted(
	_ context.Context,
	req notificationservice.NotifyPermittedRequest,
) (int, error) {
	r.sent = append(r.sent, req)
	return 1, r.err
}

type recordingRealtime struct {
	published []*realtimeports.PublishResourceInvalidationRequest
}

func (r *recordingRealtime) PublishResourceInvalidation(
	_ context.Context,
	req *realtimeports.PublishResourceInvalidationRequest,
) error {
	r.published = append(r.published, req)
	return nil
}

type twoTenants struct{ tenants []pagination.TenantInfo }

func (p twoTenants) ListTenantsPage(
	_ context.Context,
	after pulid.ID,
	_ int,
) ([]pagination.TenantInfo, error) {
	if !after.IsNil() {
		return nil, nil
	}
	return p.tenants, nil
}

func newNotices(notifier *recordingNotifier) (*Notices, []pagination.TenantInfo) {
	tenants := []pagination.TenantInfo{
		{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
	return &Notices{
		notifier: notifier,
		tenants:  twoTenants{tenants: tenants},
		realtime: &recordingRealtime{},
		clock:    func() int64 { return noticeNow },
		l:        zap.NewNop(),
	}, tenants
}

func planEnd(held *managedfeatures.State, endsAt int64) *managedfeatures.Replacement {
	return managedfeatures.Replace(held, &managedfeatures.Received{
		Features:   []platformcatalog.FeatureKey{"tms.accounting"},
		PlanEndsAt: endsAt,
	}, noticeNow)
}

const planEndsAt = int64(1_797_206_400)

func TestASetPlanEndTellsTheAdministratorsWhenTheirWorkspaceGoes(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	notices, tenants := newNotices(notifier)
	held := &managedfeatures.State{
		ID:              managedfeatures.CurrentStateID,
		Features:        []platformcatalog.FeatureKey{"tms.accounting"},
		FinalBackupDays: 7,
	}

	require.NoError(t, notices.PlanChanged(t.Context(), planEnd(held, planEndsAt)))

	require.Len(t, notifier.sent, len(tenants))
	sent := notifier.sent[0].Notification
	assert.Equal(t, "Your plan ends on Dec 14, 2026", sent.Title)
	assert.Equal(t,
		"Trenova stays available until then. After that, this workspace is taken down and a final backup is kept for 7 days. An owner or admin of your Trenova account can keep the plan before then.",
		sent.Message,
	)
	assert.Equal(t, PlanPath, sent.Data["link"])
	assert.Equal(t, "managed-plan:plan-end", sent.SubjectKey)
	assert.Equal(t, notification.PriorityHigh, sent.Priority)
}

func TestAPlanEndToldBeforeAnyDayCountSaysTheDefault(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	notices, _ := newNotices(notifier)

	require.NoError(t, notices.PlanChanged(t.Context(), planEnd(nil, planEndsAt)))

	require.NotEmpty(t, notifier.sent)
	assert.Contains(t, notifier.sent[0].Notification.Message, "kept for 30 days")
}

func TestAKeptPlanSaysItContinues(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	notices, _ := newNotices(notifier)
	held := &managedfeatures.State{
		ID:               managedfeatures.CurrentStateID,
		Features:         []platformcatalog.FeatureKey{"tms.accounting"},
		PlanEndsAt:       planEndsAt,
		PlanEndAnnounced: planEndsAt,
	}

	require.NoError(t, notices.PlanChanged(t.Context(), planEnd(held, 0)))

	require.NotEmpty(t, notifier.sent)
	sent := notifier.sent[0].Notification
	assert.Equal(t, "Your plan continues", sent.Title)
	assert.Equal(t, "It will no longer end on Dec 14, 2026.", sent.Message)
	assert.Equal(t, notification.PriorityMedium, sent.Priority)
}

func TestTheThreeDayReminderSaysWhenTheWorkspaceGoes(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	notices, _ := newNotices(notifier)
	held := &managedfeatures.State{
		ID:               managedfeatures.CurrentStateID,
		PlanEndsAt:       planEndsAt,
		PlanEndAnnounced: planEndsAt,
		FinalBackupDays:  7,
	}

	reminder := managedfeatures.RemindPlanEnd(held, planEndsAt-2*24*60*60)
	require.NoError(t, notices.PlanChanged(t.Context(), reminder))

	require.NotEmpty(t, notifier.sent)
	sent := notifier.sent[0].Notification
	assert.Equal(t, "Your plan ends in 3 days", sent.Title)
	assert.Equal(t,
		"After Dec 14, 2026, this workspace is taken down and a final backup is kept for 7 days. An owner or admin of your Trenova account can keep the plan before then.",
		sent.Message,
	)
}

func TestAPlanChangeTellsOpenPagesInEveryOrganization(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	notices, tenants := newNotices(notifier)
	realtime := &recordingRealtime{}
	notices.realtime = realtime
	held := &managedfeatures.State{
		ID:       managedfeatures.CurrentStateID,
		Features: []platformcatalog.FeatureKey{"tms.accounting"},
	}
	planURL := "https://console.trenova.cloud/instances/1/plan"

	require.NoError(t, notices.PlanChanged(t.Context(), managedfeatures.Replace(held, &managedfeatures.Received{
		Features: []platformcatalog.FeatureKey{"tms.accounting"},
		PlanURL:  &planURL,
	}, noticeNow)))

	assert.Empty(t, notifier.sent, "a new address notifies nobody")
	require.Len(t, realtime.published, len(tenants))
	for i, req := range realtime.published {
		assert.Equal(t, tenants[i].OrgID, req.OrganizationID)
		assert.Equal(t, PlanResource, req.Resource)
	}

	realtime.published = nil
	require.NoError(t, notices.PlanChanged(t.Context(), managedfeatures.Replace(
		&managedfeatures.State{ID: managedfeatures.CurrentStateID, Features: held.Features, PlanURL: planURL},
		&managedfeatures.Received{Features: held.Features, PlanURL: &planURL},
		noticeNow,
	)))
	assert.Empty(t, realtime.published, "nothing changed, so nothing is told")
}

func change(before, after []platformcatalog.FeatureKey) *managedfeatures.Replacement {
	held := &managedfeatures.State{ID: managedfeatures.CurrentStateID, Features: before}
	return managedfeatures.Replace(held, &managedfeatures.Received{Features: after}, noticeNow)
}

func TestDroppingAnAddOnTellsEachOrganizationsAdministratorsOnce(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	notices, tenants := newNotices(notifier)

	require.NoError(t, notices.PlanChanged(t.Context(), change(
		[]platformcatalog.FeatureKey{"ai.desk", "tms.dispatch_intelligence"},
		[]platformcatalog.FeatureKey{"ai.desk"},
	)))

	require.Len(t, notifier.sent, len(tenants))
	for i, req := range notifier.sent {
		assert.Equal(t, tenants[i], req.Tenant)
		assert.Equal(t, permission.ResourceOrganization, req.Resource)
		assert.Equal(t, permission.OpUpdate, req.Operation)
		assert.Equal(t, maxNoticeRecipients, req.Limit)
		assert.Equal(t, EventPlanChanged, req.Notification.EventType)
		assert.Equal(t, notification.PriorityHigh, req.Notification.Priority)
		assert.Equal(t, "Dispatch Intelligence is no longer on your plan", req.Notification.Title)
		assert.Equal(t,
			"Everything you entered is kept. It comes back if you add Dispatch Intelligence again from your Trenova account.",
			req.Notification.Message,
		)
		assert.Equal(t, PlanPath, req.Notification.Data["link"])
		assert.Equal(t, "managed-plan:DSP", req.Notification.SubjectKey)
	}
}

func TestAddingAnAddOnSaysItIsNowOnThePlan(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	notices, tenants := newNotices(notifier)

	require.NoError(t, notices.PlanChanged(t.Context(), change(
		[]platformcatalog.FeatureKey{},
		[]platformcatalog.FeatureKey{"tms.accounting", "tms.factoring"},
	)))

	require.Len(t, notifier.sent, len(tenants))
	assert.Equal(t, "Accounting & Cash is now on your plan", notifier.sent[0].Notification.Title)
	assert.Equal(t, notification.PriorityMedium, notifier.sent[0].Notification.Priority)
}

func TestANoticeNamesTheAddOnInTheSameLanguageAsItsSentence(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	notices, _ := newNotices(notifier)

	require.NoError(t, notices.PlanChanged(i18n.WithLocale(t.Context(), i18n.ES), change(
		[]platformcatalog.FeatureKey{},
		[]platformcatalog.FeatureKey{"tms.accounting"},
	)))

	require.NotEmpty(t, notifier.sent)
	title := notifier.sent[0].Notification.Title
	assert.Contains(t, title, "Contabilidad y efectivo")
	assert.NotContains(t, title, "Accounting & Cash")
	assert.Equal(
		t,
		"Accounting & Cash",
		notifier.sent[0].Notification.Data["addOn"],
		"the data names the catalogue's add-on, not a sentence",
	)
}

func TestTheStartingPlanNotifiesNobody(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	notices, _ := newNotices(notifier)

	require.NoError(t, notices.PlanChanged(t.Context(), managedfeatures.Replace(
		nil,
		&managedfeatures.Received{Features: []platformcatalog.FeatureKey{"ai.desk"}},
		noticeNow,
	)))

	assert.Empty(t, notifier.sent)
}

func TestAChangeInsideAnAddOnThatStaysOnNotifiesNobody(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	notices, _ := newNotices(notifier)

	require.NoError(t, notices.PlanChanged(t.Context(), change(
		[]platformcatalog.FeatureKey{"ai.desk"},
		[]platformcatalog.FeatureKey{"ai.desk", "ai.watchtower"},
	)))

	assert.Empty(t, notifier.sent)
}

func TestAnAddOnThatKeepsOneFeatureHasNotLeft(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	notices, _ := newNotices(notifier)

	require.NoError(t, notices.PlanChanged(t.Context(), change(
		[]platformcatalog.FeatureKey{"tms.accounting", "tms.factoring"},
		[]platformcatalog.FeatureKey{"tms.accounting"},
	)))

	assert.Empty(t, notifier.sent)
}

func TestAFailedNoticeIsReportedAndTheOtherOrganizationsAreStillTold(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{err: errors.New("notifications are down")}
	notices, tenants := newNotices(notifier)

	err := notices.PlanChanged(t.Context(), change(
		[]platformcatalog.FeatureKey{"tms.dispatch_intelligence"},
		[]platformcatalog.FeatureKey{},
	))

	require.Error(t, err)
	assert.Len(t, notifier.sent, len(tenants))
}
