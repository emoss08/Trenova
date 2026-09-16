package detector

import (
	"net/url"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/shopspring/decimal"
)

// Metric constructors.
//
// These exist so a detector states what a number means at the point it computes
// it. A bare decimal reaching the client with a key like "value" forces the
// renderer to guess at a currency symbol, and it guesses wrong.

func Money(key, label string, value decimal.Decimal, direction insight.Direction) insight.Metric {
	return insight.Metric{
		Key:       key,
		Label:     label,
		Value:     value,
		Unit:      insight.UnitCurrency,
		Direction: direction,
	}
}

func Percent(key, label string, value decimal.Decimal, direction insight.Direction) insight.Metric {
	return insight.Metric{
		Key:       key,
		Label:     label,
		Value:     value,
		Unit:      insight.UnitPercent,
		Direction: direction,
	}
}

func Count(key, label string, value int64, direction insight.Direction) insight.Metric {
	return insight.Metric{
		Key:       key,
		Label:     label,
		Value:     decimal.NewFromInt(value),
		Unit:      insight.UnitCount,
		Direction: direction,
	}
}

func Hours(key, label string, value decimal.Decimal, direction insight.Direction) insight.Metric {
	return insight.Metric{
		Key:       key,
		Label:     label,
		Value:     value,
		Unit:      insight.UnitHours,
		Direction: direction,
	}
}

func Days(key, label string, value decimal.Decimal, direction insight.Direction) insight.Metric {
	return insight.Metric{
		Key:       key,
		Label:     label,
		Value:     value,
		Unit:      insight.UnitDays,
		Direction: direction,
	}
}

func Miles(key, label string, value decimal.Decimal, direction insight.Direction) insight.Metric {
	return insight.Metric{
		Key:       key,
		Label:     label,
		Value:     value,
		Unit:      insight.UnitMiles,
		Direction: direction,
	}
}

// WithBaseline attaches the comparison a metric should be read against.
//
// A number on its own rarely says anything: 88% on-time is excellent or alarming
// depending on what it was last month and what was promised. The label is
// required because "vs 94%" without saying what 94% was is worse than no
// comparison at all.
func WithBaseline(metric insight.Metric, baseline decimal.Decimal, label string) insight.Metric {
	metric.Baseline = &baseline
	metric.BaselineLabel = label

	return metric
}

// DedupeKey builds the identity of a finding from its detector and subject.
//
// Parts are joined with a separator that cannot appear in an identifier, so
// two different subjects cannot collide into one key and silently supersede each
// other on every refresh.
func DedupeKey(detectorKey string, parts ...string) string {
	cleaned := make([]string, 0, len(parts)+1)
	cleaned = append(cleaned, detectorKey)

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, strings.ReplaceAll(trimmed, ":", "_"))
	}

	return strings.Join(cleaned, ":")
}

// FieldFilter mirrors one entry of the client's table filter state.
//
// It is duplicated here rather than imported because the client owns that type
// and this is the server building a URL the client will parse. The field names
// are the contract between them.
type FieldFilter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

// Filter operators understood by the client's table state. Only the handful a
// detector needs is spelled out; the client accepts more.
const (
	OpEq   = "eq"
	OpIn   = "in"
	OpGte  = "gte"
	OpLte  = "lte"
	OpLt   = "lt"
	OpGt   = "gt"
	OpNeq  = "ne"
	OpNull = "isnull"
)

// Client routes a detector may link to. Naming them here means a route rename
// breaks one list rather than being scattered through detector files as string
// literals, and it keeps a detector from inventing a path that does not exist.
const (
	RouteShipments       = "/shipment-management/shipments"
	RouteServiceFailures = "/shipment-management/service-failures"
	RouteBillingQueue    = "/billing/queue"
	RouteCustomers       = "/billing/configuration-files/customers"
	RouteWorkers         = "/hr/workers"
)

// FilteredLink builds a link into one of the application's tables with its
// filter state pre-applied.
//
// The whole point of an insight is that a person can go see the records behind
// it, and "go filter this yourself" is where that intent usually dies. The
// filters are encoded exactly as the table's own URL state encodes them, so the
// destination opens showing the same rows the detector counted.
//
// A filter set that cannot be encoded yields a link to the unfiltered table
// rather than no link: sending someone to the right screen is still better than
// leaving them with a number and no way in.
func FilteredLink(label, route string, count int, filters ...FieldFilter) insight.Link {
	link := insight.Link{Label: label, Path: route, Count: count}
	if len(filters) == 0 {
		return link
	}

	encoded, err := sonic.MarshalString(filters)
	if err != nil {
		return link
	}

	link.Path = route + "?fieldFilters=" + url.QueryEscape(encoded)

	return link
}

// SeverityFor grades a value against two thresholds.
//
// Thresholds are the detector's own judgement about its own metric, expressed
// once. The alternative — every detector writing its own if-ladder — is where
// inconsistent urgency creeps in, and urgency is what decides which card a
// person reads first.
//
// Both thresholds are inclusive lower bounds for a metric where a higher number
// is worse. Use SeverityForDescending when smaller is the problem.
func SeverityFor(value, warning, critical decimal.Decimal) insight.Severity {
	switch {
	case value.GreaterThanOrEqual(critical):
		return insight.SeverityCritical
	case value.GreaterThanOrEqual(warning):
		return insight.SeverityWarning
	default:
		return insight.SeverityInfo
	}
}

// SeverityForDescending grades a value where a smaller number is the problem,
// such as an on-time percentage.
func SeverityForDescending(value, warning, critical decimal.Decimal) insight.Severity {
	switch {
	case value.LessThanOrEqual(critical):
		return insight.SeverityCritical
	case value.LessThanOrEqual(warning):
		return insight.SeverityWarning
	default:
		return insight.SeverityInfo
	}
}
