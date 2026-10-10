package development

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type FiscalYearSeed struct {
	seedhelpers.BaseSeed
}

func NewFiscalYearSeed() *FiscalYearSeed {
	seed := &FiscalYearSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"FiscalYear",
		"1.0.0",
		"Opens the current calendar fiscal year with its monthly periods, so invoices and journals post",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedTestOrganizations)

	return seed
}

func (s *FiscalYearSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
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

			return s.ensure(ctx, tx, fy)
		},
	)
}

func (s *FiscalYearSeed) ensure(ctx context.Context, tx bun.Tx, fy *fiscalyear.FiscalYear) error {
	cols := buncolgen.FiscalYearColumns
	exists, err := tx.NewSelect().
		Model((*fiscalyear.FiscalYear)(nil)).
		Where(cols.OrganizationID.Eq(), fy.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), fy.BusinessUnitID).
		Where(cols.Year.Eq(), fy.Year).
		Exists(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if exists {
		return nil
	}

	if _, err = tx.NewInsert().Model(fy).Exec(ctx); err != nil {
		return fmt.Errorf("insert fiscal year %d: %w", fy.Year, err)
	}

	periods := fy.GenerateMonthlyPeriods()
	if _, err = tx.NewInsert().Model(&periods).Exec(ctx); err != nil {
		return fmt.Errorf("insert fiscal periods for %d: %w", fy.Year, err)
	}

	return nil
}
