package watchtowerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type RouterParams struct {
	fx.In

	Logger   *zap.Logger
	Controls dispatchcontrolports.DispatchControlRepository `optional:"true"`
}

type Router struct {
	l        *zap.Logger
	controls dispatchcontrolports.DispatchControlRepository
}

func NewRouter(p RouterParams) *Router {
	return &Router{
		l:        p.Logger.Named("service.watchtower.router"),
		controls: p.Controls,
	}
}

func (r *Router) Policy(
	ctx context.Context,
	tenant pagination.TenantInfo,
) dispatchcontrol.ExceptionPolicy {
	if r == nil || r.controls == nil {
		return (*dispatchcontrol.DispatchControl)(nil).ExceptionPolicy()
	}

	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	control, err := r.controls.GetByOrgID(ctx, dispatchcontrolports.GetDispatchControlRequest{
		TenantInfo: pagination.TenantInfo{OrgID: tenant.OrgID, BuID: tenant.BuID},
	})
	if err != nil {
		r.l.Debug("exception policy unavailable; using defaults",
			zap.String("organization", tenant.OrgID.String()),
			zap.Error(err))
		return (*dispatchcontrol.DispatchControl)(nil).ExceptionPolicy()
	}

	return control.ExceptionPolicy()
}

func DueAfterSeconds(policy dispatchcontrol.ExceptionPolicy, severity watchtower.Severity) int64 {
	switch severity {
	case watchtower.SeverityCritical:
		return int64(policy.DueCriticalMinutes) * 60
	case watchtower.SeverityWarning:
		return int64(policy.DueWarningMinutes) * 60
	case watchtower.SeverityInfo:
		return int64(policy.DueInfoMinutes) * 60
	}
	return int64(policy.DueInfoMinutes) * 60
}

func DueAt(policy dispatchcontrol.ExceptionPolicy, severity watchtower.Severity, now int64) int64 {
	return now + DueAfterSeconds(policy, severity)
}

func RouteAssignee(policy dispatchcontrol.ExceptionPolicy, ownerID pulid.ID) pulid.ID {
	if ownerID.IsNotNil() {
		return ownerID
	}
	return policy.DefaultAssigneeID
}
