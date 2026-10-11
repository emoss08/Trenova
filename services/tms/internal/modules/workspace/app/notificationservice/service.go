package notificationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/webpushservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger   *zap.Logger
	Repo     notificationports.NotificationRepository
	Realtime agentports.RealtimeService
	WebPush  *webpushservice.Service `optional:"true"`
	// Roles answers who may read a resource, for NotifyPermitted.
	Roles permissionports.RoleRepository `optional:"true"`
}

type Service struct {
	l        *zap.Logger
	repo     notificationports.NotificationRepository
	realtime agentports.RealtimeService
	webPush  *webpushservice.Service
	roles    permissionports.RoleRepository
}

func New(p Params) *Service {
	return &Service{
		l:        p.Logger.Named("service.notification"),
		repo:     p.Repo,
		realtime: p.Realtime,
		webPush:  p.WebPush,
		roles:    p.Roles,
	}
}

func (s *Service) ExistsRecent(
	ctx context.Context,
	req notificationports.ExistsRecentNotificationRequest,
) (bool, error) {
	return s.repo.ExistsRecent(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	log := s.l.With(zap.String("operation", "Create"))

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create notification", zap.Error(err))
		return nil, err
	}

	var buID pulid.ID
	if created.BusinessUnitID != nil {
		buID = *created.BusinessUnitID
	}

	pubErr := s.realtime.PublishResourceInvalidation(
		ctx,
		&agentports.PublishResourceInvalidationRequest{
			OrganizationID: created.OrganizationID,
			BusinessUnitID: buID,
			AudienceUserID: pulid.ConvertFromPtr(created.TargetUserID),
			Resource:       "notifications",
			Action:         "created",
			Entity:         created,
		},
	)
	if pubErr != nil {
		log.Warn("failed to publish realtime notification", zap.Error(pubErr))
	}

	s.sendWebPush(ctx, created)

	return created, nil
}

// sendWebPush delivers a targeted notification to the user's registered web
// push subscriptions in the background so it reaches drivers with the app
// closed. Global/role notifications are intentionally not pushed.
func (s *Service) sendWebPush(ctx context.Context, created *notification.Notification) {
	if s.webPush == nil || !s.webPush.Enabled() {
		return
	}
	if created.TargetUserID == nil || created.TargetUserID.IsNil() {
		return
	}

	link := ""
	if raw, ok := created.Data["link"].(string); ok {
		link = raw
	}
	targetUserID := *created.TargetUserID
	payload := &webpushservice.PushPayload{
		Title:   created.Title,
		Body:    created.Message,
		Link:    link,
		EventID: created.ID.String(),
	}

	background := context.WithoutCancel(ctx)
	go s.webPush.SendToUser(background, targetUserID, payload)
}

func (s *Service) List(
	ctx context.Context,
	req *notificationports.ListNotificationsRequest,
) (*pagination.ListResult[*notification.Notification], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *notificationports.ListNotificationConnectionRequest,
) (*pagination.CursorListResult[*notification.Notification], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) CountUnread(
	ctx context.Context,
	userID pulid.ID,
	tenantInfo pagination.TenantInfo,
	personalOnly bool,
) (int64, error) {
	return s.repo.CountUnread(ctx, userID, tenantInfo, personalOnly)
}

func (s *Service) MarkAsRead(
	ctx context.Context,
	req notificationports.NotificationActionRequest,
) error {
	return s.repo.MarkAsRead(ctx, req)
}

func (s *Service) MarkAsUnread(
	ctx context.Context,
	req notificationports.NotificationActionRequest,
) error {
	return s.repo.MarkAsUnread(ctx, req)
}

func (s *Service) Dismiss(
	ctx context.Context,
	req notificationports.NotificationActionRequest,
) error {
	return s.repo.Dismiss(ctx, req)
}

func (s *Service) Restore(
	ctx context.Context,
	req notificationports.NotificationActionRequest,
) error {
	return s.repo.Restore(ctx, req)
}

func (s *Service) MarkAllAsRead(
	ctx context.Context,
	userID pulid.ID,
	tenantInfo pagination.TenantInfo,
	personalOnly bool,
) error {
	return s.repo.MarkAllAsRead(ctx, userID, tenantInfo, personalOnly)
}
