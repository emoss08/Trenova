package board

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

type Measure string

const (
	MeasureLateness   = Measure("lateness")
	MeasureRevenue    = Measure("revenue")
	MeasureAmountDue  = Measure("amountDue")
	MeasureTotal      = Measure("total")
	MeasureAge        = Measure("age")
	MeasureAmount     = Measure("amount")
	MeasurePendingFor = Measure("pendingFor")
)

type ValueKind string

const (
	ValueCount    = ValueKind("count")
	ValueMoney    = ValueKind("money")
	ValuePercent  = ValueKind("percent")
	ValueDuration = ValueKind("duration")
	ValueNumber   = ValueKind("number")
	ValueText     = ValueKind("text")
)

var resourceMeasures = map[permission.Resource][]Measure{
	permission.ResourceShipment:     {MeasureLateness, MeasureRevenue},
	permission.ResourceInvoice:      {MeasureAmountDue, MeasureTotal, MeasureAge},
	permission.ResourceJournalEntry: {MeasureAmount, MeasurePendingFor},
	permission.ResourceCarrier:      nil,
	permission.ResourceCustomer:     nil,
	permission.ResourceWorker:       nil,
	permission.ResourceTractor:      nil,
	permission.ResourceTrailer:      nil,
	permission.ResourceLocation:     nil,
}

func Resources() []permission.Resource {
	return []permission.Resource{
		permission.ResourceShipment,
		permission.ResourceInvoice,
		permission.ResourceJournalEntry,
		permission.ResourceCarrier,
		permission.ResourceCustomer,
		permission.ResourceWorker,
		permission.ResourceTractor,
		permission.ResourceTrailer,
		permission.ResourceLocation,
	}
}

func Supports(resource permission.Resource) bool {
	_, ok := resourceMeasures[resource]

	return ok
}

func MeasuresOf(resource permission.Resource) []Measure {
	return resourceMeasures[resource]
}

func (m Measure) Of(resource permission.Resource) bool {
	return slices.Contains(resourceMeasures[resource], m)
}

func (m Measure) Kind() ValueKind {
	switch m {
	case MeasureLateness, MeasureAge, MeasurePendingFor:
		return ValueDuration
	case MeasureRevenue, MeasureAmountDue, MeasureTotal, MeasureAmount:
		return ValueMoney
	default:
		return ValueNumber
	}
}

func (m Measure) Adverse() bool {
	return m == MeasureLateness || m == MeasurePendingFor || m == MeasureAge ||
		m == MeasureAmountDue
}
