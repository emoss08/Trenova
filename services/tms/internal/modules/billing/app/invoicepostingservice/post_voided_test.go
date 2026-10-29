package invoicepostingservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceservicetest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPostRefusesAVoidedInvoice(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(f.Invoice(invoice.StatusVoided), nil).
		Once()

	_, err := newPostingTestService(
		repo,
		billingqueueportsmock.NewMockBillingQueueRepository(t),
	).Post(
		t.Context(),
		&invoiceports.PostInvoiceRequest{
			InvoiceID:   f.InvoiceID,
			TenantInfo:  f.TenantInfo,
			TriggeredBy: "manual",
		},
		f.Actor,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Voided invoices cannot be posted")
}
