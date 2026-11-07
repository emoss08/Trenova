package auditalertservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	notificationSource = "auditalertservice"
	maxRecipients      = 200
)

type RecipientLister interface {
	ListUsersWithPermission(
		ctx context.Context,
		req permissionports.ListUsersWithPermissionRequest,
	) ([]permissionports.PermittedUser, error)
}

type NotificationCreator interface {
	NotifiedUsers(
		ctx context.Context,
		req notificationports.ExistsRecentNotificationRequest,
	) ([]pulid.ID, error)
	Create(
		ctx context.Context,
		entity *notification.Notification,
	) (*notification.Notification, error)
}

type Params struct {
	fx.In

	Policies      auditports.AlertPolicyRepository
	Roles         permissionports.RoleRepository
	Notifications *notificationservice.Service
	Auditor       permissionports.SecurityAuditor
	Alerts        *auditservice.Alerts `optional:"true"`
	Logger        *zap.Logger
}

type Service struct {
	policies      auditports.AlertPolicyRepository
	recipients    RecipientLister
	notifications NotificationCreator
	auditor       permissionports.SecurityAuditor
	alerts        *auditservice.Alerts
	l             *zap.Logger
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) *Service {
	return &Service{
		policies:      p.Policies,
		recipients:    p.Roles,
		notifications: p.Notifications,
		auditor:       p.Auditor,
		alerts:        p.Alerts,
		l:             p.Logger.Named("service.audit-alerts"),
	}
}

type PolicyView struct {
	Definition audit.AlertDefinition
	Policy     *audit.AlertPolicy
}

func (s *Service) ListPolicies(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]PolicyView, error) {
	stored, err := s.policies.ListForTenant(ctx, tenant)
	if err != nil {
		return nil, err
	}
	byKind := make(map[audit.AlertKind]*audit.AlertPolicy, len(stored))
	for _, policy := range stored {
		byKind[policy.Kind] = policy
	}

	views := make([]PolicyView, 0, len(audit.AlertDefinitions))
	for _, def := range audit.AlertDefinitions {
		policy, ok := byKind[def.Kind]
		if !ok {
			policy = audit.DefaultAlertPolicy(def)
			policy.OrganizationID = tenant.OrgID
			policy.BusinessUnitID = tenant.BuID
		}
		views = append(views, PolicyView{Definition: def, Policy: policy})
	}
	return views, nil
}

type UpdatePolicyRequest struct {
	TenantInfo    pagination.TenantInfo
	Actor         kernelports.AuditActor
	Kind          audit.AlertKind
	Enabled       bool
	Threshold     int
	WindowMinutes int
}

func (s *Service) UpdatePolicy(ctx context.Context, req *UpdatePolicyRequest) (*PolicyView, error) {
	def, ok := audit.AlertDefinitionFor(req.Kind)
	if !ok {
		return nil, errortypes.NewValidationError("kind", errortypes.ErrInvalid, "Trenova raises no such alert")
	}

	views, err := s.ListPolicies(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	var before *audit.AlertPolicy
	for i := range views {
		if views[i].Definition.Kind == req.Kind {
			before = views[i].Policy
		}
	}

	policy := &audit.AlertPolicy{
		ID:             pulid.MustNew("aap_"),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Kind:           req.Kind,
		Enabled:        req.Enabled,
		Threshold:      req.Threshold,
		WindowMinutes:  req.WindowMinutes,
	}
	multiErr := errortypes.NewMultiError()
	policy.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	saved, err := s.policies.Upsert(ctx, policy)
	if err != nil {
		return nil, err
	}
	s.alerts.InvalidatePolicies(auditports.TenantKey{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	})

	if s.auditor != nil {
		s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
			Resource:       permission.ResourceSecurityAuditLog,
			ResourceID:     string(req.Kind),
			Operation:      permission.OpUpdate,
			Action:         audit.ActionAlertPolicyChanged,
			Actor:          req.Actor,
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			Before:         before,
			After:          saved,
			Comment:        "Security alert changed: " + def.Title,
		})
	}

	return &PolicyView{Definition: def, Policy: saved}, nil
}

func (s *Service) Notify(ctx context.Context, payload *auditports.AlertPayload) (int, error) {
	def, ok := audit.AlertDefinitionFor(payload.Kind)
	if !ok {
		return 0, fmt.Errorf("unknown audit alert kind %q", payload.Kind)
	}

	recipients, err := s.recipients.ListUsersWithPermission(
		ctx,
		permissionports.ListUsersWithPermissionRequest{
			OrganizationID: payload.TenantInfo.OrgID,
			BusinessUnitID: payload.TenantInfo.BuID,
			Resource:       permission.ResourceSecurityAuditLog,
			Operation:      permission.OpRead,
			Now:            timeutils.NowUnix(),
		},
	)
	if err != nil {
		return 0, fmt.Errorf("list security reviewers: %w", err)
	}
	if len(recipients) > maxRecipients {
		recipients = recipients[:maxRecipients]
	}

	message := alertMessage(def, payload)
	correlation := payload.EntryID.String()
	eventType := "audit.alert." + string(payload.Kind)
	buID := payload.TenantInfo.BuID
	notified, err := s.notifications.NotifiedUsers(ctx, notificationports.ExistsRecentNotificationRequest{
		OrganizationID: payload.TenantInfo.OrgID,
		BusinessUnitID: buID,
		EventType:      eventType,
		CorrelationID:  correlation,
		Since:          payload.OccurredAt,
	})
	if err != nil {
		return 0, fmt.Errorf("list notified security reviewers: %w", err)
	}
	already := make(map[pulid.ID]struct{}, len(notified))
	for _, id := range notified {
		already[id] = struct{}{}
	}
	sent := 0
	for _, recipient := range recipients {
		userID := recipient.UserID
		if _, ok := already[userID]; ok {
			continue
		}
		if _, err = s.notifications.Create(ctx, &notification.Notification{
			OrganizationID: payload.TenantInfo.OrgID,
			BusinessUnitID: &buID,
			TargetUserID:   &userID,
			Channel:        notification.ChannelUser,
			EventType:      eventType,
			Priority:       notification.PriorityHigh,
			Title:          def.Title,
			Message:        message,
			Source:         notificationSource,
			CorrelationID:  &correlation,
			Data: map[string]any{
				"link":          "/admin/audit-logs?entry=" + payload.EntryID.String(),
				"kind":          string(payload.Kind),
				"entryId":       payload.EntryID.String(),
				"action":        payload.Action,
				"resource":      payload.Resource.String(),
				"resourceId":    payload.ResourceID,
				"principalId":   payload.PrincipalID.String(),
				"principalType": payload.PrincipalType,
				"count":         payload.Count,
			},
			RelatedEntities: map[string]any{
				"auditEntryId": payload.EntryID.String(),
			},
		}); err != nil {
			return sent, fmt.Errorf("notify security reviewer: %w", err)
		}
		sent++
	}
	if len(already) == 0 {
		s.recordRaised(ctx, def, payload, message, len(recipients))
	}
	return sent, nil
}

func (s *Service) recordRaised(
	ctx context.Context,
	def audit.AlertDefinition,
	payload *auditports.AlertPayload,
	message string,
	recipients int,
) {
	if s.auditor == nil {
		return
	}
	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceSecurityAuditLog,
		ResourceID:     string(payload.Kind),
		Operation:      permission.OpCreate,
		Action:         audit.ActionAlertRaised,
		OrganizationID: payload.TenantInfo.OrgID,
		BusinessUnitID: payload.TenantInfo.BuID,
		After: map[string]any{
			"kind":       string(payload.Kind),
			"entryId":    payload.EntryID.String(),
			"action":     payload.Action,
			"resource":   payload.Resource.String(),
			"resourceId": payload.ResourceID,
			"count":      payload.Count,
			"recipients": recipients,
		},
		Comment: "Security alert raised: " + def.Title,
		Metadata: map[string]any{
			"message": message,
		},
	})
}

func alertMessage(def audit.AlertDefinition, payload *auditports.AlertPayload) string {
	actor := payload.ActorName
	if actor == "" {
		actor = "Someone"
	}
	if def.Counted {
		return fmt.Sprintf("%s: %s did this %d times in a short time.", def.Description, actor, payload.Count)
	}
	subject := payload.EntityRef
	if subject == "" {
		subject = payload.ResourceID
	}
	if payload.Comment != "" {
		return fmt.Sprintf("%s (%s, by %s).", payload.Comment, subject, actor)
	}
	return fmt.Sprintf("%s (%s, by %s).", def.Description, subject, actor)
}
