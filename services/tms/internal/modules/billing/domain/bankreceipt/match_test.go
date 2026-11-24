package bankreceipt_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func receiptOf(reference, memo string, amount int64) *bankreceipt.BankReceipt {
	return &bankreceipt.BankReceipt{
		ID:              pulid.MustNew("brcpt_"),
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		ReferenceNumber: reference,
		Memo:            memo,
		AmountMinor:     amount,
		ReceiptDate:     1_700_000_000,
	}
}

func TestScore_AnInvoiceNamedByTheReferenceAtItsOpenAmountIsExact(t *testing.T) {
	t.Parallel()

	receipt := receiptOf("SEED-INV-0006", "", 253_125)
	invoice := bankreceipt.InvoiceEvidence{
		ID: pulid.MustNew("inv_"), CustomerID: pulid.MustNew("cus_"),
		Number: "SEED-INV-0006", OpenMinor: 253_125,
	}

	got := bankreceipt.Score(receipt, nil, []bankreceipt.InvoiceEvidence{invoice})

	require.Len(t, got, 1)
	assert.Equal(t, bankreceipt.Candidate{
		OrganizationID:  receipt.OrganizationID,
		BusinessUnitID:  receipt.BusinessUnitID,
		BankReceiptID:   receipt.ID,
		Rank:            1,
		Kind:            bankreceipt.CandidateInvoice,
		InvoiceID:       invoice.ID,
		CustomerID:      invoice.CustomerID,
		Score:           85,
		Evidence:        []string{"NumberInReference", "AmountExact"},
		Standing:        bankreceipt.StandingExact,
		OpenMinor:       253_125,
		DifferenceMinor: 0,
	}, *got[0])
}

func TestScore_StandsShortOrOverByTheDifferenceFromTheOpenAmount(t *testing.T) {
	t.Parallel()

	invoice := bankreceipt.InvoiceEvidence{
		ID: pulid.MustNew("inv_"), Number: "SEED-INV-0003", OpenMinor: 427_500,
	}
	cases := []struct {
		name       string
		amount     int64
		standing   bankreceipt.Standing
		difference int64
	}{
		{"one cent short", 427_499, bankreceipt.StandingShort, -1},
		{"120.00 short", 415_500, bankreceipt.StandingShort, -12_000},
		{"one cent over", 427_501, bankreceipt.StandingOver, 1},
		{"45.10 over", 432_010, bankreceipt.StandingOver, 4_510},
		{"to the cent", 427_500, bankreceipt.StandingExact, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := bankreceipt.Score(
				receiptOf("PMT SEED-INV-0003", "", tc.amount),
				nil,
				[]bankreceipt.InvoiceEvidence{invoice},
			)
			require.Len(t, got, 1)
			assert.Equal(t, tc.standing, got[0].Standing)
			assert.Equal(t, tc.difference, got[0].DifferenceMinor)
		})
	}
}

func TestScore_TheSameAmountNamedOnlyInTheMemoIsNotExact(t *testing.T) {
	t.Parallel()

	got := bankreceipt.Score(
		receiptOf("ACH 77120", "REMIT SEED-INV-0010 THANK YOU", 101_250),
		nil,
		[]bankreceipt.InvoiceEvidence{{
			ID: pulid.MustNew("inv_"), Number: "SEED-INV-0010", OpenMinor: 101_250,
		}},
	)

	require.Len(t, got, 1)
	assert.Equal(t, bankreceipt.StandingSameAmount, got[0].Standing)
	assert.Equal(t, []string{"NumberInMemo", "AmountExact"}, got[0].Evidence)
	assert.Equal(t, 70, got[0].Score)
}

func TestScore_NeverAnInvoiceTheReceiptDoesNotName(t *testing.T) {
	t.Parallel()

	got := bankreceipt.Score(
		receiptOf("DEP84850047", "DEPOSIT - NO REMITTANCE DETAIL", 101_250),
		nil,
		[]bankreceipt.InvoiceEvidence{
			{ID: pulid.MustNew("inv_"), Number: "SEED-INV-0010", OpenMinor: 101_250},
			{ID: pulid.MustNew("inv_"), Number: "INV-0006", OpenMinor: 9_999},
		},
	)

	assert.Empty(t, got, "an equal open amount alone is not evidence the receipt carries")
}

func TestScore_ANumberMustBeAWholeToken(t *testing.T) {
	t.Parallel()

	got := bankreceipt.Score(
		receiptOf("SEED-INV-00061", "SEED-INV-0006X", 1),
		nil,
		[]bankreceipt.InvoiceEvidence{{ID: pulid.MustNew("inv_"), Number: "SEED-INV-0006", OpenMinor: 5}},
	)

	assert.Empty(t, got)
}

func TestScore_PaymentScoresAreTheReconciliationQueuesOwn(t *testing.T) {
	t.Parallel()

	receipt := receiptOf("BANK-1", "", 10_000)
	cases := []struct {
		name    string
		payment bankreceipt.PaymentEvidence
		score   int
		reason  string
	}{
		{
			"reference, amount and date",
			bankreceipt.PaymentEvidence{Reference: "bank 1", AmountMinor: 10_000, PaymentDate: receipt.ReceiptDate},
			100, "Exact normalized reference, exact amount, and close date match",
		},
		{
			"reference and amount",
			bankreceipt.PaymentEvidence{Reference: "BANK-1", AmountMinor: 10_000},
			95, "Exact normalized reference and amount match",
		},
		{
			"reference and near amount",
			bankreceipt.PaymentEvidence{Reference: "BANK-1", AmountMinor: 10_050},
			80, "Exact normalized reference and near amount match",
		},
		{
			"partial reference and amount",
			bankreceipt.PaymentEvidence{Reference: "BANK-12", AmountMinor: 10_000},
			75, "Partial normalized reference and exact amount match",
		},
		{
			"near amount only",
			bankreceipt.PaymentEvidence{Reference: "ZZZ", AmountMinor: 9_900},
			20, "Near amount match",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.payment.ID = pulid.MustNew("cpay_")
			got := bankreceipt.Score(receipt, []bankreceipt.PaymentEvidence{tc.payment}, nil)
			require.Len(t, got, 1)
			assert.Equal(t, tc.score, got[0].Score)
			assert.Equal(t, tc.reason, got[0].PaymentReason())
		})
	}
}

func TestScore_APaymentsCustomerIsThePayerOfItsInvoices(t *testing.T) {
	t.Parallel()

	customer := pulid.MustNew("cus_")
	receipt := receiptOf("ACH-4471", "PAYMENT INV-1001", 50_000)
	got := bankreceipt.Score(
		receipt,
		[]bankreceipt.PaymentEvidence{{
			ID: pulid.MustNew("cpay_"), CustomerID: customer, Reference: "ACH-4471", AmountMinor: 49_950,
		}},
		[]bankreceipt.InvoiceEvidence{{
			ID: pulid.MustNew("inv_"), CustomerID: customer, Number: "INV-1001", OpenMinor: 50_000,
		}},
	)

	require.Len(t, got, 2)
	assert.Equal(t, bankreceipt.CandidateCustomerPayment, got[0].Kind, "80 outranks the invoice's 75")
	assert.Equal(t, 1, got[0].Rank)
	assert.Equal(t, bankreceipt.CandidateInvoice, got[1].Kind)
	assert.Equal(t, []string{"NumberInMemo", "AmountExact", "Payer"}, got[1].Evidence)
	assert.Equal(t, 75, got[1].Score)
	assert.Equal(t, 2, got[1].Rank)
}

func TestScore_KeepsTheTenStrongest(t *testing.T) {
	t.Parallel()

	receipt := receiptOf("", "", 10_000)
	payments := make([]bankreceipt.PaymentEvidence, 0, 12)
	for range 12 {
		payments = append(payments, bankreceipt.PaymentEvidence{ID: pulid.MustNew("cpay_"), AmountMinor: 10_000})
	}

	got := bankreceipt.Score(receipt, payments, nil)

	require.Len(t, got, bankreceipt.MaxCandidates)
	assert.Equal(t, 10, got[9].Rank)
}

func TestAutoMatchPayment_OnlyAStrongPaymentWellAheadOfTheNext(t *testing.T) {
	t.Parallel()

	strong := &bankreceipt.Candidate{
		Kind: bankreceipt.CandidateCustomerPayment, CustomerPaymentID: pulid.MustNew("cpay_"), Score: 95,
	}
	near := &bankreceipt.Candidate{Kind: bankreceipt.CandidateCustomerPayment, Score: 85}
	far := &bankreceipt.Candidate{Kind: bankreceipt.CandidateCustomerPayment, Score: 75}
	invoice := &bankreceipt.Candidate{Kind: bankreceipt.CandidateInvoice, InvoiceID: pulid.MustNew("inv_"), Score: 90}

	id, ok := bankreceipt.AutoMatchPayment([]*bankreceipt.Candidate{strong, far})
	assert.True(t, ok)
	assert.Equal(t, strong.CustomerPaymentID, id)

	_, ok = bankreceipt.AutoMatchPayment([]*bankreceipt.Candidate{strong, near})
	assert.False(t, ok, "two payments within 15 points are for a person to choose")

	_, ok = bankreceipt.AutoMatchPayment([]*bankreceipt.Candidate{invoice})
	assert.False(t, ok, "a receipt reconciles to a payment, never straight to an invoice")
}

func TestLookupNumbers_OnlyTokensThatCanBeAnInvoiceNumber(t *testing.T) {
	t.Parallel()

	receipt := receiptOf("chk 581524", "Check 40418: $1,200.00 against seed-inv-0009, thanks", 1)

	assert.Equal(t, []string{
		"581524", "40418", "200.00", "seed-inv-0009", "SEED-INV-0009",
	}, receipt.LookupNumbers())
}
