package notificationtest

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/pushconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/webpushservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"go.uber.org/zap"
)

type discardedNotifications struct {
	notificationports.NotificationRepository
}

func (discardedNotifications) Create(
	_ context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	return entity, nil
}

type nobodyPermitted struct {
	permissionports.RoleRepository
}

func (nobodyPermitted) ListUsersWithPermission(
	context.Context,
	permissionports.ListUsersWithPermissionRequest,
) ([]permissionports.PermittedUser, error) {
	return nil, nil
}

func New(
	repo notificationports.NotificationRepository,
	realtime realtimeports.RealtimeService,
) *notificationservice.Service {
	return notificationservice.New(notificationservice.Params{
		Logger:   zap.NewNop(),
		Repo:     repo,
		Realtime: realtime,
		Roles:    nobodyPermitted{},
		WebPush: webpushservice.New(webpushservice.Params{
			Logger:     zap.NewNop(),
			PushConfig: &pushconfig.Config{},
		}),
		AppConfig: &appconfig.Config{},
	})
}

func Discarding() *notificationservice.Service {
	return New(discardedNotifications{}, &mocks.NoopRealtimeService{})
}
