//go:build integration

package factoringrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalpostingrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerledgerrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/paymentreminderrepository"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringwatchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/paymentreminderports"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/fiscaltest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type ledgerEnv struct {
	*factoringEnv
	svc         factoringledgerports.LedgerService
	submissions factoringsubmissionports.SubmissionRepository
	entries     factoringledgerports.ReserveEntryRepository
	attention   factoringwatchtowerports.WatchtowerReader
	ledger      customerledgerports.CustomerLedgerProjectionRepository
	accounts    factoring.PostingAccounts
	userID      pulid.ID
}

func setupLedgerEnv(t *testing.T) *ledgerEnv {
	t.Helper()
	e := setupFactoringEnv(t)
	conn := postgres.NewTestConnection(e.db)
	logger := zap.NewNop()
	p := Params{DB: conn, Logger: logger}

	var userID pulid.ID
	require.NoError(t, e.db.Pool().QueryRow(e.ctx,
		"SELECT id FROM users WHERE current_organization_id = $1 AND business_unit_id = $2 ORDER BY id LIMIT 1",
		e.tenantInfo.OrgID, e.tenantInfo.BuID,
	).
		Scan(&userID))

	account := func(code string) pulid.ID {
		var id pulid.ID
		require.NoError(t, e.db.Pool().QueryRow(e.ctx,
			"SELECT id FROM gl_accounts WHERE organization_id = $1 AND business_unit_id = $2 AND account_code = $3 LIMIT 1",
			e.tenantInfo.OrgID, e.tenantInfo.BuID, code,
		).
			Scan(&id))
		return id
	}
	accounts := factoring.PostingAccounts{
		AR:            account("1110"),
		Cash:          account("1010"),
		DueFromFactor: account("1130"),
		FeeExpense:    account("6820"),
		Liability:     account("2420"),
	}

	controls := accountingcontrolrepository.New(
		accountingcontrolrepository.Params{DB: conn, Logger: logger},
	)
	control, err := controls.GetByOrgID(e.ctx, e.tenantInfo.OrgID)
	require.NoError(t, err)
	control.DefaultARAccountID = accounts.AR
	control.DefaultCashAccountID = accounts.Cash
	control.JournalPostingMode = accountingcontrol.JournalPostingModeAutomatic
	_, err = controls.Update(e.ctx, control)
	require.NoError(t, err)
	fiscaltest.OpenPeriod(t, e.ctx, fiscaltest.OpenPeriodParams{
		DB:             e.db,
		OrganizationID: e.tenantInfo.OrgID,
		BusinessUnitID: e.tenantInfo.BuID,
		At:             time.Now().Unix(),
	})

	store := seqgen.NewSequenceStore(seqgen.SequenceStoreParams{DB: conn, Logger: logger})
	provider := seqgen.NewFormatProvider(seqgen.FormatProviderParams{DB: conn, Logger: logger})
	ledger := customerledgerrepository.New(
		customerledgerrepository.Params{DB: conn, Logger: logger},
	)
	submissions := NewSubmissionRepository(p)
	entries := NewReserveEntryRepository(p)
	svc := factoringservice.NewLedgerService(factoringservice.LedgerParams{
		DB:          conn,
		Submissions: submissions,
		Entries:     entries,
		Agreements:  e.agreements,
		Invoices:    e.invoices,
		Ledger:      ledger,
		Controls:    controls,
		Periods: fiscalperiodrepository.New(
			fiscalperiodrepository.Params{DB: conn, Logger: logger},
		),
		Sequences: seqgen.NewGenerator(
			seqgen.GeneratorParams{Store: store, Provider: provider, Logger: logger},
		),
		Journals: journalpostingrepository.New(
			journalpostingrepository.Params{DB: conn, Logger: logger},
		),
		Audit:  &mocks.NoopAuditService{},
		Logger: logger,
	})
	return &ledgerEnv{
		factoringEnv: e,
		svc:          svc,
		submissions:  submissions,
		entries:      entries,
		attention:    NewWatchtowerReader(p),
		ledger:       ledger,
		accounts:     accounts,
		userID:       userID,
	}
}

func (e *ledgerEnv) glBalance(t *testing.T, accountID pulid.ID) int64 {
	t.Helper()
	var total int64
	require.NoError(t, e.db.Pool().QueryRow(e.ctx,
		`SELECT COALESCE(SUM(jel.debit_amount - jel.credit_amount), 0)::bigint
		FROM journal_entry_lines AS jel
		WHERE jel.organization_id = $1 AND jel.business_unit_id = $2 AND jel.gl_account_id = $3
			AND jel.journal_entry_id IN (
				SELECT journal_entry_id FROM factoring_reserve_entries
				WHERE organization_id = $1 AND business_unit_id = $2)`,
		e.tenantInfo.OrgID, e.tenantInfo.BuID, accountID,
	).Scan(&total))
	return total
}

func (e *ledgerEnv) unbalancedEntries(t *testing.T) int {
	t.Helper()
	var count int
	require.NoError(t, e.db.Pool().QueryRow(e.ctx,
		`SELECT COUNT(*) FROM (SELECT jel.journal_entry_id
			FROM journal_entry_lines AS jel
			JOIN factoring_reserve_entries AS facres
				ON facres.journal_entry_id = jel.journal_entry_id
				AND facres.organization_id = jel.organization_id
				AND facres.business_unit_id = jel.business_unit_id
			WHERE jel.organization_id = $1 AND jel.business_unit_id = $2
			GROUP BY jel.journal_entry_id
			HAVING SUM(jel.debit_amount) <> SUM(jel.credit_amount)) AS unbalanced`,
		e.tenantInfo.OrgID, e.tenantInfo.BuID,
	).Scan(&count))
	return count
}

func (e *ledgerEnv) invoice(t *testing.T, id pulid.ID) *invoice.Invoice {
	t.Helper()
	inv, err := e.invoices.GetByID(e.ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         id,
		TenantInfo: e.tenantInfo,
	})
	require.NoError(t, err)
	return inv
}

func (e *ledgerEnv) ledgerBalance(t *testing.T, invoiceID pulid.ID) int64 {
	t.Helper()
	var total int64
	require.NoError(t, e.db.Pool().QueryRow(e.ctx,
		`SELECT COALESCE(SUM(cle.amount_minor), 0)::bigint FROM customer_ledger_entries AS cle
		WHERE cle.organization_id = $1 AND cle.business_unit_id = $2
			AND cle.related_invoice_id = $3 AND cle.source_object_type = $4`,
		e.tenantInfo.OrgID, e.tenantInfo.BuID, invoiceID, "FactoringSubmissionItem",
	).Scan(&total))
	return total
}

func (e *ledgerEnv) version(t *testing.T, id pulid.ID) int64 {
	t.Helper()
	sub, err := e.submissions.GetByID(e.ctx, factoringsubmissionports.GetSubmissionRequest{
		ID:         id,
		TenantInfo: e.tenantInfo,
	})
	require.NoError(t, err)
	return sub.Version
}

func TestFactoringLedgerKeepsTheBooksBalanced(t *testing.T) {
	e := setupLedgerEnv(t)
	actor := testutil.NewSessionActor(e.userID, e.tenantInfo.OrgID, e.tenantInfo.BuID)
	now := time.Now().Unix()

	factor := e.createFactor(t, "FLED")
	terms := e.agreement(factor.ID, now-3600, nil)
	terms.DueFromFactorAccountID = e.accounts.DueFromFactor
	terms.FeeExpenseAccountID = e.accounts.FeeExpense
	terms.LiabilityAccountID = e.accounts.Liability
	agreement, err := e.agreements.Create(e.ctx, terms)
	require.NoError(t, err)

	released := e.postInvoice(t, 100000, &factor.ID)
	charged := e.postInvoice(t, 50000, &factor.ID)
	rejected := e.postInvoice(t, 20000, &factor.ID)
	sub, err := e.submissions.Create(e.ctx,
		e.submission(agreement, "FS-L1", "key-l1", released, charged, rejected))
	require.NoError(t, err)
	require.NoError(t, sub.MarkSent(now-3*86400, "", 1))
	sub, err = e.submissions.Update(e.ctx, sub)
	require.NoError(t, err)

	unfunded, err := e.attention.ListUnfundedSubmissions(e.ctx,
		&factoringwatchtowerports.ListUnfundedSubmissionsRequest{TenantInfo: e.tenantInfo, At: now})
	require.NoError(t, err)
	require.Len(t, unfunded, 1)
	assert.Equal(t, sub.ID, unfunded[0].SubmissionID)
	assert.Equal(t, 3, unfunded[0].PendingCount)

	itemFor := func(invoiceID pulid.ID) *factoring.SubmissionItem {
		for _, item := range sub.Items {
			if item.InvoiceID == invoiceID {
				return item
			}
		}
		t.Fatalf("no item for invoice %s", invoiceID)
		return nil
	}
	releasedItem, chargedItem, rejectedItem :=
		itemFor(released.ID), itemFor(charged.ID), itemFor(rejected.ID)

	funding, err := e.svc.RecordFunding(e.ctx, &factoringledgerports.RecordFundingRequest{
		TenantInfo:   e.tenantInfo,
		SubmissionID: sub.ID,
		Version:      sub.Version,
		FundedOn:     now,
		Items: []factoringledgerports.FundingItemInput{
			{ItemID: releasedItem.ID, Outcome: factoring.FundingOutcomeFunded},
			{ItemID: chargedItem.ID, Outcome: factoring.FundingOutcomeFunded},
			{
				ItemID:  rejectedItem.ID,
				Outcome: factoring.FundingOutcomeRejected,
				Reason:  "Over limit",
			},
		},
	}, actor)
	require.NoError(t, err)
	require.Len(t, funding.Entries, 2)
	assert.Equal(t, factoring.SubmissionStatusFunded, funding.Submission.Status)

	balance, err := e.svc.GetBalance(e.ctx, factoringledgerports.GetReserveBalanceRequest{
		TenantInfo:         e.tenantInfo,
		FactoringCompanyID: factor.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, e.glBalance(t, e.accounts.DueFromFactor), balance.ReserveMinor,
		"the reserve ledger equals the due-from-factor account")
	assert.Positive(t, balance.ReserveMinor)
	assert.Equal(t, int64(-150000), e.glBalance(t, e.accounts.AR))
	assert.Equal(t, invoice.SettlementStatusPaid, e.invoice(t, released.ID).SettlementStatus)
	assert.Equal(t, int64(-100000), e.ledgerBalance(t, released.ID))
	assert.Zero(t, e.invoice(t, rejected.ID).AppliedAmountMinor)

	summary, err := e.states.GetSummary(e.ctx, factoringports.GetSummaryRequest{
		TenantInfo:       e.tenantInfo,
		ReadyBefore:      now,
		ChargebackBefore: now + 100*86400,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Funded.Count)
	assert.Equal(t, int64(150000), summary.Funded.AmountMinor)
	assert.Equal(t, 2, summary.ReserveHeld.Count)
	assert.Equal(t, balance.ReserveMinor, summary.ReserveHeld.AmountMinor,
		"the reserve held is what the reserve ledger says the factor holds")
	assert.Equal(t, 2, summary.ChargebacksDue.Count)
	assert.Equal(t, int64(150000)-balance.ReserveMinor, summary.ChargebacksDue.AmountMinor,
		"what would be repaid is each funded invoice's advance and fee")

	notYetDue, err := e.states.GetSummary(e.ctx, factoringports.GetSummaryRequest{
		TenantInfo:       e.tenantInfo,
		ReadyBefore:      now,
		ChargebackBefore: now,
	})
	require.NoError(t, err)
	assert.Zero(t, notYetDue.ChargebacksDue.Count)
	assert.Zero(t, notYetDue.ChargebacksDue.AmountMinor)

	unfunded, err = e.attention.ListUnfundedSubmissions(e.ctx,
		&factoringwatchtowerports.ListUnfundedSubmissionsRequest{TenantInfo: e.tenantInfo, At: now})
	require.NoError(t, err)
	assert.Empty(t, unfunded)

	due, err := e.attention.ListChargebacksDue(
		e.ctx,
		&factoringwatchtowerports.ListChargebacksDueRequest{
			TenantInfo:   e.tenantInfo,
			Before:       now + 100*86400,
			UnwarnedOnly: true,
		},
	)
	require.NoError(t, err)
	require.Len(t, due, 2)
	claimed, err := e.submissions.ClaimChargebackWarning(
		e.ctx,
		factoringsubmissionports.ClaimChargebackWarningRequest{
			TenantInfo: e.tenantInfo, ItemID: due[0].ItemID, At: now,
		},
	)
	require.NoError(t, err)
	assert.True(t, claimed)
	claimed, err = e.submissions.ClaimChargebackWarning(
		e.ctx,
		factoringsubmissionports.ClaimChargebackWarningRequest{
			TenantInfo: e.tenantInfo, ItemID: due[0].ItemID, At: now,
		},
	)
	require.NoError(t, err)
	assert.False(t, claimed, "a warning is claimed once")

	_, _, reserve := funding.Submission.Item(releasedItem.ID).Actuals()
	short := reserve - 500
	_, err = e.svc.ReleaseReserve(e.ctx, &factoringledgerports.ReserveMovementRequest{
		TenantInfo:   e.tenantInfo,
		SubmissionID: sub.ID,
		ItemID:       releasedItem.ID,
		Version:      e.version(t, sub.ID),
		OccurredOn:   now,
		CashMinor:    &short,
		Reason:       "Remittance 77",
	}, actor)
	require.NoError(t, err)

	chargeback := func(reason string) {
		_, err = e.svc.RecordChargeback(e.ctx, &factoringledgerports.ReserveMovementRequest{
			TenantInfo:   e.tenantInfo,
			SubmissionID: sub.ID,
			ItemID:       chargedItem.ID,
			Version:      e.version(t, sub.ID),
			OccurredOn:   now,
			Reason:       reason,
		}, actor)
		require.NoError(t, err)
	}
	chargeback("Customer never paid")
	_, err = e.svc.ReverseChargeback(e.ctx, &factoringledgerports.ReserveMovementRequest{
		TenantInfo:   e.tenantInfo,
		SubmissionID: sub.ID,
		ItemID:       chargedItem.ID,
		Version:      e.version(t, sub.ID),
		OccurredOn:   now,
		Reason:       "Factor withdrew it",
	}, actor)
	require.NoError(t, err)
	chargeback("Unpaid after the dispute")

	assert.Zero(t, e.unbalancedEntries(t), "every factoring entry balances")
	assert.Zero(t, e.glBalance(t, e.accounts.DueFromFactor), "no reserve is left with the factor")
	assert.Equal(t, int64(-100000), e.glBalance(t, e.accounts.AR),
		"only the charged-back invoice is back in AR")
	balance, err = e.svc.GetBalance(e.ctx, factoringledgerports.GetReserveBalanceRequest{
		TenantInfo:         e.tenantInfo,
		FactoringCompanyID: factor.ID,
	})
	require.NoError(t, err)
	assert.Zero(t, balance.ReserveMinor)
	assert.Equal(t, 6, balance.EntryCount)

	summary, err = e.states.GetSummary(e.ctx, factoringports.GetSummaryRequest{
		TenantInfo:  e.tenantInfo,
		ReadyBefore: now,
	})
	require.NoError(t, err)
	assert.Zero(
		t,
		summary.Funded.Count,
		"a settled item and a charged-back one are no longer funded",
	)
	assert.Zero(t, summary.ReserveHeld.Count)
	assert.Zero(t, summary.ReserveHeld.AmountMinor)

	reopened := e.invoice(t, charged.ID)
	assert.Zero(t, reopened.AppliedAmountMinor)
	assert.Zero(t, e.ledgerBalance(t, charged.ID))
	assert.Equal(t, invoice.SettlementStatusPaid, e.invoice(t, released.ID).SettlementStatus)

	live, err := e.submissions.ListLiveItems(e.ctx, &factoringsubmissionports.ListLiveItemsRequest{
		TenantInfo:         e.tenantInfo,
		InvoiceIDs:         []pulid.ID{charged.ID},
		IncludeChargedBack: true,
	})
	require.NoError(t, err)
	assert.Equal(t, factoring.SubmissionItemStatusChargedBack, live[charged.ID].ItemStatus)

	reminders := paymentreminderrepository.New(paymentreminderrepository.Params{
		DB: postgres.NewTestConnection(e.db), Logger: zap.NewNop(),
	})
	overdue, err := reminders.ListOverdueInvoices(
		e.ctx,
		paymentreminderports.ListOverdueInvoicesRequest{
			TenantInfo:  e.tenantInfo,
			AsOf:        now + 60*86400,
			CustomerIDs: []pulid.ID{e.customerID},
		},
	)
	require.NoError(t, err)
	overdueIDs := make([]pulid.ID, 0, len(overdue))
	for _, row := range overdue {
		overdueIDs = append(overdueIDs, row.InvoiceID)
	}
	assert.Contains(
		t,
		overdueIDs,
		charged.ID,
		"a charged-back invoice is the organization's to chase",
	)
	assert.NotContains(t, overdueIDs, rejected.ID, "a factored invoice is the factor's to chase")

	_, err = e.ledger.SumCustomerBalances(e.ctx, customerledgerports.SumCustomerBalancesRequest{
		TenantInfo:                 e.tenantInfo,
		CustomerIDs:                []pulid.ID{e.customerID},
		ExcludeNonRecourseFactored: true,
	})
	require.NoError(t, err)

	_, err = e.db.Pool().Exec(e.ctx,
		"UPDATE factoring_reserve_entries SET memo = 'edited' WHERE organization_id = $1",
		e.tenantInfo.OrgID)
	require.Error(t, err, "reserve entries are append-only")
}
