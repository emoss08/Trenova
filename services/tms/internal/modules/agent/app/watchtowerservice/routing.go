package watchtowerservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/ttlcache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	policyCacheTTL     = time.Minute
	policyCacheEntries = 4096
)

type tenantKey struct {
	orgID pulid.ID
	buID  pulid.ID
}

type RouterParams struct {
	fx.In

	Logger   *zap.Logger
	Controls dispatchcontrolports.DispatchControlRepository `optional:"true"`
}

type Router struct {
	l        *zap.Logger
	controls dispatchcontrolports.DispatchControlRepository
	cache    *ttlcache.Cache[tenantKey, dispatchcontrol.ExceptionPolicy]
}

func NewRouter(p RouterParams) *Router {
	return &Router{
		l:        p.Logger.Named("service.watchtower.router"),
		controls: p.Controls,
		cache: ttlcache.New[tenantKey, dispatchcontrol.ExceptionPolicy](ttlcache.Options{
			TTL:        policyCacheTTL,
			MaxEntries: policyCacheEntries,
		}),
	}
}

func (r *Router) Policy(
	ctx context.Context,
	tenant pagination.TenantInfo,
) dispatchcontrol.ExceptionPolicy {
	if r == nil {
		return (*dispatchcontrol.DispatchControl)(nil).ExceptionPolicy()
	}
	key := tenantKey{orgID: tenant.OrgID, buID: tenant.BuID}
	if policy, ok := r.cache.Get(key); ok {
		return policy
	}
	if r.controls == nil {
		return (*dispatchcontrol.DispatchControl)(nil).ExceptionPolicy()
	}

	control, err := r.controls.GetByOrgID(ctx, dispatchcontrolports.GetDispatchControlRequest{
		TenantInfo: pagination.TenantInfo{OrgID: tenant.OrgID, BuID: tenant.BuID},
	})
	if err != nil {
		r.l.Debug("exception policy unavailable; using defaults",
			zap.String("organization", tenant.OrgID.String()),
			zap.Error(err))
		return (*dispatchcontrol.DispatchControl)(nil).ExceptionPolicy()
	}

	policy := control.ExceptionPolicy()
	r.cache.Set(key, policy)
	return policy
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
