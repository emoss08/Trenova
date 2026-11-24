package bankreceiptservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/testutil/mocks/bankreceiptportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type scoring struct {
	payments  []bankreceipt.PaymentEvidence
	invoices  []bankreceipt.InvoiceEvidence
	refreshed []pulid.ID
}

func (s *scoring) Refresh(context.Context, pagination.TenantInfo, []pulid.ID) (int, error) {
	return 0, nil
}

func (s *scoring) RefreshReceipt(
	_ context.Context,
	receipt *bankreceipt.BankReceipt,
) ([]*bankreceipt.Candidate, error) {
	s.refreshed = append(s.refreshed, receipt.ID)

	return bankreceipt.Score(receipt, s.payments, s.invoices), nil
}

type countingWaker struct{ woken []pagination.TenantInfo }

func (w *countingWaker) Wake(_ context.Context, tenant pagination.TenantInfo) {
	w.woken = append(w.woken, tenant)
}

type storedCandidates struct {
	bankreceiptports.CandidateRepository
	views []*bankreceiptports.CandidateView
	asked *bankreceiptports.ListCandidatesRequest
}

func (s *storedCandidates) ListForReceipt(
	_ context.Context,
	req *bankreceiptports.ListCandidatesRequest,
) ([]*bankreceiptports.CandidateView, error) {
	s.asked = req

	return s.views, nil
}

func TestSuggestMatches_ReadsTheStoredPaymentCandidatesInRankOrder(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	receipt := &bankreceipt.BankReceipt{
		ID:     pulid.MustNew("brcpt_"),
		Status: bankreceipt.StatusException,
	}
	first := pulid.MustNew("cpay_")
	second := pulid.MustNew("cpay_")
	customer := pulid.MustNew("cus_")
	stored := &storedCandidates{views: []*bankreceiptports.CandidateView{
		{
			Candidate: &bankreceipt.Candidate{
				Rank:              1,
				Kind:              bankreceipt.CandidateCustomerPayment,
				CustomerPaymentID: first,
				CustomerID:        customer,
				Score:             95,
				Evidence: []string{
					string(bankreceipt.EvidenceReferenceExact),
					string(bankreceipt.EvidenceAmountExact),
				},
				OpenMinor: 10_000,
			},
			Reference: "SG-1",
		},
		{
			Candidate: &bankreceipt.Candidate{
				Rank:              2,
				Kind:              bankreceipt.CandidateCustomerPayment,
				CustomerPaymentID: second,
				Score:             20,
				Evidence:          []string{string(bankreceipt.EvidenceAmountNear)},
				OpenMinor:         10_050,
			},
			Reference: "OTHER",
		},
	}}
	repo := bankreceiptportsmock.NewMockBankReceiptRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(receipt, nil).Once()
	svc := &Service{l: zap.NewNop(), repo: repo, stored: stored}

	suggestions, err := svc.SuggestMatches(t.Context(), &bankreceiptports.GetBankReceiptRequest{
		ReceiptID:  receipt.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)

	assert.Equal(t, []*bankreceiptports.BankReceiptMatchSuggestion{
		{
			CustomerPaymentID: first,
			ReferenceNumber:   "SG-1",
			AmountMinor:       10_000,
			CustomerID:        customer,
			Score:             95,
			Reason:            "Exact normalized reference and amount match",
		},
		{
			CustomerPaymentID: second,
			ReferenceNumber:   "OTHER",
			AmountMinor:       10_050,
			Score:             20,
			Reason:            "Near amount match",
		},
	}, suggestions)
	assert.Equal(t, []bankreceipt.CandidateKind{bankreceipt.CandidateCustomerPayment},
		stored.asked.Kinds, "the page matches a receipt to a payment, so it is offered payments only")
}

func TestSuggestMatches_OffersNothingForAMatchedReceipt(t *testing.T) {
	t.Parallel()

	receipt := &bankreceipt.BankReceipt{ID: pulid.MustNew("brcpt_"), Status: bankreceipt.StatusMatched}
	repo := bankreceiptportsmock.NewMockBankReceiptRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(receipt, nil).Once()
	stored := &storedCandidates{}
	svc := &Service{l: zap.NewNop(), repo: repo, stored: stored}

	suggestions, err := svc.SuggestMatches(t.Context(), &bankreceiptports.GetBankReceiptRequest{
		ReceiptID: receipt.ID,
	})
	require.NoError(t, err)
	assert.Empty(t, suggestions)
	assert.Nil(t, stored.asked, "a matched receipt's candidates are not read")
}
