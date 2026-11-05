package edialert

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/internal/testutil/mocks/notificationportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/realtimeportsmock"
	"github.com/emoss08/trenova/internal/testutil/notificationtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestNotifyOperationalFailureSendsOneNotificationAndThrottlesDuplicates(t *testing.T) {
	t.Parallel()

	repo := notificationportsmock.NewMockNotificationRepository(t)
	realtime := realtimeportsmock.NewMockRealtimeService(t)
	notifications := notificationtest.New(repo, realtime)
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
		ExistsRecent(mock.Anything, mock.MatchedBy(func(req notificationports.ExistsRecentNotificationRequest) bool {
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
		ExistsRecent(mock.Anything, mock.MatchedBy(func(req notificationports.ExistsRecentNotificationRequest) bool {
			return req.CorrelationID == expectedCorrelation
		})).
		Return(true, nil).
		Once()

	service.NotifyOperationalFailure(t.Context(), alert)

	repo.AssertNumberOfCalls(t, "Create", 1)
}
