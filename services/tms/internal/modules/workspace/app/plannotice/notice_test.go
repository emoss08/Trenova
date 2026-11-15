package plannotice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
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
		clock:    func() int64 { return noticeNow },
		l:        zap.NewNop(),
	}, tenants
}

func change(before, after []platformcatalog.FeatureKey) *managedfeatures.Replacement {
	held := &managedfeatures.State{ID: managedfeatures.CurrentStateID, Features: before}
	return managedfeatures.Replace(held, after, noticeNow)
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

func TestTheStartingPlanNotifiesNobody(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	notices, _ := newNotices(notifier)

	require.NoError(t, notices.PlanChanged(t.Context(), managedfeatures.Replace(
		nil,
		[]platformcatalog.FeatureKey{"ai.desk"},
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
