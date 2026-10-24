package db

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear/fiscalyeardb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var dbBackfillFiscalPeriodsCmd = &cobra.Command{
	Use:   "backfill-fiscal-periods",
	Short: "Generate missing fiscal periods for fiscal years that have none",
	Long: `Find fiscal years that have zero fiscal periods and generate their monthly
periods. This repairs fiscal years that were created before period generation
was wired in, or where period generation previously failed.

Examples:
  trenova db backfill-fiscal-periods            # Backfill missing periods
  trenova db backfill-fiscal-periods --dry-run  # Preview without writing`,
	RunE: runBackfillFiscalPeriods,
}

const backfillFiscalPeriodsReason = "backfill the fiscal periods of every organization's fiscal years"

func runBackfillFiscalPeriods(_ *cobra.Command, _ []string) error {
	ctx := dbscope.WithSystem(context.Background(), backfillFiscalPeriodsReason)

	db, err := openDB(ctx)
	if err != nil {
		return fmt.Errorf("open the database: %w", err)
	}
	defer db.Close()

	years, err := fiscalYearsMissingPeriods(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to find fiscal years missing periods: %w", err)
	}

	if len(years) == 0 {
		color.Green("✓ All fiscal years already have periods. Nothing to backfill.")
		return nil
	}

	color.Cyan("→ Found %d fiscal year(s) missing periods", len(years))

	var totalPeriods int
	for _, fy := range years {
		periods := fy.GenerateMonthlyPeriods()
		totalPeriods += len(periods)

		if verbose || dryRun {
			color.White(
				"  • %s (FY %d) → %d period(s)",
				fy.Name,
				fy.Year,
				len(periods),
			)
		}

		if dryRun {
			continue
		}

		if err = insertPeriods(ctx, db, periods); err != nil {
			return fmt.Errorf("failed to backfill periods for fiscal year %s: %w", fy.ID, err)
		}
	}

	if dryRun {
		color.Yellow(
			"→ Dry run: would create %d period(s) across %d fiscal year(s)",
			totalPeriods,
			len(years),
		)
		return nil
	}

	color.Green(
		"✓ Backfilled %d period(s) across %d fiscal year(s)",
		totalPeriods,
		len(years),
	)
	return nil
}

func fiscalYearsMissingPeriods(
	ctx context.Context,
	db *keel.DB,
) ([]*fiscalyear.FiscalYear, error) {
	fy := fiscalyeardb.FiscalYears

	return fy.AllTenants().
		Where(fy.Periods().None()).
		Order(fy.OrganizationID.Asc(), fy.BusinessUnitID.Asc(), fy.Year.Asc()).
		All(ctx, db)
}

func insertPeriods(
	ctx context.Context,
	db *keel.DB,
	periods []*fiscalperiod.FiscalPeriod,
) error {
	if len(periods) == 0 {
		return nil
	}

	return fiscalperioddb.FiscalPeriods.CreateMany(ctx, db, periods)
}

func init() {
	DbCmd.AddCommand(dbBackfillFiscalPeriodsCmd)
}
