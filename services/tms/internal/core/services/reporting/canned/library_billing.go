package canned

import (
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

func revenueByCustomer() *Entry {
	return &Entry{
		Key:     "revenue-by-customer",
		Version: initialVersion,
		Name:    "Revenue by Customer",
		Description: "Freight, accessorial, and total charges per customer with volume, " +
			"miles, and revenue per mile over a rolling window",
		Category:      categoryBilling,
		Tags:          []string{tagRevenue, tagCustomers, tagBilling},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				dimCol(customerColumnID, "Customer", nameField, customerEdge),
				dimCol("customer_code", "Customer Code", codeField, customerEdge),
				countMeasure(shipmentCountColumnID, "Shipments"),
				sumMoney(totalRevenueColumnID, "Total Revenue", totalChargeFieldKey),
				sumMoney("freight_revenue", "Freight Revenue", freightChargeField),
				sumMoney("accessorial_revenue", "Accessorial Revenue", otherChargeField),
				avgMoney("avg_revenue", "Average Revenue per Shipment", totalChargeFieldKey),
				sumWeight(),
				sumPieces(),
				sumMiles(totalMilesColumnID, totalMilesLabel, movesEdge),
				perUnit(
					"revenue_per_mile", "Revenue per Mile",
					totalRevenueColumnID, totalMilesColumnID, money(), 2,
				),
				ratio(
					"accessorial_share", "Accessorial Share",
					"accessorial_revenue", totalRevenueColumnID),
			},
			Filters:    andFilters(windowFilter(createdAtFieldKey), excludeCanceled()),
			Sort:       []report.SortSpec{desc(totalRevenueColumnID)},
			Parameters: []report.ParameterDef{windowParam(30)},
		},
	}
}

func customerScorecard() *Entry {
	return &Entry{
		Key:     "customer-scorecard",
		Version: initialVersion,
		Name:    "Customer Scorecard",
		Description: "Account-level view of volume, revenue quality, freight profile, " +
			"and first and last shipment dates",
		Category:      categoryParties,
		Tags:          []string{tagCustomers, tagRevenue, "scorecard"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				dimCol(customerColumnID, "Customer", nameField, customerEdge),
				dimCol("customer_code", "Customer Code", codeField, customerEdge),
				countMeasure(shipmentCountColumnID, "Shipments"),
				sumMoney(totalRevenueColumnID, "Total Revenue", totalChargeFieldKey),
				avgMoney("avg_revenue", "Average Revenue per Shipment", totalChargeFieldKey),
				maxMoney("max_revenue", "Largest Shipment", totalChargeFieldKey),
				sumWeight(),
				sumMiles(totalMilesColumnID, totalMilesLabel, movesEdge),
				timestampMeasure("first_shipment", "First Shipment",
					reportcatalog.AggMin, createdAtFieldKey),
				timestampMeasure("last_shipment", "Last Shipment",
					reportcatalog.AggMax, createdAtFieldKey),
				perUnit(
					"revenue_per_mile", "Revenue per Mile",
					totalRevenueColumnID, totalMilesColumnID, money(), 2,
				),
				perUnit(
					"avg_weight", "Average Weight per Shipment",
					"total_weight", shipmentCountColumnID, weight(), 0,
				),
			},
			Filters:    andFilters(windowFilter(createdAtFieldKey), excludeCanceled()),
			Sort:       []report.SortSpec{desc(totalRevenueColumnID)},
			Parameters: []report.ParameterDef{windowParam(180)},
		},
	}
}

func revenueByServiceType() *Entry {
	return &Entry{
		Key:     "revenue-by-service-type",
		Version: initialVersion,
		Name:    "Revenue by Service & Shipment Type",
		Description: "Charges, volume, and revenue per mile grouped by service level " +
			"and shipment type",
		Category:      categoryBilling,
		Tags:          []string{tagRevenue, "service-types", "mix"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				dimCol("service_type", "Service Type", codeField, serviceTypeEdge),
				dimCol("shipment_type", "Shipment Type", codeField, shipmentTypeEdge),
				countMeasure(shipmentCountColumnID, "Shipments"),
				sumMoney(totalRevenueColumnID, "Total Revenue", totalChargeFieldKey),
				sumMoney("freight_revenue", "Freight Revenue", freightChargeField),
				sumMoney("accessorial_revenue", "Accessorial Revenue", otherChargeField),
				avgMoney("avg_revenue", "Average Revenue per Shipment", totalChargeFieldKey),
				sumWeight(),
				sumMiles(totalMilesColumnID, totalMilesLabel, movesEdge),
				perUnit(
					"revenue_per_mile", "Revenue per Mile",
					totalRevenueColumnID, totalMilesColumnID, money(), 2,
				),
				ratio(
					"accessorial_share", "Accessorial Share",
					"accessorial_revenue", totalRevenueColumnID),
			},
			Filters:    andFilters(windowFilter(createdAtFieldKey), excludeCanceled()),
			Sort:       []report.SortSpec{desc(totalRevenueColumnID)},
			Parameters: []report.ParameterDef{windowParam(90)},
		},
	}
}

func revenueVolumeTrend() *Entry {
	return &Entry{
		Key:     "revenue-volume-trend",
		Version: initialVersion,
		Name:    "Revenue & Volume Trend",
		Description: "Month-over-month shipments, revenue mix, tonnage, miles, " +
			"and revenue per mile",
		Category:      categoryBilling,
		Tags:          []string{tagRevenue, "trend", "volume"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				bucketDim(monthColumnID, "Month", createdAtFieldKey, report.DateBucketMonth),
				countMeasure(shipmentCountColumnID, "Shipments"),
				sumMoney(totalRevenueColumnID, "Total Revenue", totalChargeFieldKey),
				sumMoney("freight_revenue", "Freight Revenue", freightChargeField),
				sumMoney("accessorial_revenue", "Accessorial Revenue", otherChargeField),
				avgMoney("avg_revenue", "Average Revenue per Shipment", totalChargeFieldKey),
				sumWeight(),
				sumPieces(),
				sumMiles(totalMilesColumnID, totalMilesLabel, movesEdge),
				perUnit(
					"revenue_per_mile", "Revenue per Mile",
					totalRevenueColumnID, totalMilesColumnID, money(), 2,
				),
				perUnit(
					"avg_weight", "Average Weight per Shipment",
					"total_weight", shipmentCountColumnID, weight(), 0,
				),
			},
			Filters:    andFilters(windowFilter(createdAtFieldKey), excludeCanceled()),
			Sort:       []report.SortSpec{asc(monthColumnID)},
			Parameters: []report.ParameterDef{windowParam(365)},
		},
	}
}

func revenuePerMile() *Entry {
	return &Entry{
		Key:     "revenue-per-mile",
		Version: initialVersion,
		Name:    "Revenue per Mile by Customer",
		Description: "Linehaul economics per customer — charges, loaded miles, " +
			"revenue per mile, and average length of haul",
		Category:      categoryBilling,
		Tags:          []string{tagRevenue, tagMiles, "rate-analysis"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				dimCol(customerColumnID, "Customer", nameField, customerEdge),
				dimCol("service_type", "Service Type", codeField, serviceTypeEdge),
				countMeasure(shipmentCountColumnID, "Shipments"),
				sumMoney(totalRevenueColumnID, "Total Revenue", totalChargeFieldKey),
				sumMoney("freight_revenue", "Freight Revenue", freightChargeField),
				sumMiles(totalMilesColumnID, totalMilesLabel, movesEdge),
				avgMovesMiles("avg_move_miles", movesEdge),
				perUnit(
					"revenue_per_mile", "Revenue per Mile",
					totalRevenueColumnID, totalMilesColumnID, money(), 2,
				),
				perUnit(
					"miles_per_shipment", "Average Length of Haul",
					totalMilesColumnID, shipmentCountColumnID, miles(1), 1,
				),
			},
			Filters:    andFilters(windowFilter(createdAtFieldKey), excludeCanceled()),
			Sort:       []report.SortSpec{desc("revenue_per_mile")},
			Parameters: []report.ParameterDef{windowParam(90)},
		},
	}
}

func unbilledDeliveredShipments() *Entry {
	return &Entry{
		Key:     "unbilled-delivered-shipments",
		Version: initialVersion,
		Name:    "Delivered, Not Billed",
		Description: "Revenue sitting in the billing queue — delivered shipments with " +
			"no invoice, aged by delivery date",
		Category:      categoryBilling,
		Tags:          []string{tagBilling, "revenue-leakage", "aging"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				dimCol("pro_number", "PRO Number", "proNumber"),
				dimCol("bol", "BOL", "bol"),
				dimCol(customerColumnID, "Customer", nameField, customerEdge),
				dimCol("customer_code", "Customer Code", codeField, customerEdge),
				dimCol(statusFieldKey, "Status", statusFieldKey),
				dimCol("transfer_status", "Billing Transfer", "billingTransferStatus"),
				dimCol("service_type", "Service Type", codeField, serviceTypeEdge),
				dateDim("ship_date", "Ship Date", "actualShipDate"),
				dateDim("delivered_at", "Delivery Date", "actualDeliveryDate"),
				dateDim("ready_to_bill_at", "Ready to Bill", "markedReadyToBillAt"),
				dimCol("total_charge", "Total Charge", totalChargeFieldKey),
				dimCol("freight_charge", "Freight Charge", freightChargeField),
				dimCol("accessorial_charge", "Accessorial Charges", otherChargeField),
				dimCol("weight", "Weight", weightField),
				dimCol("pieces", "Pieces", piecesField),
			},
			Filters: andFilters(
				windowFilter("actualDeliveryDate"),
				isNull("billedAt"),
				inParam(statusFieldKey, statusParam),
			),
			Sort: []report.SortSpec{asc("delivered_at")},
			Parameters: []report.ParameterDef{
				windowParam(90),
				enumParam(statusParam, statusesLabel,
					[]any{"Completed", "ReadyToInvoice"},
					[]string{"Completed", "ReadyToInvoice", "PartiallyCompleted", "Invoiced"},
				),
			},
		},
	}
}

func openInvoiceAging() *Entry {
	return &Entry{
		Key:     "open-invoice-aging",
		Version: initialVersion,
		Name:    "Open Invoice Aging",
		Description: "Posted invoices carrying a balance, with amount applied and " +
			"open balance, oldest due dates first",
		Category:      categoryBilling,
		Tags:          []string{"invoices", tagAR, "aging"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityInvoice,
			Columns: []report.ColumnSpec{
				dimCol("invoice_number", "Invoice Number", "number"),
				dimCol(customerColumnID, "Customer", nameField, customerEdge),
				dimCol("bill_type", "Bill Type", "billType"),
				dimCol("settlement_status", "Settlement Status", "settlementStatus"),
				dateDim("invoice_date", "Invoice Date", "invoiceDate"),
				dateDim("due_date", "Due Date", "dueDate"),
				sumMoney("total_amount", "Invoice Total", "totalAmount"),
				sumMoney("applied_amount", "Amount Applied", "appliedAmount"),
				difference("open_balance", "Open Balance",
					"total_amount", "applied_amount", money()),
			},
			Filters: andFilters(
				statusEquals(statusPosted),
				inParam(settlementStatusKey, settlementStatusParam),
			),
			Sort: []report.SortSpec{asc("due_date")},
			Parameters: []report.ParameterDef{
				settlementStatusesParam(),
			},
		},
	}
}

func arAgingByCustomer() *Entry {
	return &Entry{
		Key:     "ar-aging-by-customer",
		Version: initialVersion,
		Name:    "AR Aging by Customer",
		Description: "Outstanding receivables rolled up per customer with collection rate " +
			"and the oldest open invoice date",
		Category:      categoryBilling,
		Tags:          []string{tagAR, "collections", tagCustomers},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityInvoice,
			Columns: []report.ColumnSpec{
				dimCol(customerColumnID, "Customer", nameField, customerEdge),
				dimCol("customer_code", "Customer Code", codeField, customerEdge),
				countMeasure("invoice_count", "Open Invoices"),
				sumMoney("total_amount", "Total Invoiced", "totalAmount"),
				sumMoney("applied_amount", "Total Applied", "appliedAmount"),
				timestampMeasure("oldest_invoice", "Oldest Invoice",
					reportcatalog.AggMin, "invoiceDate"),
				timestampMeasure("earliest_due", "Earliest Due Date",
					reportcatalog.AggMin, "dueDate"),
				difference("open_balance", "Open Balance",
					"total_amount", "applied_amount", money()),
				ratio("collection_rate", "Collection Rate",
					"applied_amount", "total_amount"),
			},
			Filters: andFilters(
				statusEquals(statusPosted),
				inParam(settlementStatusKey, settlementStatusParam),
			),
			Sort: []report.SortSpec{desc("open_balance")},
			Parameters: []report.ParameterDef{
				settlementStatusesParam(),
			},
		},
	}
}

func invoiceRevenueTrend() *Entry {
	return &Entry{
		Key:     "invoice-revenue-trend",
		Version: initialVersion,
		Name:    "Invoiced Revenue Trend (This Year)",
		Description: "Monthly posted invoice totals, collections, outstanding balance, " +
			"and collection rate for the current year",
		Category:      categoryBilling,
		Tags:          []string{"invoices", tagRevenue, "trend"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityInvoice,
			Columns: []report.ColumnSpec{
				bucketDim(monthColumnID, "Month", "invoiceDate", report.DateBucketMonth),
				countMeasure("invoice_count", "Invoices"),
				sumMoney("invoiced_total", "Invoiced", "totalAmount"),
				sumMoney("collected_total", "Collected", "appliedAmount"),
				sumMoney("other_total", "Other Charges", "otherAmount"),
				avgMoney("avg_invoice", "Average Invoice", "totalAmount"),
				difference("outstanding", "Outstanding",
					"invoiced_total", "collected_total", money()),
				ratio("collection_rate", "Collection Rate",
					"collected_total", "invoiced_total"),
			},
			Filters: andFilters(
				statusEquals(statusPosted),
				thisYear("invoiceDate"),
			),
			Sort: []report.SortSpec{asc(monthColumnID)},
		},
	}
}

func orderRevenueSummary() *Entry {
	return &Entry{
		Key:     "order-revenue-summary",
		Version: initialVersion,
		Name:    "Order Revenue Summary",
		Description: "Quoted versus actual order value by customer, with the shipment " +
			"charges booked underneath each order",
		Category:      categoryBilling,
		Tags:          []string{"orders", tagRevenue, tagCustomers},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityOrder,
			Columns: []report.ColumnSpec{
				dimCol(customerColumnID, "Customer", nameField, customerEdge),
				dimCol("customer_code", "Customer Code", codeField, customerEdge),
				countMeasure("order_count", "Orders"),
				sumMoney("quoted_total", "Quoted", "quotedAmount"),
				sumMoney("order_total", "Order Total", "totalAmount"),
				sumMoney("shipment_charges", "Shipment Charges",
					totalChargeFieldKey, shipmentsEdge),
				countMeasure(shipmentCountColumnID, "Shipments", shipmentsEdge),
				avgMoney("avg_order", "Average Order Value", "totalAmount"),
				difference("quote_variance", "Quote Variance",
					"order_total", "quoted_total", money()),
				difference("billing_variance", "Order vs Shipment Charges",
					"order_total", "shipment_charges", money()),
			},
			Filters:    andFilters(windowFilter(createdAtFieldKey)),
			Sort:       []report.SortSpec{desc("order_total")},
			Parameters: []report.ParameterDef{windowParam(90)},
		},
	}
}
