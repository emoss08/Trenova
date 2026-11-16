package supportaccessservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	ActionGranted        = "support_access.granted"
	ActionRevoked        = "support_access.revoked"
	ActionExtended       = "support_access.extended"
	ActionExpired        = "support_access.expired"
	ActionSessionStarted = "support_access.session_started"
	ActionEntryRefused   = "support_access.entry_refused"
)

type change struct {
	tenantInfo pagination.TenantInfo
	actor      kernelports.AuditActor
	grant      *supportaccess.Grant
	operation  permission.Operation
	action     string
	before     any
	after      any
	comment    string
	metadata   map[string]any
}

func (s *Service) record(ctx context.Context, c *change) {
	metadata := map[string]any{
		"supportAccess":  c.action,
		"supportGrantId": c.grant.ID.String(),
		"staffEmail":     c.grant.StaffEmail,
	}
	for k, v := range c.metadata {
		metadata[k] = v
	}
	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceSupportAccess,
		ResourceID:     c.grant.ID.String(),
		Operation:      c.operation,
		Action:         c.action,
		Actor:          c.actor,
		OrganizationID: c.tenantInfo.OrgID,
		BusinessUnitID: c.tenantInfo.BuID,
		Before:         c.before,
		After:          c.after,
		Comment:        c.comment,
		Metadata:       metadata,
	})
}

func (s *Service) notify(ctx context.Context, notice *supportaccessports.Notice) {
	noticeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), noticeTimeout)
	defer cancel()
	if err := s.notifier.SupportAccessChanged(noticeCtx, notice); err != nil {
		s.l.Error(
			"could not tell the organization's administrators about support access",
			zap.String("grantID", notice.Grant.ID.String()),
			zap.String("notice", string(notice.Kind)),
			zap.Error(err),
		)
	}
}

func grantView(g *supportaccess.Grant, now int64) map[string]any {
	view := map[string]any{
		"staffEmail":      g.StaffEmail,
		"reason":          g.Reason,
		"scope":           g.Scope.String(),
		"allowsAssistant": g.AllowsAssistant,
		"durationHours":   g.DurationHours,
		"expiresAt":       g.ExpiresAt,
		"status":          g.StatusAt(now).String(),
	}
	if g.ExtendedAt != nil {
		view["extendedAt"] = *g.ExtendedAt
	}
	if g.RevokedAt != nil {
		view["revokedAt"] = *g.RevokedAt
		view["revokeReason"] = g.RevokeReason
	}
	return view
}
