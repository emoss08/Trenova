package drivernotificationservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/portalaccessports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/shared/timeutils"
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
) (*notificationports.DriverNotificationPreview, error) {
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

	preview := &notificationports.DriverNotificationPreview{
		WorkerID:   wrk.ID,
		WorkerName: strings.TrimSpace(wrk.FirstName + " " + wrk.LastName),
		Reachable:  reachable(wrk),
		Title:      title,
		Message:    message,
		Priority:   notificationPriority(req),
		Link:       req.Link,
	}
	if !preview.Reachable {
		preview.InvitedAt, preview.InvitedEmail = s.openInvitation(ctx, req)
		if phone := strings.TrimSpace(wrk.PhoneNumber); phone != "" {
			preview.Text = &notificationports.DriverSMSPreview{
				PhoneNumber: phone,
				Message:     textOf(title, message),
			}
		}
	}

	return preview, nil
}

// openInvitation is the portal invitation a driver has not accepted yet, so
// whoever reads "the message would not reach them" is told they were already
// asked in, and when. A driver never invited, and a read that fails, answer
// with nothing: the warning stands either way.
func (s *Service) openInvitation(ctx context.Context, req *DriverNotification) (int64, string) {
	if s.portal == nil {
		return 0, ""
	}
	pending, err := s.portal.GetPendingInvitationForWorker(
		ctx,
		portalaccessports.GetPendingPortalInvitationRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
		},
	)
	if err != nil || pending == nil || !pending.IsAcceptable(timeutils.NowUnix()) {
		return 0, ""
	}

	return pending.CreatedAt, pending.Email
}

func (s *Service) recipient(ctx context.Context, req *DriverNotification) (*worker.Worker, error) {
	return s.workerRepo.GetByID(ctx, workerports.GetWorkerByIDRequest{
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
	rendered, err := s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
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

// textOf is a notification as one text message: its title, then what it
// says.
func textOf(title, message string) string {
	title, message = strings.TrimSpace(title), strings.TrimSpace(message)
	switch {
	case title == "":
		return message
	case message == "":
		return title
	default:
		return title + ": " + message
	}
}
