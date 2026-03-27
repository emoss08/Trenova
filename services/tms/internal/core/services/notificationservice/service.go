package notificationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger   *zap.Logger
	Repo     repositories.NotificationRepository
	Realtime servicesport.RealtimeService
}

type Service struct {
	l        *zap.Logger
	repo     repositories.NotificationRepository
	realtime servicesport.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:        p.Logger.Named("service.notification"),
		repo:     p.Repo,
		realtime: p.Realtime,
	}
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

	pubErr := s.realtime.PublishResourceInvalidation(ctx, &servicesport.PublishResourceInvalidationRequest{
		OrganizationID: created.OrganizationID,
		BusinessUnitID: buID,
		Resource:       "notifications",
		Action:         "created",
		Entity:         created,
	})
	if pubErr != nil {
		log.Warn("failed to publish realtime notification", zap.Error(pubErr))
	}

	return created, nil
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListNotificationsRequest,
) (*pagination.ListResult[*notification.Notification], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) CountUnread(
	ctx context.Context,
	userID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (int64, error) {
	return s.repo.CountUnread(ctx, userID, tenantInfo)
}

func (s *Service) MarkAsRead(
	ctx context.Context,
	req repositories.MarkNotificationsReadRequest,
) error {
	return s.repo.MarkAsRead(ctx, req)
}

func (s *Service) MarkAllAsRead(
	ctx context.Context,
	userID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	return s.repo.MarkAllAsRead(ctx, userID, tenantInfo)
}
