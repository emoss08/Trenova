package accountingcontrolpolicyservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCanCreateInvoiceLedgerEntry(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})
	control := &accountingcontrol.AccountingControl{
		AccountingBasis:          accountingcontrol.AccountingBasisAccrual,
		RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionOnInvoicePost,
	}

	assert.True(
		t,
		svc.CanCreateInvoiceLedgerEntry(control, accountingcontrol.JournalSourceEventInvoicePosted),
	)
	assert.True(
		t,
		svc.CanCreateInvoiceLedgerEntry(
			control,
			accountingcontrol.JournalSourceEventCreditMemoPosted,
		),
	)
	assert.False(
		t,
		svc.CanCreateInvoiceLedgerEntry(
			control,
			accountingcontrol.JournalSourceEventCustomerPaymentPosted,
		),
	)

	control.AccountingBasis = accountingcontrol.AccountingBasisCash
	assert.False(
		t,
		svc.CanCreateInvoiceLedgerEntry(control, accountingcontrol.JournalSourceEventInvoicePosted),
	)

	control.AccountingBasis = accountingcontrol.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = accountingcontrol.RevenueRecognitionOnCashReceipt
	assert.False(
		t,
		svc.CanCreateInvoiceLedgerEntry(control, accountingcontrol.JournalSourceEventInvoicePosted),
	)
}

func TestCanUseAutomaticSourcePosting(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})
	control := &accountingcontrol.AccountingControl{
		AccountingBasis:          accountingcontrol.AccountingBasisAccrual,
		RevenueRecognitionPolicy: accountingcontrol.RevenueRecognitionOnInvoicePost,
		JournalPostingMode:       accountingcontrol.JournalPostingModeAutomatic,
		AutoPostSourceEvents: []accountingcontrol.JournalSourceEventType{
			accountingcontrol.JournalSourceEventInvoicePosted,
		},
	}

	assert.True(
		t,
		svc.CanUseAutomaticSourcePosting(
			control,
			accountingcontrol.JournalSourceEventInvoicePosted,
		),
	)
	assert.False(
		t,
		svc.CanUseAutomaticSourcePosting(
			control,
			accountingcontrol.JournalSourceEventCreditMemoPosted,
		),
	)

	control.JournalPostingMode = accountingcontrol.JournalPostingModeManual
	assert.False(
		t,
		svc.CanUseAutomaticSourcePosting(
			control,
			accountingcontrol.JournalSourceEventInvoicePosted,
		),
	)
}

func TestValidateManualPeriodClose(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})
	require.NoError(t, svc.ValidateManualPeriodClose(nil))
	require.NoError(
		t,
		svc.ValidateManualPeriodClose(
			&accountingcontrol.AccountingControl{
				PeriodCloseMode: accountingcontrol.PeriodCloseModeManualOnly,
			},
		),
	)
	require.Error(
		t,
		svc.ValidateManualPeriodClose(
			&accountingcontrol.AccountingControl{
				PeriodCloseMode: accountingcontrol.PeriodCloseModeSystemScheduled,
			},
		),
	)
	require.Error(
		t,
		svc.ValidateManualPeriodClose(
			&accountingcontrol.AccountingControl{
				PeriodCloseMode:            accountingcontrol.PeriodCloseModeManualOnly,
				RequirePeriodCloseApproval: true,
			},
		),
	)
}

func TestResolveFXQuoteDate(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})

	tests := []struct {
		name     string
		policy   accountingcontrol.ExchangeRateDatePolicy
		expected int64
	}{
		{
			name:     "document date policy",
			policy:   accountingcontrol.ExchangeRateDatePolicyDocumentDate,
			expected: 1_700_000_000,
		},
		{
			name:     "accounting date policy",
			policy:   accountingcontrol.ExchangeRateDatePolicyAccountingDate,
			expected: 1_700_086_400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := svc.ResolveFXQuoteDate(ResolveFXQuoteDateRequest{
				Policy:         tt.policy,
				DocumentDate:   1_700_000_000,
				AccountingDate: 1_700_086_400,
			})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
