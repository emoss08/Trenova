package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/volumeseed"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	seedVolumeShipments int
	seedVolumeMonths    int
	seedVolumeSeed      uint64
	seedVolumeOrg       string

	errSeedVolumeEnvironment = errors.New(
		"seed-volume only runs against development or test databases",
	)
)

var dbSeedVolumeCmd = &cobra.Command{
	Use:   "seed-volume",
	Short: "Fill one organization with a large, varied data set for load and UI testing",
	Long: `Generate a realistic, internally consistent data set at volume in one
organization: customers, locations, commodities, workers, tractors, trailers,
fleet codes and carriers, then shipments across every status with orders, moves,
stops, driver and carrier assignments, commodities, accessorial charges, comments,
check calls and holds, then what follows them: billing queue items in every
review state, posted invoices, journals, ledger entries and customer payments;
invoice adjustments, approvals, reconciliation exceptions and adjustment batches;
carrier rate confirmations, cost events, invoice matching and settlements; driver
pay events, settlements, deductions, advances, escrow, expenses and disputes; fuel
cards, purchases, jurisdiction mileage and IFTA returns; bank receipts and their
reconciliation, collections, late charges, statements, factoring, and manual
journals and reversals.

Shipment, order and invoice numbers come from the organization's own sequence
formats, exactly as the application issues them. History is spread across the
past months so lists, filters, aging and dashboards have real distributions.

This is not part of 'db seed'. It requires the development seeds, refuses
production and staging, and adds to what is already there, so running it again
grows the data set.

Examples:
  trenova db seed-volume                          # 10,000 shipments, 12 months
  trenova db seed-volume --shipments 50000        # A heavier data set
  trenova db seed-volume --months 24 --seed 42    # Two years, reproducible
  trenova db seed-volume --org org_01H...         # A specific organization`,
	RunE: runSeedVolume,
}

func runSeedVolume(_ *cobra.Command, _ []string) error {
	if env := getEnvironment(); env != common.EnvDevelopment && env != common.EnvTest {
		return errSeedVolumeEnvironment
	}

	orgID := pulid.Nil
	if seedVolumeOrg != "" {
		parsed, err := pulid.Parse(seedVolumeOrg)
		if err != nil {
			return fmt.Errorf("invalid --org %q: %w", seedVolumeOrg, err)
		}
		orgID = parsed
	}

	manager, err := createManager()
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	color.Cyan(
		"→ Seeding %d shipments across %d months of history",
		seedVolumeShipments,
		seedVolumeMonths,
	)

	report, err := volumeseed.Run(context.Background(), manager.GetDB(), volumeseed.Options{
		Shipments:      seedVolumeShipments,
		HistoryMonths:  seedVolumeMonths,
		Seed:           seedVolumeSeed,
		OrganizationID: orgID,
		Logf: func(format string, args ...any) {
			color.White(format, args...)
		},
	})
	if report != nil {
		printSeedVolumeReport(report)
	}
	if err != nil {
		return fmt.Errorf("seed-volume failed: %w", err)
	}

	color.Green("✓ Volume data seeded in %s (seed %d)", report.Duration.Round(time.Millisecond), report.Seed)
	return nil
}

func printSeedVolumeReport(report *volumeseed.Report) {
	for _, count := range report.Counts {
		color.White("  %-32s %10d", count.Entity, count.Rows)
	}
}

func init() {
	DbCmd.AddCommand(dbSeedVolumeCmd)
	dbSeedVolumeCmd.Flags().
		IntVar(&seedVolumeShipments, "shipments", volumeseed.DefaultShipments, "Number of shipments to generate")
	dbSeedVolumeCmd.Flags().
		IntVar(&seedVolumeMonths, "months", volumeseed.DefaultHistoryMonths, "Months of history to spread data across")
	dbSeedVolumeCmd.Flags().
		Uint64Var(&seedVolumeSeed, "seed", 0, "Random seed for a reproducible data set (0 picks one)")
	dbSeedVolumeCmd.Flags().
		StringVar(&seedVolumeOrg, "org", "", "Organization to seed (defaults to the development organization)")
	dbSeedVolumeCmd.Flags().
		StringVar(&envOverride, "env", "", "Override environment (development, test)")
}
