package board

import (
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
)

var metricKinds = map[string]ValueKind{
	"activeShipments":        ValueCount,
	"revenueToday":           ValueMoney,
	"onTimePercent":          ValuePercent,
	"emptyMilePercent":       ValuePercent,
	"atRisk":                 ValueCount,
	"unassigned":             ValueCount,
	"readyToDispatch":        ValueCount,
	"marginPercent":          ValuePercent,
	"revenuePerMile":         ValueMoney,
	"costPerMile":            ValueMoney,
	"arTotalOpen":            ValueMoney,
	"arOverdue":              ValueMoney,
	"arDaysSalesOutstanding": ValueNumber,
	"movesLate":              ValueCount,
	"movesDueNow":            ValueCount,
	"movesDueToday":          ValueCount,
	"movesUncovered":         ValueCount,
}

func MetricKind(key string) (ValueKind, bool) {
	if _, known := homelayout.MetricDefinitionFor(key); !known {
		return "", false
	}
	kind, ok := metricKinds[key]

	return kind, ok
}

func MetricKeys() []string {
	catalog := homelayout.MetricCatalog()
	keys := make([]string, 0, len(catalog))
	for _, definition := range catalog {
		if _, ok := metricKinds[definition.Key]; ok {
			keys = append(keys, definition.Key)
		}
	}

	return keys
}
