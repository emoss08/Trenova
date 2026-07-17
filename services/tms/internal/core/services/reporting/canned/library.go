package canned

import (
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

const (
	categoryOperations = "Operations"
	categoryFleet      = "Fleet"
	categoryCompliance = "Compliance"
	categoryBilling    = "Billing"

	firstNameField = "firstName"
	lastNameField  = "lastName"
	codeField      = "code"
	statusParam    = "statuses"
	statusesLabel  = "Statuses"
	customerEdge   = "customer"
	nameField      = "name"

	entityShipment     = "shipment"
	totalMilesColumnID = "total_miles"
	statusCanceled     = "Canceled"
	monthColumnID      = "month"
	tagRevenue         = "revenue"
	tagTractors        = "tractors"
	tagCompliance      = "compliance"
)

func dimCol(id, field string, path ...string) report.ColumnSpec {
	return report.ColumnSpec{
		ID:   id,
		Ref:  report.FieldRef{Path: path, Field: field},
		Kind: report.ColumnKindDimension,
	}
}

func measureCol(id string, agg reportcatalog.Aggregation, field string) report.ColumnSpec {
	return report.ColumnSpec{
		ID:   id,
		Ref:  report.FieldRef{Field: field},
		Kind: report.ColumnKindMeasure,
		Agg:  agg,
	}
}

func andFilters(filters ...report.FieldFilter) *report.FilterGroup {
	return &report.FilterGroup{Op: report.BoolOpAnd, Filters: filters}
}

func windowParam(defaultDays float64) report.ParameterDef {
	return report.ParameterDef{
		Name:     windowDaysParam,
		Label:    windowDaysLabel,
		Type:     reportcatalog.FieldInt,
		Required: true,
		Default:  defaultDays,
	}
}

func windowFilter(field string) report.FieldFilter {
	return report.FieldFilter{
		Ref:      report.FieldRef{Field: field},
		Operator: dbtype.OpLastNDays,
		Param:    windowDaysParam,
	}
}

func revenueVolumeTrend() *Entry {
	return &Entry{
		Key:           "revenue-volume-trend",
		Version:       initialVersion,
		Name:          "Revenue & Volume Trend",
		Description:   "Monthly shipment counts, revenue, weight, and pieces over a rolling window",
		Category:      categoryBilling,
		Tags:          []string{tagRevenue, "trend", "volume"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				{
					ID:     monthColumnID,
					Ref:    report.FieldRef{Field: createdAtFieldKey},
					Kind:   report.ColumnKindDimension,
					Bucket: report.DateBucketMonth,
					Label:  "Month",
				},
				measureCol(shipmentCountColumnID, reportcatalog.AggCount, "id"),
				measureCol("total_revenue", reportcatalog.AggSum, totalChargeFieldKey),
				measureCol("freight_revenue", reportcatalog.AggSum, "freightChargeAmount"),
				measureCol("avg_revenue", reportcatalog.AggAvg, totalChargeFieldKey),
				measureCol("total_weight", reportcatalog.AggSum, "weight"),
				measureCol("total_pieces", reportcatalog.AggSum, "pieces"),
			},
			Filters: andFilters(windowFilter(createdAtFieldKey)),
			Sort: []report.SortSpec{
				{ColumnID: monthColumnID, Direction: dbtype.SortDirectionAsc},
			},
			Parameters: []report.ParameterDef{windowParam(365)},
		},
	}
}

func revenueByServiceType() *Entry {
	return &Entry{
		Key:           "revenue-by-service-type",
		Version:       initialVersion,
		Name:          "Revenue by Service & Shipment Type",
		Description:   "Charges and volume grouped by service type and shipment type",
		Category:      categoryBilling,
		Tags:          []string{tagRevenue, "service-types"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				dimCol("service_type", codeField, "serviceType"),
				dimCol("shipment_type", codeField, "shipmentType"),
				measureCol(shipmentCountColumnID, reportcatalog.AggCount, "id"),
				measureCol("total_revenue", reportcatalog.AggSum, totalChargeFieldKey),
				measureCol("avg_revenue", reportcatalog.AggAvg, totalChargeFieldKey),
			},
			Filters: andFilters(windowFilter(createdAtFieldKey)),
			Sort: []report.SortSpec{
				{ColumnID: "total_revenue", Direction: dbtype.SortDirectionDesc},
			},
			Parameters: []report.ParameterDef{windowParam(90)},
		},
	}
}

func revenuePerMile() *Entry {
	return &Entry{
		Key:           "revenue-per-mile",
		Version:       initialVersion,
		Name:          "Revenue per Mile by Customer",
		Description:   "Linehaul economics per customer — total charges divided by loaded miles",
		Category:      categoryBilling,
		Tags:          []string{tagRevenue, "miles", "rate-analysis"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				dimCol(customerColumnID, nameField, customerEdge),
				measureCol(shipmentCountColumnID, reportcatalog.AggCount, "id"),
				measureCol("total_revenue", reportcatalog.AggSum, totalChargeFieldKey),
				{
					ID:    totalMilesColumnID,
					Ref:   report.FieldRef{Path: []string{"moves"}, Field: "distance"},
					Kind:  report.ColumnKindMeasure,
					Agg:   reportcatalog.AggSum,
					Label: "Total Miles",
				},
				{
					ID:    "revenue_per_mile",
					Kind:  report.ColumnKindComputed,
					Label: "Revenue per Mile",
					Computed: &report.ComputedSpec{
						Op:      report.ComputedOpDivide,
						LeftID:  "total_revenue",
						RightID: totalMilesColumnID,
						Format:  reportcatalog.FormatMoney,
					},
				},
			},
			Filters: andFilters(
				windowFilter(createdAtFieldKey),
				report.FieldFilter{
					Ref:      report.FieldRef{Field: statusFieldKey},
					Operator: dbtype.OpNotIn,
					Value:    []any{statusCanceled},
				},
			),
			Sort: []report.SortSpec{
				{ColumnID: "revenue_per_mile", Direction: dbtype.SortDirectionDesc},
			},
			Parameters: []report.ParameterDef{windowParam(90)},
		},
	}
}

func facilityActivity() *Entry {
	return &Entry{
		Key:           "facility-activity",
		Version:       initialVersion,
		Name:          "Facility Activity",
		Description:   "Stop throughput by facility — arrivals, pieces, and weight handled",
		Category:      categoryOperations,
		Tags:          []string{"stops", "facilities", "throughput"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "stop",
			Columns: []report.ColumnSpec{
				dimCol("facility", nameField, "location"),
				dimCol("city", "city", "location"),
				dimCol("stop_type", "type"),
				measureCol("stop_count", reportcatalog.AggCount, "id"),
				measureCol("total_pieces", reportcatalog.AggSum, "pieces"),
				measureCol("total_weight", reportcatalog.AggSum, "weight"),
			},
			Filters: andFilters(
				windowFilter("actualArrival"),
				report.FieldFilter{
					Ref:      report.FieldRef{Field: "type"},
					Operator: dbtype.OpIn,
					Param:    "stopTypes",
				},
			),
			Sort: []report.SortSpec{
				{ColumnID: "stop_count", Direction: dbtype.SortDirectionDesc},
			},
			Parameters: []report.ParameterDef{
				windowParam(30),
				{
					Name:          "stopTypes",
					Label:         "Stop Types",
					Type:          reportcatalog.FieldEnum,
					Required:      true,
					Multi:         true,
					Default:       []any{"Pickup", "Delivery"},
					AllowedValues: []string{"Pickup", "Delivery", "SplitPickup", "SplitDelivery"},
				},
			},
		},
	}
}

func driverActivity() *Entry {
	return &Entry{
		Key:           "driver-activity",
		Version:       initialVersion,
		Name:          "Driver Activity",
		Description:   "Moves and miles per driver over a rolling window",
		Category:      categoryOperations,
		Tags:          []string{"drivers", "miles", "productivity"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "shipment_move",
			Columns: []report.ColumnSpec{
				dimCol("first_name", firstNameField, "assignment", "primaryWorker"),
				dimCol("last_name", lastNameField, "assignment", "primaryWorker"),
				measureCol("move_count", reportcatalog.AggCount, "id"),
				measureCol(totalMilesColumnID, reportcatalog.AggSum, "distance"),
				measureCol("avg_miles", reportcatalog.AggAvg, "distance"),
			},
			Filters: andFilters(
				windowFilter(createdAtFieldKey),
				report.FieldFilter{
					Ref:      report.FieldRef{Field: statusFieldKey},
					Operator: dbtype.OpNotIn,
					Value:    []any{statusCanceled},
				},
			),
			Sort: []report.SortSpec{
				{ColumnID: totalMilesColumnID, Direction: dbtype.SortDirectionDesc},
			},
			Parameters: []report.ParameterDef{windowParam(30)},
		},
	}
}

func fleetRoster() *Entry {
	return &Entry{
		Key:           "fleet-roster",
		Version:       initialVersion,
		Name:          "Fleet Roster — Tractors",
		Description:   "Tractor fleet with equipment type, fleet code, and registration expiry",
		Category:      categoryFleet,
		Tags:          []string{tagTractors, "fleet", "roster"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "tractor",
			Columns: []report.ColumnSpec{
				dimCol("unit", codeField),
				dimCol(statusFieldKey, statusFieldKey),
				dimCol("make", "make"),
				dimCol("model", "model"),
				dimCol("year", "year"),
				dimCol("equipment_type", codeField, "equipmentType"),
				dimCol("fleet_code", codeField, "fleetCode"),
				dimCol("registration_expiry", "registrationExpiry"),
			},
			Filters: andFilters(report.FieldFilter{
				Ref:      report.FieldRef{Field: statusFieldKey},
				Operator: dbtype.OpIn,
				Param:    statusParam,
			}),
			Sort: []report.SortSpec{
				{ColumnID: "unit", Direction: dbtype.SortDirectionAsc},
			},
			Parameters: []report.ParameterDef{
				{
					Name:          statusParam,
					Label:         statusesLabel,
					Type:          reportcatalog.FieldEnum,
					Required:      true,
					Multi:         true,
					Default:       []any{"Available", "OutOfService", "AtMaintenance"},
					AllowedValues: []string{"Available", "OutOfService", "AtMaintenance", "Sold"},
				},
			},
		},
	}
}

func expiringTractorRegistrations() *Entry {
	return &Entry{
		Key:           "expiring-tractor-registrations",
		Version:       initialVersion,
		Name:          "Expiring Tractor Registrations",
		Description:   "Tractors whose registration expires within the selected horizon",
		Category:      categoryFleet,
		Tags:          []string{tagTractors, "registrations", tagCompliance},
		DefaultFormat: report.FormatCSV,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "tractor",
			Columns: []report.ColumnSpec{
				dimCol("unit", codeField),
				dimCol(statusFieldKey, statusFieldKey),
				dimCol("license_plate", "licensePlateNumber"),
				dimCol("registration_number", "registrationNumber"),
				dimCol("registration_expiry", "registrationExpiry"),
			},
			Filters: andFilters(report.FieldFilter{
				Ref:      report.FieldRef{Field: "registrationExpiry"},
				Operator: dbtype.OpNextNDays,
				Param:    horizonDaysParam,
			}),
			Sort: []report.SortSpec{
				{ColumnID: "registration_expiry", Direction: dbtype.SortDirectionAsc},
			},
			Parameters: []report.ParameterDef{
				{
					Name:     horizonDaysParam,
					Label:    horizonDaysLabel,
					Type:     reportcatalog.FieldInt,
					Required: true,
					Default:  float64(60),
				},
			},
		},
	}
}

func expiringTrailerRegistrations() *Entry {
	return &Entry{
		Key:           "expiring-trailer-registrations",
		Version:       initialVersion,
		Name:          "Expiring Trailer Registrations",
		Description:   "Trailers whose registration expires within the selected horizon",
		Category:      categoryFleet,
		Tags:          []string{"trailers", "registrations", tagCompliance},
		DefaultFormat: report.FormatCSV,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "trailer",
			Columns: []report.ColumnSpec{
				dimCol("unit", codeField),
				dimCol(statusFieldKey, statusFieldKey),
				dimCol("license_plate", "licensePlateNumber"),
				dimCol("registration_expiry", "registrationExpiry"),
				dimCol("last_inspection", "lastInspectionDate"),
			},
			Filters: andFilters(report.FieldFilter{
				Ref:      report.FieldRef{Field: "registrationExpiry"},
				Operator: dbtype.OpNextNDays,
				Param:    horizonDaysParam,
			}),
			Sort: []report.SortSpec{
				{ColumnID: "registration_expiry", Direction: dbtype.SortDirectionAsc},
			},
			Parameters: []report.ParameterDef{
				{
					Name:     horizonDaysParam,
					Label:    horizonDaysLabel,
					Type:     reportcatalog.FieldInt,
					Required: true,
					Default:  float64(60),
				},
			},
		},
	}
}

func equipmentUtilization() *Entry {
	return &Entry{
		Key:           "equipment-utilization",
		Version:       initialVersion,
		Name:          "Equipment Utilization",
		Description:   "Assignment counts per tractor over a rolling window — spot idle equipment",
		Category:      categoryFleet,
		Tags:          []string{tagTractors, "utilization"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "assignment",
			Columns: []report.ColumnSpec{
				dimCol("unit", codeField, "tractor"),
				dimCol("fleet_code", codeField, "tractor", "fleetCode"),
				measureCol("assignment_count", reportcatalog.AggCount, "id"),
			},
			Filters: andFilters(
				windowFilter(createdAtFieldKey),
				report.FieldFilter{
					Ref:      report.FieldRef{Field: statusFieldKey},
					Operator: dbtype.OpNotIn,
					Value:    []any{statusCanceled},
				},
			),
			Sort: []report.SortSpec{
				{ColumnID: "assignment_count", Direction: dbtype.SortDirectionDesc},
			},
			Parameters: []report.ParameterDef{windowParam(30)},
		},
	}
}

func driverQualificationStatus() *Entry {
	return &Entry{
		Key:           "driver-qualification-status",
		Version:       initialVersion,
		Name:          "Driver Qualification Status",
		Description:   "Active drivers by compliance standing, license expiry, and hire date",
		Category:      categoryCompliance,
		Tags:          []string{"drivers", tagCompliance, "dq"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "worker",
			Columns: []report.ColumnSpec{
				dimCol("first_name", firstNameField),
				dimCol("last_name", lastNameField),
				dimCol("driver_type", "driverType"),
				dimCol("compliance_status", "complianceStatus", profileEdge),
				dimCol("qualified", "isQualified", profileEdge),
				dimCol("license_expiry", "licenseExpiry", profileEdge),
				dimCol("hire_date", "hireDate", profileEdge),
			},
			Filters: andFilters(
				report.FieldFilter{
					Ref:      report.FieldRef{Field: statusFieldKey},
					Operator: dbtype.OpEqual,
					Value:    "Active",
				},
				report.FieldFilter{
					Ref: report.FieldRef{
						Path:  []string{profileEdge},
						Field: "complianceStatus",
					},
					Operator: dbtype.OpIn,
					Param:    "complianceStatuses",
				},
			),
			Sort: []report.SortSpec{
				{ColumnID: "last_name", Direction: dbtype.SortDirectionAsc},
			},
			Parameters: []report.ParameterDef{
				{
					Name:          "complianceStatuses",
					Label:         "Compliance Statuses",
					Type:          reportcatalog.FieldEnum,
					Required:      true,
					Multi:         true,
					Default:       []any{"NonCompliant", "Pending"},
					AllowedValues: []string{"Compliant", "NonCompliant", "Pending"},
				},
			},
		},
	}
}

func hazmatShipmentLog() *Entry {
	return &Entry{
		Key:           "hazmat-shipment-log",
		Version:       initialVersion,
		Name:          "Hazmat Shipment Log",
		Description:   "Shipments carrying hazardous commodities over a rolling window",
		Category:      categoryCompliance,
		Tags:          []string{"hazmat", shipmentsEdge, "safety"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				dimCol("pro_number", "proNumber"),
				dimCol("bol", "bol"),
				dimCol(customerColumnID, nameField, customerEdge),
				dimCol(statusFieldKey, statusFieldKey),
				dimCol("created_at", createdAtFieldKey),
			},
			Filters: andFilters(
				windowFilter(createdAtFieldKey),
				report.FieldFilter{
					Ref: report.FieldRef{
						Path:  []string{"commodities", "commodity"},
						Field: "hazardousMaterialId",
					},
					Operator: dbtype.OpIsNotNull,
				},
			),
			Sort: []report.SortSpec{
				{ColumnID: "created_at", Direction: dbtype.SortDirectionDesc},
			},
			Parameters: []report.ParameterDef{windowParam(90)},
		},
	}
}

func unbilledDeliveredShipments() *Entry {
	return &Entry{
		Key:           "unbilled-delivered-shipments",
		Version:       initialVersion,
		Name:          "Delivered, Not Billed",
		Description:   "Shipments delivered within the window that have not been billed yet",
		Category:      categoryBilling,
		Tags:          []string{"billing", "revenue-leakage"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				dimCol("pro_number", "proNumber"),
				dimCol(customerColumnID, nameField, customerEdge),
				dimCol(statusFieldKey, statusFieldKey),
				dimCol("delivered_at", "actualDeliveryDate"),
				dimCol("total_charge", totalChargeFieldKey),
			},
			Filters: andFilters(
				windowFilter("actualDeliveryDate"),
				report.FieldFilter{
					Ref:      report.FieldRef{Field: "billedAt"},
					Operator: dbtype.OpIsNull,
				},
				report.FieldFilter{
					Ref:      report.FieldRef{Field: statusFieldKey},
					Operator: dbtype.OpIn,
					Param:    statusParam,
				},
			),
			Sort: []report.SortSpec{
				{ColumnID: "delivered_at", Direction: dbtype.SortDirectionAsc},
			},
			Parameters: []report.ParameterDef{
				windowParam(90),
				{
					Name:     statusParam,
					Label:    statusesLabel,
					Type:     reportcatalog.FieldEnum,
					Required: true,
					Multi:    true,
					Default:  []any{"Completed", "ReadyToInvoice"},
					AllowedValues: []string{
						"Completed", "ReadyToInvoice", "PartiallyCompleted", "Invoiced",
					},
				},
			},
		},
	}
}

func openInvoiceAging() *Entry {
	return &Entry{
		Key:           "open-invoice-aging",
		Version:       initialVersion,
		Name:          "Open Invoice Aging",
		Description:   "Posted invoices with an outstanding balance, oldest due dates first",
		Category:      categoryBilling,
		Tags:          []string{"invoices", "ar", "aging"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "invoice",
			Columns: []report.ColumnSpec{
				dimCol("invoice_number", "number"),
				dimCol(customerColumnID, nameField, customerEdge),
				dimCol("settlement_status", "settlementStatus"),
				dimCol("invoice_date", "invoiceDate"),
				dimCol("due_date", "dueDate"),
				dimCol("total_amount", "totalAmount"),
				dimCol("applied_amount", "appliedAmount"),
			},
			Filters: andFilters(
				report.FieldFilter{
					Ref:      report.FieldRef{Field: statusFieldKey},
					Operator: dbtype.OpEqual,
					Value:    "Posted",
				},
				report.FieldFilter{
					Ref:      report.FieldRef{Field: "settlementStatus"},
					Operator: dbtype.OpIn,
					Param:    "settlementStatuses",
				},
			),
			Sort: []report.SortSpec{
				{ColumnID: "due_date", Direction: dbtype.SortDirectionAsc},
			},
			Parameters: []report.ParameterDef{
				{
					Name:          "settlementStatuses",
					Label:         "Settlement Statuses",
					Type:          reportcatalog.FieldEnum,
					Required:      true,
					Multi:         true,
					Default:       []any{"Unpaid", "PartiallyPaid"},
					AllowedValues: []string{"Unpaid", "PartiallyPaid", "Paid"},
				},
			},
		},
	}
}

func invoiceRevenueTrend() *Entry {
	return &Entry{
		Key:           "invoice-revenue-trend",
		Version:       initialVersion,
		Name:          "Invoiced Revenue Trend (This Year)",
		Description:   "Monthly posted invoice totals and collections for the current year",
		Category:      categoryBilling,
		Tags:          []string{"invoices", tagRevenue, "trend"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "invoice",
			Columns: []report.ColumnSpec{
				{
					ID:     monthColumnID,
					Ref:    report.FieldRef{Field: "invoiceDate"},
					Kind:   report.ColumnKindDimension,
					Bucket: report.DateBucketMonth,
					Label:  "Month",
				},
				measureCol("invoice_count", reportcatalog.AggCount, "id"),
				measureCol("invoiced_total", reportcatalog.AggSum, "totalAmount"),
				measureCol("collected_total", reportcatalog.AggSum, "appliedAmount"),
			},
			Filters: andFilters(
				report.FieldFilter{
					Ref:      report.FieldRef{Field: statusFieldKey},
					Operator: dbtype.OpEqual,
					Value:    "Posted",
				},
				report.FieldFilter{
					Ref:      report.FieldRef{Field: "invoiceDate"},
					Operator: dbtype.OpThisYear,
				},
			),
			Sort: []report.SortSpec{
				{ColumnID: monthColumnID, Direction: dbtype.SortDirectionAsc},
			},
		},
	}
}
