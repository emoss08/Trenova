package canned

import (
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
)

const (
	categoryOperations = "Operations"
	categoryFleet      = "Fleet"
	categoryCompliance = "Compliance"
	categoryBilling    = "Billing"
	categoryParties    = "Customers"

	initialVersion = "1.0.0"

	entityShipment          = "shipment"
	entityShipmentMove      = "shipment_move"
	entityShipmentCommodity = "shipment_commodity"
	entityStop              = "stop"
	entityAssignment        = "assignment"
	entityOrder             = "order"
	entityInvoice           = "invoice"
	entityWorker            = "worker"
	entityTractor           = "tractor"
	entityTrailer           = "trailer"

	assignmentEdge      = "assignment"
	commodityEdge       = "commodity"
	customerEdge        = "customer"
	equipmentTypeEdge   = "equipmentType"
	manufacturerEdge    = "equipmentManufacturer"
	fleetCodeEdge       = "fleetCode"
	hazmatEdge          = "hazardousMaterial"
	locationEdge        = "location"
	movesEdge           = "moves"
	primaryWorkerEdge   = "primaryWorker"
	profileEdge         = "profile"
	serviceTypeEdge     = "serviceType"
	shipmentEdge        = "shipment"
	shipmentTypeEdge    = "shipmentType"
	shipmentsEdge       = "shipments"
	tractorEdge         = "tractor"
	trailerEdge         = "trailer"
	locationCategEdge   = "locationCategory"
	originStopEdge      = "originStop"
	destinationStopEdge = "destinationStop"
	secondaryWorkerKey  = "secondaryWorker"

	cityField           = "city"
	codeField           = "code"
	createdAtFieldKey   = "createdAt"
	distanceField       = "distance"
	firstNameField      = "firstName"
	lastNameField       = "lastName"
	nameField           = "name"
	piecesField         = "pieces"
	statusFieldKey      = "status"
	totalChargeFieldKey = "totalChargeAmount"
	freightChargeField  = "freightChargeAmount"
	otherChargeField    = "otherChargeAmount"
	weightField         = "weight"
	licensePlateField   = "licensePlateNumber"
	registrationField   = "registrationNumber"
	registrationExpiry  = "registrationExpiry"
	totalAmountField    = "totalAmount"
	appliedAmountField  = "appliedAmount"
	settlementStatusKey = "settlementStatus"

	customerColumnID      = "customer"
	customerCodeColumnID  = "customer_code"
	shipmentCountColumnID = "shipment_count"
	totalMilesColumnID    = "total_miles"
	totalRevenueColumnID  = "total_revenue"
	totalWeightColumnID   = "total_weight"
	totalPiecesColumnID   = "total_pieces"
	loadedMilesColumnID   = "split_miles"
	monthColumnID         = "month"
	moveCountColumnID     = "move_count"
	stopCountColumnID     = "stop_count"
	avgRevenueColumnID    = "avg_revenue"
	accessorialColumnID   = "accessorial_revenue"
	freightRevenueColID   = "freight_revenue"
	revenuePerMileColID   = "revenue_per_mile"
	openBalanceColumnID   = "open_balance"
	appliedColumnID       = "applied_amount"
	invoiceTotalColumnID  = "total_amount"
	avgMilesColumnID      = "avg_miles"
	lastMoveColumnID      = "last_move"
	fleetCodeColumnID     = "fleet_code"
	unitColumnID          = "unit"
	equipTypeColumnID     = "equipment_type"

	windowDaysParam        = "windowDays"
	windowDaysLabel        = "Window (days)"
	horizonDaysParam       = "horizonDays"
	horizonDaysLabel       = "Horizon (days)"
	statusParam            = "statuses"
	statusesLabel          = "Statuses"
	settlementStatusParam  = "settlementStatuses"
	settlementStatusLabel  = "Settlement Statuses"
	equipmentStatusDefault = "Available"

	statusCanceled      = "Canceled"
	statusPosted        = "Posted"
	statusActive        = "Active"
	statusUnpaid        = "Unpaid"
	statusPartiallyPaid = "PartiallyPaid"
	statusOutOfService  = "OutOfService"
	statusAtMaintenance = "AtMaintenance"

	avgMilesLabel     = "Average Miles per Move"
	avgRevenueLabel   = "Average Revenue per Shipment"
	totalRevenueLabel = "Total Revenue"
	totalMilesLabel   = "Total Miles"
	fleetLabel        = "Fleet"
	unitLabelTractor  = "Tractor Unit"
	customerLabel     = "Customer"
	customerCodeLabel = "Customer Code"
	equipTypeLabel    = "Equipment Type"
	milesLabel        = "Miles"
	weightLabel       = "Weight"
	piecesLabel       = "Pieces"
	statusLabel       = "Status"

	tagRevenue    = "revenue"
	tagCompliance = "compliance"
	tagTractors   = "tractors"
	tagTrailers   = "trailers"
	tagDrivers    = "drivers"
	tagAssets     = "assets"
	tagMiles      = "miles"
	tagUtilizatn  = "utilization"
	tagBilling    = "billing"
	tagAR         = "ar"
	tagCustomers  = "customers"
	tagOperations = "operations"
	tagRoster     = "roster"
	tagFleet      = "fleet"
)

func dimCol(id, label, field string, path ...string) report.ColumnSpec {
	return report.ColumnSpec{
		ID:    id,
		Ref:   report.FieldRef{Path: path, Field: field},
		Kind:  report.ColumnKindDimension,
		Label: label,
	}
}

func styledDim(
	id, label, field string,
	display *reportfmt.Spec,
	path ...string,
) report.ColumnSpec {
	col := dimCol(id, label, field, path...)
	col.Display = display
	return col
}

func dateDim(id, label, field string, path ...string) report.ColumnSpec {
	return styledDim(id, label, field, dateOnly(), path...)
}

func bucketDim(id, label, field string, bucket report.DateBucket) report.ColumnSpec {
	return report.ColumnSpec{
		ID:     id,
		Ref:    report.FieldRef{Field: field},
		Kind:   report.ColumnKindDimension,
		Bucket: bucket,
		Label:  label,
	}
}

// measureSpec groups the measure knobs so builders stay readable as the
// column set grows.
type measureSpec struct {
	id        string
	label     string
	agg       reportcatalog.Aggregation
	field     string
	path      []string
	display   *reportfmt.Spec
	transform *report.TransformSpec
}

func newMeasure(spec *measureSpec) report.ColumnSpec {
	return report.ColumnSpec{
		ID:        spec.id,
		Ref:       report.FieldRef{Path: spec.path, Field: spec.field},
		Kind:      report.ColumnKindMeasure,
		Agg:       spec.agg,
		Label:     spec.label,
		Display:   spec.display,
		Transform: spec.transform,
	}
}

func countMeasure(id, label string, path ...string) report.ColumnSpec {
	return newMeasure(&measureSpec{
		id:      id,
		label:   label,
		agg:     reportcatalog.AggCount,
		field:   "id",
		path:    path,
		display: counted(),
	})
}

func distinctMeasure(id, label, field string, path ...string) report.ColumnSpec {
	return newMeasure(&measureSpec{
		id:      id,
		label:   label,
		agg:     reportcatalog.AggCountDistinct,
		field:   field,
		path:    path,
		display: counted(),
	})
}

func sumMoney(id, label, field string, path ...string) report.ColumnSpec {
	return newMeasure(&measureSpec{
		id:      id,
		label:   label,
		agg:     reportcatalog.AggSum,
		field:   field,
		path:    path,
		display: money(),
	})
}

func maxMoney(id, label, field string) report.ColumnSpec {
	return newMeasure(&measureSpec{
		id:      id,
		label:   label,
		agg:     reportcatalog.AggMax,
		field:   field,
		display: money(),
	})
}

// avgMoney rounds in SQL as well as at display time: an average of
// 1082.0672643987443142 belongs in the export as 1082.07, not as the raw
// repeating quotient.
func avgMoney(id, label, field string) report.ColumnSpec {
	return newMeasure(&measureSpec{
		id:        id,
		label:     label,
		agg:       reportcatalog.AggAvg,
		field:     field,
		display:   money(),
		transform: roundTo(2),
	})
}

func sumMiles(id, label string, path ...string) report.ColumnSpec {
	return newMeasure(&measureSpec{
		id:        id,
		label:     label,
		agg:       reportcatalog.AggSum,
		field:     distanceField,
		path:      path,
		display:   miles(0),
		transform: roundTo(0),
	})
}

func avgMovesMiles(id string, path ...string) report.ColumnSpec {
	return newMeasure(&measureSpec{
		id:        id,
		label:     avgMilesLabel,
		agg:       reportcatalog.AggAvg,
		field:     distanceField,
		path:      path,
		display:   miles(1),
		transform: roundTo(1),
	})
}

func sumWeight() report.ColumnSpec {
	return newMeasure(&measureSpec{
		id:      totalWeightColumnID,
		label:   "Total Weight",
		agg:     reportcatalog.AggSum,
		field:   weightField,
		display: weight(),
	})
}

func sumPieces() report.ColumnSpec {
	return newMeasure(&measureSpec{
		id:      totalPiecesColumnID,
		label:   "Total Pieces",
		agg:     reportcatalog.AggSum,
		field:   piecesField,
		display: counted(),
	})
}

func timestampMeasure(
	id, label string,
	agg reportcatalog.Aggregation,
	field string,
) report.ColumnSpec {
	return newMeasure(&measureSpec{
		id:      id,
		label:   label,
		agg:     agg,
		field:   field,
		display: asDate(),
	})
}

func computed(
	id, label string,
	op report.ComputedOp,
	leftID, rightID string,
	display *reportfmt.Spec,
) report.ColumnSpec {
	return report.ColumnSpec{
		ID:    id,
		Kind:  report.ColumnKindComputed,
		Label: label,
		Computed: &report.ComputedSpec{
			Op:      op,
			LeftID:  leftID,
			RightID: rightID,
		},
		Display: display,
	}
}

// ratio renders a share as a percentage; the SQL transform keeps the stored
// quotient at three decimals so exports carry a tenth-of-a-percent value
// rather than a repeating fraction.
func ratio(id, label, leftID, rightID string) report.ColumnSpec {
	col := computed(id, label, report.ComputedOpDivide, leftID, rightID, percent(1))
	col.Transform = roundTo(3)
	return col
}

func perUnit(
	id, label, leftID, rightID string,
	display *reportfmt.Spec,
	precision int,
) report.ColumnSpec {
	col := computed(id, label, report.ComputedOpDivide, leftID, rightID, display)
	col.Transform = roundTo(precision)
	return col
}

func difference(id, label, leftID, rightID string, display *reportfmt.Spec) report.ColumnSpec {
	return computed(id, label, report.ComputedOpSubtract, leftID, rightID, display)
}

func andFilters(filters ...report.FieldFilter) *report.FilterGroup {
	return &report.FilterGroup{Op: report.BoolOpAnd, Filters: filters}
}

func orGroup(filters ...report.FieldFilter) report.FilterGroup {
	return report.FilterGroup{Op: report.BoolOpOr, Filters: filters}
}

func withGroups(group *report.FilterGroup, nested ...report.FilterGroup) *report.FilterGroup {
	group.Groups = append(group.Groups, nested...)
	return group
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

func horizonParam(defaultDays float64) report.ParameterDef {
	return report.ParameterDef{
		Name:     horizonDaysParam,
		Label:    horizonDaysLabel,
		Type:     reportcatalog.FieldInt,
		Required: true,
		Default:  defaultDays,
	}
}

func enumParam(name, label string, defaults []any, allowed []string) report.ParameterDef {
	return report.ParameterDef{
		Name:          name,
		Label:         label,
		Type:          reportcatalog.FieldEnum,
		Required:      true,
		Multi:         true,
		Default:       defaults,
		AllowedValues: allowed,
	}
}

func equipmentStatusParam() report.ParameterDef {
	return enumParam(statusParam, statusesLabel,
		[]any{equipmentStatusDefault, statusOutOfService, statusAtMaintenance},
		[]string{equipmentStatusDefault, statusOutOfService, statusAtMaintenance, "Sold"},
	)
}

func settlementStatusesParam() report.ParameterDef {
	return enumParam(settlementStatusParam, settlementStatusLabel,
		[]any{statusUnpaid, statusPartiallyPaid},
		[]string{statusUnpaid, statusPartiallyPaid, "Paid"},
	)
}

func windowFilter(field string, path ...string) report.FieldFilter {
	return report.FieldFilter{
		Ref:      report.FieldRef{Path: path, Field: field},
		Operator: dbtype.OpLastNDays,
		Param:    windowDaysParam,
	}
}

func horizonFilter(field string, path ...string) report.FieldFilter {
	return report.FieldFilter{
		Ref:      report.FieldRef{Path: path, Field: field},
		Operator: dbtype.OpNextNDays,
		Param:    horizonDaysParam,
	}
}

func statusEquals(value string) report.FieldFilter {
	return report.FieldFilter{
		Ref:      report.FieldRef{Field: statusFieldKey},
		Operator: dbtype.OpEqual,
		Value:    value,
	}
}

func notIn(field string, values []any, path ...string) report.FieldFilter {
	return report.FieldFilter{
		Ref:      report.FieldRef{Path: path, Field: field},
		Operator: dbtype.OpNotIn,
		Value:    values,
	}
}

func inParam(field, param string, path ...string) report.FieldFilter {
	return report.FieldFilter{
		Ref:      report.FieldRef{Path: path, Field: field},
		Operator: dbtype.OpIn,
		Param:    param,
	}
}

func isNull(field string) report.FieldFilter {
	return report.FieldFilter{
		Ref:      report.FieldRef{Field: field},
		Operator: dbtype.OpIsNull,
	}
}

func isNotNull(field string, path ...string) report.FieldFilter {
	return report.FieldFilter{
		Ref:      report.FieldRef{Path: path, Field: field},
		Operator: dbtype.OpIsNotNull,
	}
}

func thisYear(field string) report.FieldFilter {
	return report.FieldFilter{
		Ref:      report.FieldRef{Field: field},
		Operator: dbtype.OpThisYear,
	}
}

func excludeCanceled() report.FieldFilter {
	return notIn(statusFieldKey, []any{statusCanceled})
}

func desc(columnID string) report.SortSpec {
	return report.SortSpec{ColumnID: columnID, Direction: dbtype.SortDirectionDesc}
}

func asc(columnID string) report.SortSpec {
	return report.SortSpec{ColumnID: columnID, Direction: dbtype.SortDirectionAsc}
}

func loadedPivot(measureIDs ...string) *report.PivotSpec {
	return &report.PivotSpec{
		Ref:        report.FieldRef{Field: "loaded"},
		Values:     []string{"true", "false"},
		Labels:     []string{"Loaded", "Empty"},
		MeasureIDs: measureIDs,
	}
}

func roundTo(precision int) *report.TransformSpec {
	return &report.TransformSpec{Op: report.TransformRound, Precision: &precision}
}

func money() *reportfmt.Spec {
	decimals := 2
	return &reportfmt.Spec{
		Style:    reportfmt.StyleCurrency,
		Decimals: &decimals,
		Negative: reportfmt.NegativeParentheses,
	}
}

func percent(decimals int) *reportfmt.Spec {
	return &reportfmt.Spec{Style: reportfmt.StylePercent, Decimals: &decimals}
}

func miles(decimals int) *reportfmt.Spec {
	return &reportfmt.Spec{
		Style:    reportfmt.StyleNumber,
		Decimals: &decimals,
		Suffix:   " mi",
	}
}

func weight() *reportfmt.Spec {
	decimals := 0
	return &reportfmt.Spec{
		Style:    reportfmt.StyleNumber,
		Decimals: &decimals,
		Suffix:   " lbs",
	}
}

func counted() *reportfmt.Spec {
	decimals := 0
	grouping := true
	return &reportfmt.Spec{
		Style:    reportfmt.StyleNumber,
		Decimals: &decimals,
		Grouping: &grouping,
	}
}

func dateOnly() *reportfmt.Spec {
	return &reportfmt.Spec{Style: reportfmt.StyleDate, DateStyle: reportfmt.DateStyleDate}
}

func asDate() *reportfmt.Spec {
	return &reportfmt.Spec{Style: reportfmt.StyleDate, DateStyle: reportfmt.DateStyleDateTime}
}

func dwell() *reportfmt.Spec {
	return &reportfmt.Spec{
		Style:         reportfmt.StyleDuration,
		DurationUnit:  reportfmt.DurationUnitSeconds,
		DurationStyle: reportfmt.DurationStyleCompact,
	}
}

func yesNo() *reportfmt.Spec {
	return &reportfmt.Spec{Style: reportfmt.StyleBool, BoolStyle: reportfmt.BoolStyleYesNo}
}

func emptyDash() *reportfmt.Spec {
	return &reportfmt.Spec{NullText: "—"}
}
