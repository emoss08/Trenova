package shipmentbillingtransfer

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentbillingreadiness"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type transferBatchEntry struct {
	entity    *shipment.Shipment
	markReady bool
	readiness *shipmentbillingports.ShipmentBillingReadiness
	failure   shipmentbillingports.BillingTransferFailureCode
	err       error
}

type transferBatch struct {
	entries []transferBatchEntry
	pending []shipmentbillingreadiness.PendingReadiness
	payers  map[pulid.ID]*customer.Customer
}

func (s *Service) readTransferBatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
	markCompleted bool,
) (*transferBatch, error) {
	shipments, err := s.repo.GetByIDs(ctx, &shipmentports.GetShipmentsByIDsRequest{
		TenantInfo:     tenantInfo,
		ShipmentIDs:    ids,
		IncludeCharges: true,
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[pulid.ID]*shipment.Shipment, len(shipments))
	for _, entity := range shipments {
		if entity != nil {
			byID[entity.ID] = entity
		}
	}

	batch := &transferBatch{
		entries: make([]transferBatchEntry, len(ids)),
		pending: make([]shipmentbillingreadiness.PendingReadiness, 0, len(ids)),
	}
	for idx, id := range ids {
		entry := &batch.entries[idx]
		entry.entity = byID[id]
		if entry.entity == nil {
			entry.failure = shipmentbillingports.BillingTransferFailureNotFound
			entry.err = errortypes.NewNotFoundError("Shipment not found")
			continue
		}

		markReady, code, gateErr := transferGate(entry.entity, markCompleted)
		if gateErr != nil {
			entry.failure, entry.err = code, gateErr
			continue
		}
		entry.markReady = markReady

		resolution, shareErr := chargeallocation.ResolveShares(
			ctx,
			entry.entity,
			entry.entity.ChargeAllocations,
		)
		if shareErr != nil {
			entry.failure = shipmentbillingports.BillingTransferFailureUnexpected
			entry.err = shareErr
			continue
		}
		batch.pending = append(batch.pending, shipmentbillingreadiness.PendingReadiness{
			Index:      idx,
			Entity:     entry.entity,
			Resolution: resolution,
			MarkReady:  markReady,
		})
	}

	sources, err := s.readiness.ReadinessSourcesFor(ctx, tenantInfo, batch.pending)
	if err != nil {
		return nil, err
	}
	batch.payers = sources.Payers
	for _, pending := range batch.pending {
		batch.entries[pending.Index].readiness = s.readiness.ReadinessFrom(
			pending.Entity,
			pending.Resolution,
			sources,
		)
	}

	return batch, nil
}
