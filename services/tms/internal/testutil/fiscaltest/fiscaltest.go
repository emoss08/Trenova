package fiscaltest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear/fiscalyeardb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glbalance"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glbalance/glbalancedb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

type OpenPeriodParams struct {
	DB             *keel.DB
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

	ctx = tenantContext(ctx, p.OrganizationID, p.BusinessUnitID)
	fy := findYear(t, ctx, p)
	if fy == nil {
		fy = createCalendarYear(t, ctx, p)
	}

	y := fiscalyeardb.FiscalYears
	_, err := y.Where(y.ID.Eq(fy.ID)).Update(ctx, p.DB,
		y.Status.To(fiscalyear.StatusOpen),
		y.AllowAdjustingEntries.To(true),
	)
	require.NoError(t, err)
	fy.Status = fiscalyear.StatusOpen
	fy.AllowAdjustingEntries = true

	fpt := fiscalperioddb.FiscalPeriods
	fp, err := fpt.Where(
		fpt.FiscalYearID.Eq(fy.ID),
		fpt.IsAdjusting.IsFalse(),
		fpt.StartDate.Lte(p.At),
		fpt.EndDate.Gte(p.At),
	).Limit(1).One(ctx, p.DB)
	require.NoError(t, err, "fiscal year %s has no operating period covering %d", fy.ID, p.At)

	_, err = fpt.Where(fpt.ID.Eq(fp.ID)).Update(ctx, p.DB,
		fpt.Status.To(fiscalperiod.StatusOpen),
		fpt.AllowAdjustingEntries.To(true),
	)
	require.NoError(t, err)
	fp.Status = fiscalperiod.StatusOpen
	fp.AllowAdjustingEntries = true

	return fy, fp
}

func findYear(t testing.TB, ctx context.Context, p OpenPeriodParams) *fiscalyear.FiscalYear {
	t.Helper()

	y := fiscalyeardb.FiscalYears
	fy, err := y.Where(y.StartDate.Lte(p.At), y.EndDate.Gte(p.At)).Limit(1).One(ctx, p.DB)
	if errors.Is(err, keel.ErrNotFound) {
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
	require.NoError(t, fiscalyeardb.FiscalYears.Create(ctx, p.DB, fy))
	require.NoError(t, fiscalperioddb.FiscalPeriods.CreateMany(ctx, p.DB, fy.GenerateMonthlyPeriods()))

	return fy
}

type PeriodBalanceParams struct {
	DB             *keel.DB
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	GLAccountID    pulid.ID
	FiscalPeriodID pulid.ID
}

func PeriodBalance(
	t testing.TB,
	ctx context.Context,
	p PeriodBalanceParams,
) glbalance.GLAccountPeriodBalance {
	t.Helper()

	b := glbalancedb.GLAccountPeriodBalances
	balance, err := b.Where(
		b.GLAccountID.Eq(p.GLAccountID),
		b.FiscalPeriodID.Eq(p.FiscalPeriodID),
	).One(tenantContext(ctx, p.OrganizationID, p.BusinessUnitID), p.DB)
	if errors.Is(err, keel.ErrNotFound) {
		return glbalance.GLAccountPeriodBalance{}
	}
	require.NoError(t, err)

	return *balance
}

func tenantContext(ctx context.Context, orgID, buID pulid.ID) context.Context {
	return dbscope.WithTenant(ctx, dbscope.Tenant{OrganizationID: orgID, BusinessUnitID: buID})
}
