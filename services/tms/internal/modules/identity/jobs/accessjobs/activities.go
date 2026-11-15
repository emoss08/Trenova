package accessjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/supportaccessservice"
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

type SupportExpiries interface {
	RecordExpiries(ctx context.Context, now int64) (*supportaccessservice.ExpirySweepResult, error)
}

type ActivitiesParams struct {
	fx.In

	Sweeper         Sweeper
	SupportExpiries SupportExpiries
}

type Activities struct {
	sweeper         Sweeper
	supportExpiries SupportExpiries
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{sweeper: p.Sweeper, supportExpiries: p.SupportExpiries}
}

func (a *Activities) RecordSupportAccessExpiriesActivity(
	ctx context.Context,
) (*supportaccessservice.ExpirySweepResult, error) {
	return a.supportExpiries.RecordExpiries(ctx, timeutils.NowUnix())
}

func (a *Activities) SweepOrphanedOrganizationAccessActivity(
	ctx context.Context,
) (*userservice.OrganizationAccessSweepResult, error) {
	return a.sweeper.SweepOrphanedOrganizationAccess(ctx, timeutils.NowUnix())
}
