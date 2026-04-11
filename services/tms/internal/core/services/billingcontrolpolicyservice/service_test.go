package billingcontrolpolicyservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCanAutoCreateInvoiceDraft(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})
	assert.False(t, svc.CanAutoCreateInvoiceDraft(nil))
	assert.False(t, svc.CanAutoCreateInvoiceDraft(&tenant.BillingControl{InvoiceDraftCreationMode: tenant.InvoiceDraftCreationModeManualOnly}))
	assert.True(t, svc.CanAutoCreateInvoiceDraft(&tenant.BillingControl{InvoiceDraftCreationMode: tenant.InvoiceDraftCreationModeAutomaticWhenTransferred}))
}

func TestCanAutoPostInvoice(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})
	control := &tenant.BillingControl{
		InvoiceDraftCreationMode: tenant.InvoiceDraftCreationModeAutomaticWhenTransferred,
		InvoicePostingMode:       tenant.InvoicePostingModeAutomaticWhenNoBlockingExceptions,
	}

	assert.True(t, svc.CanAutoPostInvoice(control, nil))
	assert.False(t, svc.CanAutoPostInvoice(control, &customer.Customer{BillingProfile: &customer.CustomerBillingProfile{AutoBill: false}}))
	assert.True(t, svc.CanAutoPostInvoice(control, &customer.Customer{BillingProfile: &customer.CustomerBillingProfile{AutoBill: true}}))

	control.InvoicePostingMode = tenant.InvoicePostingModeManualReviewRequired
	assert.False(t, svc.CanAutoPostInvoice(control, nil))
}

func TestValidateInvoicePosting(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})
	require.NoError(t, svc.ValidateInvoicePosting(nil, "manual"))
	require.Error(t, svc.ValidateInvoicePosting(nil, AutoPostInvoiceTrigger))
	require.Error(t, svc.ValidateInvoicePosting(&tenant.BillingControl{InvoicePostingMode: tenant.InvoicePostingModeManualReviewRequired}, AutoPostInvoiceTrigger))
	require.Error(t, svc.ValidateInvoicePosting(&tenant.BillingControl{InvoicePostingMode: tenant.InvoicePostingModeAutomaticWhenNoBlockingExceptions, InvoiceDraftCreationMode: tenant.InvoiceDraftCreationModeManualOnly}, AutoPostInvoiceTrigger))
	require.NoError(t, svc.ValidateInvoicePosting(&tenant.BillingControl{InvoicePostingMode: tenant.InvoicePostingModeAutomaticWhenNoBlockingExceptions, InvoiceDraftCreationMode: tenant.InvoiceDraftCreationModeAutomaticWhenTransferred}, AutoPostInvoiceTrigger))
}
