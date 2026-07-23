package drivernotificationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const source = "driver_portal"

type Params struct {
	fx.In

	Logger        *zap.Logger
	WorkerRepo    repositories.WorkerRepository
	Notifications *notificationservice.Service
}

type Service struct {
	l             *zap.Logger
	workerRepo    repositories.WorkerRepository
	notifications *notificationservice.Service
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.driver-notification"),
		workerRepo:    p.WorkerRepo,
		notifications: p.Notifications,
	}
}

type DriverNotification struct {
	TenantInfo      pagination.TenantInfo
	WorkerID        pulid.ID
	EventType       string
	Priority        notification.Priority
	Title           string
	Message         string
	Link            string
	RelatedEntities map[string]any
}

// Notify sends an in-app (and, via the notification pipeline, push) notification
// to the portal user linked to a worker. Workers without portal access are
// skipped silently — notifying drivers is always best-effort and never fails
// the calling operation.
func (s *Service) Notify(ctx context.Context, req *DriverNotification) {
	s.NotifyWithCorrelation(ctx, req, "")
}

// NotifyWithCorrelation is Notify with a correlation ID stamped on the
// notification so periodic emitters (e.g. credential-expiry sweeps) can dedupe
// via ExistsRecent.
func (s *Service) NotifyWithCorrelation(
	ctx context.Context,
	req *DriverNotification,
	correlationID string,
) {
	if req == nil || req.WorkerID.IsNil() {
		return
	}

	wrk, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         req.WorkerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		s.l.Warn("failed to load worker for driver notification",
			zap.String("workerId", req.WorkerID.String()),
			zap.Error(err))
		return
	}
	if wrk.UserID.IsNil() {
		return
	}

	priority := req.Priority
	if priority == "" {
		priority = notification.PriorityMedium
	}
	data := map[string]any{}
	if req.Link != "" {
		data["link"] = req.Link
	}

	targetUserID := wrk.UserID
	buID := req.TenantInfo.BuID
	entity := &notification.Notification{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  &buID,
		TargetUserID:    &targetUserID,
		EventType:       req.EventType,
		Priority:        priority,
		Channel:         notification.ChannelUser,
		Title:           req.Title,
		Message:         req.Message,
		Data:            data,
		RelatedEntities: req.RelatedEntities,
		Source:          source,
	}
	if correlationID != "" {
		entity.CorrelationID = &correlationID
	}
	if _, err = s.notifications.Create(ctx, entity); err != nil {
		s.l.Warn("failed to create driver notification",
			zap.String("workerId", req.WorkerID.String()),
			zap.String("eventType", req.EventType),
			zap.Error(err))
	}
}
