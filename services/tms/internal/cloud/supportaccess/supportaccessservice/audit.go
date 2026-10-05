package supportaccessservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	supportctx "github.com/emoss08/trenova/internal/cloud/supportaccess"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	maxNameLength         = 255
	maxNotifiedAdmins     = 25
	notificationSource    = "support_access"
	eventSessionStarted   = "support_access.session_started"
	metadataSupportAccess = "supportAccess"
)

type sessionOperation string

const (
	sessionStarted  sessionOperation = "session_started"
	sessionElevated sessionOperation = "session_elevated"
	sessionDemoted  sessionOperation = "session_returned_to_read_only"
	sessionEnded    sessionOperation = "session_ended"
	sessionRefused  sessionOperation = "request_refused"
)

type grantOperation string

const (
	grantCreated grantOperation = "grant_created"
	grantRevoked grantOperation = "grant_revoked"
)

func (o sessionOperation) permissionOperation() permission.Operation {
	switch o {
	case sessionStarted:
		return permission.OpCreate
	case sessionEnded:
		return permission.OpClose
	case sessionRefused:
		return permission.OpRead
	case sessionElevated, sessionDemoted:
		return permission.OpUpdate
	default:
		return permission.OpUpdate
	}
}

type sessionEvent struct {
	session   *supportaccess.Session
	operation sessionOperation
	comment   string
	metadata  map[string]any
}

func (s *Service) recordSessionEvent(ctx context.Context, event *sessionEvent) {
	recordSessionAudit(ctx, s.auditor, event)
}

func recordSessionAudit(
	ctx context.Context,
	auditor services.SecurityAuditor,
	event *sessionEvent,
) {
	if auditor == nil {
		return
	}

	session := event.session
	metadata := map[string]any{
		metadataSupportAccess: string(event.operation),
		"supportSessionId":    session.ID.String(),
		"grantId":             session.GrantID.String(),
		"staffUserId":         session.StaffUserID.String(),
		"staffName":           session.StaffName,
		"reason":              session.Reason,
	}
	if session.TicketReference != "" {
		metadata["ticketReference"] = session.TicketReference
	}
	if session.ElevationReason != "" && event.operation == sessionElevated {
		metadata["elevationReason"] = session.ElevationReason
	}
	for key, value := range event.metadata {
		metadata[key] = value
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  session.OrganizationID,
		BuID:   session.BusinessUnitID,
		UserID: session.PrincipalUserID,
	}

	auditor.RecordChange(
		tenantScope(ctx, session.OrganizationID, session.BusinessUnitID, session.PrincipalUserID),
		&services.SecurityChange{
			Resource:       permission.ResourceOrganization,
			ResourceID:     session.ID.String(),
			Operation:      event.operation.permissionOperation(),
			Actor:          services.UserActor(tenantInfo).AuditActor(),
			OrganizationID: session.OrganizationID,
			BusinessUnitID: session.BusinessUnitID,
			Comment:        event.comment,
			Metadata:       metadata,
		},
	)
}

type grantEvent struct {
	tenantInfo    pagination.TenantInfo
	grantID       pulid.ID
	operation     grantOperation
	before        *supportaccess.Grant
	after         *supportaccess.Grant
	comment       string
	endedSessions []pulid.ID
}

func (s *Service) recordGrantEvent(ctx context.Context, event *grantEvent) {
	if s.auditor == nil {
		return
	}

	ended := make([]string, 0, len(event.endedSessions))
	for _, id := range event.endedSessions {
		ended = append(ended, id.String())
	}

	op := permission.OpCreate
	if event.operation == grantRevoked {
		op = permission.OpDelete
	}

	change := &services.SecurityChange{
		Resource:       permission.ResourceOrganization,
		ResourceID:     event.grantID.String(),
		Operation:      op,
		Actor:          services.UserActor(event.tenantInfo).AuditActor(),
		OrganizationID: event.tenantInfo.OrgID,
		BusinessUnitID: event.tenantInfo.BuID,
		Comment:        event.comment,
		Metadata: map[string]any{
			metadataSupportAccess:    string(event.operation),
			"grantId":                event.grantID.String(),
			"endedSupportSessionIds": ended,
		},
	}
	if event.before != nil {
		change.Before = event.before
	}
	if event.after != nil {
		change.After = event.after
	}

	s.auditor.RecordChange(ctx, change)
}

type RefusalRequest struct {
	Active  *supportctx.Active
	Method  string
	Route   string
	Subject string
	Denied  bool
}

func (s *Service) RecordRefusal(ctx context.Context, req *RefusalRequest) {
	if s.auditor == nil || req.Active == nil {
		return
	}

	why := "the session is read-only"
	if req.Denied {
		why = "support sessions may not do this"
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  req.Active.OrganizationID,
		BuID:   req.Active.BusinessUnitID,
		UserID: req.Active.PrincipalUserID,
	}

	s.auditor.RecordChange(
		tenantScope(
			ctx,
			req.Active.OrganizationID,
			req.Active.BusinessUnitID,
			req.Active.PrincipalUserID,
		),
		&services.SecurityChange{
			Resource:       permission.ResourceOrganization,
			ResourceID:     req.Active.SessionID.String(),
			Operation:      sessionRefused.permissionOperation(),
			Actor:          services.UserActor(tenantInfo).AuditActor(),
			OrganizationID: req.Active.OrganizationID,
			BusinessUnitID: req.Active.BusinessUnitID,
			Comment: "Refused " + req.Subject + " for Trenova support (" + req.Active.StaffName +
				") because " + why,
			Metadata: map[string]any{
				metadataSupportAccess: string(sessionRefused),
				"supportSessionId":    req.Active.SessionID.String(),
				"staffUserId":         req.Active.StaffUserID.String(),
				"method":              req.Method,
				"route":               req.Route,
				"mode":                req.Active.EffectiveMode().String(),
			},
		},
	)
}

func (s *Service) notifyAdmins(
	ctx context.Context,
	session *supportaccess.Session,
	grant *supportaccess.Grant,
) {
	if s.notifier == nil {
		return
	}

	tenantInfo := pagination.TenantInfo{OrgID: session.OrganizationID, BuID: session.BusinessUnitID}
	correlation := session.ID.String()
	if _, err := s.notifier.NotifyPermitted(
		tenantScope(ctx, session.OrganizationID, session.BusinessUnitID, session.PrincipalUserID),
		notificationservice.NotifyPermittedRequest{
			Tenant:    tenantInfo,
			Resource:  permission.ResourceOrganization,
			Operation: permission.OpUpdate,
			Limit:     maxNotifiedAdmins,
			Now:       session.StartedAt,
			Notification: notification.Notification{
				EventType: eventSessionStarted,
				Priority:  notification.PriorityHigh,
				Title:     "Trenova support opened a session",
				Message: session.StaffName + " from Trenova support opened a session (" +
					accessModeText(grant.AccessMode) + " allowed): " + session.Reason,
				Source:        notificationSource,
				CorrelationID: &correlation,
				Data: map[string]any{
					"link":             "/admin/support-access",
					"supportSessionId": correlation,
					"staffName":        session.StaffName,
				},
				RelatedEntities: map[string]any{"supportSessionId": correlation},
			},
		},
	); err != nil {
		s.l.Warn("could not tell administrators a support session started",
			zap.String("supportSessionId", correlation),
			zap.Error(err),
		)
	}
}

func endReasonText(reason supportaccess.EndReason) string {
	switch reason {
	case supportaccess.EndReasonExited:
		return "the staff member exited"
	case supportaccess.EndReasonExpired:
		return "the session reached its time limit"
	case supportaccess.EndReasonGrantRevoked:
		return "the organization revoked support access"
	case supportaccess.EndReasonGrantExpired:
		return "the organization's support access expired"
	case supportaccess.EndReasonStaffRemoved:
		return "the staff member is no longer Trenova staff"
	case supportaccess.EndReasonSignedOut:
		return "the staff member signed out"
	case supportaccess.EndReasonReplaced:
		return "the staff member opened another session"
	case supportaccess.EndReasonAssuranceLost:
		return "the staff member's sign-in no longer meets two-factor requirements"
	default:
		return string(reason)
	}
}

func accessModeText(mode supportaccess.AccessMode) string {
	if mode == supportaccess.AccessModeReadWrite {
		return "read-write"
	}
	return "read-only"
}

func ticketSuffix(ticket string) string {
	if strings.TrimSpace(ticket) == "" {
		return ""
	}
	return " (ticket " + strings.TrimSpace(ticket) + ")"
}

func truncate(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}

func asEnded(err error, target **SessionEndedError) bool {
	return errors.As(err, target)
}

func AsSessionEnded(err error) (*SessionEndedError, bool) {
	var ended *SessionEndedError
	if errors.As(err, &ended) {
		return ended, true
	}
	return nil, false
}
