package auditservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const securityChangeKind = "security_change"

type SecurityAuditorParams struct {
	fx.In

	AuditService services.AuditService
	Metrics      *metrics.Registry
	Logger       *zap.Logger
}

type SecurityAuditor struct {
	audit   services.AuditService
	metrics *metrics.Registry
	l       *zap.Logger
}

func NewSecurityAuditor(p SecurityAuditorParams) services.SecurityAuditor {
	return &SecurityAuditor{
		audit:   p.AuditService,
		metrics: p.Metrics,
		l:       p.Logger.Named("service.security-audit"),
	}
}

func (a *SecurityAuditor) RecordChange(ctx context.Context, change *services.SecurityChange) {
	change = withRequestTenant(ctx, change)

	params := &services.LogActionParams{
		Resource:       change.Resource,
		ResourceID:     change.ResourceID,
		Operation:      change.Operation,
		UserID:         change.Actor.UserID,
		PrincipalType:  change.Actor.PrincipalType,
		PrincipalID:    change.Actor.PrincipalID,
		APIKeyID:       change.Actor.APIKeyID,
		OrganizationID: change.OrganizationID,
		BusinessUnitID: change.BusinessUnitID,
		Critical:       true,
	}
	params.PreviousState = a.stateOf(change.Before, "before")
	params.CurrentState = a.stateOf(change.After, "after")

	opts := make([]services.LogOption, 0, 5)
	opts = append(opts, WithCategory(audit.CategoryUser), WithRequest(ctx))
	if change.Comment != "" {
		opts = append(opts, WithComment(change.Comment))
	}
	if params.PreviousState != nil && params.CurrentState != nil {
		opts = append(opts, WithDiff(params.PreviousState, params.CurrentState))
	}
	if len(change.Metadata) > 0 {
		opts = append(opts, WithMetadata(change.Metadata))
	}

	err := a.audit.LogAction(params, opts...)
	a.recordMetric(err == nil)
	if err == nil {
		return
	}

	meta, _ := requestmeta.From(ctx)
	a.l.Error("failed to record security change",
		zap.Error(err),
		zap.String("resource", change.Resource.String()),
		zap.String("resourceID", change.ResourceID),
		zap.String("operation", string(change.Operation)),
		zap.String("principalType", string(change.Actor.PrincipalType)),
		zap.String("principalID", change.Actor.PrincipalID.String()),
		zap.String("organizationID", change.OrganizationID.String()),
		zap.String("businessUnitID", change.BusinessUnitID.String()),
		zap.String("comment", change.Comment),
		zap.String("requestID", meta.RequestID),
		zap.String("ipAddress", meta.ClientIP),
		zap.String("userAgent", meta.UserAgent),
	)
}

func withRequestTenant(
	ctx context.Context,
	source *services.SecurityChange,
) *services.SecurityChange {
	change := *source
	tenant, ok := dbscope.From(ctx).Tenant()
	if !ok {
		if change.Actor.PrincipalType == "" {
			change.Actor = services.SystemAuditActor()
		}
		return &change
	}

	if change.OrganizationID.IsNil() || change.BusinessUnitID.IsNil() {
		change.OrganizationID = tenant.OrganizationID
		change.BusinessUnitID = tenant.BusinessUnitID
	}

	if change.Actor.PrincipalType == "" {
		if tenant.UserID.IsNotNil() {
			change.Actor = services.UserActor(pagination.TenantInfo{
				OrgID:  tenant.OrganizationID,
				BuID:   tenant.BusinessUnitID,
				UserID: tenant.UserID,
			}).AuditActor()
		} else {
			change.Actor = services.SystemAuditActor()
		}
	}

	return &change
}

func (a *SecurityAuditor) stateOf(value any, label string) map[string]any {
	if value == nil {
		return nil
	}

	state, err := jsonutils.ToJSONDocument(value)
	if err != nil {
		a.l.Warn(
			"security change state could not be serialized",
			zap.String("state", label),
			zap.Error(err),
		)
		return nil
	}

	return state
}

func (a *SecurityAuditor) recordMetric(success bool) {
	if a.metrics == nil || a.metrics.Audit == nil {
		return
	}
	a.metrics.Audit.RecordSecurityEvent(securityChangeKind, success)
}
