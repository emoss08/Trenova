package billingcontrolpolicyservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCanAutoCreateInvoiceDraft(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})
	assert.False(t, svc.CanAutoCreateInvoiceDraft(nil))
	assert.False(
		t,
		svc.CanAutoCreateInvoiceDraft(
			&billingcontrol.BillingControl{
				InvoiceDraftCreationMode: billingcontrol.InvoiceDraftCreationModeManualOnly,
			},
		),
	)
	assert.True(
		t,
		svc.CanAutoCreateInvoiceDraft(
			&billingcontrol.BillingControl{
				InvoiceDraftCreationMode: billingcontrol.InvoiceDraftCreationModeAutomaticWhenTransferred,
			},
		),
	)
}

func TestCanAutoPostInvoice(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})
	control := &billingcontrol.BillingControl{
		InvoiceDraftCreationMode: billingcontrol.InvoiceDraftCreationModeAutomaticWhenTransferred,
		InvoicePostingMode:       billingcontrol.InvoicePostingModeAutomaticWhenNoBlockingExceptions,
	}

	assert.True(t, svc.CanAutoPostInvoice(control, nil))
	assert.False(
		t,
		svc.CanAutoPostInvoice(
			control,
			&customer.Customer{BillingProfile: &customer.CustomerBillingProfile{AutoBill: false}},
		),
	)
	assert.True(
		t,
		svc.CanAutoPostInvoice(
			control,
			&customer.Customer{BillingProfile: &customer.CustomerBillingProfile{AutoBill: true}},
		),
	)

	control.InvoicePostingMode = billingcontrol.InvoicePostingModeManualReviewRequired
	assert.False(t, svc.CanAutoPostInvoice(control, nil))
}

func TestValidateInvoicePosting(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()})
	require.NoError(t, svc.ValidateInvoicePosting(nil, "manual"))
	require.Error(t, svc.ValidateInvoicePosting(nil, AutoPostInvoiceTrigger))
	require.Error(
		t,
		svc.ValidateInvoicePosting(
			&billingcontrol.BillingControl{
				InvoicePostingMode: billingcontrol.InvoicePostingModeManualReviewRequired,
			},
			AutoPostInvoiceTrigger,
		),
	)
	require.Error(
		t,
		svc.ValidateInvoicePosting(
			&billingcontrol.BillingControl{
				InvoicePostingMode:       billingcontrol.InvoicePostingModeAutomaticWhenNoBlockingExceptions,
				InvoiceDraftCreationMode: billingcontrol.InvoiceDraftCreationModeManualOnly,
			},
			AutoPostInvoiceTrigger,
		),
	)
	require.NoError(
		t,
		svc.ValidateInvoicePosting(
			&billingcontrol.BillingControl{
				InvoicePostingMode:       billingcontrol.InvoicePostingModeAutomaticWhenNoBlockingExceptions,
				InvoiceDraftCreationMode: billingcontrol.InvoiceDraftCreationModeAutomaticWhenTransferred,
			},
			AutoPostInvoiceTrigger,
		),
	)
}
