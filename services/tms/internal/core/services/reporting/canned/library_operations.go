package canned

import (
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

func shipmentVolumeByStatus() *Entry {
	return &Entry{
		Key:     "shipment-volume-by-status",
		Version: initialVersion,
		Name:    "Shipment Volume by Status",
		Description: "Where the book of business is sitting — counts, revenue, and " +
			"freight profile per lifecycle status",
		Category:      categoryOperations,
		Tags:          []string{"shipments", tagOperations, "pipeline"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				dimCol(statusFieldKey, "Status", statusFieldKey),
				dimCol("service_type", "Service Type", codeField, serviceTypeEdge),
				countMeasure(shipmentCountColumnID, "Shipments"),
				sumMoney(totalRevenueColumnID, "Total Revenue", totalChargeFieldKey),
				avgMoney("avg_revenue", "Average Revenue per Shipment", totalChargeFieldKey),
				sumWeight(),
				sumPieces(),
				sumMiles(totalMilesColumnID, totalMilesLabel, movesEdge),
				timestampMeasure("oldest_shipment", "Oldest Shipment",
					reportcatalog.AggMin, createdAtFieldKey),
				timestampMeasure("newest_shipment", "Newest Shipment",
					reportcatalog.AggMax, createdAtFieldKey),
			},
			Filters:    andFilters(windowFilter(createdAtFieldKey)),
			Sort:       []report.SortSpec{desc(shipmentCountColumnID)},
			Parameters: []report.ParameterDef{windowParam(30)},
		},
	}
}

func facilityActivity() *Entry {
	return &Entry{
		Key:     "facility-activity",
		Version: initialVersion,
		Name:    "Facility Activity",
		Description: "Throughput by facility — stops handled, pieces, weight, and the " +
			"average time trucks spend on site",
		Category:      categoryOperations,
		Tags:          []string{"stops", "facilities", "throughput"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityStop,
			Columns: []report.ColumnSpec{
				dimCol("facility", "Facility", nameField, locationEdge),
				dimCol("city", "City", "city", locationEdge),
				dimCol("facility_type", "Facility Type", "facilityType",
					locationEdge, locationCategEdge),
				dimCol("stop_type", "Stop Type", "type"),
				countMeasure(stopCountColumnID, "Stops"),
				sumPieces(),
				sumWeight(),
				timestampMeasure("avg_arrival", "Average Arrival",
					reportcatalog.AggAvg, "actualArrival"),
				timestampMeasure("avg_departure", "Average Departure",
					reportcatalog.AggAvg, "actualDeparture"),
				difference("avg_dwell", "Average Dwell",
					"avg_departure", "avg_arrival", dwell()),
			},
			Filters: andFilters(
				windowFilter("actualArrival"),
				isNotNull("actualDeparture"),
				inParam("type", "stopTypes"),
			),
			Sort: []report.SortSpec{desc(stopCountColumnID)},
			Parameters: []report.ParameterDef{
				windowParam(30),
				enumParam("stopTypes", "Stop Types",
					[]any{"Pickup", "Delivery"},
					[]string{"Pickup", "Delivery", "SplitPickup", "SplitDelivery"},
				),
			},
		},
	}
}

func stopDwellAndDetention() *Entry {
	return &Entry{
		Key:     "stop-dwell-and-detention",
		Version: initialVersion,
		Name:    "Dwell & Detention Exposure",
		Description: "Where drivers are losing hours — average dwell on site and how far " +
			"arrivals drift from the scheduled window, by facility",
		Category:      categoryOperations,
		Tags:          []string{"detention", "dwell", "facilities", tagDrivers},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityStop,
			Columns: []report.ColumnSpec{
				dimCol("facility", "Facility", nameField, locationEdge),
				dimCol("city", "City", "city", locationEdge),
				dimCol("stop_type", "Stop Type", "type"),
				countMeasure(stopCountColumnID, "Stops"),
				timestampMeasure("avg_scheduled_end", "Average Window Close",
					reportcatalog.AggAvg, "scheduledWindowEnd"),
				timestampMeasure("avg_arrival", "Average Arrival",
					reportcatalog.AggAvg, "actualArrival"),
				timestampMeasure("avg_departure", "Average Departure",
					reportcatalog.AggAvg, "actualDeparture"),
				difference("avg_dwell", "Average Dwell",
					"avg_departure", "avg_arrival", dwell()),
				difference("avg_window_variance", "Average Arrival vs Window Close",
					"avg_arrival", "avg_scheduled_end", dwell()),
			},
			Filters: andFilters(
				windowFilter("actualArrival"),
				isNotNull("actualDeparture"),
				isNotNull("scheduledWindowEnd"),
			),
			Sort:       []report.SortSpec{desc("avg_dwell")},
			Parameters: []report.ParameterDef{windowParam(30)},
		},
	}
}

func hazmatShipmentLog() *Entry {
	return &Entry{
		Key:     "hazmat-shipment-log",
		Version: initialVersion,
		Name:    "Hazmat Manifest Log",
		Description: "Line-level record of hazardous freight moved — UN number, class, " +
			"packing group, and placarding, by shipment",
		Category:      categoryCompliance,
		Tags:          []string{"hazmat", "safety", tagCompliance},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipmentCommodity,
			Columns: []report.ColumnSpec{
				dimCol("pro_number", "PRO Number", "proNumber", shipmentEdge),
				dimCol("bol", "BOL", "bol", shipmentEdge),
				dimCol(customerColumnID, "Customer", nameField, shipmentEdge, customerEdge),
				dimCol("shipment_status", "Shipment Status", statusFieldKey, shipmentEdge),
				dateDim("ship_date", "Ship Date", "actualShipDate", shipmentEdge),
				dimCol("commodity", "Commodity", nameField, commodityEdge),
				dimCol("freight_class", "Freight Class", "freightClass", commodityEdge),
				dimCol("hazmat_class", "Hazard Class", "class", commodityEdge, hazmatEdge),
				dimCol("un_number", "UN Number", "unNumber", commodityEdge, hazmatEdge),
				dimCol("packing_group", "Packing Group", "packingGroup",
					commodityEdge, hazmatEdge),
				dimCol("shipping_name", "Proper Shipping Name", "properShippingName",
					commodityEdge, hazmatEdge),
				styledDim(
					"placard_required",
					"Placard Required",
					"placardRequired",
					yesNo(),
					commodityEdge,
					hazmatEdge,
				),
				dimCol("weight", "Weight", weightField),
				dimCol("pieces", "Pieces", piecesField),
			},
			Filters: andFilters(
				windowFilter(createdAtFieldKey, shipmentEdge),
				isNotNull("hazardousMaterialId", commodityEdge),
			),
			Sort:       []report.SortSpec{desc("ship_date")},
			Parameters: []report.ParameterDef{windowParam(90)},
		},
	}
}

// lanePerformance is the report the pick edges exist for. Origin and
// destination are particular stops of a shipment, so before lane accessors the
// only path to them crossed a to-many edge and multiplied every measure by the
// stop count. Grouping on two picked rows keeps the money honest.
func lanePerformance() *Entry {
	return &Entry{
		Key:     "lane-performance",
		Version: initialVersion,
		Name:    "Lane Performance",
		Description: "Revenue, volume, and miles by origin and destination city — " +
			"which lanes carry the book and what they earn",
		Category:      categoryOperations,
		Tags:          []string{"lanes", tagOperations, "network", "revenue"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				dimCol("origin_city", "Origin", cityField, originStopEdge, locationEdge),
				dimCol("dest_city", "Destination", cityField, destinationStopEdge, locationEdge),
				countMeasure(shipmentCountColumnID, "Shipments"),
				sumMoney(totalRevenueColumnID, "Total Revenue", totalChargeFieldKey),
				avgMoney("avg_revenue", "Average Revenue per Shipment", totalChargeFieldKey),
				sumMiles(totalMilesColumnID, totalMilesLabel, movesEdge),
			},
			Filters:    andFilters(windowFilter(createdAtFieldKey)),
			Sort:       []report.SortSpec{desc(totalRevenueColumnID)},
			Parameters: []report.ParameterDef{windowParam(90)},
			Totals:     true,
		},
	}
}
