package drivernotificationservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

var errRenderingNotConfigured = errors.New(
	"template rendering is not configured, so the driver would be sent nothing",
)

// Preview renders the notification as Notify would send it, and says whether
// the driver has the portal access it needs to reach them. It creates
// nothing.
func (s *Service) Preview(
	ctx context.Context,
	req *DriverNotification,
) (*services.DriverNotificationPreview, error) {
	if req == nil || req.WorkerID.IsNil() {
		return nil, errors.New("a driver notification needs a driver")
	}

	wrk, err := s.recipient(ctx, req)
	if err != nil {
		return nil, err
	}

	title, message, err := s.renderNotification(ctx, req, wrk)
	if err != nil {
		return nil, err
	}

	return &services.DriverNotificationPreview{
		WorkerID:   wrk.ID,
		WorkerName: strings.TrimSpace(wrk.FirstName + " " + wrk.LastName),
		Reachable:  reachable(wrk),
		Title:      title,
		Message:    message,
		Priority:   notificationPriority(req),
		Link:       req.Link,
	}, nil
}

func (s *Service) recipient(ctx context.Context, req *DriverNotification) (*worker.Worker, error) {
	return s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         req.WorkerID,
		TenantInfo: req.TenantInfo,
	})
}

// reachable is a driver with a portal login: the notification is addressed
// to that user, and a driver without one is never told.
func reachable(wrk *worker.Worker) bool {
	return wrk != nil && wrk.UserID.IsNotNil()
}

func notificationPriority(req *DriverNotification) notification.Priority {
	if req.Priority == "" {
		return notification.PriorityMedium
	}

	return req.Priority
}

// renderNotification turns the event into the words a driver reads, with
// FallbackToBuiltIn: a template an organization broke costs styling, never
// the notification.
func (s *Service) renderNotification(
	ctx context.Context,
	req *DriverNotification,
	wrk *worker.Worker,
) (title, message string, err error) {
	if s.templates == nil {
		return "", "", errRenderingNotConfigured
	}

	rendered, err := s.templates.RenderMessage(ctx, &services.RenderMessageRequest{
		TenantInfo:        req.TenantInfo,
		Kind:              notificationKind(req.EventType),
		Data:              fillRecipient(req.Context, wrk),
		ReferenceID:       req.WorkerID,
		FallbackToBuiltIn: true,
	})
	if err != nil {
		return "", "", err
	}

	return rendered.Subject, rendered.Text, nil
}
