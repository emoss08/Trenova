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

type resourceMeasure struct {
	resource permission.Resource
	measures []Measure
}

var resourceMeasures = []resourceMeasure{
	{resource: permission.ResourceShipment, measures: []Measure{MeasureLateness, MeasureRevenue}},
	{
		resource: permission.ResourceInvoice,
		measures: []Measure{MeasureAmountDue, MeasureTotal, MeasureAge},
	},
	{
		resource: permission.ResourceJournalEntry,
		measures: []Measure{MeasureAmount, MeasurePendingFor},
	},
	{resource: permission.ResourceCarrier},
	{resource: permission.ResourceCustomer},
	{resource: permission.ResourceWorker},
	{resource: permission.ResourceTractor},
	{resource: permission.ResourceTrailer},
	{resource: permission.ResourceLocation},
}

func Resources() []permission.Resource {
	resources := make([]permission.Resource, 0, len(resourceMeasures))
	for idx := range resourceMeasures {
		resources = append(resources, resourceMeasures[idx].resource)
	}

	return resources
}

func Supports(resource permission.Resource) bool {
	return slices.ContainsFunc(resourceMeasures, func(entry resourceMeasure) bool {
		return entry.resource == resource
	})
}

func MeasuresOf(resource permission.Resource) []Measure {
	for idx := range resourceMeasures {
		if resourceMeasures[idx].resource == resource {
			return resourceMeasures[idx].measures
		}
	}

	return nil
}

func (m Measure) Of(resource permission.Resource) bool {
	return slices.Contains(MeasuresOf(resource), m)
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
