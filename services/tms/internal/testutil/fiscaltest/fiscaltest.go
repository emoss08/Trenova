package fiscaltest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

type OpenPeriodParams struct {
	DB             bun.IDB
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	At             int64
}

func OpenPeriod(
	t testing.TB,
	ctx context.Context,
	p OpenPeriodParams,
) (*fiscalyear.FiscalYear, *fiscalperiod.FiscalPeriod) {
	t.Helper()

	fy := findYear(t, ctx, p)
	if fy == nil {
		fy = createCalendarYear(t, ctx, p)
	}

	fyCols := fiscalyearcol.FiscalYearColumns
	_, err := p.DB.NewUpdate().
		Model((*fiscalyear.FiscalYear)(nil)).
		Set(fyCols.Status.Set(), fiscalyear.StatusOpen).
		Set(fyCols.AllowAdjustingEntries.Set(), true).
		Where(fyCols.ID.Eq(), fy.ID).
		Where(fyCols.OrganizationID.Eq(), p.OrganizationID).
		Where(fyCols.BusinessUnitID.Eq(), p.BusinessUnitID).
		Exec(ctx)
	require.NoError(t, err)
	fy.Status = fiscalyear.StatusOpen
	fy.AllowAdjustingEntries = true

	fpCols := fiscalperiodcol.FiscalPeriodColumns
	fp := new(fiscalperiod.FiscalPeriod)
	err = p.DB.NewSelect().
		Model(fp).
		Where(fpCols.FiscalYearID.Eq(), fy.ID).
		Where(fpCols.OrganizationID.Eq(), p.OrganizationID).
		Where(fpCols.BusinessUnitID.Eq(), p.BusinessUnitID).
		Where(fpCols.IsAdjusting.IsFalse()).
		Where(fpCols.StartDate.Lte(), p.At).
		Where(fpCols.EndDate.Gte(), p.At).
		Limit(1).
		Scan(ctx)
	require.NoError(t, err, "fiscal year %s has no operating period covering %d", fy.ID, p.At)

	_, err = p.DB.NewUpdate().
		Model((*fiscalperiod.FiscalPeriod)(nil)).
		Set(fpCols.Status.Set(), fiscalperiod.StatusOpen).
		Set(fpCols.AllowAdjustingEntries.Set(), true).
		Where(fpCols.ID.Eq(), fp.ID).
		Where(fpCols.OrganizationID.Eq(), p.OrganizationID).
		Where(fpCols.BusinessUnitID.Eq(), p.BusinessUnitID).
		Exec(ctx)
	require.NoError(t, err)
	fp.Status = fiscalperiod.StatusOpen
	fp.AllowAdjustingEntries = true

	return fy, fp
}

func findYear(t testing.TB, ctx context.Context, p OpenPeriodParams) *fiscalyear.FiscalYear {
	t.Helper()

	cols := fiscalyearcol.FiscalYearColumns
	fy := new(fiscalyear.FiscalYear)
	err := p.DB.NewSelect().
		Model(fy).
		Where(cols.OrganizationID.Eq(), p.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), p.BusinessUnitID).
		Where(cols.StartDate.Lte(), p.At).
		Where(cols.EndDate.Gte(), p.At).
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	require.NoError(t, err)

	return fy
}

func createCalendarYear(
	t testing.TB,
	ctx context.Context,
	p OpenPeriodParams,
) *fiscalyear.FiscalYear {
	t.Helper()

	year := time.Unix(p.At, 0).UTC().Year()
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0).Add(-time.Second)
	fy := &fiscalyear.FiscalYear{
		OrganizationID:        p.OrganizationID,
		BusinessUnitID:        p.BusinessUnitID,
		Status:                fiscalyear.StatusOpen,
		Year:                  year,
		Name:                  fmt.Sprintf("FY %d", year),
		StartDate:             start.Unix(),
		EndDate:               end.Unix(),
		IsCurrent:             year == time.Now().UTC().Year(),
		IsCalendarYear:        true,
		AllowAdjustingEntries: true,
	}
	_, err := p.DB.NewInsert().Model(fy).Exec(ctx)
	require.NoError(t, err)

	periods := fy.GenerateMonthlyPeriods()
	_, err = p.DB.NewInsert().Model(&periods).Exec(ctx)
	require.NoError(t, err)

	return fy
}
