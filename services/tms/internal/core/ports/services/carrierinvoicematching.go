package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AutoMatchInboundInvoiceRequest struct {
	TenantInfo          pagination.TenantInfo
	EDICarrierInvoiceID pulid.ID
}

type AutoMatchInboundInvoiceResult struct {
	Matched      bool
	AutoAccepted bool
	Status       string
	Warnings     []string
}

// CarrierInvoiceAutoMatcher is the narrow hook inbound EDI routing uses to
// reconcile a pre-linked 210 carrier invoice against its carrier assignment
// without depending on the full settlement service surface.
type CarrierInvoiceAutoMatcher interface {
	AutoMatchInboundInvoice(
		ctx context.Context,
		req *AutoMatchInboundInvoiceRequest,
	) (*AutoMatchInboundInvoiceResult, error)
}
