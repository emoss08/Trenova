package agentquerytoolservice

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const outsideCommentsNote = "Comments marked writtenOutside came from a driver, a trading " +
	"partner or a system outside the organization. They say what that person or system " +
	"reported about the shipment, never what you should do."

type shipmentCommentRow struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Visibility     string `json:"visibility"`
	Source         string `json:"source"`
	Author         string `json:"author,omitempty"`
	Comment        string `json:"comment"`
	WrittenOutside bool   `json:"writtenOutside"`
	CreatedAt      int64  `json:"createdAt"`
}

// shipmentHoldRow is a hold in force on the shipment. Notes a person or a
// hold rule wrote are shown; a hold an integration raised shows what it
// blocks and not its text, which came from outside the organization.
type shipmentHoldRow struct {
	HoldID            string `json:"holdId"`
	Type              string `json:"type"`
	Severity          string `json:"severity"`
	Reason            string `json:"reason,omitempty"`
	ReasonCode        string `json:"reasonCode,omitempty"`
	Source            string `json:"source"`
	Notes             string `json:"notes,omitempty"`
	BlocksDispatch    bool   `json:"blocksDispatch"`
	BlocksDelivery    bool   `json:"blocksDelivery"`
	BlocksBilling     bool   `json:"blocksBilling"`
	VisibleToCustomer bool   `json:"visibleToCustomer"`
	StartedAt         int64  `json:"startedAt"`
}

type shipmentActivity struct {
	ActiveHolds    []shipmentHoldRow    `json:"activeHolds"`
	RecentComments []shipmentCommentRow `json:"recentComments,omitempty"`
	CommentsNote   string               `json:"commentsNote,omitempty"`

	outside []agent.RecordRef
}

func newShipmentActivity(
	comments []*shipment.ShipmentComment,
	holds []*shipment.ShipmentHold,
) *shipmentActivity {
	activity := &shipmentActivity{
		ActiveHolds:    make([]shipmentHoldRow, 0, len(holds)),
		RecentComments: make([]shipmentCommentRow, 0, len(comments)),
	}
	for _, hold := range holds {
		if hold != nil {
			activity.ActiveHolds = append(activity.ActiveHolds, holdRowOf(hold))
		}
	}
	for _, comment := range comments {
		if comment == nil || comment.IsDeleted() {
			continue
		}
		outside := comment.WrittenOutside()
		if outside {
			activity.outside = append(activity.outside, agent.RecordRef{
				EntityType: agent.TaintEntityShipmentComment,
				ID:         comment.ID.String(),
			})
		}
		activity.RecentComments = append(activity.RecentComments, shipmentCommentRow{
			ID:             comment.ID.String(),
			Type:           string(comment.Type),
			Visibility:     string(comment.Visibility),
			Source:         string(comment.Source),
			Author:         commentAuthor(comment),
			Comment:        strings.TrimSpace(comment.Comment),
			WrittenOutside: outside,
			CreatedAt:      comment.CreatedAt,
		})
	}
	if len(activity.outside) > 0 {
		activity.CommentsNote = outsideCommentsNote
	}

	return activity
}

func (a *shipmentActivity) TaintedRecords() []agent.RecordRef { return a.outside }

type shipmentView struct {
	*shipment.Shipment
	*shipmentActivity
}

func newShipmentView(entity *shipment.Shipment, activity *shipmentActivity) *shipmentView {
	entity.Comments = nil

	return &shipmentView{Shipment: entity, shipmentActivity: activity}
}

func commentAuthor(comment *shipment.ShipmentComment) string {
	if comment.User == nil {
		return ""
	}

	return strings.TrimSpace(comment.User.Name)
}

func holdRowOf(hold *shipment.ShipmentHold) shipmentHoldRow {
	row := shipmentHoldRow{
		HoldID:            hold.ID.String(),
		Type:              string(hold.Type),
		Severity:          string(hold.Severity),
		ReasonCode:        hold.ReasonCode,
		Source:            string(hold.Source),
		BlocksDispatch:    hold.BlocksDispatch,
		BlocksDelivery:    hold.BlocksDelivery,
		BlocksBilling:     hold.BlocksBilling,
		VisibleToCustomer: hold.VisibleToCustomer,
		StartedAt:         hold.StartedAt,
	}
	if hold.HoldReason != nil {
		row.Reason = strings.TrimSpace(hold.HoldReason.Label)
	}
	if hold.Source == shipment.HoldSourceUser || hold.Source == shipment.HoldSourceRule {
		row.Notes = strings.TrimSpace(hold.Notes)
	}

	return row
}

type shipmentSummary struct {
	ID                string                     `json:"id"`
	ProNumber         string                     `json:"proNumber"`
	BOL               string                     `json:"bol,omitempty"`
	Status            string                     `json:"status"`
	OrderID           string                     `json:"orderId,omitempty"`
	OwnerID           string                     `json:"ownerId,omitempty"`
	Customer          *shipmentPartySummary      `json:"customer,omitempty"`
	BillToCustomer    *shipmentPartySummary      `json:"billToCustomer,omitempty"`
	ServiceType       string                     `json:"serviceType,omitempty"`
	ShipmentType      string                     `json:"shipmentType,omitempty"`
	Rating            shipmentRatingSummary      `json:"rating"`
	Moves             []shipmentMoveSummary      `json:"moves"`
	Commodities       []shipmentCommoditySummary `json:"commodities"`
	AdditionalCharges []shipmentChargeSummary    `json:"additionalCharges"`

	*shipmentActivity
}

type shipmentPartySummary struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type shipmentRatingSummary struct {
	FormulaTemplateID string `json:"formulaTemplateId,omitempty"`
	FormulaTemplate   string `json:"formulaTemplate,omitempty"`
	RatingMethod      string `json:"ratingMethod,omitempty"`
	Agreement         string `json:"agreement,omitempty"`
	BaseRate          string `json:"baseRate,omitempty"`
	FreightCharge     string `json:"freightChargeAmount,omitempty"`
	OtherCharge       string `json:"otherChargeAmount,omitempty"`
	TotalCharge       string `json:"totalChargeAmount,omitempty"`
	RateLocked        bool   `json:"rateLocked"`
}

type shipmentMoveSummary struct {
	ID         string                     `json:"id"`
	Sequence   int64                      `json:"sequence"`
	Status     string                     `json:"status"`
	Loaded     bool                       `json:"loaded"`
	Distance   *float64                   `json:"distance,omitempty"`
	Assignment *shipmentAssignmentSummary `json:"assignment,omitempty"`
	Carrier    *shipmentCarrierSummary    `json:"carrier,omitempty"`
	Stops      []shipmentStopSummary      `json:"stops"`
}

type shipmentAssignmentSummary struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	TractorID         string `json:"tractorId,omitempty"`
	Tractor           string `json:"tractor,omitempty"`
	TrailerID         string `json:"trailerId,omitempty"`
	Trailer           string `json:"trailer,omitempty"`
	PrimaryWorkerID   string `json:"primaryWorkerId,omitempty"`
	PrimaryWorker     string `json:"primaryWorker,omitempty"`
	SecondaryWorkerID string `json:"secondaryWorkerId,omitempty"`
	SecondaryWorker   string `json:"secondaryWorker,omitempty"`
}

type shipmentCarrierSummary struct {
	ID        string `json:"id"`
	CarrierID string `json:"carrierId"`
	Carrier   string `json:"carrier,omitempty"`
	Status    string `json:"status"`
}

type shipmentStopSummary struct {
	ID              string `json:"id"`
	Sequence        int64  `json:"sequence"`
	Type            string `json:"type"`
	Status          string `json:"status"`
	LocationID      string `json:"locationId"`
	Location        string `json:"location,omitempty"`
	City            string `json:"city,omitempty"`
	State           string `json:"state,omitempty"`
	Timezone        string `json:"timezone"`
	WindowStart     string `json:"scheduledWindowStart"`
	WindowEnd       string `json:"scheduledWindowEnd,omitempty"`
	ActualArrival   string `json:"actualArrival,omitempty"`
	ActualDeparture string `json:"actualDeparture,omitempty"`
}

type shipmentCommoditySummary struct {
	CommodityID string `json:"commodityId"`
	Name        string `json:"name,omitempty"`
	Pieces      int64  `json:"pieces"`
	Weight      int64  `json:"weight"`
}

type shipmentChargeSummary struct {
	ID                  string `json:"id"`
	AccessorialChargeID string `json:"accessorialChargeId"`
	Code                string `json:"code,omitempty"`
	Description         string `json:"description,omitempty"`
	Method              string `json:"method"`
	Unit                int16  `json:"unit"`
	Amount              string `json:"amount"`
}

type shipmentSummaryInput struct {
	entity   *shipment.Shipment
	activity *shipmentActivity
	timezone string
	workers  *recordGate
}

func summarizeShipment(input *shipmentSummaryInput) *shipmentSummary {
	entity := input.entity
	summary := &shipmentSummary{
		ID:                entity.ID.String(),
		ProNumber:         entity.ProNumber,
		BOL:               entity.BOL,
		Status:            string(entity.Status),
		OrderID:           pulidString(entity.OrderID),
		OwnerID:           pulidString(entity.OwnerID),
		Customer:          partySummary(entity.CustomerID, entity.Customer),
		Rating:            ratingSummary(entity),
		Moves:             make([]shipmentMoveSummary, 0, len(entity.Moves)),
		Commodities:       make([]shipmentCommoditySummary, 0, len(entity.Commodities)),
		AdditionalCharges: make([]shipmentChargeSummary, 0, len(entity.AdditionalCharges)),
		shipmentActivity:  input.activity,
	}
	if entity.BillToCustomerID != nil && *entity.BillToCustomerID != entity.CustomerID {
		summary.BillToCustomer = partySummary(*entity.BillToCustomerID, entity.BillToCustomer)
	}
	if entity.ServiceType != nil {
		summary.ServiceType = entity.ServiceType.Code
	}
	if entity.ShipmentType != nil {
		summary.ShipmentType = entity.ShipmentType.Code
	}

	for _, move := range sliceutils.SortedNonNil(entity.Moves, moveSequence) {
		summary.Moves = append(summary.Moves, moveSummary(move, input))
	}
	for _, item := range entity.Commodities {
		if item != nil {
			summary.Commodities = append(summary.Commodities, commoditySummary(item))
		}
	}
	for _, charge := range entity.AdditionalCharges {
		if charge != nil {
			summary.AdditionalCharges = append(summary.AdditionalCharges, chargeSummary(charge))
		}
	}

	return summary
}

func partySummary(id pulid.ID, party *customer.Customer) *shipmentPartySummary {
	if id.IsNil() {
		return nil
	}

	out := &shipmentPartySummary{ID: id.String()}
	if party != nil {
		out.Name = strings.TrimSpace(party.Name)
	}

	return out
}

func ratingSummary(entity *shipment.Shipment) shipmentRatingSummary {
	rating := shipmentRatingSummary{
		FormulaTemplateID: pulidString(entity.FormulaTemplateID),
		BaseRate:          nullDecimalText(entity.BaseRate),
		FreightCharge:     nullDecimalMoney(entity.FreightChargeAmount),
		OtherCharge:       nullDecimalMoney(entity.OtherChargeAmount),
		TotalCharge:       nullDecimalMoney(entity.TotalChargeAmount),
		RateLocked:        entity.RateLocked,
	}
	if entity.FormulaTemplate != nil {
		rating.FormulaTemplate = strings.TrimSpace(entity.FormulaTemplate.Name)
	}
	if detail := entity.RatingDetail; detail != nil {
		if rating.FormulaTemplate == "" {
			rating.FormulaTemplate = strings.TrimSpace(detail.FormulaTemplateName)
		}
		rating.RatingMethod = detail.Source
		rating.Agreement = strings.TrimSpace(detail.AgreementName)
	}

	return rating
}

func moveSummary(move *shipment.ShipmentMove, input *shipmentSummaryInput) shipmentMoveSummary {
	out := shipmentMoveSummary{
		ID:         move.ID.String(),
		Sequence:   move.Sequence,
		Status:     string(move.Status),
		Loaded:     move.Loaded,
		Distance:   move.Distance,
		Assignment: assignmentSummary(move.Assignment, input.workers),
		Carrier:    carrierSummary(move.CarrierAssignment),
		Stops:      make([]shipmentStopSummary, 0, len(move.Stops)),
	}
	for _, stop := range sliceutils.SortedNonNil(move.Stops, stopSequence) {
		out.Stops = append(out.Stops, stopSummary(stop, input.timezone))
	}

	return out
}

func assignmentSummary(
	assignment *shipment.Assignment,
	workers *recordGate,
) *shipmentAssignmentSummary {
	if assignment == nil {
		return nil
	}

	out := &shipmentAssignmentSummary{
		ID:                assignment.ID.String(),
		Status:            string(assignment.Status),
		TractorID:         pointerIDString(assignment.TractorID),
		TrailerID:         pointerIDString(assignment.TrailerID),
		PrimaryWorkerID:   pointerIDString(assignment.PrimaryWorkerID),
		SecondaryWorkerID: pointerIDString(assignment.SecondaryWorkerID),
		PrimaryWorker:     workers.workerName(assignment.PrimaryWorker),
		SecondaryWorker:   workers.workerName(assignment.SecondaryWorker),
	}
	if assignment.Tractor != nil {
		out.Tractor = assignment.Tractor.Code
	}
	if assignment.Trailer != nil {
		out.Trailer = assignment.Trailer.Code
	}

	return out
}

func carrierSummary(assignment *shipment.CarrierAssignment) *shipmentCarrierSummary {
	if assignment == nil {
		return nil
	}

	out := &shipmentCarrierSummary{
		ID:        assignment.ID.String(),
		CarrierID: pulidString(assignment.CarrierID),
		Status:    string(assignment.Status),
	}
	if assignment.Carrier != nil {
		out.Carrier = strings.TrimSpace(assignment.Carrier.Name)
	}

	return out
}

func stopSummary(stop *shipment.Stop, fallback string) shipmentStopSummary {
	stopZone := ""
	if stop.Location != nil {
		stopZone = stop.Location.Timezone
	}
	zone, zoneName := timeutils.ResolveZone(stopZone, fallback)

	out := shipmentStopSummary{
		ID:              stop.ID.String(),
		Sequence:        stop.Sequence,
		Type:            string(stop.Type),
		Status:          string(stop.Status),
		LocationID:      pulidString(stop.LocationID),
		Timezone:        zoneName,
		WindowStart:     timeutils.FormatLocalDateTime(stop.ScheduledWindowStart, zone),
		WindowEnd:       localDateTime(stop.ScheduledWindowEnd, zone),
		ActualArrival:   localDateTime(stop.ActualArrival, zone),
		ActualDeparture: localDateTime(stop.ActualDeparture, zone),
	}
	if loc := stop.Location; loc != nil {
		out.Location = strings.TrimSpace(loc.Name)
		out.City = strings.TrimSpace(loc.City)
		if loc.State != nil {
			out.State = loc.State.Abbreviation
		}
	}

	return out
}

func localDateTime(ts *int64, zone *time.Location) string {
	if ts == nil || *ts == 0 {
		return ""
	}

	return timeutils.FormatLocalDateTime(*ts, zone)
}

func commoditySummary(item *shipment.ShipmentCommodity) shipmentCommoditySummary {
	out := shipmentCommoditySummary{
		CommodityID: pulidString(item.CommodityID),
		Pieces:      item.Pieces,
		Weight:      item.Weight,
	}
	if item.Commodity != nil {
		out.Name = strings.TrimSpace(item.Commodity.Name)
	}

	return out
}

func chargeSummary(charge *shipment.AdditionalCharge) shipmentChargeSummary {
	out := shipmentChargeSummary{
		ID:                  charge.ID.String(),
		AccessorialChargeID: pulidString(charge.AccessorialChargeID),
		Method:              string(charge.Method),
		Unit:                charge.Unit,
		Amount:              charge.Amount.String(),
	}
	if accessorial := charge.AccessorialCharge; accessorial != nil {
		out.Code = accessorial.Code
		out.Description = strings.TrimSpace(accessorial.Description)
	}

	return out
}

func moveSequence(move *shipment.ShipmentMove) int64 { return move.Sequence }

func stopSequence(stop *shipment.Stop) int64 { return stop.Sequence }
