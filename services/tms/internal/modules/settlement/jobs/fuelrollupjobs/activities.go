package fuelrollupjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Refresher fuelpurchaseports.TractorDayRefresher
	Logger    *zap.Logger
}

type Activities struct {
	refresher fuelpurchaseports.TractorDayRefresher
	l         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{refresher: p.Refresher, l: p.Logger.Named("temporal.fuel-tractor-days")}
}

func (a *Activities) RefreshFuelTractorDaysActivity(
	ctx context.Context,
	input *RefreshInput,
) (*RefreshResult, error) {
	rows, err := a.refresher.RefreshTractorDays(ctx, input.Days)
	if err != nil {
		return nil, err
	}
	a.l.Debug("fuel tractor days refreshed", zap.Int("days", input.Days), zap.Int64("rows", rows))

	return &RefreshResult{Rows: rows}, nil
}
