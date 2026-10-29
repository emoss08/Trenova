package factoringservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringwatchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type ledgerHarness struct {
	svc         factoringledgerports.LedgerService
	submissions *fakeSubmissions
	entries     *fakeReserveEntries
	agreements  *fakeAgreements
	invoices    *fakeInvoices
	projection  *fakeProjection
	controls    *fakeControls
	periods     *fakePeriods
	journals    *fakeJournals
	requester   *fakeRequester
	events      *fakeEvents
	actor       *permissionports.RequestActor
	tenantInfo  pagination.TenantInfo
	accounts    factoring.PostingAccounts
	sub         *factoringsubmission.Submission
	first       *factoringsubmission.SubmissionItem
	second      *factoringsubmission.SubmissionItem
}

func newLedgerHarness(t *testing.T, treatment factoring.AccountingTreatment) *ledgerHarness {
	t.Helper()
	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	h := &ledgerHarness{
		submissions: newFakeSubmissions(),
		entries:     &fakeReserveEntries{},
		invoices:    &fakeInvoices{byID: map[pulid.ID]*invoice.Invoice{}},
		projection:  &fakeProjection{},
		journals:    &fakeJournals{},
		requester:   &fakeRequester{},
		events:      &fakeEvents{},
		tenantInfo:  pagination.TenantInfo{OrgID: orgID, BuID: buID},
		accounts: factoring.PostingAccounts{
			AR:            pulid.MustNew("gla_"),
			Cash:          pulid.MustNew("gla_"),
			DueFromFactor: pulid.MustNew("gla_"),
			FeeExpense:    pulid.MustNew("gla_"),
			Liability:     pulid.MustNew("gla_"),
		},
		actor: &permissionports.RequestActor{
			PrincipalType: permissionports.PrincipalTypeUser,
			PrincipalID:   pulid.MustNew("usr_"),
			UserID:        pulid.MustNew("usr_"),
		},
	}
	h.controls = &fakeControls{control: &accountingcontrol.AccountingControl{
		OrganizationID:            orgID,
		BusinessUnitID:            buID,
		DefaultARAccountID:        h.accounts.AR,
		DefaultCashAccountID:      h.accounts.Cash,
		JournalPostingMode:        accountingcontrol.JournalPostingModeAutomatic,
		ClosedPeriodPostingPolicy: accountingcontrol.ClosedPeriodPostingPolicyPostToNextOpen,
	}}
	h.periods = &fakePeriods{periods: []*fiscalperiod.FiscalPeriod{{
		ID:           pulid.MustNew("fp_"),
		FiscalYearID: pulid.MustNew("fy_"),
		PeriodNumber: 1,
		Name:         "Period 1",
		Status:       fiscalperiod.StatusOpen,
		StartDate:    testNow - 30*timeutils.SecondsPerDay,
		EndDate:      testNow + 30*timeutils.SecondsPerDay,
	}}}
	h.agreements = &fakeAgreements{agreement: &factoring.Agreement{
		ID:                     pulid.MustNew("facagr_"),
		OrganizationID:         orgID,
		BusinessUnitID:         buID,
		FactoringCompanyID:     pulid.MustNew("fac_"),
		Status:                 factoring.AgreementStatusActive,
		Recourse:               factoring.RecourseFull,
		AccountingTreatment:    treatment,
		ChargebackWindowDays:   90,
		DueFromFactorAccountID: h.accounts.DueFromFactor,
		FeeExpenseAccountID:    h.accounts.FeeExpense,
		LiabilityAccountID:     h.accounts.Liability,
	}}

	sub := &factoringsubmission.Submission{
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		Number:             "FS2610000001",
		FactoringCompanyID: h.agreements.agreement.FactoringCompanyID,
		AgreementID:        h.agreements.agreement.ID,
		Status:             factoringsubmission.SubmissionStatusSent,
		CurrencyCode:       "USD",
		IdempotencyKey:     "key",
	}
	for _, number := range []string{"INV-1", "INV-2"} {
		inv := &invoice.Invoice{
			ID:               pulid.MustNew("inv_"),
			OrganizationID:   orgID,
			BusinessUnitID:   buID,
			CustomerID:       pulid.MustNew("cus_"),
			Number:           number,
			TotalAmountMinor: 100_000,
			SettlementStatus: invoice.SettlementStatusUnpaid,
		}
		h.invoices.byID[inv.ID] = inv
		sub.Items = append(sub.Items, &factoringsubmission.SubmissionItem{
			OrganizationID:       orgID,
			BusinessUnitID:       buID,
			InvoiceID:            inv.ID,
			InvoiceNumber:        number,
			CustomerID:           inv.CustomerID,
			FaceMinor:            100_000,
			ExpectedAdvanceMinor: 90_000,
			ExpectedFeeMinor:     3_000,
			ExpectedReserveMinor: 7_000,
			Status:               factoringsubmission.SubmissionItemStatusPending,
		})
	}
	sub.Totalize()
	created, err := h.submissions.Create(t.Context(), sub)
	require.NoError(t, err)
	h.sub = created
	h.first, h.second = created.Items[0], created.Items[1]

	h.svc = NewLedgerService(LedgerParams{
		DB:          fakeTx{},
		Submissions: h.submissions,
		Entries:     h.entries,
		Agreements:  h.agreements,
		Invoices:    h.invoices,
		Ledger:      h.projection,
		Controls:    h.controls,
		Periods:     h.periods,
		Sequences:   &fakeJournalNumbers{},
		Journals:    h.journals,
		Requester:   h.requester,
		Events:      h.events,
		Logger:      zap.NewNop(),
	})
	h.svc.(*LedgerService).now = func() int64 { return testNow }
	return h
}

func (h *ledgerHarness) version(t *testing.T) int64 {
	t.Helper()
	current, err := h.submissions.GetByID(
		t.Context(),
		factoringsubmissionports.GetSubmissionRequest{ID: h.sub.ID},
	)
	require.NoError(t, err)
	return current.Version
}

func (h *ledgerHarness) fundFirst(t *testing.T) {
	t.Helper()
	_, err := h.svc.RecordFunding(t.Context(), &factoringledgerports.RecordFundingRequest{
		TenantInfo:   h.tenantInfo,
		SubmissionID: h.sub.ID,
		Version:      h.version(t),
		FundedOn:     testNow,
		Items: []factoringledgerports.FundingItemInput{
			{ItemID: h.first.ID, Outcome: factoringledger.FundingOutcomeFunded},
			{
				ItemID:  h.second.ID,
				Outcome: factoringledger.FundingOutcomeRejected,
				Reason:  "Over limit",
			},
		},
	}, h.actor)
	require.NoError(t, err)
}

func (h *ledgerHarness) itemRequest(
	t *testing.T,
	cash *int64,
	reason string,
) *factoringledgerports.ReserveMovementRequest {
	t.Helper()
	return &factoringledgerports.ReserveMovementRequest{
		TenantInfo:   h.tenantInfo,
		SubmissionID: h.sub.ID,
		ItemID:       h.first.ID,
		Version:      h.version(t),
		OccurredOn:   testNow + timeutils.SecondsPerDay,
		CashMinor:    cash,
		Reason:       reason,
	}
}

func (h *ledgerHarness) invoice(id pulid.ID) *invoice.Invoice {
	return h.invoices.byID[id]
}

func TestRecordFundingSale(t *testing.T) {
	t.Parallel()
	h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
	advance, fee := int64(88_000), int64(2_500)

	req := &factoringledgerports.RecordFundingRequest{
		TenantInfo:   h.tenantInfo,
		SubmissionID: h.sub.ID,
		Version:      h.version(t),
		FundedOn:     testNow,
		Items: []factoringledgerports.FundingItemInput{
			{
				ItemID:       h.first.ID,
				Outcome:      factoringledger.FundingOutcomeFunded,
				AdvanceMinor: &advance,
				FeeMinor:     &fee,
			},
			{
				ItemID:  h.second.ID,
				Outcome: factoringledger.FundingOutcomeRejected,
				Reason:  "Debtor over limit",
			},
		},
	}
	plan, err := h.svc.PlanFunding(t.Context(), req, h.actor)
	require.NoError(t, err)
	require.Len(t, plan.Movements, 1)
	assert.Equal(t, 1, plan.FundedCount)
	assert.Equal(t, 1, plan.RejectCount)
	assert.Equal(t, int64(9_500), plan.ReserveMinor)
	assert.Equal(t, factoringsubmission.SubmissionStatusFunded, plan.After.Status)
	assert.Equal(t, []factoringledgerports.PlannedLine{
		{Role: factoring.AccountRoleCash, AccountID: h.accounts.Cash, Debit: 88_000},
		{Role: factoring.AccountRoleFeeExpense, AccountID: h.accounts.FeeExpense, Debit: 2_500},
		{
			Role:      factoring.AccountRoleDueFromFactor,
			AccountID: h.accounts.DueFromFactor,
			Debit:     9_500,
		},
		{Role: factoring.AccountRoleAR, AccountID: h.accounts.AR, Credit: 100_000},
	}, plan.Movements[0].Lines)
	assert.Equal(t, "Period 1", plan.Movements[0].FiscalPeriodName)
	assert.False(t, plan.Movements[0].RequiresApproval)
	assert.Empty(t, h.journals.postings, "planning books nothing")

	result, err := h.svc.RecordFunding(t.Context(), req, h.actor)
	require.NoError(t, err)
	require.Len(t, result.Entries, 1)
	assert.Equal(t, factoringsubmission.SubmissionStatusFunded, result.Submission.Status)
	require.Len(t, h.journals.postings, 1)
	posting := h.journals.postings[0]
	assert.Equal(
		t,
		accountingcontrol.JournalSourceEventFactoringFunded.String(),
		posting.SourceEventType,
	)
	assert.Equal(t, "factoring-funded:"+h.first.ID.String(), posting.SourceIdempotencyKey)
	assert.Equal(t, int64(100_000), posting.TotalDebit)
	assert.True(t, posting.IsPosted)

	entry := result.Entries[0]
	assert.Equal(t, posting.EntryID, entry.JournalEntryID)
	assert.Equal(t, int64(9_500), entry.ReserveMinor)
	assert.Equal(t, int64(-100_000), entry.ARMinor)

	settled := h.invoice(h.first.InvoiceID)
	assert.Equal(t, int64(100_000), settled.AppliedAmountMinor)
	assert.Equal(t, invoice.SettlementStatusPaid, settled.SettlementStatus)
	assert.Equal(t, int64(-100_000), h.projection.balance(h.first.InvoiceID))
	assert.Zero(t, h.invoice(h.second.InvoiceID).AppliedAmountMinor)

	stored, err := h.submissions.GetByID(
		t.Context(),
		factoringsubmissionports.GetSubmissionRequest{ID: h.sub.ID},
	)
	require.NoError(t, err)
	assert.Equal(t, factoringsubmission.SubmissionItemStatusFunded, stored.Item(h.first.ID).Status)
	assert.Equal(
		t,
		factoringsubmission.SubmissionItemStatusRejected,
		stored.Item(h.second.ID).Status,
	)
	assert.Equal(t, "Debtor over limit", stored.Item(h.second.ID).RejectionReason)

	assert.Equal(t, []string{string(agenttrigger.EventFactoringSubmissionFunded)}, h.events.kinds())
	require.Len(t, h.requester.targets, 1)
	assert.ElementsMatch(
		t,
		[]pulid.ID{h.first.InvoiceID, h.second.InvoiceID},
		h.requester.targets[0].InvoiceIDs,
	)

	_, err = h.svc.RecordFunding(t.Context(), req, h.actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err), "a replay with the old version is refused")
}

func TestRecordFundingSecuredBorrowingLeavesTheInvoiceOpen(t *testing.T) {
	t.Parallel()
	h := newLedgerHarness(t, factoring.AccountingTreatmentSecuredBorrowing)

	h.fundFirst(t)

	require.Len(t, h.journals.postings, 1)
	assert.Equal(t, int64(90_000), h.journals.balance(h.accounts.Cash))
	assert.Equal(t, int64(3_000), h.journals.balance(h.accounts.FeeExpense))
	assert.Equal(t, int64(-93_000), h.journals.balance(h.accounts.Liability))
	assert.Zero(t, h.journals.balance(h.accounts.AR))
	assert.Zero(t, h.invoice(h.first.InvoiceID).AppliedAmountMinor)
	assert.Empty(t, h.projection.entries)
}

func TestRecordFundingRefusals(t *testing.T) {
	t.Parallel()

	t.Run("missing accounts", func(t *testing.T) {
		t.Parallel()
		h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
		h.agreements.agreement.DueFromFactorAccountID = pulid.Nil
		h.agreements.agreement.FeeExpenseAccountID = pulid.Nil

		_, err := h.svc.PlanFunding(t.Context(), &factoringledgerports.RecordFundingRequest{
			TenantInfo: h.tenantInfo, SubmissionID: h.sub.ID, Version: h.version(t), FundedOn: testNow,
			Items: []factoringledgerports.FundingItemInput{
				{ItemID: h.first.ID, Outcome: factoringledger.FundingOutcomeFunded},
			},
		}, h.actor)
		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr)
		assert.Len(t, multiErr.Errors, 2)
	})

	t.Run("no person behind the actor", func(t *testing.T) {
		t.Parallel()
		h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
		apiKey := &permissionports.RequestActor{
			PrincipalType: permissionports.PrincipalTypeAPIKey,
			PrincipalID:   pulid.MustNew("ak_"),
		}
		_, err := h.svc.PlanFunding(t.Context(), &factoringledgerports.RecordFundingRequest{
			TenantInfo: h.tenantInfo, SubmissionID: h.sub.ID, Version: h.version(t), FundedOn: testNow,
			Items: []factoringledgerports.FundingItemInput{
				{ItemID: h.first.ID, Outcome: factoringledger.FundingOutcomeFunded},
			},
		}, apiKey)
		require.Error(t, err)
		assert.True(t, errortypes.IsAuthorizationError(err))
	})

	t.Run("future date, duplicate and unknown items", func(t *testing.T) {
		t.Parallel()
		h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
		_, err := h.svc.PlanFunding(t.Context(), &factoringledgerports.RecordFundingRequest{
			TenantInfo: h.tenantInfo, SubmissionID: h.sub.ID, Version: h.version(t),
			FundedOn: testNow + 3*timeutils.SecondsPerDay,
			Items: []factoringledgerports.FundingItemInput{
				{ItemID: h.first.ID, Outcome: factoringledger.FundingOutcomeFunded},
			},
		}, h.actor)
		require.Error(t, err)

		_, err = h.svc.PlanFunding(t.Context(), &factoringledgerports.RecordFundingRequest{
			TenantInfo: h.tenantInfo, SubmissionID: h.sub.ID, Version: h.version(t), FundedOn: testNow,
			Items: []factoringledgerports.FundingItemInput{
				{ItemID: h.first.ID, Outcome: factoringledger.FundingOutcomeFunded},
				{ItemID: h.first.ID, Outcome: factoringledger.FundingOutcomeRejected, Reason: "x"},
			},
		}, h.actor)
		require.Error(t, err)

		_, err = h.svc.PlanFunding(t.Context(), &factoringledgerports.RecordFundingRequest{
			TenantInfo: h.tenantInfo, SubmissionID: h.sub.ID, Version: h.version(t), FundedOn: testNow,
			Items: []factoringledgerports.FundingItemInput{
				{ItemID: pulid.MustNew("facsit_"), Outcome: factoringledger.FundingOutcomeFunded},
			},
		}, h.actor)
		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr)
	})

	t.Run("invoice paid down since it was scheduled", func(t *testing.T) {
		t.Parallel()
		h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
		h.invoice(h.first.InvoiceID).ApplyPaymentMinor(10_000)
		_, err := h.svc.PlanFunding(t.Context(), &factoringledgerports.RecordFundingRequest{
			TenantInfo: h.tenantInfo, SubmissionID: h.sub.ID, Version: h.version(t), FundedOn: testNow,
			Items: []factoringledgerports.FundingItemInput{
				{ItemID: h.first.ID, Outcome: factoringledger.FundingOutcomeFunded},
			},
		}, h.actor)
		require.Error(t, err)
		assert.True(t, errortypes.IsBusinessError(err))
	})

	t.Run("schedule not waiting on funding", func(t *testing.T) {
		t.Parallel()
		h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
		h.fundFirst(t)
		_, err := h.svc.PlanFunding(t.Context(), &factoringledgerports.RecordFundingRequest{
			TenantInfo: h.tenantInfo, SubmissionID: h.sub.ID, Version: h.version(t), FundedOn: testNow,
			Items: []factoringledgerports.FundingItemInput{
				{ItemID: h.first.ID, Outcome: factoringledger.FundingOutcomeFunded},
			},
		}, h.actor)
		require.Error(t, err)
		assert.True(t, errortypes.IsBusinessError(err))
	})
}

func TestFundingInAClosedPeriodMovesToTheNextOpenOne(t *testing.T) {
	t.Parallel()
	h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
	closed := h.periods.periods[0]
	closed.Status = fiscalperiod.StatusClosed
	closed.EndDate = testNow + timeutils.SecondsPerDay/2
	h.periods.periods = append(h.periods.periods, &fiscalperiod.FiscalPeriod{
		ID:           pulid.MustNew("fp_"),
		FiscalYearID: closed.FiscalYearID,
		PeriodNumber: 2,
		Name:         "Period 2",
		Status:       fiscalperiod.StatusOpen,
		StartDate:    closed.EndDate + 1,
		EndDate:      closed.EndDate + 30*timeutils.SecondsPerDay,
	})

	plan, err := h.svc.PlanFunding(t.Context(), &factoringledgerports.RecordFundingRequest{
		TenantInfo: h.tenantInfo, SubmissionID: h.sub.ID, Version: h.version(t), FundedOn: testNow,
		Items: []factoringledgerports.FundingItemInput{
			{ItemID: h.first.ID, Outcome: factoringledger.FundingOutcomeFunded},
		},
	}, h.actor)
	require.NoError(t, err)
	movement := plan.Movements[0]
	assert.Equal(t, "Period 2", movement.FiscalPeriodName)
	assert.True(t, movement.Redated())
	assert.Equal(t, closed.EndDate+1, movement.AccountingDate)
}

func TestReleaseReserveSale(t *testing.T) {
	t.Parallel()
	h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
	h.fundFirst(t)

	short := int64(6_000)
	plan, err := h.svc.PlanRelease(t.Context(), h.itemRequest(t, &short, ""), h.actor)
	require.NoError(t, err)
	assert.Equal(t, []factoringledgerports.PlannedLine{
		{Role: factoring.AccountRoleCash, AccountID: h.accounts.Cash, Debit: 6_000},
		{Role: factoring.AccountRoleFeeExpense, AccountID: h.accounts.FeeExpense, Debit: 1_000},
		{
			Role:      factoring.AccountRoleDueFromFactor,
			AccountID: h.accounts.DueFromFactor,
			Credit:    7_000,
		},
	}, plan.Movement.Lines)

	result, err := h.svc.ReleaseReserve(
		t.Context(),
		h.itemRequest(t, &short, "Remittance 42"),
		h.actor,
	)
	require.NoError(t, err)
	require.NotNil(t, result.Entry)
	assert.Equal(
		t,
		factoringsubmission.SubmissionItemStatusSettled,
		result.Submission.Item(h.first.ID).Status,
	)
	assert.Zero(t, h.journals.balance(h.accounts.DueFromFactor), "the reserve is cleared")
	assert.Equal(t, int64(96_000), h.journals.balance(h.accounts.Cash))
	assert.Equal(t, int64(4_000), h.journals.balance(h.accounts.FeeExpense))
	assert.Equal(t, "Remittance 42", result.Entry.Memo)

	_, err = h.svc.ReleaseReserve(t.Context(), h.itemRequest(t, nil, ""), h.actor)
	require.Error(t, err, "a settled item has nothing left to release")

	_, err = h.svc.RecordChargeback(t.Context(), h.itemRequest(t, nil, "unpaid"), h.actor)
	require.Error(t, err, "a settled item cannot be charged back")
}

func TestReleaseReserveSecuredBorrowingSettlesTheInvoice(t *testing.T) {
	t.Parallel()
	h := newLedgerHarness(t, factoring.AccountingTreatmentSecuredBorrowing)
	h.fundFirst(t)

	_, err := h.svc.ReleaseReserve(t.Context(), h.itemRequest(t, nil, ""), h.actor)
	require.NoError(t, err)

	assert.Zero(t, h.journals.balance(h.accounts.Liability))
	assert.Equal(t, int64(-100_000), h.journals.balance(h.accounts.AR))
	assert.Equal(t, int64(97_000), h.journals.balance(h.accounts.Cash))
	assert.Equal(t, int64(3_000), h.journals.balance(h.accounts.FeeExpense))
	assert.Equal(t, invoice.SettlementStatusPaid, h.invoice(h.first.InvoiceID).SettlementStatus)
	assert.Equal(t, int64(-100_000), h.projection.balance(h.first.InvoiceID))
}

func TestChargebackAndReversalSale(t *testing.T) {
	t.Parallel()
	h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
	h.fundFirst(t)

	_, err := h.svc.RecordChargeback(t.Context(), h.itemRequest(t, nil, ""), h.actor)
	require.Error(t, err, "a chargeback needs a reason")

	result, err := h.svc.RecordChargeback(
		t.Context(),
		h.itemRequest(t, nil, "Customer never paid"),
		h.actor,
	)
	require.NoError(t, err)
	item := result.Submission.Item(h.first.ID)
	assert.Equal(t, factoringsubmission.SubmissionItemStatusChargedBack, item.Status)
	assert.Zero(t, h.journals.balance(h.accounts.AR), "AR is back on the books")
	assert.Zero(t, h.journals.balance(h.accounts.DueFromFactor))
	assert.Equal(t, int64(-3_000), h.journals.balance(h.accounts.Cash))
	reopened := h.invoice(h.first.InvoiceID)
	assert.Zero(t, reopened.AppliedAmountMinor)
	assert.Equal(t, invoice.SettlementStatusUnpaid, reopened.SettlementStatus)
	assert.Zero(t, h.projection.balance(h.first.InvoiceID))
	chargebackKey := h.journals.postings[len(h.journals.postings)-1].SourceIdempotencyKey
	assert.Equal(t, "factoring-charged-back:"+h.first.ID.String()+":1", chargebackKey)

	cash := int64(1)
	_, err = h.svc.ReverseChargeback(
		t.Context(),
		h.itemRequest(t, &cash, "Factor withdrew it"),
		h.actor,
	)
	require.Error(t, err, "a reversal takes no amount")
	_, err = h.svc.ReverseChargeback(t.Context(), h.itemRequest(t, nil, ""), h.actor)
	require.Error(t, err, "a reversal needs a reason")

	reversal, err := h.svc.PlanChargebackReversal(
		t.Context(),
		h.itemRequest(t, nil, "Factor withdrew it"),
		h.actor,
	)
	require.NoError(t, err)
	require.NotNil(t, reversal.Reverses)
	assert.Equal(t, factoringledger.ReserveEntryChargeback, reversal.Reverses.Kind)

	reversed, err := h.svc.ReverseChargeback(
		t.Context(),
		h.itemRequest(t, nil, "Factor withdrew it"),
		h.actor,
	)
	require.NoError(t, err)
	assert.Equal(
		t,
		factoringsubmission.SubmissionItemStatusFunded,
		reversed.Submission.Item(h.first.ID).Status,
	)
	assert.Equal(t, reversal.Reverses.ID, reversed.Entry.ReversesEntryID)
	assert.Equal(t, int64(-100_000), h.journals.balance(h.accounts.AR))
	assert.Equal(t, int64(7_000), h.journals.balance(h.accounts.DueFromFactor))
	assert.Equal(t, invoice.SettlementStatusPaid, h.invoice(h.first.InvoiceID).SettlementStatus)
	assert.Equal(t, int64(-100_000), h.projection.balance(h.first.InvoiceID))

	_, err = h.svc.ReverseChargeback(t.Context(), h.itemRequest(t, nil, "again"), h.actor)
	require.Error(t, err, "nothing is left to reverse")

	_, err = h.svc.RecordChargeback(t.Context(), h.itemRequest(t, nil, "Unpaid after all"), h.actor)
	require.NoError(t, err)
	secondKey := h.journals.postings[len(h.journals.postings)-1].SourceIdempotencyKey
	assert.Equal(t, "factoring-charged-back:"+h.first.ID.String()+":2", secondKey)
}

func TestChargebackSecuredBorrowingLeavesAROpen(t *testing.T) {
	t.Parallel()
	h := newLedgerHarness(t, factoring.AccountingTreatmentSecuredBorrowing)
	h.fundFirst(t)

	_, err := h.svc.RecordChargeback(
		t.Context(),
		h.itemRequest(t, nil, "Customer never paid"),
		h.actor,
	)
	require.NoError(t, err)

	assert.Zero(t, h.journals.balance(h.accounts.Liability))
	assert.Zero(t, h.journals.balance(h.accounts.AR))
	assert.Equal(t, int64(-3_000), h.journals.balance(h.accounts.Cash))
	assert.Zero(t, h.invoice(h.first.InvoiceID).AppliedAmountMinor)
}

func TestChargebackRefusedWithoutRecourse(t *testing.T) {
	t.Parallel()
	h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
	h.fundFirst(t)
	h.agreements.agreement.Recourse = factoring.RecourseNone

	_, err := h.svc.PlanChargeback(t.Context(), h.itemRequest(t, nil, "unpaid"), h.actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestReleaseUsesTheTreatmentTheItemWasFundedUnder(t *testing.T) {
	t.Parallel()
	h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
	h.fundFirst(t)
	h.agreements.agreement.AccountingTreatment = factoring.AccountingTreatmentSecuredBorrowing

	plan, err := h.svc.PlanRelease(t.Context(), h.itemRequest(t, nil, ""), h.actor)
	require.NoError(t, err)
	assert.Equal(t, factoring.AccountingTreatmentSale, plan.Movement.Movement.Treatment)
}

func TestReleaseOfAnEmptyReserveBooksNothing(t *testing.T) {
	t.Parallel()
	h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
	advance, fee := int64(97_000), int64(3_000)
	_, err := h.svc.RecordFunding(t.Context(), &factoringledgerports.RecordFundingRequest{
		TenantInfo: h.tenantInfo, SubmissionID: h.sub.ID, Version: h.version(t), FundedOn: testNow,
		Items: []factoringledgerports.FundingItemInput{
			{
				ItemID:       h.first.ID,
				Outcome:      factoringledger.FundingOutcomeFunded,
				AdvanceMinor: &advance,
				FeeMinor:     &fee,
			},
		},
	}, h.actor)
	require.NoError(t, err)

	result, err := h.svc.ReleaseReserve(t.Context(), h.itemRequest(t, nil, ""), h.actor)
	require.NoError(t, err)
	assert.Nil(t, result.Entry)
	assert.Len(t, h.journals.postings, 1)
	assert.Equal(
		t,
		factoringsubmission.SubmissionItemStatusSettled,
		result.Submission.Item(h.first.ID).Status,
	)
}

func TestSweepWarnsOnceAboutChargebacksComingDue(t *testing.T) {
	t.Parallel()
	h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
	h.fundFirst(t)
	attention := &fakeAttention{due: []*factoringwatchtowerports.ChargebackDue{
		{
			ItemID:        h.first.ID,
			SubmissionID:  h.sub.ID,
			InvoiceID:     h.first.InvoiceID,
			InvoiceNumber: h.first.InvoiceNumber,
			DeadlineAt:    testNow + 3*timeutils.SecondsPerDay,
		},
		{
			ItemID:        h.second.ID,
			SubmissionID:  h.sub.ID,
			InvoiceID:     h.second.InvoiceID,
			InvoiceNumber: h.second.InvoiceNumber,
			DeadlineAt:    testNow + 3*timeutils.SecondsPerDay,
		},
	}}
	events := &fakeEvents{}
	svc := NewReadinessService(ReadinessParams{
		Submissions: h.submissions,
		Attention:   attention,
		Events:      events,
		Logger:      zap.NewNop(),
	}).(*ReadinessService)
	svc.now = func() int64 { return testNow }

	warned, err := svc.warnChargebacksDue(t.Context(), h.tenantInfo)
	require.NoError(t, err)
	assert.Equal(t, 1, warned, "the rejected item is not funded and is skipped")
	require.NotNil(t, attention.req)
	assert.True(t, attention.req.UnwarnedOnly)
	assert.Equal(t, testNow+7*timeutils.SecondsPerDay, attention.req.Before)
	assert.Equal(t, []string{string(agenttrigger.EventFactoringChargebackDue)}, events.kinds())

	warned, err = svc.warnChargebacksDue(t.Context(), h.tenantInfo)
	require.NoError(t, err)
	assert.Zero(t, warned, "a warned item is not announced again")
}

func TestLedgerRowsCarryTheirOwnIDs(t *testing.T) {
	t.Parallel()
	h := newLedgerHarness(t, factoring.AccountingTreatmentSale)
	_, err := h.svc.RecordFunding(t.Context(), &factoringledgerports.RecordFundingRequest{
		TenantInfo: h.tenantInfo, SubmissionID: h.sub.ID, Version: h.version(t), FundedOn: testNow,
		Items: []factoringledgerports.FundingItemInput{
			{ItemID: h.first.ID, Outcome: factoringledger.FundingOutcomeFunded},
			{ItemID: h.second.ID, Outcome: factoringledger.FundingOutcomeFunded},
		},
	}, h.actor)
	require.NoError(t, err)
	require.Len(t, h.projection.entries, 2)
	assert.True(t, h.projection.entries[0].ID.IsNotNil())
	assert.NotEqual(t, h.projection.entries[0].ID, h.projection.entries[1].ID)
}
