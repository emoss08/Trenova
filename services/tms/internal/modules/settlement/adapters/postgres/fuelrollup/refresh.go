package fuelrollup

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
)

const (
	refreshStatement = "SELECT fuel_tractor_daily_refresh($1)"
	systemReason     = "rebuild every organization's trailing fuel and miles by tractor and day"
	minDays          = 0
	maxDays          = 400
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Refresher struct{ db *postgres.Connection }

var _ fuelpurchaseports.TractorDayRefresher = (*Refresher)(nil)

func New(p Params) *Refresher { return &Refresher{db: p.DB} }

func (r *Refresher) RefreshTractorDays(ctx context.Context, days int) (int64, error) {
	if days < minDays || days > maxDays {
		return 0, fmt.Errorf("refresh fuel tractor days: %d days is outside 0 to %d", days, maxDays)
	}
	ctx = dbscope.WithSystem(ctx, systemReason)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		written, err := keel.QueryValue[int64](ctx, r.db.Keel(ctx), refreshStatement, days)
		if err != nil {
			return 0, fmt.Errorf("refresh fuel tractor days: %w", err)
		}

		return written, nil
	})
}
