package drivernotificationservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func previewNotificationService(
	t *testing.T,
	wrk *worker.Worker,
) (*Service, *mocks.MockNotificationRepository) {
	t.Helper()

	workers := mocks.NewMockWorkerRepository(t)
	workers.EXPECT().GetByID(mock.Anything, mock.Anything).Return(wrk, nil)
	resolver := mocks.NewMockDocumentTemplateResolver(t)
	resolver.EXPECT().
		RenderMessage(mock.Anything, mock.Anything).
		Return(&services.RenderedMessage{
			Subject: "Delivery moved to 3 PM",
			Text:    "Marcus, the receiver moved your delivery to 3 PM.",
		}, nil)
	repo := mocks.NewMockNotificationRepository(t)
	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Maybe()

	return &Service{
		l:          zap.NewNop(),
		workerRepo: workers,
		templates:  resolver,
		notifications: notificationservice.New(notificationservice.Params{
			Logger:   zap.NewNop(),
			Repo:     repo,
			Realtime: realtime,
		}),
	}, repo
}

func dispatchNotice(workerID pulid.ID) *DriverNotification {
	return &DriverNotification{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		WorkerID:   workerID,
		EventType:  "dash.dispatch_message",
		Priority:   notification.PriorityHigh,
		Context: documenttemplate.DriverNotificationContext{
			AlertTitle:   "Delivery moved to 3 PM",
			AlertMessage: "The receiver moved your delivery to 3 PM.",
		},
		Link: "/loads/shp_1",
	}
}

// The repository carries no Create expectation while the preview runs, so a
// notification it created would fail the test.
func TestPreview_IsTheNotificationNotifySends(t *testing.T) {
	t.Parallel()

	wrk := notifiedWorker()
	svc, repo := previewNotificationService(t, wrk)
	req := dispatchNotice(wrk.ID)

	preview, err := svc.Preview(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, preview.Reachable)
	assert.Equal(t, "Marcus Dell", preview.WorkerName)

	var created *notification.Notification
	repo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *notification.Notification) (*notification.Notification, error) {
			created = entity
			return entity, nil
		}).
		Once()
	svc.Notify(t.Context(), req)

	require.NotNil(t, created)
	assert.Equal(t, created.Title, preview.Title)
	assert.Equal(t, created.Message, preview.Message)
	assert.Equal(t, created.Priority, preview.Priority)
	assert.Equal(t, wrk.UserID, *created.TargetUserID)
	assert.Equal(t, created.Data["link"], preview.Link)
}

func TestPreview_SaysADriverWithoutPortalAccessIsUnreachable(t *testing.T) {
	t.Parallel()

	wrk := notifiedWorker()
	wrk.UserID = pulid.Nil
	svc, _ := previewNotificationService(t, wrk)

	preview, err := svc.Preview(t.Context(), dispatchNotice(wrk.ID))
	require.NoError(t, err)
	assert.False(t, preview.Reachable)
	assert.Equal(t, "Delivery moved to 3 PM", preview.Title)

	svc.Notify(t.Context(), dispatchNotice(wrk.ID))
}
