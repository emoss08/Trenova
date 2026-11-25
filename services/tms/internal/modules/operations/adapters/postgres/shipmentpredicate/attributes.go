package shipmentpredicate

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype/equipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
)

func attributePredicates(opts *shipmentports.ShipmentOptions) []keel.Pred {
	preds := make([]keel.Pred, 0, 6)

	if opts.Unassigned != nil {
		preds = append(preds, CoveragePredicate(*opts.Unassigned))
	}
	if pred := equipmentPredicate(opts); pred != nil {
		preds = append(preds, pred)
	}
	if text := strings.TrimSpace(opts.CommodityText); text != "" {
		preds = append(preds, commodityTextPredicate(text))
	}
	if opts.BillingReady != nil {
		ready := BillingTransferCandidateCondition()
		if !*opts.BillingReady {
			ready = keel.Not(ready)
		}
		preds = append(preds, ready)
	}
	if opts.InStatusSince > 0 {
		preds = append(preds, shipmentdb.Shipments.StatusChangedAt.Lte(opts.InStatusSince))
	}

	return preds
}

func stopWindowPredicates(
	st *shipmentdb.StopsTable,
	place *shipmentports.ShipmentStopPlace,
) []keel.Pred {
	if place.WindowFrom <= 0 && place.WindowTo <= 0 {
		return nil
	}

	preds := []keel.Pred{st.ScheduledWindowStart.Gt(0)}
	if len(place.Types) == 0 {
		preds = append(preds, st.Type.In(
			append(shipmentvocab.PickupStopTypes(), shipmentvocab.DeliveryStopTypes()...)...,
		))
	}
	if place.WindowTo > 0 {
		preds = append(preds, st.ScheduledWindowStart.Lte(place.WindowTo))
	}
	if place.WindowFrom > 0 {
		preds = append(preds,
			st.ScheduledWindowEnd.Coalesce(st.ScheduledWindowStart).Gte(place.WindowFrom))
	}

	return preds
}

func equipmentPredicate(opts *shipmentports.ShipmentOptions) keel.Pred {
	sp := shipmentdb.Shipments
	text := strings.TrimSpace(opts.EquipmentTypeText)

	switch {
	case len(opts.EquipmentTypeIDs) > 0:
		return keel.Or(
			sp.TrailerTypeID.In(opts.EquipmentTypeIDs...),
			sp.TractorTypeID.In(opts.EquipmentTypeIDs...),
		)
	case text != "":
		et := equipmenttypedb.EquipmentTypes
		types := et.Pick(et.ID).
			Where(keel.Or(et.Code.ContainsFold(text), et.Description.ContainsFold(text)))

		return keel.Or(sp.TrailerTypeID.InQuery(types), sp.TractorTypeID.InQuery(types))
	default:
		return nil
	}
}

func commodityTextPredicate(text string) keel.Pred {
	sc := shipmentdb.Shipments.Commodities()
	com := commoditydb.Commodities

	return sc.Any(sc.CommodityID.InQuery(com.Pick(com.ID).Where(com.Name.ContainsFold(text))))
}

func CoveragePredicate(unassigned bool) keel.Pred {
	sp := shipmentdb.Shipments
	m := sp.Moves()
	ca := m.CarrierAssignment()
	carrier := m.Any(
		m.Status.Ne(shipmentvocab.MoveStatusCanceled),
		ca.Status.Ne(shipmentvocab.CarrierAssignmentStatusCanceled),
		ca.CarrierID.IsNotNull(),
	)

	if unassigned {
		return keel.And(sp.HasActiveAssignment.IsFalse(), keel.Not(carrier))
	}

	return keel.Or(sp.HasActiveAssignment.IsTrue(), carrier)
}
