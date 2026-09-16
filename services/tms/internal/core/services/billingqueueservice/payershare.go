package billingqueueservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// attachPayerShare fills the item's view of its own bill. It needs the shipment
// with charges and allocations already loaded; names for payers the shipment does
// not already carry are fetched in one lookup.
func (s *service) attachPayerShare(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) {
	if item == nil || item.Shipment == nil || item.BillType != billingqueue.BillTypeInvoice {
		return
	}
	payerID := item.BillToCustomerID
	if payerID.IsNil() {
		payerID = item.Shipment.PayerID()
	}

	names := s.payerNames(ctx, item, tenantInfo)
	item.PayerShare = billingqueue.BuildPayerShare(item.Shipment, payerID, names)
}

func (s *service) payerNames(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	tenantInfo pagination.TenantInfo,
) map[pulid.ID]billingqueue.PayerRef {
	shp := item.Shipment
	names := make(map[pulid.ID]billingqueue.PayerRef, 4)
	remember := func(id pulid.ID, name, code string) {
		if id.IsNotNil() && name != "" {
			names[id] = billingqueue.PayerRef{ID: id, Name: name, Code: code}
		}
	}
	if item.BillToCustomer != nil {
		remember(item.BillToCustomer.ID, item.BillToCustomer.Name, item.BillToCustomer.Code)
	}
	if shp.Customer != nil {
		remember(shp.Customer.ID, shp.Customer.Name, shp.Customer.Code)
	}
	if shp.BillToCustomer != nil {
		remember(shp.BillToCustomer.ID, shp.BillToCustomer.Name, shp.BillToCustomer.Code)
	}

	missing := make([]pulid.ID, 0, len(shp.ChargeAllocations)+1)
	want := func(id pulid.ID) {
		if id.IsNil() {
			return
		}
		if _, ok := names[id]; ok {
			return
		}
		for _, existing := range missing {
			if existing == id {
				return
			}
		}
		missing = append(missing, id)
	}
	want(shp.PayerID())
	for _, allocation := range shp.ChargeAllocations {
		if allocation == nil {
			continue
		}
		if allocation.BillToCustomer != nil {
			remember(allocation.BillToCustomerID, allocation.BillToCustomer.Name, allocation.BillToCustomer.Code)
			continue
		}
		if allocation.ChargeKind != shipment.ChargeAllocationKindOrderCharge {
			want(allocation.BillToCustomerID)
		}
	}

	if len(missing) == 0 || s.customerRepo == nil {
		return names
	}
	found, err := s.customerRepo.GetByIDs(ctx, repositories.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: missing,
	})
	if err != nil {
		s.l.Warn("failed to load payer names for billing queue item",
			zap.String("billingQueueItemId", item.ID.String()),
			zap.Error(err),
		)
		return names
	}
	for _, cus := range found {
		if cus != nil {
			remember(cus.ID, cus.Name, cus.Code)
		}
	}

	return names
}
