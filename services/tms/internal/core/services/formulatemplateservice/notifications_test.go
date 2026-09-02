package formulatemplateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	notificationdomain "github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// withNotifications arms the service with a real notification service backed
// by a mock repository, and returns a pointer that captures the last
// notification written.
func withNotifications(t *testing.T, deps *testDeps) **notificationdomain.Notification {
	t.Helper()

	notifRepo := mocks.NewMockNotificationRepository(t)
	deps.svc.notifications = notificationservice.New(notificationservice.Params{
		Logger:   zap.NewNop(),
		Repo:     notifRepo,
		Realtime: &mocks.NoopRealtimeService{},
	})

	captured := new(*notificationdomain.Notification)
	notifRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			entity, ok := args.Get(1).(*notificationdomain.Notification)
			require.True(t, ok)
			*captured = entity
		}).
		Return(&notificationdomain.Notification{}, nil).
		Maybe()

	return captured
}

func TestSubmit_NotifiesReviewers(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	captured := withNotifications(t, deps)

	tenant := newTenantInfo()
	template := newTestTemplate()
	template.Status = formulatemplate.StatusDraft

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	_, err := deps.svc.Submit(t.Context(), &ApprovalActionRequest{
		TenantInfo: tenant,
		EntityID:   template.ID,
		Comment:    "ready for review",
	})

	require.NoError(t, err)
	notified := *captured
	require.NotNil(t, notified)
	assert.Equal(t, notificationdomain.ChannelGlobal, notified.Channel)
	assert.Equal(t, eventTemplateSubmitted, notified.EventType)
	assert.Equal(t, tenant.OrgID, notified.OrganizationID)
	assert.Contains(t, notified.Message, template.Name)
	assert.Equal(t, templateReviewLink(template.ID), notified.Data["link"])
}

func TestApprove_NotifiesSubmitter(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	captured := withNotifications(t, deps)

	tenant := newTenantInfo()
	submitterID := pulid.MustNew("usr_")
	submittedAt := timeutils.NowUnix() - 3600

	template := newTestTemplate()
	template.Status = formulatemplate.StatusInReview
	template.SubmittedByID = &submitterID
	template.SubmittedAt = &submittedAt

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("Create", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Approve(t.Context(), &ApprovalActionRequest{
		TenantInfo: tenant,
		EntityID:   template.ID,
		Comment:    "looks right",
	})

	require.NoError(t, err)
	notified := *captured
	require.NotNil(t, notified)
	assert.Equal(t, notificationdomain.ChannelUser, notified.Channel)
	assert.Equal(t, eventTemplateApproved, notified.EventType)
	require.NotNil(t, notified.TargetUserID)
	assert.Equal(t, submitterID, *notified.TargetUserID)
	assert.Contains(t, notified.Message, "approved")
	assert.Contains(t, notified.Message, "looks right")
}

func TestReject_NotifiesSubmitterWithComment(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	captured := withNotifications(t, deps)

	submitterID := pulid.MustNew("usr_")
	submittedAt := timeutils.NowUnix() - 3600

	template := newTestTemplate()
	template.Status = formulatemplate.StatusInReview
	template.SubmittedByID = &submitterID
	template.SubmittedAt = &submittedAt

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("ClearScheduled", mock.Anything, mock.Anything).Return(int64(0), nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Reject(t.Context(), &ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   template.ID,
		Comment:    "expression is wrong",
	})

	require.NoError(t, err)
	notified := *captured
	require.NotNil(t, notified)
	assert.Equal(t, notificationdomain.ChannelUser, notified.Channel)
	assert.Equal(t, eventTemplateRejected, notified.EventType)
	assert.Equal(t, notificationdomain.PriorityHigh, notified.Priority)
	require.NotNil(t, notified.TargetUserID)
	assert.Equal(t, submitterID, *notified.TargetUserID)
	assert.Contains(t, notified.Message, "expression is wrong")
}

func TestReject_SkipsNotificationWhenRejectorIsSubmitter(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	captured := withNotifications(t, deps)

	tenant := newTenantInfo()
	submitterID := tenant.UserID
	submittedAt := timeutils.NowUnix() - 3600

	template := newTestTemplate()
	template.Status = formulatemplate.StatusInReview
	template.SubmittedByID = &submitterID
	template.SubmittedAt = &submittedAt

	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("ClearScheduled", mock.Anything, mock.Anything).Return(int64(0), nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Reject(t.Context(), &ApprovalActionRequest{
		TenantInfo: tenant,
		EntityID:   template.ID,
		Comment:    "withdrawing my own submission",
	})

	require.NoError(t, err)
	assert.Nil(t, *captured)
}
