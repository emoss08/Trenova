package ediservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNotifyOperationalFailureSendsOneNotificationAndThrottlesDuplicates(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockNotificationRepository(t)
	realtime := mocks.NewMockRealtimeService(t)
	notifications := notificationservice.New(notificationservice.Params{
		Logger:   zap.NewNop(),
		Repo:     repo,
		Realtime: realtime,
	})
	service := &Service{l: zap.NewNop(), notifications: notifications}

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	partnerID := pulid.MustNew("edip_")
	messageID := pulid.MustNew("edimsg_")
	alert := &EDIOperationalAlert{
		OrganizationID:  orgID,
		BusinessUnitID:  buID,
		EventType:       EDIAlertEventMessageDeadLettered,
		PartnerID:       partnerID,
		Title:           "EDI message dead-lettered",
		Message:         "Outbound 204 message exhausted its delivery retries",
		RelatedEntities: map[string]any{"messageId": messageID},
		Data:            map[string]any{"error": "connection refused"},
	}
	expectedCorrelation := EDIAlertEventMessageDeadLettered + ":" + partnerID.String()

	repo.EXPECT().
		ExistsRecent(mock.Anything, mock.MatchedBy(func(req repositories.ExistsRecentNotificationRequest) bool {
			return req.OrganizationID == orgID &&
				req.BusinessUnitID == buID &&
				req.EventType == EDIAlertEventMessageDeadLettered &&
				req.CorrelationID == expectedCorrelation &&
				req.Since > 0
		})).
		Return(false, nil).
		Once()
	repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(entity *notification.Notification) bool {
			return entity.OrganizationID == orgID &&
				entity.BusinessUnitID != nil && *entity.BusinessUnitID == buID &&
				entity.EventType == EDIAlertEventMessageDeadLettered &&
				entity.Channel == notification.ChannelGlobal &&
				entity.Priority == notification.PriorityHigh &&
				entity.CorrelationID != nil && *entity.CorrelationID == expectedCorrelation
		})).
		RunAndReturn(
			func(_ context.Context, entity *notification.Notification) (*notification.Notification, error) {
				return entity, nil
			},
		).
		Once()
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.Anything).
		Return(nil).
		Once()

	service.NotifyOperationalFailure(t.Context(), alert)

	repo.EXPECT().
		ExistsRecent(mock.Anything, mock.MatchedBy(func(req repositories.ExistsRecentNotificationRequest) bool {
			return req.CorrelationID == expectedCorrelation
		})).
		Return(true, nil).
		Once()

	service.NotifyOperationalFailure(t.Context(), alert)

	repo.AssertNumberOfCalls(t, "Create", 1)
}

func TestNotifyOperationalFailureIsNoopWithoutNotificationService(t *testing.T) {
	t.Parallel()

	service := &Service{l: zap.NewNop()}
	require.NotPanics(t, func() {
		service.NotifyOperationalFailure(t.Context(), &EDIOperationalAlert{
			EventType: EDIAlertEventInboundFileQuarantined,
		})
	})
}
