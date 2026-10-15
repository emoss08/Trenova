package permissionchange

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	Resource      = "permissions"
	ActionChanged = "changed"
)

type Params struct {
	fx.In

	Cache    permissionports.PermissionCacheRepository
	Realtime agentports.RealtimeService
	Logger   *zap.Logger
}

type Notifier struct {
	cache    permissionports.PermissionCacheRepository
	realtime agentports.RealtimeService
	l        *zap.Logger
}

func New(p Params) permissionports.PermissionChangeNotifier {
	return &Notifier{
		cache:    p.Cache,
		realtime: p.Realtime,
		l:        p.Logger.Named("service.permission-change"),
	}
}

func (n *Notifier) OrganizationChanged(ctx context.Context, tenant pagination.TenantInfo) {
	if err := n.cache.InvalidateOrganization(ctx, tenant.OrgID); err != nil {
		n.l.Warn(
			"failed to invalidate the organization's permission cache",
			zap.String("orgID", tenant.OrgID.String()),
			zap.Error(err),
		)
	}
	n.announce(ctx, tenant, pulid.Nil)
}

func (n *Notifier) UserChanged(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) {
	if err := n.cache.Delete(ctx, userID, tenant.OrgID); err != nil {
		n.l.Warn(
			"failed to invalidate a user's permission cache",
			zap.String("orgID", tenant.OrgID.String()),
			zap.String("userID", userID.String()),
			zap.Error(err),
		)
	}
	n.announce(ctx, tenant, userID)
}

func (n *Notifier) announce(
	ctx context.Context,
	tenant pagination.TenantInfo,
	audience pulid.ID,
) {
	if n.realtime == nil || tenant.OrgID.IsNil() || tenant.BuID.IsNil() {
		return
	}

	if err := n.realtime.PublishResourceInvalidation(
		ctx,
		&agentports.PublishResourceInvalidationRequest{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			AudienceUserID: audience,
			Resource:       Resource,
			Action:         ActionChanged,
			ActorUserID:    tenant.UserID,
		},
	); err != nil {
		n.l.Warn(
			"failed to announce a permission change",
			zap.String("orgID", tenant.OrgID.String()),
			zap.Error(err),
		)
	}
}
