package canned

import (
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

const (
	initialVersion = "1.0.0"

	customerColumnID      = "customer"
	shipmentCountColumnID = "shipment_count"
	statusFieldKey        = "status"
	createdAtFieldKey     = "createdAt"
	totalChargeFieldKey   = "totalChargeAmount"
	profileEdge           = "profile"
	windowDaysParam       = "windowDays"
	windowDaysLabel       = "Window (days)"
	horizonDaysParam      = "horizonDays"
	horizonDaysLabel      = "Horizon (days)"
	shipmentsEdge         = "shipments"
)

type Entry struct {
	Key           string
	Version       string
	Name          string
	Description   string
	Category      string
	Tags          []string
	DefaultFormat report.Format
	Definition    *report.Definition
}

type Registry struct {
	entries map[string]*Entry
	ordered []*Entry
}

func NewRegistry(entries []*Entry) *Registry {
	r := &Registry{
		entries: make(map[string]*Entry, len(entries)),
		ordered: make([]*Entry, 0, len(entries)),
	}
	for _, entry := range entries {
		r.entries[entry.Key] = entry
		r.ordered = append(r.ordered, entry)
	}
	sort.Slice(r.ordered, func(i, j int) bool { return r.ordered[i].Key < r.ordered[j].Key })
	return r
}

func (r *Registry) Get(key string) (*Entry, bool) {
	entry, ok := r.entries[key]
	return entry, ok
}

func (r *Registry) All() []*Entry {
	return r.ordered
}

func Default() *Registry {
	return NewRegistry([]*Entry{
		revenueByCustomer(),
		shipmentVolumeByStatus(),
		expiringWorkerCredentials(),
		orderRevenueSummary(),
		revenueVolumeTrend(),
		revenueByServiceType(),
		revenuePerMile(),
		facilityActivity(),
		driverActivity(),
		fleetRoster(),
		expiringTractorRegistrations(),
		expiringTrailerRegistrations(),
		equipmentUtilization(),
		driverQualificationStatus(),
		hazmatShipmentLog(),
		unbilledDeliveredShipments(),
		openInvoiceAging(),
		invoiceRevenueTrend(),
	})
}

func revenueByCustomer() *Entry {
	return &Entry{
		Key:           "revenue-by-customer",
		Version:       initialVersion,
		Name:          "Revenue by Customer",
		Description:   "Total shipment charges grouped by customer over a rolling window",
		Category:      "Billing",
		Tags:          []string{tagRevenue, "customers"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				{
					ID:   customerColumnID,
					Ref:  report.FieldRef{Path: []string{"customer"}, Field: "name"},
					Kind: report.ColumnKindDimension,
				},
				{
					ID:   "total_charges",
					Ref:  report.FieldRef{Field: totalChargeFieldKey},
					Kind: report.ColumnKindMeasure,
					Agg:  reportcatalog.AggSum,
				},
				{
					ID:   shipmentCountColumnID,
					Ref:  report.FieldRef{Field: "id"},
					Kind: report.ColumnKindMeasure,
					Agg:  reportcatalog.AggCount,
				},
				{
					ID:   "avg_charge",
					Ref:  report.FieldRef{Field: totalChargeFieldKey},
					Kind: report.ColumnKindMeasure,
					Agg:  reportcatalog.AggAvg,
				},
			},
			Filters: &report.FilterGroup{
				Op: report.BoolOpAnd,
				Filters: []report.FieldFilter{
					{
						Ref:      report.FieldRef{Field: createdAtFieldKey},
						Operator: dbtype.OpLastNDays,
						Param:    windowDaysParam,
					},
				},
			},
			Sort: []report.SortSpec{
				{ColumnID: "total_charges", Direction: dbtype.SortDirectionDesc},
			},
			Parameters: []report.ParameterDef{
				{
					Name:     windowDaysParam,
					Label:    windowDaysLabel,
					Type:     reportcatalog.FieldInt,
					Required: true,
					Default:  float64(30),
				},
			},
		},
	}
}

func shipmentVolumeByStatus() *Entry {
	return &Entry{
		Key:           "shipment-volume-by-status",
		Version:       initialVersion,
		Name:          "Shipment Volume by Status",
		Description:   "Shipment counts per lifecycle status over a rolling window",
		Category:      "Operations",
		Tags:          []string{shipmentsEdge, "operations"},
		DefaultFormat: report.FormatCSV,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipment,
			Columns: []report.ColumnSpec{
				{
					ID:   statusFieldKey,
					Ref:  report.FieldRef{Field: statusFieldKey},
					Kind: report.ColumnKindDimension,
				},
				{
					ID:   shipmentCountColumnID,
					Ref:  report.FieldRef{Field: "id"},
					Kind: report.ColumnKindMeasure,
					Agg:  reportcatalog.AggCount,
				},
			},
			Filters: &report.FilterGroup{
				Op: report.BoolOpAnd,
				Filters: []report.FieldFilter{
					{
						Ref:      report.FieldRef{Field: createdAtFieldKey},
						Operator: dbtype.OpLastNDays,
						Param:    windowDaysParam,
					},
				},
			},
			Sort: []report.SortSpec{
				{ColumnID: shipmentCountColumnID, Direction: dbtype.SortDirectionDesc},
			},
			Parameters: []report.ParameterDef{
				{
					Name:     windowDaysParam,
					Label:    windowDaysLabel,
					Type:     reportcatalog.FieldInt,
					Required: true,
					Default:  float64(30),
				},
			},
		},
	}
}

func expiringWorkerCredentials() *Entry {
	return &Entry{
		Key:           "expiring-worker-credentials",
		Version:       initialVersion,
		Name:          "Expiring Worker Credentials",
		Description:   "Workers whose licenses expire within the selected horizon",
		Category:      "Compliance",
		Tags:          []string{"workers", tagCompliance},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "worker",
			Columns: []report.ColumnSpec{
				{
					ID:   "first_name",
					Ref:  report.FieldRef{Field: "firstName"},
					Kind: report.ColumnKindDimension,
				},
				{
					ID:   "last_name",
					Ref:  report.FieldRef{Field: "lastName"},
					Kind: report.ColumnKindDimension,
				},
				{
					ID:   statusFieldKey,
					Ref:  report.FieldRef{Field: statusFieldKey},
					Kind: report.ColumnKindDimension,
				},
				{
					ID:   "license_expiry",
					Ref:  report.FieldRef{Path: []string{profileEdge}, Field: "licenseExpiry"},
					Kind: report.ColumnKindDimension,
				},
				{
					ID:   "medical_card_expiry",
					Ref:  report.FieldRef{Path: []string{profileEdge}, Field: "medicalCardExpiry"},
					Kind: report.ColumnKindDimension,
				},
			},
			Filters: &report.FilterGroup{
				Op: report.BoolOpAnd,
				Filters: []report.FieldFilter{
					{
						Ref: report.FieldRef{
							Path:  []string{profileEdge},
							Field: "licenseExpiry",
						},
						Operator: dbtype.OpNextNDays,
						Param:    horizonDaysParam,
					},
				},
			},
			Sort: []report.SortSpec{
				{ColumnID: "license_expiry", Direction: dbtype.SortDirectionAsc},
			},
			Parameters: []report.ParameterDef{
				{
					Name:     horizonDaysParam,
					Label:    horizonDaysLabel,
					Type:     reportcatalog.FieldInt,
					Required: true,
					Default:  float64(30),
				},
			},
		},
	}
}

func orderRevenueSummary() *Entry {
	return &Entry{
		Key:           "order-revenue-summary",
		Version:       initialVersion,
		Name:          "Order Revenue Summary",
		Description:   "Order totals and shipment charges grouped by customer",
		Category:      "Billing",
		Tags:          []string{"orders", tagRevenue},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "order",
			Columns: []report.ColumnSpec{
				{
					ID:   customerColumnID,
					Ref:  report.FieldRef{Path: []string{"customer"}, Field: "name"},
					Kind: report.ColumnKindDimension,
				},
				{
					ID:   "order_count",
					Ref:  report.FieldRef{Field: "id"},
					Kind: report.ColumnKindMeasure,
					Agg:  reportcatalog.AggCount,
				},
				{
					ID:   "order_total",
					Ref:  report.FieldRef{Field: "totalAmount"},
					Kind: report.ColumnKindMeasure,
					Agg:  reportcatalog.AggSum,
				},
				{
					ID: "shipment_charges",
					Ref: report.FieldRef{
						Path:  []string{shipmentsEdge},
						Field: totalChargeFieldKey,
					},
					Kind: report.ColumnKindMeasure,
					Agg:  reportcatalog.AggSum,
				},
			},
			Filters: &report.FilterGroup{
				Op: report.BoolOpAnd,
				Filters: []report.FieldFilter{
					{
						Ref:      report.FieldRef{Field: createdAtFieldKey},
						Operator: dbtype.OpLastNDays,
						Param:    windowDaysParam,
					},
				},
			},
			Sort: []report.SortSpec{
				{ColumnID: "order_total", Direction: dbtype.SortDirectionDesc},
			},
			Parameters: []report.ParameterDef{
				{
					Name:     windowDaysParam,
					Label:    windowDaysLabel,
					Type:     reportcatalog.FieldInt,
					Required: true,
					Default:  float64(90),
				},
			},
		},
	}
}
