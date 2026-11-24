package bankreceiptcardservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCards struct {
	asked *bankreceiptports.GetDeskCardRequest
	card  *bankreceiptports.DeskCard
}

func (f *fakeCards) GetDeskCard(
	_ context.Context,
	req *bankreceiptports.GetDeskCardRequest,
) (*bankreceiptports.DeskCard, error) {
	f.asked = req

	return f.card, nil
}

func tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestGetPassesTheSectionsTheReaderMaySeeToTheRead(t *testing.T) {
	t.Parallel()

	receipt := &bankreceipt.BankReceipt{ID: pulid.MustNew("brcpt_"), Status: bankreceipt.StatusMatched}
	cards := &fakeCards{card: &bankreceiptports.DeskCard{Receipt: receipt, PaymentHidden: true}}
	svc := New(Params{Cards: cards, Candidates: &fakeCandidates{}})

	card, err := svc.Get(t.Context(), &bankreceiptports.GetDeskCardRequest{
		TenantInfo:      tenantInfo(),
		ReceiptID:       receipt.ID,
		IncludeWorkItem: true,
	})
	require.NoError(t, err)

	require.NotNil(t, cards.asked)
	assert.False(t, cards.asked.IncludePayment)
	assert.True(t, cards.asked.IncludeWorkItem)
	assert.Same(t, receipt, card.Receipt)
	assert.True(t, card.PaymentHidden)
}

func TestGetRefusesARequestWithNoReceiptOrNoOrganization(t *testing.T) {
	t.Parallel()

	cards := &fakeCards{}
	svc := New(Params{Cards: cards, Candidates: &fakeCandidates{}})

	_, err := svc.Get(t.Context(), &bankreceiptports.GetDeskCardRequest{TenantInfo: tenantInfo()})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))

	_, err = svc.Get(t.Context(), &bankreceiptports.GetDeskCardRequest{
		ReceiptID: pulid.MustNew("brcpt_"),
	})
	require.Error(t, err)
	assert.Nil(t, cards.asked, "nothing is read without a receipt and an organization")
}

type fakeCandidates struct {
	bankreceiptports.CandidateRepository
	asked *bankreceiptports.ListCandidatesRequest
	views []*bankreceiptports.CandidateView
}

func (f *fakeCandidates) ListForReceipt(
	_ context.Context,
	req *bankreceiptports.ListCandidatesRequest,
) ([]*bankreceiptports.CandidateView, error) {
	f.asked = req

	return f.views, nil
}

func TestGetShowsTheCandidatesOfAnUnappliedReceipt(t *testing.T) {
	t.Parallel()

	receipt := &bankreceipt.BankReceipt{ID: pulid.MustNew("brcpt_"), Status: bankreceipt.StatusException}
	invoiceID := pulid.MustNew("inv_")
	candidates := &fakeCandidates{views: []*bankreceiptports.CandidateView{{
		Candidate: &bankreceipt.Candidate{
			Rank:            1,
			Kind:            bankreceipt.CandidateInvoice,
			InvoiceID:       invoiceID,
			Standing:        bankreceipt.StandingShort,
			Evidence:        []string{"NumberInReference"},
			OpenMinor:       427_500,
			DifferenceMinor: -12_000,
		},
		InvoiceNumber: "SEED-INV-0003",
		CustomerName:  "Premier Farms",
	}}}
	svc := New(Params{
		Cards:      &fakeCards{card: &bankreceiptports.DeskCard{Receipt: receipt}},
		Candidates: candidates,
	})

	card, err := svc.Get(t.Context(), &bankreceiptports.GetDeskCardRequest{
		TenantInfo:      tenantInfo(),
		ReceiptID:       receipt.ID,
		IncludePayment:  true,
		IncludeInvoices: true,
	})
	require.NoError(t, err)

	assert.Equal(t, []bankreceiptports.DeskCardCandidate{{
		Rank:            1,
		Kind:            "Invoice",
		InvoiceID:       invoiceID,
		InvoiceNumber:   "SEED-INV-0003",
		CustomerName:    "Premier Farms",
		Standing:        "Short",
		Evidence:        []string{"NumberInReference"},
		OpenMinor:       427_500,
		DifferenceMinor: -12_000,
	}}, card.Candidates)
	assert.Empty(t, card.CandidatesWithheld)
	assert.ElementsMatch(t,
		[]bankreceipt.CandidateKind{bankreceipt.CandidateCustomerPayment, bankreceipt.CandidateInvoice},
		candidates.asked.Kinds)
}

func TestGetWithholdsTheCandidatesOfAKindTheReaderMayNotRead(t *testing.T) {
	t.Parallel()

	receipt := &bankreceipt.BankReceipt{ID: pulid.MustNew("brcpt_"), Status: bankreceipt.StatusException}
	candidates := &fakeCandidates{}
	svc := New(Params{
		Cards:      &fakeCards{card: &bankreceiptports.DeskCard{Receipt: receipt}},
		Candidates: candidates,
	})

	card, err := svc.Get(t.Context(), &bankreceiptports.GetDeskCardRequest{
		TenantInfo:     tenantInfo(),
		ReceiptID:      receipt.ID,
		IncludePayment: true,
	})
	require.NoError(t, err)

	assert.Equal(t, []bankreceipt.CandidateKind{bankreceipt.CandidateCustomerPayment}, candidates.asked.Kinds)
	assert.Equal(t, []string{"Invoice"}, card.CandidatesWithheld)
}

func TestGetReadsNoCandidatesForAMatchedReceipt(t *testing.T) {
	t.Parallel()

	receipt := &bankreceipt.BankReceipt{ID: pulid.MustNew("brcpt_"), Status: bankreceipt.StatusMatched}
	candidates := &fakeCandidates{}
	svc := New(Params{
		Cards:      &fakeCards{card: &bankreceiptports.DeskCard{Receipt: receipt}},
		Candidates: candidates,
	})

	card, err := svc.Get(t.Context(), &bankreceiptports.GetDeskCardRequest{
		TenantInfo: tenantInfo(), ReceiptID: receipt.ID, IncludePayment: true, IncludeInvoices: true,
	})
	require.NoError(t, err)

	assert.Nil(t, candidates.asked)
	assert.Empty(t, card.Candidates)
}
