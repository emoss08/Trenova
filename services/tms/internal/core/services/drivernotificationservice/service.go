package drivernotificationservice

import (
	"context"
	"reflect"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
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
	Templates     services.DocumentTemplateResolver
}

type Service struct {
	l             *zap.Logger
	workerRepo    repositories.WorkerRepository
	notifications *notificationservice.Service
	templates     services.DocumentTemplateResolver
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.driver-notification"),
		workerRepo:    p.WorkerRepo,
		notifications: p.Notifications,
		templates:     p.Templates,
	}
}

type DriverNotification struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	EventType  string
	Priority   notification.Priority

	// Context is the event's data, in the shape its notification family declares.
	// The wording that used to sit at the call site now lives in the organization's
	// template, so a carrier can reword what their drivers are told.
	//
	// The recipient block is filled here rather than by the caller: this is where
	// the worker record is already loaded, and thirty call sites restating a
	// driver's name is thirty places for it to drift.
	Context any

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

	title, message, ok := s.render(ctx, req, wrk)
	if !ok {
		return
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
		Title:           title,
		Message:         message,
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

// render turns the event into the words a driver reads.
//
// It renders with FallbackToBuiltIn: a driver who is never told their load moved
// is worse off than one told in the shipped wording, so a template an
// organization broke costs styling rather than the notification. A failure that
// even the built-in cannot survive drops the notification and logs, matching how
// this service already treats a worker without portal access — driver
// notification is best-effort by design and never fails the operation that
// triggered it.
func (s *Service) render(
	ctx context.Context,
	req *DriverNotification,
	wrk *worker.Worker,
) (title, message string, ok bool) {
	if s.templates == nil {
		s.l.Warn("template rendering is not configured; dropping driver notification",
			zap.String("eventType", req.EventType))
		return "", "", false
	}

	data := fillRecipient(req.Context, wrk)

	rendered, err := s.templates.RenderMessage(ctx, &services.RenderMessageRequest{
		TenantInfo:        req.TenantInfo,
		Kind:              notificationKind(req.EventType),
		Data:              data,
		ReferenceID:       req.WorkerID,
		FallbackToBuiltIn: true,
	})
	if err != nil {
		s.l.Warn("failed to render driver notification",
			zap.String("eventType", req.EventType),
			zap.String("workerId", req.WorkerID.String()),
			zap.Error(err))
		return "", "", false
	}

	return rendered.Subject, rendered.Text, true
}

// fillRecipient returns the context with the driver's name in it, accepting
// either a value or a pointer.
//
// The reflection is not decoration. A struct value stored in an interface is not
// addressable, so a pointer-receiver setter cannot be reached through a plain type
// assertion — a call site that passed a value rather than a pointer would silently
// send a notification with no name in it, and nothing would fail. Cloning into an
// addressable value closes that, so both spellings work at the call site.
func fillRecipient(data any, wrk *worker.Worker) any {
	if data == nil {
		return nil
	}

	full := strings.TrimSpace(wrk.FirstName + " " + wrk.LastName)

	if aware, ok := data.(documenttemplate.RecipientAware); ok {
		aware.SetRecipient(wrk.FirstName, wrk.LastName, full)
		return data
	}

	value := reflect.ValueOf(data)
	if value.Kind() != reflect.Struct {
		return data
	}

	clone := reflect.New(value.Type())
	clone.Elem().Set(value)

	aware, ok := clone.Interface().(documenttemplate.RecipientAware)
	if !ok {
		return data
	}
	aware.SetRecipient(wrk.FirstName, wrk.LastName, full)

	return clone.Elem().Interface()
}

// notificationKind maps an event type onto its template kind.
//
// The key is the event type verbatim, so the registry and
// notifications.event_type cannot drift: renaming an event without registering
// the new kind fails the render rather than silently sending the wrong wording.
func notificationKind(eventType string) documenttemplate.Kind {
	return documenttemplate.Kind("notification." + eventType)
}
