//go:build integration

package invoiceadjustmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalpostingrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalrepairservice"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerledgerrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/journalrepairrepository"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/app/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestJournalRepairRestoresAdjustmentCreditMemoJournals(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	h := newIntegrationHarness(
		t,
		ctx,
		db,
		&fakeWorkflowStarter{enabled: false},
		decimal.NewFromInt(100),
	)
	ctx = h.ctx
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
	assert.Contains(t, planned.Skipped[0].Reason, "no payable was booked")
	assert.Nil(t, h.creditMemoJournal(t, credit.CreditMemoInvoiceID))
	assert.Empty(t, h.customerLedgerLines(t, credit.CreditMemoInvoiceID))

	payableID, cashID := h.lookupGLAccount(t, "2010"), h.lookupGLAccount(t, "1010")
	h.execSQL(t,
		`UPDATE driver_settlements SET posted_payable_account_id = $1
		WHERE organization_id = $2 AND business_unit_id = $3 AND id = $4`,
		payableID, h.orgID, h.buID, settlementID,
	)
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

	h.execSQL(t,
		`UPDATE driver_settlements SET paid_journal_batch_id = NULL
		WHERE organization_id = $1 AND business_unit_id = $2 AND id = $3`,
		h.orgID, h.buID, settlementID,
	)

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

	adjustment, err := h.service.Submit(h.ctx, &invoiceadjustmentports.InvoiceAdjustmentRequest{
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

	h.execSQL(t,
		`DELETE FROM customer_ledger_entries
		WHERE organization_id = $1 AND business_unit_id = $2 AND source_object_id = $3`,
		h.orgID, h.buID, creditMemoID.String(),
	)
}

func (h *integrationHarness) execSQL(t *testing.T, sql string, args ...any) {
	t.Helper()

	_, err := h.db.Pool().Exec(h.ctx, sql, args...)
	require.NoError(t, err)
}

func (h *integrationHarness) forgetCreditMemoJournal(t *testing.T, creditMemoID pulid.ID) {
	t.Helper()

	h.forgetCustomerLedgerLines(t, creditMemoID)
	h.execSQL(t,
		`DELETE FROM journal_sources
		WHERE organization_id = $1 AND business_unit_id = $2 AND source_object_type = 'Invoice'
			AND source_object_id = $3 AND source_event_type = 'CreditMemoPosted'`,
		h.orgID, h.buID, creditMemoID.String(),
	)
}

func (h *integrationHarness) seededPaidSettlement(t *testing.T) (pulid.ID, int64) {
	t.Helper()

	var id pulid.ID
	var netPayMinor int64
	require.NoError(t, h.db.Pool().QueryRow(h.ctx,
		`SELECT id, net_pay_minor FROM driver_settlements
		WHERE organization_id = $1 AND business_unit_id = $2 AND status = 'Paid'
			AND paid_journal_batch_id IS NULL
		ORDER BY id LIMIT 1`,
		h.orgID, h.buID,
	).Scan(&id, &netPayMinor))
	require.Positive(t, netPayMinor)

	h.execSQL(
		t,
		`UPDATE driver_settlements SET posted_payable_account_id = NULL, posted_journal_batch_id = NULL
		WHERE organization_id = $1 AND business_unit_id = $2 AND id = $3`,
		h.orgID,
		h.buID,
		id,
	)

	return id, netPayMinor
}

func (h *integrationHarness) settlementPaymentJournal(
	t *testing.T,
	settlementID pulid.ID,
) *creditMemoJournalRow {
	t.Helper()

	var batchID *pulid.ID
	require.NoError(t, h.db.Pool().QueryRow(h.ctx,
		`SELECT paid_journal_batch_id FROM driver_settlements
		WHERE organization_id = $1 AND business_unit_id = $2 AND id = $3`,
		h.orgID, h.buID, settlementID,
	).Scan(&batchID))
	if batchID == nil {
		return nil
	}

	row := &creditMemoJournalRow{SourceEventType: "DriverSettlementPaid"}
	var journalEntryID pulid.ID
	require.NoError(t, h.db.Pool().QueryRow(h.ctx,
		`SELECT status, journal_entry_id FROM journal_sources
		WHERE organization_id = $1 AND business_unit_id = $2 AND journal_batch_id = $3
			AND source_object_type = 'DriverSettlement' AND source_object_id = $4
			AND source_event_type = 'DriverSettlementPaid'`,
		h.orgID, h.buID, *batchID, settlementID.String(),
	).Scan(&row.Status, &journalEntryID))

	row.Lines = h.journalEntryLines(t, journalEntryID)
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

func TestWriteOffJournalIsStampedWhenItIsWrittenNotOnItsAccountingDate(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	h := newIntegrationHarness(
		t,
		ctx,
		db,
		&fakeWorkflowStarter{enabled: false},
		decimal.NewFromInt(100),
	)
	ctx = h.ctx
	h.ensureOpenFiscalPeriod(t)
	h.ensureAccountingDefaults(t)
	h.setControls(t, func(control *tenant.InvoiceAdjustmentControl) {
		control.WriteOffApprovalPolicy = tenant.WriteOffApprovalPolicyRequireApprovalAboveThreshold
		control.WriteOffApprovalThreshold = decimal.NewFromInt(10_000)
	})

	entity := h.createPostedInvoice(t, []invoice.InvoiceLine{
		makeInvoiceLine(1, invoice.InvoiceLineTypeFreight, "Open balance", 1, 80),
	}, invoice.SettlementStatusUnpaid, decimal.Zero)
	accountingDate := timeutils.NowUnix() - 3_600
	h.execSQL(t,
		`UPDATE invoices SET invoice_date = $1
		WHERE organization_id = $2 AND business_unit_id = $3 AND id = $4`,
		accountingDate, h.orgID, h.buID, entity.ID,
	)

	writtenAfter := timeutils.NowUnix()
	adjustment, err := h.service.Submit(ctx, &invoiceadjustmentports.InvoiceAdjustmentRequest{
		InvoiceID:      entity.ID,
		Kind:           invoiceadjustment.KindWriteOff,
		IdempotencyKey: "writeoff-stamp-" + entity.ID.String(),
		Reason:         "Bad debt",
		TenantInfo:     h.tenantInfo(),
	}, h.actor())
	require.NoError(t, err)
	require.Equal(t, invoiceadjustment.StatusExecuted, adjustment.Status)
	require.Equal(t, accountingDate, adjustment.AccountingDate)

	var journalAccountingDate int64
	var postedAt *int64
	require.NoError(t, h.db.Pool().QueryRow(ctx,
		`SELECT accounting_date, posted_at FROM journal_entries
		WHERE organization_id = $1 AND business_unit_id = $2 AND reference_id = $3 LIMIT 1`,
		h.orgID, h.buID, adjustment.ID.String(),
	).Scan(&journalAccountingDate, &postedAt))
	assert.Equal(t, accountingDate, journalAccountingDate)
	require.NotNil(t, postedAt)
	assert.GreaterOrEqual(t, *postedAt, writtenAfter)
}
