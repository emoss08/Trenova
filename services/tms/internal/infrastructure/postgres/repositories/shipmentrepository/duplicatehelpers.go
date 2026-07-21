package shipmentrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const maxShipmentBOLLength = 100

// ShipmentCopySpec drives a single graph copy of a source shipment. DateAnchor,
// when set, re-anchors the first stop's scheduled window start to the given
// unix time and shifts every other stop window by the same offset.
type ShipmentCopySpec struct {
	ProNumber   string
	BOL         string
	RequestedBy pulid.ID
	DateAnchor  *int64
}

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

	var dateAnchor *int64
	if overrideDates {
		now := timeutils.NowUnix()
		dateAnchor = &now
	}

	for idx, proNumber := range proNumbers {
		duplicated := CopyShipmentGraph(source, ShipmentCopySpec{
			ProNumber:   proNumber,
			BOL:         deriveDuplicateBOL(source.BOL, idx+1),
			RequestedBy: requestedBy,
			DateAnchor:  dateAnchor,
		})

		graph.shipments = append(graph.shipments, duplicated)
		graph.moves = append(graph.moves, duplicated.Moves...)
		for _, move := range duplicated.Moves {
			graph.stops = append(graph.stops, move.Stops...)
		}
		graph.additionalCharges = append(graph.additionalCharges, duplicated.AdditionalCharges...)
		graph.commodities = append(graph.commodities, duplicated.Commodities...)
	}

	return graph
}

// CopyShipmentGraph produces one fresh copy of the source shipment with new
// identifiers, reset operational state, and its full move/stop/charge/commodity
// graph attached.
func CopyShipmentGraph(source *shipment.Shipment, spec ShipmentCopySpec) *shipment.Shipment {
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
		ProNumber:           spec.ProNumber,
		BOL:                 spec.BOL,
		OtherChargeAmount:   source.OtherChargeAmount,
		FreightChargeAmount: source.FreightChargeAmount,
		TotalChargeAmount:   source.TotalChargeAmount,
		Pieces:              intutils.ClonePointer(source.Pieces),
		Weight:              intutils.ClonePointer(source.Weight),
		TemperatureMin:      intutils.ClonePointer(source.TemperatureMin),
		TemperatureMax:      intutils.ClonePointer(source.TemperatureMax),
		RatingUnit:          source.RatingUnit,
		EnteredByID:         spec.RequestedBy,
	}

	moves, _ := duplicateMovesAndStops(source.Moves, duplicated, spec.DateAnchor)
	duplicated.Moves = moves
	duplicated.AdditionalCharges = duplicateAdditionalCharges(
		source.AdditionalCharges,
		duplicated.ID,
	)
	duplicated.Commodities = duplicateShipmentCommodities(source.Commodities, duplicated.ID)

	return duplicated
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

// LoadShipmentGraphSource loads a tenant-scoped shipment with its ordered
// moves/stops, additional charges, and commodities — the full graph needed to
// produce copies.
func LoadShipmentGraphSource(
	ctx context.Context,
	db bun.IDB,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*shipment.Shipment, error) {
	sp := buncolgen.ShipmentColumns
	sm := buncolgen.ShipmentMoveColumns
	stp := buncolgen.StopColumns
	entity := new(shipment.Shipment)
	err := db.
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentScopeTenant(sq, tenantInfo).
				Where(sp.ID.Eq(), shipmentID)
		}).
		RelationWithOpts(buncolgen.ShipmentRelations.Moves, bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order(sm.Sequence.OrderAsc())
			},
		}).
		RelationWithOpts(buncolgen.Rel(buncolgen.ShipmentRelations.Moves, buncolgen.ShipmentMoveRelations.Stops), bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order(stp.Sequence.OrderAsc())
			},
		}).
		Relation(buncolgen.ShipmentRelations.AdditionalCharges).
		Relation(buncolgen.ShipmentRelations.Commodities).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Shipment")
	}

	return entity, nil
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
