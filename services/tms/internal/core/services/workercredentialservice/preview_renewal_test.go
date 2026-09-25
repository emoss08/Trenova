package workercredentialservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The notification repository has no Create expectation while the preview
// runs, so a message it sent would fail the test.
func TestPreviewRenewal_IsTheAskRequestRenewalSends(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.wrk.UserID = pulid.MustNew("usr_")
	h.wrk.FirstName, h.wrk.LastName = "Marcus", "Dell"
	cdl := h.credential(h.cdl, 1_900_000_000)
	cdl.ID = pulid.MustNew("wcred_")
	cdl.Status = worker.CredentialStatusActive
	h.repo.credentials[cdl.ID] = cdl

	resolver := mocks.NewMockDocumentTemplateResolver(t)
	resolver.EXPECT().
		RenderMessage(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *services.RenderMessageRequest) (*services.RenderedMessage, error) {
			data, ok := req.Data.(documenttemplate.DriverNotificationContext)
			require.True(t, ok)
			return &services.RenderedMessage{
				Subject: "Renew your " + data.CredentialName,
				Text:    data.FullName + ": " + data.Reason,
			}, nil
		})
	notifications := mocks.NewMockNotificationRepository(t)
	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Maybe()
	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Maybe()

	svc := workercredentialservice.New(workercredentialservice.Params{
		Logger:       zap.NewNop(),
		Repo:         h.repo,
		WorkerRepo:   h.workerRepo,
		DocumentRepo: h.docRepo,
		AuditService: audit,
		DriverNotify: drivernotificationservice.New(drivernotificationservice.Params{
			Logger:     zap.NewNop(),
			WorkerRepo: h.workerRepo,
			Templates:  resolver,
			Notifications: notificationservice.New(notificationservice.Params{
				Logger:   zap.NewNop(),
				Repo:     notifications,
				Realtime: realtime,
			}),
		}),
	})
	request := workercredentialservice.RenewalRequest{
		TenantInfo:    h.tenant,
		WorkerID:      h.wrk.ID,
		CredentialIDs: []pulid.ID{cdl.ID},
		Note:          "Send the renewed CDL to the safety desk by Friday.",
		RequestedByID: h.userID,
	}

	preview, err := svc.PreviewRenewal(t.Context(), request)
	require.NoError(t, err)
	require.Len(t, preview.Credentials, 1)
	assert.True(t, preview.Notification.Reachable)

	var sent *notification.Notification
	notifications.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *notification.Notification) (*notification.Notification, error) {
			sent = entity
			return entity, nil
		}).
		Once()
	require.NoError(t, svc.RequestRenewal(t.Context(), request))

	require.NotNil(t, sent)
	assert.Equal(t, "Renew your CDL", preview.Notification.Title)
	assert.Equal(t, sent.Title, preview.Notification.Title)
	assert.Equal(t, sent.Message, preview.Notification.Message)
	assert.Equal(t, sent.Priority, preview.Notification.Priority)
}
