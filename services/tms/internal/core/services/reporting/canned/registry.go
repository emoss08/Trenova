package canned

import (
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/report"
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
		customerScorecard(),
		revenueByServiceType(),
		revenueVolumeTrend(),
		revenuePerMile(),
		unbilledDeliveredShipments(),
		openInvoiceAging(),
		arAgingByCustomer(),
		invoiceRevenueTrend(),
		orderRevenueSummary(),

		shipmentVolumeByStatus(),
		facilityActivity(),
		lanePerformance(),
		stopDwellAndDetention(),
		hazmatShipmentLog(),

		driverProductivity(),
		tractorUtilization(),
		trailerUtilization(),
		equipmentUtilization(),
		deadheadAnalysis(),
		fleetRoster(),
		trailerRoster(),
		expiringTractorRegistrations(),
		expiringTrailerRegistrations(),

		driverRoster(),
		driverQualificationStatus(),
		expiringWorkerCredentials(),
	})
}
