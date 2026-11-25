package shipmentwrite

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/stopactualnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// announceStopActuals records the stop events for the stop times a save set
// or corrected, and tells the agents and waits watching each move of the
// arrivals and departures it first recorded, once the save has committed.
func (s *Service) announceStopActuals(
	ctx context.Context,
	updated *shipment.Shipment,
	before map[pulid.ID]stopactualnotice.Snapshot,
	actor kernelports.AuditActor,
) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  updated.OrganizationID,
		BuID:   updated.BusinessUnitID,
		UserID: actor.UserID,
	}
	visits := make([]stopactualnotice.Visit, 0)
	for _, move := range updated.Moves {
		if move == nil {
			continue
		}
		previous := before[move.ID]
		for _, params := range stopactualnotice.Events(tenantInfo, actor, move, previous) {
			s.effects.RecordShipmentEvent(ctx, params)
		}
		visits = append(visits, stopactualnotice.Visits(move, previous)...)
	}
	if len(visits) == 0 {
		return
	}

	kernelports.AfterCommit(ctx, func(committed context.Context) {
		for _, visit := range visits {
			stopactualnotice.Publish(committed, s.agentEvents, tenantInfo, visit)
		}
	})
}
