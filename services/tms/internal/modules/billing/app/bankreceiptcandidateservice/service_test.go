package bankreceiptcandidateservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRepo struct {
	bankreceiptports.CandidateRepository
	touching []*bankreceipt.BankReceipt
	payments map[pulid.ID][]bankreceipt.PaymentEvidence
	invoices []bankreceipt.InvoiceEvidence
	numbers  [][]string
	asked    [][]pulid.ID
	replaced []*bankreceiptports.ReplaceCandidatesRequest
}

func (f *fakeRepo) ReceiptsTouching(
	_ context.Context,
	req *bankreceiptports.ReceiptsTouchingRequest,
) ([]*bankreceipt.BankReceipt, error) {
	f.asked = append(f.asked, req.SubjectIDs)

	return f.touching, nil
}

func (f *fakeRepo) PaymentEvidence(
	_ context.Context,
	receipts []*bankreceipt.BankReceipt,
) (map[pulid.ID][]bankreceipt.PaymentEvidence, error) {
	out := make(map[pulid.ID][]bankreceipt.PaymentEvidence, len(receipts))
	for _, receipt := range receipts {
		out[receipt.ID] = f.payments[receipt.ID]
	}

	return out, nil
}

func (f *fakeRepo) OpenInvoicesByNumber(
	_ context.Context,
	numbers []string,
) ([]bankreceipt.InvoiceEvidence, error) {
	f.numbers = append(f.numbers, numbers)

	return f.invoices, nil
}

func (f *fakeRepo) Replace(_ context.Context, req *bankreceiptports.ReplaceCandidatesRequest) error {
	f.replaced = append(f.replaced, req)

	return nil
}

func tenantOf(receipt *bankreceipt.BankReceipt) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: receipt.OrganizationID, BuID: receipt.BusinessUnitID}
}

func newReceipt(status bankreceipt.Status, reference string, amount int64) *bankreceipt.BankReceipt {
	return &bankreceipt.BankReceipt{
		ID:              pulid.MustNew("brcpt_"),
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		Status:          status,
		ReferenceNumber: reference,
		AmountMinor:     amount,
		ReceiptDate:     1_700_000_000,
	}
}

func TestRefresh_ReplacesEachTouchedReceiptsCandidatesAsOneSet(t *testing.T) {
	t.Parallel()

	open := newReceipt(bankreceipt.StatusException, "SEED-INV-0003", 415_500)
	matched := newReceipt(bankreceipt.StatusMatched, "SEED-INV-0003", 427_500)
	invoice := bankreceipt.InvoiceEvidence{
		ID: pulid.MustNew("inv_"), Number: "SEED-INV-0003", OpenMinor: 427_500,
	}
	repo := &fakeRepo{
		touching: []*bankreceipt.BankReceipt{open, matched},
		invoices: []bankreceipt.InvoiceEvidence{invoice},
	}
	svc := New(Params{Repo: repo, Logger: zap.NewNop()})

	touched, err := svc.Refresh(t.Context(), tenantOf(open), []pulid.ID{invoice.ID})
	require.NoError(t, err)

	assert.Equal(t, 2, touched)
	require.Len(t, repo.replaced, 1)
	write := repo.replaced[0]
	assert.ElementsMatch(t, []pulid.ID{open.ID, matched.ID}, write.ReceiptIDs,
		"a matched receipt is in the set so its old candidates are cleared")
	require.Len(t, write.Candidates, 1, "only the receipt waiting to be applied gets candidates")
	assert.Equal(t, open.ID, write.Candidates[0].BankReceiptID)
	assert.Equal(t, bankreceipt.StandingShort, write.Candidates[0].Standing)
	assert.Equal(t, int64(-12_000), write.Candidates[0].DifferenceMinor)
}

func TestRefresh_ClearsANamedReceiptThatNoLongerExists(t *testing.T) {
	t.Parallel()

	gone := pulid.MustNew("brcpt_")
	repo := &fakeRepo{}
	svc := New(Params{Repo: repo, Logger: zap.NewNop()})

	_, err := svc.Refresh(t.Context(), pagination.TenantInfo{}, []pulid.ID{gone})
	require.NoError(t, err)

	require.Len(t, repo.replaced, 1)
	assert.Equal(t, []pulid.ID{gone}, repo.replaced[0].ReceiptIDs)
	assert.Empty(t, repo.replaced[0].Candidates)
}

func TestRefresh_IsTheSameWriteWhenRunAgain(t *testing.T) {
	t.Parallel()

	receipt := newReceipt(bankreceipt.StatusException, "ACH-1", 10_000)
	repo := &fakeRepo{
		touching: []*bankreceipt.BankReceipt{receipt},
		payments: map[pulid.ID][]bankreceipt.PaymentEvidence{receipt.ID: {{
			ID: pulid.MustNew("cpay_"), Reference: "ACH-1", AmountMinor: 10_000,
		}}},
	}
	svc := New(Params{Repo: repo, Logger: zap.NewNop()})

	for range 2 {
		_, err := svc.Refresh(t.Context(), tenantOf(receipt), []pulid.ID{receipt.ID})
		require.NoError(t, err)
	}

	require.Len(t, repo.replaced, 2)
	assert.Equal(t, repo.replaced[0].ReceiptIDs, repo.replaced[1].ReceiptIDs)
	assert.Equal(t, repo.replaced[0].Candidates, repo.replaced[1].Candidates,
		"a retried page writes the same rows: the activity is idempotent")
}

func TestRefresh_TouchesNothingWhenNoSubjectReachesAReceipt(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc := New(Params{Repo: repo, Logger: zap.NewNop()})

	touched, err := svc.Refresh(t.Context(), pagination.TenantInfo{}, []pulid.ID{pulid.MustNew("inv_")})
	require.NoError(t, err)

	assert.Zero(t, touched)
	assert.Empty(t, repo.replaced)
}

func TestRefreshReceipt_StoresAndReturnsTheScoredSet(t *testing.T) {
	t.Parallel()

	receipt := newReceipt(bankreceipt.StatusImported, "SEED-INV-0006", 253_125)
	repo := &fakeRepo{invoices: []bankreceipt.InvoiceEvidence{{
		ID: pulid.MustNew("inv_"), Number: "SEED-INV-0006", OpenMinor: 253_125,
	}}}
	svc := New(Params{Repo: repo, Logger: zap.NewNop()})

	got, err := svc.RefreshReceipt(t.Context(), receipt)
	require.NoError(t, err)

	require.Len(t, got, 1)
	assert.Equal(t, bankreceipt.StandingExact, got[0].Standing)
	require.Len(t, repo.replaced, 1)
	assert.Equal(t, []pulid.ID{receipt.ID}, repo.replaced[0].ReceiptIDs)
	assert.Equal(t, got, repo.replaced[0].Candidates)
	assert.Equal(t, [][]string{{"SEED-INV-0006"}}, repo.numbers)
}
