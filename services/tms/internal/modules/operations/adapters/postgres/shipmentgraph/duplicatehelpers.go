package shipmentgraph

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxShipmentBOLLength = 100

// CopySpec drives a single graph copy of a source shipment. DateAnchor,
// when set, re-anchors the first stop's scheduled window start to the given
// unix time and shifts every other stop window by the same offset.
type CopySpec struct {
	ProNumber   string
	BOL         string
	RequestedBy pulid.ID
	DateAnchor  *int64
}

type Batch struct {
	Shipments         []*shipment.Shipment
	Moves             []*shipment.ShipmentMove
	Stops             []*shipment.Stop
	AdditionalCharges []*shipment.AdditionalCharge
	ChargeAllocations []*shipment.ChargeAllocation
	Commodities       []*shipment.ShipmentCommodity
}

func BuildBatch(
	source *shipment.Shipment,
	proNumbers []string,
	dateAnchor *int64,
	requestedBy pulid.ID,
) *Batch {
	graph := &Batch{
		Shipments: make([]*shipment.Shipment, 0, len(proNumbers)),
		Moves:     make([]*shipment.ShipmentMove, 0, len(source.Moves)*len(proNumbers)),
		Stops:     make([]*shipment.Stop, 0, countStops(source)*len(proNumbers)),
		AdditionalCharges: make(
			[]*shipment.AdditionalCharge,
			0,
			len(source.AdditionalCharges)*len(proNumbers),
		),
		Commodities: make(
			[]*shipment.ShipmentCommodity,
			0,
			len(source.Commodities)*len(proNumbers),
		),
	}

	for idx, proNumber := range proNumbers {
		duplicated := Copy(source, CopySpec{
			ProNumber:   proNumber,
			BOL:         deriveDuplicateBOL(source.BOL, idx+1),
			RequestedBy: requestedBy,
			DateAnchor:  dateAnchor,
		})

		graph.Shipments = append(graph.Shipments, duplicated)
		graph.Moves = append(graph.Moves, duplicated.Moves...)
		for _, move := range duplicated.Moves {
			graph.Stops = append(graph.Stops, move.Stops...)
		}
		graph.AdditionalCharges = append(graph.AdditionalCharges, duplicated.AdditionalCharges...)
		graph.ChargeAllocations = append(graph.ChargeAllocations, duplicated.ChargeAllocations...)
		graph.Commodities = append(graph.Commodities, duplicated.Commodities...)
	}

	return graph
}

// Copy produces one fresh copy of the source shipment with new
// identifiers, reset operational state, and its full move/stop/charge/commodity
// graph attached.
func Copy(source *shipment.Shipment, spec CopySpec) *shipment.Shipment {
	duplicated := &shipment.Shipment{
		ID:                  pulid.MustNew("shp_"),
		BusinessUnitID:      source.BusinessUnitID,
		OrganizationID:      source.OrganizationID,
		ServiceTypeID:       source.ServiceTypeID,
		ShipmentTypeID:      source.ShipmentTypeID,
		CustomerID:          source.CustomerID,
		BillToCustomerID:    pulid.ClonePointer(source.BillToCustomerID),
		FreightTerms:        source.FreightTerms,
		TractorTypeID:       source.TractorTypeID,
		TrailerTypeID:       source.TrailerTypeID,
		FormulaTemplateID:   source.FormulaTemplateID,
		TariffInputs:        source.TariffInputs.Clone(),
		FreightLines:        slices.Clone(source.FreightLines),
		Status:              shipment.StatusNew,
		ProNumber:           spec.ProNumber,
		BOL:                 spec.BOL,
		OtherChargeAmount:   source.OtherChargeAmount,
		FreightChargeAmount: source.FreightChargeAmount,
		TotalChargeAmount:   source.TotalChargeAmount,
		Pieces:              intutils.ClonePointer(source.Pieces),
		Weight:              intutils.ClonePointer(source.Weight),
		TemperatureMin:      intutils.ClonePointer(source.TemperatureMin),
		TemperatureMax:      intutils.ClonePointer(source.TemperatureMax),
		EnteredByID:         spec.RequestedBy,
	}

	moves, _ := duplicateMovesAndStops(source.Moves, duplicated, spec.DateAnchor)
	duplicated.Moves = moves
	duplicated.AdditionalCharges = duplicateAdditionalCharges(
		source.AdditionalCharges,
		duplicated.ID,
	)
	duplicated.ChargeAllocations = duplicateChargeAllocations(
		source,
		duplicated,
	)
	duplicated.Commodities = duplicateShipmentCommodities(source.Commodities, duplicated.ID)

	return duplicated
}

// duplicateChargeAllocations carries the payer split onto the copy. Freight
// rows re-point at the new shipment; accessorial rows re-point at the copied
// charge that sits at the same position as the one they split. Invoice links
// are not copied: the copy has not been billed.
func duplicateChargeAllocations(
	source *shipment.Shipment,
	duplicated *shipment.Shipment,
) []*shipment.ChargeAllocation {
	if len(source.ChargeAllocations) == 0 {
		return nil
	}

	copiedChargeBySource := make(map[pulid.ID]pulid.ID, len(source.AdditionalCharges))
	position := 0
	for _, sourceCharge := range source.AdditionalCharges {
		if sourceCharge == nil {
			continue
		}
		if position < len(duplicated.AdditionalCharges) &&
			duplicated.AdditionalCharges[position] != nil {
			copiedChargeBySource[sourceCharge.ID] = duplicated.AdditionalCharges[position].ID
		}
		position++
	}

	rows := make([]*shipment.ChargeAllocation, 0, len(source.ChargeAllocations))
	for _, allocation := range source.ChargeAllocations {
		if allocation == nil || allocation.ChargeKind == shipment.ChargeAllocationKindOrderCharge {
			continue
		}
		row := &shipment.ChargeAllocation{
			ID:               pulid.MustNew("chal_"),
			BusinessUnitID:   allocation.BusinessUnitID,
			OrganizationID:   allocation.OrganizationID,
			ShipmentID:       &duplicated.ID,
			ChargeKind:       allocation.ChargeKind,
			BillToCustomerID: allocation.BillToCustomerID,
			Method:           allocation.Method,
			Percent:          allocation.Percent,
			Amount:           allocation.Amount,
			Sequence:         allocation.Sequence,
		}
		if allocation.ChargeKind == shipment.ChargeAllocationKindAccessorial {
			if allocation.AdditionalChargeID == nil {
				continue
			}
			copied, ok := copiedChargeBySource[*allocation.AdditionalChargeID]
			if !ok {
				continue
			}
			row.AdditionalChargeID = &copied
		}
		rows = append(rows, row)
	}

	return rows
}

// BuildAutoOrder wraps a copied shipment in its own single-leg commercial
// order, preserving the "everything has a commercial parent" invariant.
func BuildAutoOrder(entity *shipment.Shipment, orderNumber string) *order.Order {
	return &order.Order{
		ID:             pulid.MustNew("ord_"),
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		CustomerID:     entity.CustomerID,
		OwnerID:        entity.OwnerID,
		EnteredByID:    entity.EnteredByID,
		Status:         order.StatusConfirmed,
		OrderNumber:    orderNumber,
		CurrencyCode:   "USD",
		TotalAmount:    entity.TotalChargeAmount,
	}
}

func duplicateMovesAndStops(
	sourceMoves []*shipment.ShipmentMove,
	parent *shipment.Shipment,
	dateAnchor *int64,
) ([]*shipment.ShipmentMove, []*shipment.Stop) {
	moves := make([]*shipment.ShipmentMove, 0, len(sourceMoves))
	stops := make([]*shipment.Stop, 0, countMoveStops(sourceMoves))

	var offset int64
	var applyOffset bool
	if dateAnchor != nil {
		if firstStop := firstSourceStop(sourceMoves); firstStop != nil {
			offset = *dateAnchor - firstStop.ScheduledWindowStart
			applyOffset = true
		}
	}

	for _, sourceMove := range sourceMoves {
		duplicatedMove := &shipment.ShipmentMove{
			ID:             pulid.MustNew("sm_"),
			BusinessUnitID: sourceMove.BusinessUnitID,
			OrganizationID: sourceMove.OrganizationID,
			ShipmentID:     parent.ID,
			Status:         shipment.MoveStatusNew,
			Loaded:         sourceMove.Loaded,
			Sequence:       sourceMove.Sequence,
			Distance:       intutils.ClonePointer(sourceMove.Distance),
		}

		moveStops := duplicateStops(sourceMove.Stops, duplicatedMove.ID, offset, applyOffset)
		duplicatedMove.Stops = moveStops

		moves = append(moves, duplicatedMove)
		stops = append(stops, moveStops...)
	}

	return moves, stops
}

func firstSourceStop(sourceMoves []*shipment.ShipmentMove) *shipment.Stop {
	for _, move := range sourceMoves {
		if move == nil {
			continue
		}
		if len(move.Stops) > 0 {
			return move.Stops[0]
		}
	}

	return nil
}

func duplicateStops(
	sourceStops []*shipment.Stop,
	moveID pulid.ID,
	offset int64,
	applyOffset bool,
) []*shipment.Stop {
	stops := make([]*shipment.Stop, 0, len(sourceStops))

	for _, sourceStop := range sourceStops {
		scheduledWindowStart := sourceStop.ScheduledWindowStart
		scheduledWindowEnd := intutils.ClonePointer(sourceStop.ScheduledWindowEnd)
		if applyOffset {
			scheduledWindowStart += offset
			if scheduledWindowEnd != nil {
				*scheduledWindowEnd += offset
			}
		}

		stops = append(stops, &shipment.Stop{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       sourceStop.BusinessUnitID,
			OrganizationID:       sourceStop.OrganizationID,
			ShipmentMoveID:       moveID,
			LocationID:           sourceStop.LocationID,
			Status:               shipment.StopStatusNew,
			Type:                 sourceStop.Type,
			Sequence:             sourceStop.Sequence,
			Pieces:               intutils.ClonePointer(sourceStop.Pieces),
			Weight:               intutils.ClonePointer(sourceStop.Weight),
			ScheduledWindowStart: scheduledWindowStart,
			ScheduledWindowEnd:   scheduledWindowEnd,
			AddressLine:          sourceStop.AddressLine,
		})
	}

	return stops
}

func deriveDuplicateBOL(source string, copyNumber int) string {
	suffix := fmt.Sprintf("-COPY-%02d", copyNumber)
	base := strings.TrimSpace(source)
	if base == "" {
		base = "BOL"
	}

	baseRunes := []rune(base)
	suffixRunes := []rune(suffix)
	maxBaseLength := max(maxShipmentBOLLength-len(suffixRunes), 1)

	if len(baseRunes) > maxBaseLength {
		baseRunes = baseRunes[:maxBaseLength]
	}

	return string(baseRunes) + suffix
}

func duplicateAdditionalCharges(
	sourceCharges []*shipment.AdditionalCharge,
	shipmentID pulid.ID,
) []*shipment.AdditionalCharge {
	charges := make([]*shipment.AdditionalCharge, 0, len(sourceCharges))

	for _, sourceCharge := range sourceCharges {
		if sourceCharge == nil {
			continue
		}

		charges = append(charges, &shipment.AdditionalCharge{
			ID:                  pulid.MustNew("ac_"),
			BusinessUnitID:      sourceCharge.BusinessUnitID,
			OrganizationID:      sourceCharge.OrganizationID,
			ShipmentID:          shipmentID,
			AccessorialChargeID: sourceCharge.AccessorialChargeID,
			Method:              sourceCharge.Method,
			Amount:              sourceCharge.Amount,
			Unit:                sourceCharge.Unit,
		})
	}

	return charges
}

func duplicateShipmentCommodities(
	sourceCommodities []*shipment.ShipmentCommodity,
	shipmentID pulid.ID,
) []*shipment.ShipmentCommodity {
	commodities := make([]*shipment.ShipmentCommodity, 0, len(sourceCommodities))

	for _, sourceCommodity := range sourceCommodities {
		if sourceCommodity == nil {
			continue
		}

		commodities = append(commodities, &shipment.ShipmentCommodity{
			ID:             pulid.MustNew("sc_"),
			BusinessUnitID: sourceCommodity.BusinessUnitID,
			OrganizationID: sourceCommodity.OrganizationID,
			ShipmentID:     shipmentID,
			CommodityID:    sourceCommodity.CommodityID,
			Weight:         sourceCommodity.Weight,
			Pieces:         sourceCommodity.Pieces,
		})
	}

	return commodities
}

func countStops(entity *shipment.Shipment) int {
	if entity == nil {
		return 0
	}

	return countMoveStops(entity.Moves)
}

func countMoveStops(moves []*shipment.ShipmentMove) int {
	total := 0
	for _, move := range moves {
		total += len(move.Stops)
	}

	return total
}
