package shipmentrepository

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const maxShipmentBOLLength = 100

type duplicatedShipmentGraph struct {
	shipments         []*shipment.Shipment
	moves             []*shipment.ShipmentMove
	stops             []*shipment.Stop
	additionalCharges []*shipment.AdditionalCharge
	commodities       []*shipment.ShipmentCommodity
}

func buildDuplicatedShipmentGraph(
	source *shipment.Shipment,
	proNumbers []string,
	overrideDates bool,
	requestedBy pulid.ID,
) *duplicatedShipmentGraph {
	graph := &duplicatedShipmentGraph{
		shipments: make([]*shipment.Shipment, 0, len(proNumbers)),
		moves:     make([]*shipment.ShipmentMove, 0, len(source.Moves)*len(proNumbers)),
		stops:     make([]*shipment.Stop, 0, countStops(source)*len(proNumbers)),
		additionalCharges: make(
			[]*shipment.AdditionalCharge,
			0,
			len(source.AdditionalCharges)*len(proNumbers),
		),
		commodities: make(
			[]*shipment.ShipmentCommodity,
			0,
			len(source.Commodities)*len(proNumbers),
		),
	}

	for idx, proNumber := range proNumbers {
		duplicatedShipment := duplicateShipment(source, proNumber, idx+1, requestedBy)
		graph.shipments = append(graph.shipments, duplicatedShipment)

		duplicatedMoves, duplicatedStops := duplicateMovesAndStops(
			source.Moves,
			duplicatedShipment,
			overrideDates,
		)
		duplicatedShipment.Moves = duplicatedMoves
		duplicatedCharges := duplicateAdditionalCharges(
			source.AdditionalCharges,
			duplicatedShipment.ID,
		)
		duplicatedShipment.AdditionalCharges = duplicatedCharges
		duplicatedCommodities := duplicateShipmentCommodities(
			source.Commodities,
			duplicatedShipment.ID,
		)
		duplicatedShipment.Commodities = duplicatedCommodities
		graph.moves = append(graph.moves, duplicatedMoves...)
		graph.stops = append(graph.stops, duplicatedStops...)
		graph.additionalCharges = append(graph.additionalCharges, duplicatedCharges...)
		graph.commodities = append(graph.commodities, duplicatedCommodities...)
	}

	return graph
}

func duplicateShipment(
	source *shipment.Shipment,
	proNumber string,
	copyNumber int,
	requestedBy pulid.ID,
) *shipment.Shipment {
	duplicated := &shipment.Shipment{
		ID:                  pulid.MustNew("shp_"),
		BusinessUnitID:      source.BusinessUnitID,
		OrganizationID:      source.OrganizationID,
		ServiceTypeID:       source.ServiceTypeID,
		ShipmentTypeID:      source.ShipmentTypeID,
		CustomerID:          source.CustomerID,
		TractorTypeID:       source.TractorTypeID,
		TrailerTypeID:       source.TrailerTypeID,
		FormulaTemplateID:   source.FormulaTemplateID,
		Status:              shipment.StatusNew,
		ProNumber:           proNumber,
		BOL:                 deriveDuplicateBOL(source.BOL, copyNumber),
		OtherChargeAmount:   source.OtherChargeAmount,
		FreightChargeAmount: source.FreightChargeAmount,
		TotalChargeAmount:   source.TotalChargeAmount,
		Pieces:              intutils.ClonePointer(source.Pieces),
		Weight:              intutils.ClonePointer(source.Weight),
		TemperatureMin:      intutils.ClonePointer(source.TemperatureMin),
		TemperatureMax:      intutils.ClonePointer(source.TemperatureMax),
		RatingUnit:          source.RatingUnit,
		EnteredByID:         requestedBy,
	}

	return duplicated
}

func duplicateMovesAndStops(
	sourceMoves []*shipment.ShipmentMove,
	parent *shipment.Shipment,
	overrideDates bool,
) ([]*shipment.ShipmentMove, []*shipment.Stop) {
	moves := make([]*shipment.ShipmentMove, 0, len(sourceMoves))
	stops := make([]*shipment.Stop, 0, countMoveStops(sourceMoves))

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

		moveStops := duplicateStops(sourceMove.Stops, duplicatedMove.ID, overrideDates)
		duplicatedMove.Stops = moveStops

		moves = append(moves, duplicatedMove)
		stops = append(stops, moveStops...)
	}

	return moves, stops
}

func duplicateStops(
	sourceStops []*shipment.Stop,
	moveID pulid.ID,
	overrideDates bool,
) []*shipment.Stop {
	stops := make([]*shipment.Stop, 0, len(sourceStops))

	var offset int64
	if overrideDates && len(sourceStops) > 0 {
		offset = timeutils.NowUnix() - sourceStops[0].ScheduledWindowStart
	}

	for _, sourceStop := range sourceStops {
		scheduledWindowStart := sourceStop.ScheduledWindowStart
		scheduledWindowEnd := intutils.ClonePointer(sourceStop.ScheduledWindowEnd)
		if overrideDates {
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
