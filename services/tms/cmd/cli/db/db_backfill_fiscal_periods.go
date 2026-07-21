package db

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
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

func runBackfillFiscalPeriods(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	manager, err := createManager()
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	db := manager.GetDB()

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
	db *bun.DB,
) ([]*fiscalyear.FiscalYear, error) {
	years := make([]*fiscalyear.FiscalYear, 0)
	err := db.NewSelect().
		Model(&years).
		Where(
			"NOT EXISTS (?)",
			db.NewSelect().
				Model((*fiscalperiod.FiscalPeriod)(nil)).
				Where("fp.fiscal_year_id = fy.id").
				Where("fp.organization_id = fy.organization_id").
				Where("fp.business_unit_id = fy.business_unit_id"),
		).
		Order("fy.organization_id", "fy.business_unit_id", "fy.year").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return years, nil
}

func insertPeriods(
	ctx context.Context,
	db *bun.DB,
	periods []*fiscalperiod.FiscalPeriod,
) error {
	if len(periods) == 0 {
		return nil
	}

	return db.RunInTx(ctx, nil, func(txCtx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(&periods).Exec(txCtx)
		return err
	})
}

func init() {
	DbCmd.AddCommand(dbBackfillFiscalPeriodsCmd)
}
