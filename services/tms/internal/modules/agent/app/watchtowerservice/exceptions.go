package watchtowerservice

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	EventAssigned  = "watchtower.assigned"
	EventCommented = "watchtower.commented"
	EventEscalated = "watchtower.escalated"
)

type ExceptionsParams struct {
	fx.In

	Logger        *zap.Logger
	Repo          watchtowerports.WatchtowerRepository
	Projector     *Projector
	Router        *Router
	Permissions   permissionports.PermissionEngine
	Audit         auditports.AuditService      `optional:"true"`
	Notifications *notificationservice.Service `optional:"true"`
}

type Exceptions struct {
	l             *zap.Logger
	repo          watchtowerports.WatchtowerRepository
	projector     *Projector
	router        *Router
	permissions   permissionports.PermissionEngine
	access        kindAccess
	audit         auditports.AuditService
	notifications *notificationservice.Service
	now           func() int64
}

func NewExceptions(
	p ExceptionsParams, //nolint:gocritic // fx params are passed by value
) *Exceptions {
	return &Exceptions{
		l:             p.Logger.Named("service.watchtower.exceptions"),
		repo:          p.Repo,
		projector:     p.Projector,
		router:        p.Router,
		permissions:   p.Permissions,
		access:        kindAccess{permissions: p.Permissions},
		audit:         p.Audit,
		notifications: p.Notifications,
		now:           timeutils.NowUnix,
	}
}

var _ watchtowerports.ExceptionService = (*Exceptions)(nil)

func AsExceptionService(e *Exceptions) watchtowerports.ExceptionService { return e }

type exceptionChange struct {
	target *watchtowerports.ExceptionTarget
	actor  *permissionports.RequestActor
	mutate func(item *watchtower.Item, now int64) ([]*watchtower.Event, error)
	after  func(ctx context.Context, before, after *watchtower.Item)
}

func (s *Exceptions) applyException(
	ctx context.Context,
	change *exceptionChange,
) (*watchtower.Item, error) {
	if change.target == nil || change.target.ItemID.IsNil() {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrRequired,
			"Choose an exception",
		)
	}

	before, err := s.GetException(ctx, watchtowerports.GetWatchtowerItemRequest{
		ID:         change.target.ItemID,
		TenantInfo: change.target.TenantInfo,
	}, change.actor)
	if err != nil {
		return nil, err
	}

	expected := change.target.ExpectedVersion
	if expected == 0 {
		expected = before.Version
	}
	if expected != before.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"This exception changed since you opened it; refresh and try again",
		)
	}

	working := *before
	now := s.now()
	events, err := change.mutate(&working, now)
	if err != nil {
		return nil, err
	}
	if change.target.DryRun {
		return &working, nil
	}

	saved, err := s.repo.UpdateWorkflow(ctx, &watchtowerports.UpdateWatchtowerWorkflowRequest{
		Item:            &working,
		ExpectedVersion: expected,
		Events:          events,
	})
	if err != nil {
		return nil, err
	}

	s.projector.publish(ctx, saved, "updated")
	s.auditException(before, saved, change.actor, events)
	if change.after != nil {
		change.after(ctx, before, saved)
	}

	return saved, nil
}

func (s *Exceptions) GetException(
	ctx context.Context,
	req watchtowerports.GetWatchtowerItemRequest,
	actor *permissionports.RequestActor,
) (*watchtower.Item, error) {
	item, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}
	if err = s.access.assertMaySee(ctx, actor, item); err != nil {
		return nil, err
	}
	return item, nil
}

func actorUserID(actor *permissionports.RequestActor) pulid.ID {
	return actor.ExecutorUserID()
}

func (s *Exceptions) AssignException(
	ctx context.Context,
	req *watchtowerports.AssignExceptionRequest,
	actor *permissionports.RequestActor,
) (*watchtower.Item, error) {
	return s.applyException(ctx, &exceptionChange{
		target: &req.ExceptionTarget,
		actor:  actor,
		mutate: func(item *watchtower.Item, _ int64) ([]*watchtower.Event, error) {
			if req.AssigneeID.IsNotNil() {
				if err := s.assertAssignable(ctx, item, req.AssigneeID); err != nil {
					return nil, err
				}
			}
			previous := item.AssigneeID
			if err := item.Assign(req.AssigneeID); err != nil {
				return nil, err
			}
			kind := watchtower.EventAssigned
			if req.AssigneeID.IsNil() {
				kind = watchtower.EventUnassigned
			}
			return []*watchtower.Event{
				watchtower.NewEvent(item, kind, actorUserID(actor)).
					WithChange(previous.String(), req.AssigneeID.String()).
					WithNote(req.Note),
			}, nil
		},
		after: func(ctx context.Context, _, after *watchtower.Item) {
			if after.AssigneeID.IsNotNil() && after.AssigneeID != actorUserID(actor) {
				s.notifyAssignee(ctx, after, assignedNotice)
			}
		},
	})
}

func (s *Exceptions) assertAssignable(
	ctx context.Context,
	item *watchtower.Item,
	assigneeID pulid.ID,
) error {
	ok, err := mayWorkException(ctx, s.access, item, pagination.TenantInfo{
		OrgID:  item.OrganizationID,
		BuID:   item.BusinessUnitID,
		UserID: assigneeID,
	})
	if err != nil {
		return err
	}
	if !ok {
		return errortypes.NewValidationError(
			"assigneeId",
			errortypes.ErrInvalid,
			"That person cannot see this exception",
		)
	}
	return nil
}

func (s *Exceptions) AcknowledgeException(
	ctx context.Context,
	req *watchtowerports.ExceptionTarget,
	actor *permissionports.RequestActor,
) (*watchtower.Item, error) {
	return s.applyException(ctx, &exceptionChange{
		target: req,
		actor:  actor,
		mutate: func(item *watchtower.Item, now int64) ([]*watchtower.Event, error) {
			previous := item.Status
			if err := item.Acknowledge(now); err != nil {
				return nil, err
			}
			events := []*watchtower.Event{
				watchtower.NewEvent(item, watchtower.EventAcknowledged, actorUserID(actor)).
					WithChange(previous.String(), item.Status.String()),
			}
			return append(events, claimIfUnowned(item, actor)...), nil
		},
	})
}

func (s *Exceptions) StartException(
	ctx context.Context,
	req *watchtowerports.ExceptionTarget,
	actor *permissionports.RequestActor,
) (*watchtower.Item, error) {
	return s.applyException(ctx, &exceptionChange{
		target: req,
		actor:  actor,
		mutate: func(item *watchtower.Item, now int64) ([]*watchtower.Event, error) {
			previous := item.Status
			if err := item.Start(now); err != nil {
				return nil, err
			}
			events := []*watchtower.Event{
				watchtower.NewEvent(item, watchtower.EventStarted, actorUserID(actor)).
					WithChange(previous.String(), item.Status.String()),
			}
			return append(events, claimIfUnowned(item, actor)...), nil
		},
	})
}

func claimIfUnowned(
	item *watchtower.Item,
	actor *permissionports.RequestActor,
) []*watchtower.Event {
	userID := actorUserID(actor)
	if item.AssigneeID.IsNotNil() || userID.IsNil() || !actor.IsUser() {
		return nil
	}
	item.AssigneeID = userID
	return []*watchtower.Event{
		watchtower.NewEvent(item, watchtower.EventAssigned, userID).WithChange("", userID.String()),
	}
}

func (s *Exceptions) SnoozeException(
	ctx context.Context,
	req *watchtowerports.SnoozeExceptionRequest,
	actor *permissionports.RequestActor,
) (*watchtower.Item, error) {
	return s.applyException(ctx, &exceptionChange{
		target: &req.ExceptionTarget,
		actor:  actor,
		mutate: func(item *watchtower.Item, now int64) ([]*watchtower.Event, error) {
			previous := item.Status
			if err := item.Snooze(req.Until, now); err != nil {
				return nil, err
			}
			return []*watchtower.Event{
				watchtower.NewEvent(item, watchtower.EventSnoozed, actorUserID(actor)).
					WithChange(previous.String(), strconv.FormatInt(req.Until, 10)).
					WithNote(req.Note),
			}, nil
		},
	})
}

func (s *Exceptions) SetExceptionDue(
	ctx context.Context,
	req *watchtowerports.SetExceptionDueRequest,
	actor *permissionports.RequestActor,
) (*watchtower.Item, error) {
	return s.applyException(ctx, &exceptionChange{
		target: &req.ExceptionTarget,
		actor:  actor,
		mutate: func(item *watchtower.Item, now int64) ([]*watchtower.Event, error) {
			previous := formatOptionalTime(item.DueAt)
			if err := item.SetDue(req.DueAt, now); err != nil {
				return nil, err
			}
			return []*watchtower.Event{
				watchtower.NewEvent(item, watchtower.EventDueChanged, actorUserID(actor)).
					WithChange(previous, formatOptionalTime(item.DueAt)).
					WithNote(req.Note),
			}, nil
		},
	})
}

func formatOptionalTime(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func (s *Exceptions) CommentOnException(
	ctx context.Context,
	req *watchtowerports.CommentOnExceptionRequest,
	actor *permissionports.RequestActor,
) (*watchtower.Event, error) {
	body, err := watchtower.ValidateComment(req.Body)
	if err != nil {
		return nil, err
	}
	item, err := s.GetException(ctx, watchtowerports.GetWatchtowerItemRequest{
		ID:         req.ItemID,
		TenantInfo: req.TenantInfo,
	}, actor)
	if err != nil {
		return nil, err
	}

	event := watchtower.NewEvent(item, watchtower.EventCommented, actorUserID(actor)).WithNote(body)
	if req.DryRun {
		return event, nil
	}
	if err = s.repo.InsertEvents(ctx, []*watchtower.Event{event}); err != nil {
		return nil, err
	}
	s.projector.publish(ctx, item, "commented")
	s.auditException(item, item, actor, []*watchtower.Event{event})
	if item.AssigneeID.IsNotNil() && item.AssigneeID != actorUserID(actor) {
		s.notifyAssignee(ctx, item, commentedNotice)
	}

	return event, nil
}

func (s *Exceptions) ResolveException(
	ctx context.Context,
	req *watchtowerports.ResolveExceptionRequest,
	actor *permissionports.RequestActor,
) (*watchtower.Item, error) {
	return s.applyException(ctx, &exceptionChange{
		target: &req.ExceptionTarget,
		actor:  actor,
		mutate: func(item *watchtower.Item, now int64) ([]*watchtower.Event, error) {
			previous := item.Status
			resolver := actorUserID(actor)
			if err := item.Resolve(req.Code, req.Note, resolver, now); err != nil {
				return nil, err
			}
			return []*watchtower.Event{
				watchtower.NewEvent(item, watchtower.EventResolved, resolver).
					WithChange(previous.String(), req.Code.String()).
					WithNote(item.ResolutionNote),
			}, nil
		},
	})
}

func (s *Exceptions) ReopenException(
	ctx context.Context,
	req *watchtowerports.ReopenExceptionRequest,
	actor *permissionports.RequestActor,
) (*watchtower.Item, error) {
	return s.applyException(ctx, &exceptionChange{
		target: &req.ExceptionTarget,
		actor:  actor,
		mutate: func(item *watchtower.Item, now int64) ([]*watchtower.Event, error) {
			previousCode := item.ResolutionCode
			if err := item.Reopen(); err != nil {
				return nil, err
			}
			s.applyDuePolicy(ctx, item, now)
			return []*watchtower.Event{
				watchtower.NewEvent(item, watchtower.EventReopened, actorUserID(actor)).
					WithChange(previousCode.String(), item.Status.String()).
					WithNote(req.Note),
			}, nil
		},
	})
}

func (s *Exceptions) UnsnoozeException(
	ctx context.Context,
	req *watchtowerports.UnsnoozeExceptionRequest,
	actor *permissionports.RequestActor,
) (*watchtower.Item, error) {
	return s.applyException(ctx, &exceptionChange{
		target: &req.ExceptionTarget,
		actor:  actor,
		mutate: func(item *watchtower.Item, _ int64) ([]*watchtower.Event, error) {
			until := formatOptionalTime(item.SnoozedUntil)
			if err := item.Unsnooze(); err != nil {
				return nil, err
			}
			return []*watchtower.Event{
				watchtower.NewEvent(item, watchtower.EventWoke, actorUserID(actor)).
					WithChange(until, item.Status.String()),
			}, nil
		},
	})
}

func (s *Exceptions) ExceptionHistory(
	ctx context.Context,
	req *watchtowerports.ExceptionHistoryRequest,
	actor *permissionports.RequestActor,
) ([]*watchtower.Event, error) {
	if _, err := s.GetException(ctx, watchtowerports.GetWatchtowerItemRequest{
		ID:         req.ItemID,
		TenantInfo: req.TenantInfo,
	}, actor); err != nil {
		return nil, err
	}

	return s.repo.ListEvents(ctx, watchtowerports.ListWatchtowerEventsRequest{
		TenantInfo: req.TenantInfo,
		ItemID:     req.ItemID,
		Limit:      req.Limit,
	})
}

func (s *Exceptions) auditException(
	before, after *watchtower.Item,
	actor *permissionports.RequestActor,
	events []*watchtower.Event,
) {
	if s.audit == nil || len(events) == 0 {
		return
	}

	auditActor := actor.AuditActorOrSystem()
	entries := make([]auditports.BulkLogEntry, 0, len(events))
	for _, event := range events {
		entries = append(entries, auditports.BulkLogEntry{
			Params: &auditports.LogActionParams{
				Resource:       permission.ResourceWatchtower,
				ResourceID:     after.ID.String(),
				Operation:      permission.OpUpdate,
				PreviousState:  exceptionState(before),
				CurrentState:   exceptionState(after),
				UserID:         auditActor.UserID,
				PrincipalType:  auditActor.PrincipalType,
				PrincipalID:    auditActor.PrincipalID,
				APIKeyID:       auditActor.APIKeyID,
				OrganizationID: after.OrganizationID,
				BusinessUnitID: after.BusinessUnitID,
			},
			Options: []auditports.LogOption{
				auditservice.WithComment(exceptionAuditComment(event)),
			},
		})
	}
	if err := s.audit.LogActions(entries); err != nil {
		s.l.Error("failed to audit exception change",
			zap.String("item", after.ID.String()),
			zap.Error(err))
	}
}

func exceptionAuditComment(event *watchtower.Event) string {
	comment := "Exception " + event.Kind.String()
	if event.ToValue != "" {
		comment += ": " + event.ToValue
	}
	return comment
}

func exceptionState(item *watchtower.Item) map[string]any {
	return jsonutils.MustToJSON(map[string]any{
		"sourceKind":      item.SourceKind,
		"sourceId":        item.SourceID,
		"severity":        item.Severity,
		"status":          item.Status,
		"assigneeId":      item.AssigneeID,
		"dueAt":           item.DueAt,
		"snoozedUntil":    item.SnoozedUntil,
		"resolutionCode":  item.ResolutionCode,
		"resolutionNote":  item.ResolutionNote,
		"resolvedById":    item.ResolvedByID,
		"resolvedAt":      item.ResolvedAt,
		"escalationLevel": item.EscalationLevel,
		"dueSource":       item.DueSource,
		"agentId":         item.AgentDefinitionID,
		"agentRunId":      item.AgentRunID,
	})
}

type assigneeNotice int

const (
	assignedNotice assigneeNotice = iota
	commentedNotice
)

func (s *Exceptions) applyDuePolicy(ctx context.Context, item *watchtower.Item, now int64) {
	policy := s.router.Policy(ctx, pagination.TenantInfo{
		OrgID: item.OrganizationID,
		BuID:  item.BusinessUnitID,
	})
	if item.DueSource == watchtower.DueSourceSource && item.DueAt != nil {
		return
	}
	due := DueAt(policy, item.Severity, now)
	item.DueAt = &due
	item.DueLabel = ""
	item.DueSource = watchtower.DueSourcePolicy
}

func (s *Exceptions) notifyAssignee(
	ctx context.Context,
	item *watchtower.Item,
	notice assigneeNotice,
) {
	if s.notifications == nil || item.AssigneeID.IsNil() {
		return
	}

	eventType := EventAssigned
	title := "Exception assigned to you: " + item.Title
	if notice == commentedNotice {
		eventType = EventCommented
		title = "New comment on your exception: " + item.Title
	}
	sendExceptionNotice(ctx, s.notifications, s.l, &exceptionNotice{
		item:      item,
		userID:    item.AssigneeID,
		eventType: eventType,
		title:     title,
		priority:  notification.PriorityMedium,
	})
}

type exceptionNotice struct {
	item      *watchtower.Item
	userID    pulid.ID
	eventType string
	title     string
	priority  notification.Priority
}

func sendExceptionNotice(
	ctx context.Context,
	notifications *notificationservice.Service,
	l *zap.Logger,
	notice *exceptionNotice,
) {
	item := notice.item
	buID := item.BusinessUnitID
	target := notice.userID
	correlation := item.ID.String()
	link := FeedPath + "?item=" + item.ID.String()
	if _, err := notifications.Create(ctx, &notification.Notification{
		OrganizationID:  item.OrganizationID,
		BusinessUnitID:  &buID,
		TargetUserID:    &target,
		EventType:       notice.eventType,
		Priority:        notice.priority,
		Channel:         notification.ChannelUser,
		Title:           notice.title,
		Message:         item.Summary,
		Source:          notificationSource,
		CorrelationID:   &correlation,
		Data:            noticeData(item, link),
		RelatedEntities: map[string]any{"watchtowerItemId": item.ID.String()},
	}); err != nil {
		l.Warn("could not notify about an exception",
			zap.String("item", item.ID.String()),
			zap.String("user", target.String()),
			zap.Error(err))
	}
}

func noticeData(item *watchtower.Item, link string) map[string]any {
	return map[string]any{
		"link":       link,
		"itemId":     item.ID.String(),
		"sourceKind": string(item.SourceKind),
		"sourceId":   item.SourceID,
		"severity":   string(item.Severity),
	}
}
