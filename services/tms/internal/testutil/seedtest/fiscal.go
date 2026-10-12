package seedtest

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/pkg/buncolgen/fiscalperiodcol"
	"github.com/emoss08/trenova/pkg/buncolgen/fiscalyearcol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func OpenFiscalPeriod(
	t *testing.T,
	ctx context.Context,
	db bun.IDB,
	orgID, buID pulid.ID,
) (*fiscalyear.FiscalYear, *fiscalperiod.FiscalPeriod) {
	t.Helper()

	now := time.Now().UTC()
	fy := currentFiscalYear(t, ctx, db, orgID, buID, now)

	yearCols := fiscalyearcol.FiscalYearColumns
	_, err := db.NewUpdate().
		Model((*fiscalyear.FiscalYear)(nil)).
		Set(yearCols.AllowAdjustingEntries.Set(), true).
		Where(yearCols.ID.Eq(), fy.ID).
		Exec(ctx)
	require.NoError(t, err)
	fy.AllowAdjustingEntries = true

	periodCols := fiscalperiodcol.FiscalPeriodColumns
	period := new(fiscalperiod.FiscalPeriod)
	require.NoError(t, db.NewSelect().
		Model(period).
		Where(periodCols.FiscalYearID.Eq(), fy.ID).
		Where(periodCols.PeriodType.NotEq(), fiscalperiod.PeriodTypeAdjusting).
		Where(periodCols.StartDate.Lte(), now.Unix()).
		Where(periodCols.EndDate.Gte(), now.Unix()).
		Limit(1).
		Scan(ctx))

	_, err = db.NewUpdate().
		Model((*fiscalperiod.FiscalPeriod)(nil)).
		Set(periodCols.Status.Set(), fiscalperiod.StatusOpen).
		Set(periodCols.AllowAdjustingEntries.Set(), true).
		Where(periodCols.ID.Eq(), period.ID).
		Exec(ctx)
	require.NoError(t, err)
	period.Status = fiscalperiod.StatusOpen
	period.AllowAdjustingEntries = true

	return fy, period
}

func currentFiscalYear(
	t *testing.T,
	ctx context.Context,
	db bun.IDB,
	orgID, buID pulid.ID,
	now time.Time,
) *fiscalyear.FiscalYear {
	t.Helper()

	cols := fiscalyearcol.FiscalYearColumns
	fy := new(fiscalyear.FiscalYear)
	err := db.NewSelect().
		Model(fy).
		Where(cols.OrganizationID.Eq(), orgID).
		Where(cols.BusinessUnitID.Eq(), buID).
		Where(cols.StartDate.Lte(), now.Unix()).
		Where(cols.EndDate.Gte(), now.Unix()).
		Limit(1).
		Scan(ctx)
	if err == nil {
		return fy
	}
	require.True(t, errors.Is(err, sql.ErrNoRows), "read current fiscal year: %v", err)

	start := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
	fy = &fiscalyear.FiscalYear{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         fiscalyear.StatusOpen,
		Year:           now.Year(),
		Name:           "FY " + strconv.Itoa(now.Year()),
		StartDate:      start.Unix(),
		EndDate:        start.AddDate(1, 0, 0).Add(-time.Second).Unix(),
		IsCurrent:      true,
		IsCalendarYear: true,
	}
	_, err = db.NewInsert().Model(fy).Exec(ctx)
	require.NoError(t, err)

	periods := fy.GenerateMonthlyPeriods()
	_, err = db.NewInsert().Model(&periods).Exec(ctx)
	require.NoError(t, err)

	return fy
}
