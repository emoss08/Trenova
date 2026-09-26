package db

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/services/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/core/services/journalrepairservice"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/customerledgerrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalpostingrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalrepairrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/userrepository"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var repairJournalsOrg string

var dbRepairJournalsCmd = &cobra.Command{
	Use:   "repair-journals",
	Short: "Write the journals earlier releases left out for credit memos and driver payments",
	Long: `Find records that should have booked to the general ledger but did not, and
write what is missing, once per record:

  - Adjustment credit memos without a customer ledger line get their
    CreditMemoPosted journal (Dr revenue, Cr accounts receivable) and the
    ledger line. Write-off memos, whose journal is written by the write-off
    itself, get only the ledger line. A memo whose journal already exists gets
    only the ledger line.
  - Paid driver settlements without a payment journal get one (Dr the posted
    payable account, Cr the default cash account) on their paid date. A
    settlement whose payment journal already exists is linked to it.

Each journal is dated on its own record's date. When that period is closed, the
organization's closed-period policy decides: the journal moves to the next open
period, or the record is reported as skipped. With journal posting set to Manual
the journals wait on Journals to post for review; with Automatic they post.

Records that cannot be repaired (no accounting accounts configured, a period
that refuses postings, revenue not recognized on invoice post) are listed with
the reason and left unchanged; run again after correcting them.

Examples:
  trenova db repair-journals --dry-run            # Preview without writing
  trenova db repair-journals                      # Repair every organization
  trenova db repair-journals --org org_01H...     # Repair one organization`,
	RunE: runRepairJournals,
}

func runRepairJournals(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	orgID := pulid.Nil
	if repairJournalsOrg != "" {
		parsed, err := pulid.Parse(repairJournalsOrg)
		if err != nil {
			return fmt.Errorf("invalid --org %q: %w", repairJournalsOrg, err)
		}
		orgID = parsed
	}

	manager, err := createManager()
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	report, err := repairJournals(ctx, postgres.WrapDB(manager.GetDB()), orgID, dryRun)
	if err != nil {
		return err
	}

	printRepairReport(report)
	return nil
}

func repairJournals(
	ctx context.Context,
	conn *postgres.Connection,
	orgID pulid.ID,
	preview bool,
) (*journalrepairservice.Report, error) {
	logger := zap.NewNop()

	systemUser, err := userrepository.New(userrepository.Params{DB: conn, Logger: logger}).
		GetSystemUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load the system user: %w", err)
	}

	service := journalrepairservice.New(&journalrepairservice.Deps{
		DB: conn,
		Repo: journalrepairrepository.New(
			journalrepairrepository.Params{DB: conn, Logger: logger},
		),
		Controls: accountingcontrolrepository.New(
			accountingcontrolrepository.Params{DB: conn, Logger: logger},
		),
		Periods: fiscalperiodrepository.New(
			fiscalperiodrepository.Params{DB: conn, Logger: logger},
		),
		Journals: journalpostingrepository.New(
			journalpostingrepository.Params{DB: conn, Logger: logger},
		),
		Ledger: customerledgerrepository.New(
			customerledgerrepository.Params{DB: conn, Logger: logger},
		),
		Numbers: seqgen.NewGenerator(seqgen.GeneratorParams{
			Store: seqgen.NewSequenceStore(
				seqgen.SequenceStoreParams{DB: conn, Logger: logger},
			),
			Provider: seqgen.NewFormatProvider(
				seqgen.FormatProviderParams{DB: conn, Logger: logger},
			),
			Logger: logger,
		}),
		Policy: accountingcontrolpolicyservice.New(
			accountingcontrolpolicyservice.Params{Logger: logger},
		),
		Now: timeutils.NowUnix,
	})

	report, err := service.Repair(ctx, &journalrepairservice.Request{
		OrganizationID: orgID,
		ActorID:        systemUser.ID,
		DryRun:         preview,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to repair journals: %w", err)
	}
	return report, nil
}

func printRepairReport(report *journalrepairservice.Report) {
	verb := "Repaired"
	if dryRun {
		verb = "Dry run: would repair"
	}

	total := report.CreditMemosJournaled + report.CreditMemosLedgerOnly +
		report.PaymentsJournaled + report.PaymentsLinked
	if total == 0 && len(report.Skipped) == 0 {
		color.Green(
			"✓ Nothing to repair. Every credit memo and paid driver settlement is journaled.",
		)
		return
	}

	color.Cyan("→ %s:", verb)
	color.White(
		"  • %d adjustment credit memo(s) journaled with a customer ledger line",
		report.CreditMemosJournaled,
	)
	color.White(
		"  • %d credit memo(s) given only their customer ledger line",
		report.CreditMemosLedgerOnly,
	)
	color.White("  • %d driver settlement payment journal(s) written", report.PaymentsJournaled)
	color.White(
		"  • %d driver settlement(s) linked to an existing payment journal",
		report.PaymentsLinked,
	)

	if len(report.Skipped) == 0 {
		return
	}
	color.Yellow("→ Skipped %d record(s):", len(report.Skipped))
	for _, skip := range report.Skipped {
		color.Yellow(
			"  • %s %s (%s) in %s: %s",
			skip.Kind,
			skip.Number,
			skip.ID,
			skip.OrganizationID,
			skip.Reason,
		)
	}
}

func init() {
	dbRepairJournalsCmd.Flags().
		StringVar(&repairJournalsOrg, "org", "", "Repair only this organization")
	DbCmd.AddCommand(dbRepairJournalsCmd)
}
