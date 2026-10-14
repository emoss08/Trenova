package db

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editenderrepairrepository"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderrepairservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var repairEDITenderReferencesOrg string

var dbRepairEDITenderReferencesCmd = &cobra.Command{
	Use:   "repair-edi-tender-references",
	Short: "Re-read the shipment reference of inbound load tenders read one B2 element early",
	Long: `Earlier releases read an inbound 204's B2 segment one element early: a partner
sending X12's layout (SCAC in B202, shipment identification number in B204,
method of payment in B206) had its SCAC kept as the tender's shipment reference
and its shipment identification number kept as the method of payment.

This re-reads the B2 of every inbound 204 whose X12 is still kept, the way the
inbound parser reads it now, and where the reference it finds differs from the
one kept, corrects it in three places:

  - the inbound message's decoded tender;
  - the transfers that tender opened, while they still hold the old reference;
  - the shipments those transfers built, while their external reference is
    still the old one. A reference someone has since changed is left alone.

A 204 written in Trenova's earlier layout reads the same as it did and is left
unchanged. Tenders whose X12 was purged under the retention policy cannot be
re-read; they are counted and left as they are.

Examples:
  trenova db repair-edi-tender-references --dry-run         # Preview without writing
  trenova db repair-edi-tender-references                   # Repair every organization
  trenova db repair-edi-tender-references --org org_01H...  # Repair one organization
  trenova db repair-edi-tender-references --dry-run --verbose  # List each tender as it is read`,
	RunE: runRepairEDITenderReferences,
}

func runRepairEDITenderReferences(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	orgID := pulid.Nil
	if repairEDITenderReferencesOrg != "" {
		parsed, err := pulid.Parse(repairEDITenderReferencesOrg)
		if err != nil {
			return fmt.Errorf("invalid --org %q: %w", repairEDITenderReferencesOrg, err)
		}
		orgID = parsed
	}

	manager, err := createManager()
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	report, err := repairEDITenderReferences(
		ctx,
		postgres.WrapDB(manager.GetDB()),
		&editenderrepairservice.Request{
			OrganizationID: orgID,
			DryRun:         dryRun,
			OnRepair:       printRepairedTender(verbose),
			OnSkip:         printSkippedTender(verbose),
		},
	)
	if err != nil {
		return err
	}

	printEDITenderReferenceReport(report, dryRun)
	return nil
}

func repairEDITenderReferences(
	ctx context.Context,
	conn *postgres.Connection,
	req *editenderrepairservice.Request,
) (*editenderrepairservice.Report, error) {
	service := editenderrepairservice.New(editenderrepairservice.Deps{
		Repo: editenderrepairrepository.New(editenderrepairrepository.Params{
			DB:     conn,
			Logger: zap.NewNop(),
		}),
	})

	report, err := service.Repair(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to repair EDI tender references: %w", err)
	}
	return report, nil
}

func printRepairedTender(verbose bool) func(editenderrepairservice.Repair) {
	if !verbose {
		return nil
	}
	return func(repair editenderrepairservice.Repair) {
		color.White(
			"  • %s in %s: shipment %q → %q, payment %q → %q",
			repair.MessageID,
			repair.OrganizationID,
			repair.From.ShipmentID,
			repair.To.ShipmentID,
			repair.From.PaymentMethod,
			repair.To.PaymentMethod,
		)
	}
}

func printSkippedTender(verbose bool) func(editenderrepairservice.Skip) {
	if !verbose {
		return nil
	}
	return func(skip editenderrepairservice.Skip) {
		color.Yellow("  • skipped %s in %s: %s", skip.MessageID, skip.OrganizationID, skip.Reason)
	}
}

func printEDITenderReferenceReport(report *editenderrepairservice.Report, preview bool) {
	if report.Repaired == 0 {
		color.Green(
			"✓ Nothing to repair. %d inbound tender(s) already read their B2 right.",
			report.Unchanged,
		)
	} else {
		verb := "Repaired"
		if preview {
			verb = "Dry run: would repair"
		}
		color.Cyan("→ %s %d inbound tender(s)", verb, report.Repaired)
		if !preview {
			color.White("  • %d transfer(s) corrected", report.TransfersRepaired)
			color.White(
				"  • %d shipment(s) given their external reference",
				report.ShipmentsRepaired,
			)
		}
		color.White("  • %d inbound tender(s) already read right", report.Unchanged)
	}

	if report.Unreadable > 0 {
		color.Yellow(
			"→ %d inbound tender(s) could not be re-read: their X12 was purged under the retention policy",
			report.Unreadable,
		)
	}
	if report.Skipped > 0 {
		color.Yellow(
			"→ %d inbound tender(s) skipped: the kept transaction has no B2 segment",
			report.Skipped,
		)
	}
}

func init() {
	dbRepairEDITenderReferencesCmd.Flags().
		StringVar(&repairEDITenderReferencesOrg, "org", "", "Repair only this organization")
	DbCmd.AddCommand(dbRepairEDITenderReferencesCmd)
}
