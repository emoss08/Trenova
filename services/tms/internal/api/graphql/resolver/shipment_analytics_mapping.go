package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func shipmentAnalyticsToModel(data services.AnalyticsData) (*gqlmodel.ShipmentAnalytics, error) {
	page := services.ShipmentAnalyticsPage
	if value, ok := data["page"].(string); ok && value != "" {
		page = services.AnalyticsPage(value)
	}
	return &gqlmodel.ShipmentAnalytics{
		Page:               string(page),
		SavedViewCounts:    savedViewCountsFromAnalytics(data["savedViewCounts"]),
		ActiveShipments:    activeShipmentsFromAnalytics(data["activeShipments"]),
		OnTimePercent:      onTimeFromAnalytics(data["onTimePercent"]),
		RevenueToday:       revenueTodayFromAnalytics(data["revenueToday"]),
		EmptyMilePercent:   emptyMileFromAnalytics(data["emptyMilePercent"]),
		AtRisk:             atRiskFromAnalytics(data["atRisk"]),
		Unassigned:         unassignedFromAnalytics(data["unassigned"]),
		ReadyToDispatch:    readyToDispatchFromAnalytics(data["readyToDispatch"]),
		DetentionWatchlist: detentionWatchlistFromAnalytics(data["detentionWatchlist"]),
		CustomerMix:        customerMixFromAnalytics(data["customerMix"]),
		TomorrowsPickups:   tomorrowsPickupsFromAnalytics(data["tomorrowsPickups"]),
		LaneHeatmap:        laneHeatmapFromAnalytics(data["laneHeatmap"]),
		Profitability:      profitabilityFromAnalytics(data["profitability"]),
	}, nil
}

func analyticsDateRange(startDate, endDate *int) *services.DateRange {
	if startDate == nil && endDate == nil {
		return nil
	}
	return &services.DateRange{
		StartDate: int64Value(startDate),
		EndDate:   int64Value(endDate),
	}
}

func savedViewCountsFromAnalytics(value any) *gqlmodel.ShipmentSavedViewCounts {
	countsMap := analyticsObject(value)
	if countsMap == nil {
		return nil
	}
	return &gqlmodel.ShipmentSavedViewCounts{
		All:             intutils.IntPtrValue(countsMap["all"]),
		Transit:         intutils.IntPtrValue(countsMap["transit"]),
		AtRisk:          intutils.IntPtrValue(countsMap["at-risk"]),
		Unassigned:      intutils.IntPtrValue(countsMap["unassigned"]),
		DeliveringToday: intutils.IntPtrValue(countsMap["delivering-today"]),
	}
}

func activeShipmentsFromAnalytics(value any) *gqlmodel.ShipmentActiveShipments {
	card := analyticsObject(value)
	if card == nil {
		return nil
	}
	return &gqlmodel.ShipmentActiveShipments{
		Count:               intutils.IntValue(card["count"]),
		ChangeFromYesterday: intutils.IntValue(card["changeFromYesterday"]),
		Sparkline:           sparklineToModel(card["sparkline"]),
		Breakdown: activeShipmentBreakdownToModel(
			card["breakdown"],
		),
	}
}

func activeShipmentBreakdownToModel(
	value any,
) *gqlmodel.ShipmentActiveShipmentsBreakdown {
	breakdown := analyticsObject(value)
	if breakdown == nil {
		return &gqlmodel.ShipmentActiveShipmentsBreakdown{}
	}
	return &gqlmodel.ShipmentActiveShipmentsBreakdown{
		InTransit: intutils.IntValue(breakdown["inTransit"]),
		AtRisk:    intutils.IntValue(breakdown["atRisk"]),
		Loading:   intutils.IntValue(breakdown["loading"]),
		Done:      intutils.IntValue(breakdown["done"]),
	}
}

func onTimeFromAnalytics(value any) *gqlmodel.ShipmentOnTime {
	card := analyticsObject(value)
	if card == nil {
		return nil
	}
	return &gqlmodel.ShipmentOnTime{
		Percent:         floatutils.FloatValue(card["percent"]),
		OnTimeCount:     intutils.IntValue(card["onTimeCount"]),
		TotalCount:      intutils.IntValue(card["totalCount"]),
		Target:          floatutils.FloatPtrValue(card["target"]),
		DeltaPp:         floatutils.FloatValue(card["deltaPp"]),
		SevenDayPercent: floatutils.FloatValue(card["sevenDayPercent"]),
	}
}

func revenueTodayFromAnalytics(value any) *gqlmodel.ShipmentRevenueToday {
	card := analyticsObject(value)
	if card == nil {
		return nil
	}
	return &gqlmodel.ShipmentRevenueToday{
		Total:     floatutils.FloatValue(card["total"]),
		Sparkline: sparklineToModel(card["sparkline"]),
		DeltaPct:  floatutils.FloatValue(card["deltaPct"]),
		Rpm:       floatutils.FloatValue(card["rpm"]),
	}
}

func sparklineToModel(value any) []*gqlmodel.ShipmentSparklinePoint {
	points := analyticsSlice(value)
	items := make([]*gqlmodel.ShipmentSparklinePoint, 0, len(points))
	for _, item := range points {
		point := analyticsObject(item)
		if point == nil {
			continue
		}
		items = append(items, &gqlmodel.ShipmentSparklinePoint{
			Hour:  sliceutils.StringValue(point["hour"]),
			Value: floatutils.FloatValue(point["value"]),
		})
	}
	return items
}

func emptyMileFromAnalytics(value any) *gqlmodel.ShipmentEmptyMile {
	card := analyticsObject(value)
	if card == nil {
		return nil
	}
	return &gqlmodel.ShipmentEmptyMile{
		Percent:    floatutils.FloatValue(card["percent"]),
		EmptyMiles: floatutils.FloatValue(card["emptyMiles"]),
		TotalMiles: floatutils.FloatValue(card["totalMiles"]),
		DeltaPp:    floatutils.FloatValue(card["deltaPp"]),
	}
}

func atRiskFromAnalytics(value any) *gqlmodel.ShipmentAtRisk {
	card := analyticsObject(value)
	if card == nil {
		return nil
	}
	return &gqlmodel.ShipmentAtRisk{
		Count:   intutils.IntValue(card["count"]),
		Delta:   intutils.IntValue(card["delta"]),
		EtaSlip: intutils.IntValue(card["etaSlip"]),
		Weather: intutils.IntValue(card["weather"]),
		Reefer:  intutils.IntValue(card["reefer"]),
	}
}

func unassignedFromAnalytics(value any) *gqlmodel.ShipmentUnassignedAnalytics {
	card := analyticsObject(value)
	if card == nil {
		return nil
	}
	return &gqlmodel.ShipmentUnassignedAnalytics{
		Count:          intutils.IntValue(card["count"]),
		Delta:          intutils.IntValue(card["delta"]),
		RevenueWaiting: floatutils.FloatValue(card["revenueWaiting"]),
	}
}

func readyToDispatchFromAnalytics(value any) *gqlmodel.ShipmentReadyToDispatch {
	card := analyticsObject(value)
	if card == nil {
		return nil
	}
	return &gqlmodel.ShipmentReadyToDispatch{
		Count:       intutils.IntValue(card["count"]),
		Delta:       intutils.IntValue(card["delta"]),
		Unassigned:  intutils.IntValue(card["unassigned"]),
		DriverReady: intutils.IntValue(card["driverReady"]),
	}
}

func detentionWatchlistFromAnalytics(value any) *gqlmodel.ShipmentDetentionWatchlist {
	card := analyticsObject(value)
	if card == nil {
		return nil
	}
	sourceItems := analyticsSlice(card["items"])
	items := make([]*gqlmodel.ShipmentDetentionWatchlistItem, 0, len(sourceItems))
	for _, sourceItem := range sourceItems {
		item := analyticsObject(sourceItem)
		if item == nil {
			continue
		}
		items = append(items, &gqlmodel.ShipmentDetentionWatchlistItem{
			ShipmentID: sliceutils.StringValue(item["shipmentId"]),
			Customer:   sliceutils.StringValue(item["customer"]),
			DwellLabel: sliceutils.StringValue(item["dwellLabel"]),
			Tone:       sliceutils.StringValue(item["tone"]),
		})
	}
	return &gqlmodel.ShipmentDetentionWatchlist{Items: items}
}

func customerMixFromAnalytics(value any) *gqlmodel.ShipmentCustomerMix {
	card := analyticsObject(value)
	if card == nil {
		return nil
	}
	sourceEntries := analyticsSlice(card["entries"])
	entries := make([]*gqlmodel.ShipmentCustomerMixEntry, 0, len(sourceEntries))
	for _, sourceEntry := range sourceEntries {
		entry := analyticsObject(sourceEntry)
		if entry == nil {
			continue
		}
		entries = append(entries, &gqlmodel.ShipmentCustomerMixEntry{
			CustomerID: sliceutils.StringValue(entry["customerId"]),
			Name:       sliceutils.StringValue(entry["name"]),
			Revenue:    floatutils.FloatValue(entry["revenue"]),
			Share:      floatutils.FloatValue(entry["share"]),
			Loads:      intutils.IntValue(entry["loads"]),
			Trend:      floatutils.FloatValue(entry["trend"]),
		})
	}
	return &gqlmodel.ShipmentCustomerMix{
		WindowDays: intutils.IntValue(card["windowDays"]),
		Entries:    entries,
	}
}

func tomorrowsPickupsFromAnalytics(value any) *gqlmodel.ShipmentTomorrowsPickups {
	card := analyticsObject(value)
	if card == nil {
		return nil
	}
	sourcePickups := analyticsSlice(card["pickups"])
	pickups := make([]*gqlmodel.ShipmentTomorrowPickup, 0, len(sourcePickups))
	for _, sourcePickup := range sourcePickups {
		pickup := analyticsObject(sourcePickup)
		if pickup == nil {
			continue
		}
		pickups = append(pickups, &gqlmodel.ShipmentTomorrowPickup{
			ShipmentID:        sliceutils.StringValue(pickup["shipmentId"]),
			ProNumber:         sliceutils.StringValue(pickup["proNumber"]),
			PickupWindowStart: intutils.IntValue(pickup["pickupWindowStart"]),
			Customer:          sliceutils.StringValue(pickup["customer"]),
			Origin:            sliceutils.StringValue(pickup["origin"]),
			Destination:       sliceutils.StringValue(pickup["destination"]),
			Driver:            sliceutils.StringValue(pickup["driver"]),
			Status:            sliceutils.StringValue(pickup["status"]),
		})
	}
	return &gqlmodel.ShipmentTomorrowsPickups{
		Date:    sliceutils.StringValue(card["date"]),
		Pickups: pickups,
	}
}

func laneHeatmapFromAnalytics(value any) *gqlmodel.ShipmentLaneHeatmap {
	card := analyticsObject(value)
	if card == nil {
		return nil
	}
	sourceCells := analyticsSlice(card["cells"])
	cells := make([]*gqlmodel.ShipmentLaneHeatmapCell, 0, len(sourceCells))
	for _, sourceCell := range sourceCells {
		cell := analyticsObject(sourceCell)
		if cell == nil {
			continue
		}
		cells = append(cells, &gqlmodel.ShipmentLaneHeatmapCell{
			Origin:      sliceutils.StringValue(cell["origin"]),
			Destination: sliceutils.StringValue(cell["destination"]),
			Count:       intutils.IntValue(cell["count"]),
		})
	}
	return &gqlmodel.ShipmentLaneHeatmap{
		WindowDays: intutils.IntValue(card["windowDays"]),
		Cells:      cells,
		Total:      intutils.IntValue(card["total"]),
	}
}

func profitabilityFromAnalytics(value any) *gqlmodel.ShipmentProfitabilityAnalytics {
	card := analyticsObject(value)
	if card == nil {
		return nil
	}
	hasMargin, _ := card["hasMargin"].(bool)
	return &gqlmodel.ShipmentProfitabilityAnalytics{
		AvgCpm:            floatutils.FloatValue(card["avgCpm"]),
		AvgMarginPct:      floatutils.FloatValue(card["avgMarginPct"]),
		HasMargin:         hasMargin,
		UnprofitableCount: intutils.IntValue(card["unprofitableCount"]),
		ShipmentCount:     intutils.IntValue(card["shipmentCount"]),
		TotalMiles:        floatutils.FloatValue(card["totalMiles"]),
	}
}

func analyticsObject(value any) map[string]any {
	if value == nil {
		return nil
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	object, err := optionalJSON(value)
	if err != nil {
		return nil
	}
	return object
}

func analyticsSlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case nil:
		return nil
	default:
		return nil
	}
}
