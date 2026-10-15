package accessjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type Sweeper interface {
	SweepOrphanedOrganizationAccess(
		ctx context.Context,
		now int64,
	) (*userservice.OrganizationAccessSweepResult, error)
}

type ActivitiesParams struct {
	fx.In

	Sweeper Sweeper
}

type Activities struct {
	sweeper Sweeper
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{sweeper: p.Sweeper}
}

func (a *Activities) SweepOrphanedOrganizationAccessActivity(
	ctx context.Context,
) (*userservice.OrganizationAccessSweepResult, error) {
	return a.sweeper.SweepOrphanedOrganizationAccess(ctx, timeutils.NowUnix())
}
