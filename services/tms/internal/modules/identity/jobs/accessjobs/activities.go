package accessjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/supportaccessservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/supportlogservice"
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

type SupportLogTimeouts interface {
	FailOverdue(ctx context.Context, now int64) (*supportlogservice.SweepResult, error)
}

type ActivitiesParams struct {
	fx.In

	Sweeper            Sweeper
	SupportExpiries    SupportExpiries
	SupportLogTimeouts SupportLogTimeouts
}

type Activities struct {
	sweeper            Sweeper
	supportExpiries    SupportExpiries
	supportLogTimeouts SupportLogTimeouts
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		sweeper:            p.Sweeper,
		supportExpiries:    p.SupportExpiries,
		supportLogTimeouts: p.SupportLogTimeouts,
	}
}

func (a *Activities) RecordSupportAccessExpiriesActivity(
	ctx context.Context,
) (*supportaccessservice.ExpirySweepResult, error) {
	return a.supportExpiries.RecordExpiries(ctx, timeutils.NowUnix())
}

func (a *Activities) FailOverdueSupportLogRequestsActivity(
	ctx context.Context,
) (*supportlogservice.SweepResult, error) {
	return a.supportLogTimeouts.FailOverdue(ctx, timeutils.NowUnix())
}

func (a *Activities) SweepOrphanedOrganizationAccessActivity(
	ctx context.Context,
) (*userservice.OrganizationAccessSweepResult, error) {
	return a.sweeper.SweepOrphanedOrganizationAccess(ctx, timeutils.NowUnix())
}
