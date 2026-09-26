//go:build integration

package invoiceadjustmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/core/services/journalrepairservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/customerledgerrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalpostingrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/journalrepairrepository"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestJournalRepairRestoresAdjustmentCreditMemoJournals(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	seedRegistry := seeder.NewRegistry()
	seeds.Register(seedRegistry)
	engine := seeder.NewEngine(db, seedRegistry, &config.Config{
		System: config.SystemConfig{SystemUserPassword: "test-system-password"},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	h := newIntegrationHarness(
		t,
		ctx,
		db,
		&fakeWorkflowStarter{enabled: false},
		decimal.NewFromInt(100),
	)
	h.ensureOpenFiscalPeriod(t)
	h.ensureAccountingDefaults(t)
	h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
		control.StandardAdjustmentApprovalPolicy = tenant.ApprovalPolicyAmountThreshold
		control.StandardAdjustmentApprovalThreshold = decimal.NewFromInt(10_000)
		control.WriteOffApprovalPolicy = tenant.WriteOffApprovalPolicyRequireApprovalAboveThreshold
		control.WriteOffApprovalThreshold = decimal.NewFromInt(10_000)
	})

	credit := h.executeAdjustment(t, invoiceadjustment.KindCreditOnly)
	writeOff := h.executeAdjustment(t, invoiceadjustment.KindWriteOff)
	halfDone := h.executeAdjustment(t, invoiceadjustment.KindCreditOnly)
	halfDoneJournal := h.creditMemoJournal(t, halfDone.CreditMemoInvoiceID)
	require.NotNil(t, halfDoneJournal)

	h.forgetCreditMemoJournal(t, credit.CreditMemoInvoiceID)
	h.forgetCreditMemoJournal(t, writeOff.CreditMemoInvoiceID)
	h.forgetCustomerLedgerLines(t, halfDone.CreditMemoInvoiceID)
	require.Nil(t, h.creditMemoJournal(t, credit.CreditMemoInvoiceID))
	require.Empty(t, h.customerLedgerLines(t, credit.CreditMemoInvoiceID))
	require.Empty(t, h.customerLedgerLines(t, writeOff.CreditMemoInvoiceID))

	repair := h.repairService()
	request := &journalrepairservice.Request{OrganizationID: h.orgID, ActorID: h.userID}

	settlementID, netPay := h.seededPaidSettlement(t)

	dryRun := *request
	dryRun.DryRun = true
	planned, err := repair.Repair(ctx, &dryRun)
	require.NoError(t, err)
	assert.Equal(t, 1, planned.CreditMemosJournaled)
	assert.Equal(t, 2, planned.CreditMemosLedgerOnly)
	assert.Zero(t, planned.PaymentsJournaled)
	require.Len(t, planned.Skipped, 1)
	assert.Equal(t, journalrepairservice.KindDriverSettlement, planned.Skipped[0].Kind)
	assert.Equal(t, settlementID, planned.Skipped[0].ID)
	assert.Contains(t, planned.Skipped[0].Reason, "no posted payable account")
	assert.Nil(t, h.creditMemoJournal(t, credit.CreditMemoInvoiceID))
	assert.Empty(t, h.customerLedgerLines(t, credit.CreditMemoInvoiceID))

	payableID, cashID := h.lookupGLAccount(t, "2010"), h.lookupGLAccount(t, "1010")
	_, err = db.NewUpdate().
		Table("driver_settlements").
		Set("posted_payable_account_id = ?", payableID).
		Where("id = ?", settlementID).
		Exec(ctx)
	require.NoError(t, err)
	control, err := h.accountingRepo.GetByOrgID(ctx, h.orgID)
	require.NoError(t, err)
	control.DefaultCashAccountID = cashID
	_, err = h.accountingRepo.Update(ctx, control)
	require.NoError(t, err)

	report, err := repair.Repair(ctx, request)
	require.NoError(t, err)
	assert.Equal(t, 1, report.CreditMemosJournaled)
	assert.Equal(t, 2, report.CreditMemosLedgerOnly)
	assert.Equal(t, 1, report.PaymentsJournaled)
	assert.Empty(t, report.Skipped)

	journal := h.creditMemoJournal(t, credit.CreditMemoInvoiceID)
	require.NotNil(t, journal)
	assert.Equal(t, "CreditMemoPosted", journal.SourceEventType)
	assert.Equal(t, "Posted", journal.Status)
	require.Len(t, journal.Lines, 2)
	assert.Equal(t, h.lookupGLAccount(t, "4000"), journal.Lines[0].GLAccountID)
	assert.Equal(t, int64(12500), journal.Lines[0].DebitAmount)
	assert.Equal(t, h.lookupGLAccount(t, "1110"), journal.Lines[1].GLAccountID)
	assert.Equal(t, int64(12500), journal.Lines[1].CreditAmount)

	ledger := h.customerLedgerLines(t, credit.CreditMemoInvoiceID)
	require.Len(t, ledger, 1)
	assert.Equal(t, int64(-12500), ledger[0].AmountMinor)
	assert.Equal(t, "CreditMemoPosted", ledger[0].SourceEventType)
	assert.Equal(t, credit.OriginalInvoiceID, ledger[0].RelatedInvoiceID)

	assert.Nil(t, h.creditMemoJournal(t, writeOff.CreditMemoInvoiceID))
	writeOffLedger := h.customerLedgerLines(t, writeOff.CreditMemoInvoiceID)
	require.Len(t, writeOffLedger, 1)
	assert.Equal(t, int64(-12500), writeOffLedger[0].AmountMinor)

	assert.Equal(t, halfDoneJournal, h.creditMemoJournal(t, halfDone.CreditMemoInvoiceID))
	halfDoneLedger := h.customerLedgerLines(t, halfDone.CreditMemoInvoiceID)
	require.Len(t, halfDoneLedger, 1)
	assert.Equal(t, int64(-12500), halfDoneLedger[0].AmountMinor)

	payment := h.settlementPaymentJournal(t, settlementID)
	require.NotNil(t, payment)
	assert.Equal(t, "Posted", payment.Status)
	require.Len(t, payment.Lines, 2)
	assert.Equal(t, payableID, payment.Lines[0].GLAccountID)
	assert.Equal(t, netPay, payment.Lines[0].DebitAmount)
	assert.Equal(t, cashID, payment.Lines[1].GLAccountID)
	assert.Equal(t, netPay, payment.Lines[1].CreditAmount)

	_, err = db.NewUpdate().
		Table("driver_settlements").
		Set("paid_journal_batch_id = NULL").
		Where("id = ?", settlementID).
		Exec(ctx)
	require.NoError(t, err)

	again, err := repair.Repair(ctx, request)
	require.NoError(t, err)
	assert.Zero(t, again.CreditMemosJournaled)
	assert.Zero(t, again.CreditMemosLedgerOnly)
	assert.Zero(t, again.PaymentsJournaled)
	assert.Equal(t, 1, again.PaymentsLinked)
	assert.Empty(t, again.Skipped)
	assert.Equal(t, payment, h.settlementPaymentJournal(t, settlementID))

	settled, err := repair.Repair(ctx, request)
	require.NoError(t, err)
	assert.Zero(t, settled.PaymentsLinked)
	assert.Empty(t, settled.Skipped)
	assert.Len(t, h.customerLedgerLines(t, credit.CreditMemoInvoiceID), 1)

	otherOrg, err := repair.Repair(ctx, &journalrepairservice.Request{
		OrganizationID: pulid.MustNew("org_"),
		ActorID:        h.userID,
		DryRun:         true,
	})
	require.NoError(t, err)
	assert.Zero(t, otherOrg.CreditMemosJournaled)
	assert.Zero(t, otherOrg.PaymentsJournaled)
	assert.Empty(t, otherOrg.Skipped)
}

func (h *integrationHarness) executeAdjustment(
	t *testing.T,
	kind invoiceadjustment.Kind,
) *invoiceadjustment.InvoiceAdjustment {
	t.Helper()

	entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
		makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Base freight", 1, 100),
		makeInvoiceLine(2, invoice.InvoiceLineTypeAccessorial, "Fuel", 1, 25),
	}, invoice.SettlementStatusUnpaid, decimal.Zero)

	adjustment, err := h.service.Submit(h.ctx, &servicesports.InvoiceAdjustmentRequest{
		InvoiceID:      entity.ID,
		Kind:           kind,
		IdempotencyKey: string(kind) + "-" + entity.ID.String(),
		Reason:         "Repair scenario",
		TenantInfo:     h.tenantInfo(),
	}, h.actor())
	require.NoError(t, err)
	require.Equal(t, invoiceadjustment.StatusExecuted, adjustment.Status)
	require.True(t, adjustment.CreditMemoInvoiceID.IsNotNil())
	return adjustment
}

func (h *integrationHarness) forgetCustomerLedgerLines(t *testing.T, creditMemoID pulid.ID) {
	t.Helper()

	_, err := h.db.NewDelete().
		Table("customer_ledger_entries").
		Where("organization_id = ?", h.orgID).
		Where("source_object_id = ?", creditMemoID.String()).
		Exec(h.ctx)
	require.NoError(t, err)
}

func (h *integrationHarness) forgetCreditMemoJournal(t *testing.T, creditMemoID pulid.ID) {
	t.Helper()

	h.forgetCustomerLedgerLines(t, creditMemoID)
	_, err := h.db.NewDelete().
		Table("journal_sources").
		Where("organization_id = ?", h.orgID).
		Where("source_object_type = ?", "Invoice").
		Where("source_object_id = ?", creditMemoID.String()).
		Where("source_event_type = ?", "CreditMemoPosted").
		Exec(h.ctx)
	require.NoError(t, err)
}

func (h *integrationHarness) seededPaidSettlement(t *testing.T) (pulid.ID, int64) {
	t.Helper()

	var row struct {
		ID          pulid.ID `bun:"id"`
		NetPayMinor int64    `bun:"net_pay_minor"`
	}
	require.NoError(t, h.db.NewSelect().
		Table("driver_settlements").
		Column("id", "net_pay_minor").
		Where("organization_id = ?", h.orgID).
		Where("status = ?", "Paid").
		Where("paid_journal_batch_id IS NULL").
		Scan(h.ctx, &row))
	require.Positive(t, row.NetPayMinor)
	return row.ID, row.NetPayMinor
}

func (h *integrationHarness) settlementPaymentJournal(
	t *testing.T,
	settlementID pulid.ID,
) *creditMemoJournalRow {
	t.Helper()

	var settlement struct {
		BatchID *pulid.ID `bun:"paid_journal_batch_id"`
	}
	require.NoError(t, h.db.NewSelect().
		Table("driver_settlements").
		Column("paid_journal_batch_id").
		Where("id = ?", settlementID).
		Scan(h.ctx, &settlement))
	if settlement.BatchID == nil {
		return nil
	}

	var source struct {
		Status         string   `bun:"status"`
		JournalEntryID pulid.ID `bun:"journal_entry_id"`
	}
	require.NoError(t, h.db.NewSelect().
		Table("journal_sources").
		Column("status", "journal_entry_id").
		Where("journal_batch_id = ?", *settlement.BatchID).
		Where("source_object_type = ?", "DriverSettlement").
		Where("source_object_id = ?", settlementID.String()).
		Where("source_event_type = ?", "DriverSettlementPaid").
		Scan(h.ctx, &source))

	row := &creditMemoJournalRow{SourceEventType: "DriverSettlementPaid", Status: source.Status}
	require.NoError(t, h.db.NewSelect().
		Table("journal_entry_lines").
		Column("gl_account_id", "debit_amount", "credit_amount").
		Where("journal_entry_id = ?", source.JournalEntryID).
		OrderExpr("line_number ASC").
		Scan(h.ctx, &row.Lines))
	return row
}

func (h *integrationHarness) repairService() *journalrepairservice.Service {
	logger := zap.NewNop()
	return journalrepairservice.New(&journalrepairservice.Deps{
		DB: h.conn,
		Repo: journalrepairrepository.New(
			journalrepairrepository.Params{DB: h.conn, Logger: logger},
		),
		Controls: h.accountingRepo,
		Periods: fiscalperiodrepository.New(
			fiscalperiodrepository.Params{DB: h.conn, Logger: logger},
		),
		Journals: journalpostingrepository.New(
			journalpostingrepository.Params{DB: h.conn, Logger: logger},
		),
		Ledger: customerledgerrepository.New(
			customerledgerrepository.Params{DB: h.conn, Logger: logger},
		),
		Numbers: testutil.UniqueJournalSequenceGenerator{
			TestSequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "REPAIR-SEQ"},
		},
		Policy: accountingcontrolpolicyservice.New(
			accountingcontrolpolicyservice.Params{Logger: logger},
		),
		Now: func() int64 { return 1_800_000_000 },
	})
}
