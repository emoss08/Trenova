package platformplan

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
)

var (
	ErrUnknownMeterOverride = errors.New("free plan override names a meter the plan does not limit")
	ErrNegativeLimit        = errors.New("plan limit must not be negative")
	ErrUnknownPlan          = errors.New("unknown plan")
)

const (
	FreeDemoShipments         int64 = 12
	FreeDemoRecurringSeries   int64 = 1
	FreeDemoCustomers         int64 = 8
	FreeDemoLocations         int64 = 25
	FreeDemoWorkers           int64 = 3
	FreeDemoTractors          int64 = 3
	FreeDemoTrailers          int64 = 3
	FreeDemoUserSeats         int64 = 1
	FreeDemoDocumentUploads   int64 = 25
	FreeDemoDocumentStorage   int64 = 100 * 1024 * 1024
	FreeDemoDocumentFileBytes int64 = 10 * 1024 * 1024
	FreeDemoAssistantMessages int64 = 25
	FreeDemoAISpendCents      int64 = 150
	freeDemoName                    = "Free demo"
	unlimitedName                   = "Unlimited"
)

type Limit struct {
	Max    int64  `json:"max"`
	Window Window `json:"window"`
}

type Plan struct {
	Key                    PlanKey                            `json:"key"`
	Name                   string                             `json:"name"`
	Limits                 map[platformcatalog.MeterKey]Limit `json:"limits"`
	RestrictedCapabilities []Capability                       `json:"restrictedCapabilities"`
	TrialEndingMeters      []platformcatalog.MeterKey         `json:"trialEndingMeters"`
}

func DefaultFreeDemoLimits() map[platformcatalog.MeterKey]Limit {
	return map[platformcatalog.MeterKey]Limit{
		platformcatalog.MeterShipmentsTotal:          {Max: FreeDemoShipments, Window: WindowLifetime},
		platformcatalog.MeterRecurringShipmentSeries: {Max: FreeDemoRecurringSeries, Window: WindowLifetime},
		platformcatalog.MeterCustomersTotal:          {Max: FreeDemoCustomers, Window: WindowLifetime},
		platformcatalog.MeterLocationsTotal:          {Max: FreeDemoLocations, Window: WindowLifetime},
		platformcatalog.MeterWorkersTotal:            {Max: FreeDemoWorkers, Window: WindowLifetime},
		platformcatalog.MeterTractorsTotal:           {Max: FreeDemoTractors, Window: WindowLifetime},
		platformcatalog.MeterTrailersTotal:           {Max: FreeDemoTrailers, Window: WindowLifetime},
		platformcatalog.MeterUserSeats:               {Max: FreeDemoUserSeats, Window: WindowLifetime},
		platformcatalog.MeterDocumentUploads:         {Max: FreeDemoDocumentUploads, Window: WindowLifetime},
		platformcatalog.MeterDocumentStorageBytes:    {Max: FreeDemoDocumentStorage, Window: WindowLifetime},
		platformcatalog.MeterDocumentFileBytes:       {Max: FreeDemoDocumentFileBytes, Window: WindowPerItem},
		platformcatalog.MeterAIAssistantMessages:     {Max: FreeDemoAssistantMessages, Window: WindowMonthly},
		platformcatalog.MeterAISpendCents:            {Max: FreeDemoAISpendCents, Window: WindowMonthly},
	}
}

func FreeDemoRestrictedCapabilities() []Capability {
	return AllCapabilities()
}

func FreeDemoTrialEndingMeters() []platformcatalog.MeterKey {
	return []platformcatalog.MeterKey{platformcatalog.MeterShipmentsTotal}
}

func FreeDemo(overrides map[platformcatalog.MeterKey]int64) (*Plan, error) {
	limits := DefaultFreeDemoLimits()

	for _, meter := range slices.Sorted(maps.Keys(overrides)) {
		value := overrides[meter]
		limit, ok := limits[meter]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownMeterOverride, meter)
		}
		if value < 0 {
			return nil, fmt.Errorf("%w: %s is %d", ErrNegativeLimit, meter, value)
		}
		limit.Max = value
		limits[meter] = limit
	}

	return &Plan{
		Key:                    PlanKeyFreeDemo,
		Name:                   freeDemoName,
		Limits:                 limits,
		RestrictedCapabilities: FreeDemoRestrictedCapabilities(),
		TrialEndingMeters:      FreeDemoTrialEndingMeters(),
	}, nil
}

func ParseLimitOverrides(raw map[string]int64) map[platformcatalog.MeterKey]int64 {
	overrides := make(map[platformcatalog.MeterKey]int64, len(raw))
	for key, value := range raw {
		overrides[platformcatalog.MeterKey(key)] = value
	}

	return overrides
}

func Unlimited() *Plan {
	return &Plan{
		Key:                    PlanKeyUnlimited,
		Name:                   unlimitedName,
		Limits:                 map[platformcatalog.MeterKey]Limit{},
		RestrictedCapabilities: []Capability{},
		TrialEndingMeters:      []platformcatalog.MeterKey{},
	}
}

func (p *Plan) Limit(meter platformcatalog.MeterKey) (Limit, bool) {
	if p == nil {
		return Limit{}, false
	}

	limit, ok := p.Limits[meter]
	return limit, ok
}

func (p *Plan) Restricts(capability Capability) bool {
	if p == nil {
		return false
	}

	return slices.Contains(p.RestrictedCapabilities, capability)
}

func (p *Plan) EndsTrial(meter platformcatalog.MeterKey) bool {
	if p == nil {
		return false
	}

	return slices.Contains(p.TrialEndingMeters, meter)
}

func (p *Plan) Allows(capability Capability) bool {
	return !p.Restricts(capability)
}

func (p *Plan) IsUnlimited() bool {
	return p == nil || (len(p.Limits) == 0 && len(p.RestrictedCapabilities) == 0)
}

func (p *Plan) MeterKeys() []platformcatalog.MeterKey {
	if p == nil {
		return []platformcatalog.MeterKey{}
	}

	return slices.Sorted(maps.Keys(p.Limits))
}

type Catalog struct {
	plans map[PlanKey]*Plan
}

func NewCatalog(freeDemoOverrides map[platformcatalog.MeterKey]int64) (*Catalog, error) {
	freeDemo, err := FreeDemo(freeDemoOverrides)
	if err != nil {
		return nil, err
	}

	return &Catalog{
		plans: map[PlanKey]*Plan{
			PlanKeyFreeDemo:  freeDemo,
			PlanKeyUnlimited: Unlimited(),
		},
	}, nil
}

func (c *Catalog) Get(key PlanKey) (*Plan, error) {
	plan, ok := c.plans[key]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownPlan, key)
	}

	return plan, nil
}

func (c *Catalog) FreeDemo() *Plan {
	return c.plans[PlanKeyFreeDemo]
}

func (c *Catalog) Unlimited() *Plan {
	return c.plans[PlanKeyUnlimited]
}
