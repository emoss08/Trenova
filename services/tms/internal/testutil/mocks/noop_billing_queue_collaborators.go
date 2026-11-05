package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var _ billingqueueports.RecheckRequester = (*NoopRecheckRequester)(nil)

type NoopRecheckRequester struct{}

func (*NoopRecheckRequester) RequestRecheck(context.Context, pagination.TenantInfo, ...pulid.ID) {}

type NoopInvoiceCreation struct {
	invoiceports.InvoiceCreationService
}

func (*NoopInvoiceCreation) CreateFromApprovedBillingQueueItem(
	context.Context,
	*invoiceports.CreateInvoiceFromBillingQueueRequest,
	*permissionports.RequestActor,
) (*invoiceports.CreateInvoiceFromBillingQueueResult, error) {
	return &invoiceports.CreateInvoiceFromBillingQueueResult{DeferredToStatement: true}, nil
}

type NoopDetentionBilling struct {
	detentionports.DetentionBillingService
}

func (*NoopDetentionBilling) HoldsForShipments(
	context.Context,
	*detentionports.DetentionBillingHoldsRequest,
) ([]*detention.DetentionOccurrence, error) {
	return nil, nil
}
