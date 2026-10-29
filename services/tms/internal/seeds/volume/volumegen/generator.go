package volumegen

import (
	"errors"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DefaultShipments     = 10_000
	DefaultHistoryMonths = 12
	MaxShipments         = 250_000
)

var (
	ErrShipmentCount = errors.New("shipment count must be between 1 and 250000")
	ErrHistoryMonths = errors.New("history must be between 1 and 36 months")
)

type Options struct {
	Shipments      int
	HistoryMonths  int
	Seed           uint64
	OrganizationID pulid.ID
	Logf           func(format string, args ...any)
}

func (o *Options) Normalize() error {
	if o.Shipments == 0 {
		o.Shipments = DefaultShipments
	}
	if o.Shipments < 1 || o.Shipments > MaxShipments {
		return ErrShipmentCount
	}
	if o.HistoryMonths == 0 {
		o.HistoryMonths = DefaultHistoryMonths
	}
	if o.HistoryMonths < 1 || o.HistoryMonths > 36 {
		return ErrHistoryMonths
	}
	if o.Seed == 0 {
		o.Seed = uint64(time.Now().UnixNano()) //nolint:gosec // any seed is fine
	}
	if o.Logf == nil {
		o.Logf = func(string, ...any) {}
	}

	return nil
}

type Scale struct {
	Customers   int
	Locations   int
	Drivers     int
	Tractors    int
	Trailers    int
	Carriers    int
	FleetCodes  int
	CommentRate float64
}

func ScaleFor(shipments int) Scale {
	return Scale{
		Customers:   Clamp(shipments/40, 40, 5_000),
		Locations:   Clamp(shipments/8, 150, 25_000),
		Drivers:     Clamp(shipments/25, 30, 10_000),
		Tractors:    Clamp(shipments*9/250, 25, 9_000),
		Trailers:    Clamp(shipments*3/50, 40, 15_000),
		Carriers:    Clamp(shipments/80, 20, 3_000),
		FleetCodes:  Clamp(shipments/800, 6, 40),
		CommentRate: 2.4,
	}
}

type Report struct {
	Seed     uint64
	Counts   []Count
	Duration time.Duration
}

type Count struct {
	Entity string
	Rows   int
}

func (r *Report) Add(entity string, rows int) {
	for i := range r.Counts {
		if r.Counts[i].Entity == entity {
			r.Counts[i].Rows += rows
			return
		}
	}
	r.Counts = append(r.Counts, Count{Entity: entity, Rows: rows})
}

type Generator struct {
	DB      *keel.DB
	Opts    Options
	Scale   Scale
	Fake    *Faker
	Numbers *Numberer
	Refs    *TenantRefs
	Report  *Report
	Now     time.Time
	Start   time.Time

	Customers   []*CustomerRecord
	Locations   []*LocationRecord
	Commodities []*CommodityRecord
	Drivers     []*DriverRecord
	Crews       []*Crew
	Carriers    []pulid.ID
	Shipments   []*ShipmentRecord
	Payments    []*customerpayment.Payment
	Factoring   *FactorState
}

func Clamp(value, low, high int) int {
	return max(low, min(value, high))
}
