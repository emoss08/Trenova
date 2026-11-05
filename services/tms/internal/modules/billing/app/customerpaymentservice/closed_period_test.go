package customerpaymentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/app/journalposting"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerledgerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fiscalperiodportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/realtimeportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const nextPeriodStart = 900

var (
	orgForValidation = pulid.MustNew("org_")
	buForValidation  = pulid.MustNew("bu_")
)

func unappliedPaymentWorld(t *testing.T) (*paymentWorld, *invoice.Invoice) {
	t.Helper()

	payment := postedPayment(0, 15000)
	target := openInvoiceFor(payment, 6000, 0)
	w := newPaymentWorld(t, payment, target)
	ledger := customerledgerportsmock.NewMockCustomerLedgerProjectionRepository(t)
	ledger.EXPECT().AppendEntries(mock.Anything, mock.Anything).Return(nil).Maybe()
	w.svc.customerLedgerRepo = ledger
	w.svc.receiptMatches = &recordingReceiptReleaser{}
	realtime := realtimeportsmock.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Maybe()
	w.svc.realtime = realtime
	w.svc.l = zap.NewNop()

	return w, target
}

func applyRequestFor(
	w *paymentWorld,
	target *invoice.Invoice,
) *customerpaymentports.ApplyCustomerPaymentRequest {
	return &customerpaymentports.ApplyCustomerPaymentRequest{
		PaymentID:      w.payment.ID,
		AccountingDate: 200,
		Applications: []*customerpaymentports.CustomerPaymentApplicationInput{
			{InvoiceID: target.ID, AppliedAmountMinor: 1000},
		},
		TenantInfo: w.tenantInfo,
	}
}

func reverseRequestFor(w *paymentWorld) *customerpaymentports.ReverseCustomerPaymentRequest {
	return &customerpaymentports.ReverseCustomerPaymentRequest{
		PaymentID:      w.payment.ID,
		AccountingDate: 200,
		Reason:         "Chargeback from the bank",
		TenantInfo:     w.tenantInfo,
	}
}

func allowNextOpenPeriod(w *paymentWorld) *fiscalperiod.FiscalPeriod {
	next := &fiscalperiod.FiscalPeriod{
		ID:           pulid.MustNew("fp_"),
		FiscalYearID: w.period.FiscalYearID,
		PeriodNumber: w.period.PeriodNumber + 1,
		StartDate:    nextPeriodStart,
		Status:       fiscalperiod.StatusOpen,
	}
	w.laterPeriods = []*fiscalperiod.FiscalPeriod{next}
	w.control.ClosedPeriodPostingPolicy = accountingcontrol.ClosedPeriodPostingPolicyPostToNextOpen
	w.period.Status = fiscalperiod.StatusClosed
	return next
}

func TestApplyUnappliedRefusesAPeriodThatDoesNotAcceptPostings(t *testing.T) {
	t.Parallel()

	for status, message := range map[fiscalperiod.Status]string{
		fiscalperiod.StatusClosed:   "closed fiscal period",
		fiscalperiod.StatusInactive: "inactive fiscal period",
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			w, target := unappliedPaymentWorld(t)
			w.period.Status = status

			_, err := w.svc.ApplyUnapplied(t.Context(), applyRequestFor(w, target), w.actor)

			require.Error(t, err)
			assert.Contains(t, err.Error(), message)
			assert.Zero(t, w.writes)
			assert.Empty(t, w.postings)
		})
	}
}

func TestReverseRefusesAPeriodThatDoesNotAcceptPostings(t *testing.T) {
	t.Parallel()

	for status, message := range map[fiscalperiod.Status]string{
		fiscalperiod.StatusClosed:   "closed fiscal period",
		fiscalperiod.StatusInactive: "inactive fiscal period",
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			w, _ := unappliedPaymentWorld(t)
			w.period.Status = status

			_, err := w.svc.Reverse(t.Context(), reverseRequestFor(w), w.actor)

			require.Error(t, err)
			assert.Contains(t, err.Error(), message)
			assert.Zero(t, w.writes)
			assert.Empty(t, w.postings)
		})
	}
}

func TestApplyUnappliedPostsToTheNextOpenPeriodWhenThePolicyAllows(t *testing.T) {
	t.Parallel()

	w, target := unappliedPaymentWorld(t)
	next := allowNextOpenPeriod(w)

	preview, err := w.svc.PreviewApplyUnapplied(t.Context(), applyRequestFor(w, target), w.actor)
	require.NoError(t, err)
	assert.Equal(t, int64(nextPeriodStart), preview.Journal.AccountingDate)
	assert.Equal(t, next.ID, preview.Journal.FiscalPeriodID)

	_, err = w.svc.ApplyUnapplied(t.Context(), applyRequestFor(w, target), w.actor)
	require.NoError(t, err)
	require.Len(t, w.postings, 1)
	assert.Equal(t, int64(nextPeriodStart), w.postings[0].AccountingDate)
	assert.Equal(t, next.ID, w.postings[0].FiscalPeriodID)
}

func TestReversePostsToTheNextOpenPeriodWhenThePolicyAllows(t *testing.T) {
	t.Parallel()

	w, _ := unappliedPaymentWorld(t)
	next := allowNextOpenPeriod(w)

	_, err := w.svc.Reverse(t.Context(), reverseRequestFor(w), w.actor)

	require.NoError(t, err)
	require.Len(t, w.postings, 1)
	assert.Equal(t, int64(nextPeriodStart), w.postings[0].AccountingDate)
	assert.Equal(t, next.ID, w.postings[0].FiscalPeriodID)
}

func TestValidatePostAndApplyRefusesAPeriodThatDoesNotAcceptPostings(t *testing.T) {
	t.Parallel()

	for status, message := range map[fiscalperiod.Status]string{
		fiscalperiod.StatusClosed:   "closed fiscal period",
		fiscalperiod.StatusInactive: "inactive fiscal period",
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			_, _, err := validateNewPayment(t, &fiscalperiod.FiscalPeriod{
				ID:           pulid.MustNew("fp_"),
				FiscalYearID: pulid.MustNew("fy_"),
				Status:       status,
			}, &accountingcontrol.AccountingControl{})

			require.Error(t, err)
			assert.Contains(t, err.Error(), message)
		})
	}
}

func TestValidatePostAndApplyResolvesTheNextOpenPeriodWhenThePolicyAllows(t *testing.T) {
	t.Parallel()

	nextID := pulid.MustNew("fp_")
	fiscalYearID := pulid.MustNew("fy_")
	_, resolved, err := validateNewPayment(
		t,
		&fiscalperiod.FiscalPeriod{
			ID:           pulid.MustNew("fp_"),
			FiscalYearID: fiscalYearID,
			PeriodNumber: 1,
			Status:       fiscalperiod.StatusClosed,
		},
		&accountingcontrol.AccountingControl{
			ClosedPeriodPostingPolicy: accountingcontrol.ClosedPeriodPostingPolicyPostToNextOpen,
		},
		&fiscalperiod.FiscalPeriod{
			ID:           nextID,
			FiscalYearID: fiscalYearID,
			PeriodNumber: 2,
			StartDate:    nextPeriodStart,
			Status:       fiscalperiod.StatusOpen,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, nextID, resolved.Period.ID)
	assert.Equal(t, int64(nextPeriodStart), resolved.AccountingDate)
}

func TestApplyCreditMemoRefusesAPeriodThatDoesNotAcceptPostings(t *testing.T) {
	t.Parallel()

	f := newCreditMemoFixture(t)
	f.fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, mock.Anything).
		Return(&fiscalperiod.FiscalPeriod{
			ID:     pulid.MustNew("fp_"),
			Status: fiscalperiod.StatusClosed,
		}, nil).
		Once()

	_, err := f.svc.ApplyCreditMemo(t.Context(), f.applyRequest(1000), f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed fiscal period")
}

func TestPreviewApplyCreditMemoDatesTheApplicationInTheNextOpenPeriod(t *testing.T) {
	t.Parallel()

	f := newCreditMemoFixture(t)
	f.control.ClosedPeriodPostingPolicy = accountingcontrol.ClosedPeriodPostingPolicyPostToNextOpen
	fiscalYearID := pulid.MustNew("fy_")
	f.fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, mock.Anything).
		Return(&fiscalperiod.FiscalPeriod{
			ID:           pulid.MustNew("fp_"),
			FiscalYearID: fiscalYearID,
			PeriodNumber: 1,
			Status:       fiscalperiod.StatusClosed,
		}, nil).
		Once()
	f.fiscalRepo.EXPECT().
		ListByFiscalYearID(mock.Anything, mock.Anything).
		Return([]*fiscalperiod.FiscalPeriod{{
			ID:           pulid.MustNew("fp_"),
			FiscalYearID: fiscalYearID,
			PeriodNumber: 2,
			StartDate:    nextPeriodStart,
			Status:       fiscalperiod.StatusOpen,
		}}, nil).
		Once()
	f.invoiceRepo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: f.memo.ID, TenantInfo: f.tenantInfo}).
		Return(f.memo, nil).
		Once()
	f.invoiceRepo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: f.target.ID, TenantInfo: f.tenantInfo}).
		Return(f.target, nil).
		Once()

	preview, err := f.svc.PreviewApplyCreditMemo(t.Context(), f.applyRequest(1000), f.actor)

	require.NoError(t, err)
	require.Len(t, preview.Applications, 1)
	assert.Equal(t, int64(nextPeriodStart), preview.Applications[0].AccountingDate)
}

func validateNewPayment(
	t *testing.T,
	period *fiscalperiod.FiscalPeriod,
	control *accountingcontrol.AccountingControl,
	later ...*fiscalperiod.FiscalPeriod,
) ([]*invoice.Invoice, *journalposting.ResolvedPeriod, error) {
	t.Helper()

	fiscalRepo := fiscalperiodportsmock.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, fiscalperiodports.GetPeriodByDateRequest{
			OrgID: orgForValidation,
			BuID:  buForValidation,
			Date:  100,
		}).
		Return(period, nil)
	if len(later) > 0 {
		fiscalRepo.EXPECT().ListByFiscalYearID(mock.Anything, mock.Anything).Return(later, nil)
	}
	validator := NewValidator(ValidatorParams{
		InvoiceRepo:      invoiceportsmock.NewMockInvoiceRepository(t),
		FiscalPeriodRepo: fiscalRepo,
	})

	control.DefaultCashAccountID = pulid.MustNew("gla_")
	control.DefaultUnappliedCashAccountID = pulid.MustNew("gla_")

	return validator.ValidatePostAndApply(t.Context(), &customerpayment.Payment{
		OrganizationID: orgForValidation,
		BusinessUnitID: buForValidation,
		CustomerID:     pulid.MustNew("cus_"),
		PaymentDate:    100,
		AccountingDate: 100,
		AmountMinor:    1000,
		CurrencyCode:   "USD",
		PaymentMethod:  customerpayment.MethodACH,
	}, invoiceports.GetInvoiceByIDRequest{}, control)
}
