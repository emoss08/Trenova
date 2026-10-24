package fixtures

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear/fiscalyeardb"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/shared/timeutils"
)

type FiscalYearSeed struct {
}

var FiscalYear = Seeds.Add(seed.Seed{
	Name:        "fiscal_year",
	Kind:        seed.Fixture,
	Description: "Opens the current calendar fiscal year with its monthly periods, so invoices and journals post",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&FiscalYearSeed{}).Run,
})

func (s *FiscalYearSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	year := time.Unix(timeutils.NowUnix(), 0).UTC().Year()
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0).Add(-time.Second)
	fy := &fiscalyear.FiscalYear{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Status:         fiscalyear.StatusOpen,
		Year:           year,
		Name:           "FY " + strconv.Itoa(year),
		Description:    "Calendar fiscal year opened by the development seed",
		StartDate:      start.Unix(),
		EndDate:        end.Unix(),
		IsCurrent:      true,
		IsCalendarYear: true,
	}

	return s.ensure(ctx, db, fy)
}

func (s *FiscalYearSeed) ensure(ctx context.Context, db *keel.DB, fy *fiscalyear.FiscalYear) error {

	if err := fiscalyeardb.FiscalYears.Create(ctx, db, fy); err != nil {
		return fmt.Errorf("insert fiscal year %d: %w", fy.Year, err)
	}

	periods := fy.GenerateMonthlyPeriods()
	if err := fiscalperioddb.FiscalPeriods.CreateMany(ctx, db, periods); err != nil {
		return fmt.Errorf("insert fiscal periods for %d: %w", fy.Year, err)
	}

	return nil
}
