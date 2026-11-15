package shipmenteffects

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// KeepPlanFields keeps a shipment write from changing the values the plan
// withholds: the shipment's own (its tender and detention charges, and a
// rating method of the organization's own while Rating is off the plan),
// each move's (its driver's acknowledgement) and each stop's (its detention
// override). A part is kept against the stored part with its id; a new one
// takes nothing the plan withholds. original is nil on a create.
func (s *Effects) KeepPlanFields(ctx context.Context, entity, original *shipment.Shipment) error {
	if entity == nil {
		return nil
	}
	if err := s.planFields.Keep(ctx, permission.ResourceShipment, entity, original); err != nil {
		return err
	}
	if err := s.keepPricingTemplate(ctx, entity, original); err != nil {
		return err
	}

	var storedMoves map[pulid.ID]*shipment.ShipmentMove
	var storedStops map[pulid.ID]*shipment.Stop
	var storedCharges map[pulid.ID]*shipment.AdditionalCharge
	if original != nil {
		storedMoves = byID(original.Moves, func(m *shipment.ShipmentMove) pulid.ID { return m.ID })
		storedStops = make(map[pulid.ID]*shipment.Stop)
		for _, move := range original.Moves {
			if move == nil {
				continue
			}
			for id, stop := range byID(move.Stops, func(st *shipment.Stop) pulid.ID { return st.ID }) {
				storedStops[id] = stop
			}
		}
		storedCharges = byID(
			original.AdditionalCharges,
			func(c *shipment.AdditionalCharge) pulid.ID { return c.ID },
		)
	}

	for _, move := range entity.Moves {
		if move == nil {
			continue
		}
		if err := s.planFields.Keep(
			ctx, permission.ResourceShipmentMove, move, storedMoves[move.ID],
		); err != nil {
			return err
		}
		for _, stop := range move.Stops {
			if stop == nil {
				continue
			}
			if err := s.planFields.Keep(
				ctx, permission.ResourceShipmentStop, stop, storedStops[stop.ID],
			); err != nil {
				return err
			}
		}
	}
	for _, charge := range entity.AdditionalCharges {
		if charge == nil {
			continue
		}
		if err := s.planFields.Keep(
			ctx, permission.ResourceShipment, charge, storedCharges[charge.ID],
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *Effects) keepPricingTemplate(
	ctx context.Context,
	entity, original *shipment.Shipment,
) error {
	if entity.FormulaTemplateID.IsNil() {
		return nil
	}

	tenant := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
	if original == nil || original.FormulaTemplateID != entity.FormulaTemplateID {
		return s.pricingTemplates.RefuseOffPlan(ctx, tenant, entity.FormulaTemplateID)
	}

	off, err := s.pricingTemplates.OffPlan(ctx, tenant, entity.FormulaTemplateID)
	if err != nil {
		return err
	}
	if _, withheld := off[entity.FormulaTemplateID]; withheld {
		entity.TariffInputs = original.TariffInputs
	}

	return nil
}

// byID indexes stored parts by their id; a part without one is new and has no
// stored match.
func byID[T any](parts []*T, id func(*T) pulid.ID) map[pulid.ID]*T {
	out := make(map[pulid.ID]*T, len(parts))
	for _, part := range parts {
		if part == nil || id(part).IsNil() {
			continue
		}
		out[id(part)] = part
	}

	return out
}
