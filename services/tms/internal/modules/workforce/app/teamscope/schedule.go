package teamscope

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/identity/app/orgstructureservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/weekhoursports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ScheduleNarrowerParams struct {
	fx.In

	Permissions permissionports.PermissionEngine
	Managers    *orgstructureservice.Service
}

type ScheduleNarrower struct {
	guard ListGuard
}

func NewScheduleNarrower(p ScheduleNarrowerParams) weekhoursports.TeamNarrower {
	return &ScheduleNarrower{guard: ListGuard{Permissions: p.Permissions, Managers: p.Managers}}
}

func (n *ScheduleNarrower) Narrow(
	ctx context.Context,
	actor *permissionports.RequestActor,
	tenant pagination.TenantInfo,
) ([]pulid.ID, error) {
	ids, err := n.guard.Narrow(ctx, &Request{
		Actor:      actor,
		TenantInfo: tenant,
		Resource:   permission.ResourceWorkerSchedule,
		Operation:  permission.OpRead,
	})
	if errors.Is(err, ErrOutsideTeam) {
		return nil, weekhoursports.ErrNoTeam
	}

	return ids, err
}
