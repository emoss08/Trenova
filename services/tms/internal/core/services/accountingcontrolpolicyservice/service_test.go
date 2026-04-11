package accountingcontrolpolicyservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCanCreateInvoiceLedgerEntry(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})
	control := &tenant.AccountingControl{
		AccountingBasis:          tenant.AccountingBasisAccrual,
		RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost,
	}

	assert.True(t, svc.CanCreateInvoiceLedgerEntry(control, tenant.JournalSourceEventInvoicePosted))
	assert.True(t, svc.CanCreateInvoiceLedgerEntry(control, tenant.JournalSourceEventCreditMemoPosted))
	assert.False(t, svc.CanCreateInvoiceLedgerEntry(control, tenant.JournalSourceEventCustomerPaymentPosted))

	control.AccountingBasis = tenant.AccountingBasisCash
	assert.False(t, svc.CanCreateInvoiceLedgerEntry(control, tenant.JournalSourceEventInvoicePosted))

	control.AccountingBasis = tenant.AccountingBasisAccrual
	control.RevenueRecognitionPolicy = tenant.RevenueRecognitionOnCashReceipt
	assert.False(t, svc.CanCreateInvoiceLedgerEntry(control, tenant.JournalSourceEventInvoicePosted))
}

func TestCanUseAutomaticSourcePosting(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})
	control := &tenant.AccountingControl{
		AccountingBasis:          tenant.AccountingBasisAccrual,
		RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost,
		JournalPostingMode:       tenant.JournalPostingModeAutomatic,
		AutoPostSourceEvents:     []tenant.JournalSourceEventType{tenant.JournalSourceEventInvoicePosted},
	}

	assert.True(t, svc.CanUseAutomaticSourcePosting(control, tenant.JournalSourceEventInvoicePosted))
	assert.False(t, svc.CanUseAutomaticSourcePosting(control, tenant.JournalSourceEventCreditMemoPosted))

	control.JournalPostingMode = tenant.JournalPostingModeManual
	assert.False(t, svc.CanUseAutomaticSourcePosting(control, tenant.JournalSourceEventInvoicePosted))
}

func TestValidateManualPeriodClose(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})
	require.NoError(t, svc.ValidateManualPeriodClose(nil))
	require.NoError(t, svc.ValidateManualPeriodClose(&tenant.AccountingControl{PeriodCloseMode: tenant.PeriodCloseModeManualOnly}))
	require.Error(t, svc.ValidateManualPeriodClose(&tenant.AccountingControl{PeriodCloseMode: tenant.PeriodCloseModeSystemScheduled}))
	require.Error(t, svc.ValidateManualPeriodClose(&tenant.AccountingControl{PeriodCloseMode: tenant.PeriodCloseModeManualOnly, RequirePeriodCloseApproval: true}))
}
