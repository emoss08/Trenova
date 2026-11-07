package demandforecastrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast/demandforecastdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxSeriesPerCell = 60
	maxModelRates    = 200_000
	maxModelAreas    = 5_000
	maxModelCurves   = 20_000
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) demandforecastports.Repository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.demand-forecast-repository"),
	}
}

func (r *repository) Replace(ctx context.Context, req *demandforecastports.ReplaceRequest) error {
	if req.Fitted == nil || req.Quality == nil {
		return errors.New("replace demand forecast: a fit and its quality are required")
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		rates := demandforecastdb.Rates
		areas := demandforecastdb.Areas
		curves := demandforecastdb.BookingCurves

		if _, err := rates.Where(keel.True()).Delete(ctx, db); err != nil {
			return fmt.Errorf("clear forecast rates: %w", err)
		}
		if _, err := areas.Where(keel.True()).Delete(ctx, db); err != nil {
			return fmt.Errorf("clear forecast areas: %w", err)
		}
		if _, err := curves.Where(keel.True()).Delete(ctx, db); err != nil {
			return fmt.Errorf("clear booking curves: %w", err)
		}
		if len(req.Fitted.Rates) > 0 {
			if err := rates.CreateMany(ctx, db, req.Fitted.Rates); err != nil {
				return fmt.Errorf("write forecast rates: %w", err)
			}
		}
		if len(req.Fitted.Areas) > 0 {
			if err := areas.CreateMany(ctx, db, req.Fitted.Areas); err != nil {
				return fmt.Errorf("write forecast areas: %w", err)
			}
		}
		if len(req.Fitted.Curves) > 0 {
			if err := curves.CreateMany(ctx, db, req.Fitted.Curves); err != nil {
				return fmt.Errorf("write booking curves: %w", err)
			}
		}
		days := demandforecastdb.Days
		if _, err := days.Where(keel.True()).Delete(ctx, db); err != nil {
			return fmt.Errorf("clear forecast days: %w", err)
		}
		if len(req.Days) > 0 {
			if err := days.CreateMany(ctx, db, req.Days); err != nil {
				return fmt.Errorf("write forecast days: %w", err)
			}
		}

		q := demandforecastdb.Qualities
		if _, err := q.Upsert(ctx, db, []*demandforecast.Quality{req.Quality},
			keel.OnConflict(q.OrganizationID, q.BusinessUnitID).Update(
				q.Status, q.HistoryWeeks, q.HistoryLoads, q.WeeksTested, q.LoadsTested,
				q.IntervalCoverage, q.MAE, q.NaiveMAE, q.Skill, q.FittedAt,
			)); err != nil {
			return fmt.Errorf("write forecast quality: %w", err)
		}

		return nil
	})
}

func (r *repository) Quality(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*demandforecast.Quality, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*demandforecast.Quality, error) {
		return r.quality(ctx, tenantInfo)
	})
}

func (r *repository) quality(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*demandforecast.Quality, error) {
	q := demandforecastdb.Qualities
	rows, err := q.Query().Limit(1).List(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, fmt.Errorf("read forecast quality: %w", err)
	}
	if len(rows) == 0 {
		return &demandforecast.Quality{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Status:         demandforecast.StatusInsufficientHistory,
		}, nil
	}

	return rows[0], nil
}

func (r *repository) Days(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*demandforecast.Day, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*demandforecast.Day, error) {
		d := demandforecastdb.Days
		rows, err := d.Query().
			Order(d.DayStart.Asc()).
			Limit(demandforecast.RecordDays).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read forecast days: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) ModelForCells(
	ctx context.Context,
	req *demandforecastports.CellsRequest,
) (*demandforecastports.Model, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*demandforecastports.Model, error) {
		return r.model(ctx, req.TenantInfo, req.Cells, true)
	})
}

func (r *repository) Model(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*demandforecastports.Model, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*demandforecastports.Model, error) {
		return r.model(ctx, tenantInfo, nil, false)
	})
}

func (r *repository) model(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	cells []int64,
	byCell bool,
) (*demandforecastports.Model, error) {
	quality, err := r.quality(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	model := &demandforecastports.Model{Quality: quality}
	if !quality.Ready() || (byCell && len(cells) == 0) {
		return model, nil
	}

	db := r.db.Keel(ctx)
	rates := demandforecastdb.Rates
	areas := demandforecastdb.Areas
	curves := demandforecastdb.BookingCurves
	rateLimit, areaLimit := maxModelRates, maxModelAreas
	if byCell {
		rateLimit = min(len(cells)*int(dispatchmarket.SlotsPerWeek)*maxSeriesPerCell, maxModelRates)
		areaLimit = min(len(cells), maxModelAreas)
	}
	if model.Rates, err = rates.Where(keel.When(byCell, rates.Cell.In(cells...))).Limit(rateLimit).List(ctx, db); err != nil {
		return nil, fmt.Errorf("read forecast rates: %w", err)
	}
	if model.Areas, err = areas.Where(keel.When(byCell, areas.Cell.In(cells...))).Limit(areaLimit).List(ctx, db); err != nil {
		return nil, fmt.Errorf("read forecast areas: %w", err)
	}
	if model.Curves, err = curves.Query().Limit(maxModelCurves).List(ctx, db); err != nil {
		return nil, fmt.Errorf("read booking curves: %w", err)
	}

	return model, nil
}
