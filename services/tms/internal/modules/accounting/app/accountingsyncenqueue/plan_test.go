package accountingsyncenqueue

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type countedConnections struct {
	accountingconnectionports.AccountingConnectionRepository
	listed int
}

func (c *countedConnections) ListByTenant(
	context.Context,
	pagination.TenantInfo,
) ([]*accountingconnection.AccountingConnection, error) {
	c.listed++
	return nil, nil
}

func postedInvoiceRequest() *accountingsyncports.AccountingSyncEnqueueRequest {
	tenant := plantest.Tenant()
	return accountingsyncports.InvoiceSyncRequest(&invoice.Invoice{
		ID:             pulid.MustNew("inv_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Number:         "INV-1001",
		BillType:       billingvocab.BillTypeInvoice,
	}, accountingsync.PostedSourceEvent(billingvocab.BillTypeInvoice))
}

func TestPostingAnInvoiceQueuesNoAccountingSyncWithoutAccounting(t *testing.T) {
	t.Parallel()

	connections := &countedConnections{}
	enqueuer := New(Params{
		Logger:      zap.NewNop(),
		Connections: connections,
		Features:    plantest.BaseOnly(),
	})

	require.NoError(t, enqueuer.Enqueue(t.Context(), postedInvoiceRequest()))
	assert.Zero(t, connections.listed, "no connection is read and no record is queued")

	destinations, err := enqueuer.Destinations(t.Context(), postedInvoiceRequest())
	require.NoError(t, err)
	assert.Empty(t, destinations)
}

func TestPostingAnInvoiceLooksForConnectionsWithAccounting(t *testing.T) {
	t.Parallel()

	connections := &countedConnections{}
	enqueuer := New(Params{
		Logger:      zap.NewNop(),
		Connections: connections,
		Features:    plantest.AllFeatures(),
	})

	require.NoError(t, enqueuer.Enqueue(t.Context(), postedInvoiceRequest()))
	assert.Equal(t, 1, connections.listed)
}
