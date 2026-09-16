package detectors

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detector"
	"github.com/shopspring/decimal"
)

// UnbilledAgingKey identifies the delivered-but-unbilled detector.
const UnbilledAgingKey = "unbilled-aging"

// Unbilled thresholds.
//
// The age cutoff is what makes this a finding rather than a description of
// normal work: freight delivered this morning is not late to bill. Seven days is
// past any reasonable paperwork cycle, so anything older is sitting still.
//
// The amount thresholds are deliberately in money rather than shipment count. A
// hundred small shipments and one large one are not equally urgent, and the
// question this answers is how much cash is not moving.
const unbilledAgeDays = 7

var (
	unbilledWarningAmount  = decimal.NewFromInt(5_000)
	unbilledCriticalAmount = decimal.NewFromInt(25_000)
	// minUnbilledShipments stops a single stuck shipment from producing a card.
	// One is a task; a pattern is an insight.
	minUnbilledShipments = int64(3)
)

// UnbilledAging reports revenue that has been earned and not invoiced.
//
// It groups by customer because that is who the conversation is with and
// because a single customer's shipments usually stall for one reason — a missing
// signed bill of lading, a rate that never got agreed, a portal nobody has
// access to — which is a fixable thing rather than a hundred separate ones.
type UnbilledAging struct {
	metrics repositories.InsightMetricsRepository
}

func NewUnbilledAging(metrics repositories.InsightMetricsRepository) *UnbilledAging {
	return &UnbilledAging{metrics: metrics}
}

func (d *UnbilledAging) Key() string                     { return UnbilledAgingKey }
func (d *UnbilledAging) Category() insight.Category      { return insight.CategoryCashFlow }
func (d *UnbilledAging) Resource() permission.Resource   { return permission.ResourceShipment }
func (d *UnbilledAging) Operation() permission.Operation { return permission.OpRead }

// Surfaces puts unbilled revenue in front of the people who invoice.
func (d *UnbilledAging) Surfaces() []insight.Surface {
	return []insight.Surface{insight.SurfaceAccounting}
}

func (d *UnbilledAging) Explain() detector.Explanation {
	return detector.Explanation{
		Measures: "Delivered shipments that have never been billed, grouped by customer, " +
			"with the revenue they represent and the age of the oldest.",
		Threshold: "Reported when at least 3 shipments worth 5,000 or more are waiting, " +
			"and treated as critical past 25,000.",
		Excludes: "Anything delivered within the last 7 days, which is still inside a " +
			"normal paperwork cycle, and any customer whose total will not parse as a " +
			"number — reporting zero there would read as good news.",
	}
}

func (d *UnbilledAging) Detect(
	ctx context.Context,
	params detector.Params,
) ([]detector.Finding, error) {
	cutoff := params.WindowEnd - int64(unbilledAgeDays)*secondsPerDay

	rows, err := d.metrics.UnbilledDeliveredShipments(ctx, repositories.UnbilledShipmentsRequest{
		TenantInfo:      params.TenantInfo,
		DeliveredBefore: cutoff,
	})
	if err != nil {
		return nil, fmt.Errorf("read unbilled shipments: %w", err)
	}

	findings := make([]detector.Finding, 0, len(rows))
	for _, row := range rows {
		if finding, ok := d.findingFor(row, params); ok {
			findings = append(findings, finding)
		}
	}

	return findings, nil
}

func (d *UnbilledAging) findingFor(
	row repositories.UnbilledShipmentRow,
	params detector.Params,
) (detector.Finding, bool) {
	if row.ShipmentCount < minUnbilledShipments {
		return detector.Finding{}, false
	}

	amount, err := decimal.NewFromString(row.TotalAmount)
	if err != nil {
		// A sum that will not parse is a repository bug, and reporting an amount
		// of zero would be worse than saying nothing: it reads as "no money here".
		return detector.Finding{}, false
	}

	if amount.LessThan(unbilledWarningAmount) {
		return detector.Finding{}, false
	}

	oldestDays := daysBetween(row.OldestDelivery, params.WindowEnd)

	return detector.Finding{
		DedupeKey: detector.DedupeKey(UnbilledAgingKey, row.CustomerID.String()),
		Subject:   row.CustomerName,
		Headline: fmt.Sprintf(
			"%d delivered shipments for %s worth %s have not been billed, the oldest %s days ago",
			row.ShipmentCount,
			row.CustomerName,
			amount.StringFixed(2),
			oldestDays.String(),
		),
		Severity: detector.SeverityFor(amount, unbilledWarningAmount, unbilledCriticalAmount),
		Metrics: []insight.Metric{
			detector.Money(
				"unbilledAmount",
				"Delivered but unbilled",
				amount,
				insight.DirectionHigherIsWorse,
			),
			detector.Count(
				"unbilledShipments",
				"Shipments waiting",
				row.ShipmentCount,
				insight.DirectionHigherIsWorse,
			),
			detector.Days(
				"oldestAgeDays",
				"Oldest delivery",
				oldestDays,
				insight.DirectionHigherIsWorse,
			),
		},
		Links: []insight.Link{
			detector.FilteredLink(
				"Unbilled shipments",
				detector.RouteShipments,
				int(row.ShipmentCount),
				detector.FieldFilter{
					Field:    "customerId",
					Operator: detector.OpEq,
					Value:    row.CustomerID.String(),
				},
				detector.FieldFilter{
					Field:    "status",
					Operator: detector.OpEq,
					Value:    "Completed",
				},
			),
			detector.FilteredLink("Billing queue", detector.RouteBillingQueue, 0),
		},
	}, true
}

const secondsPerDay = int64(86400)

// daysBetween reports whole days elapsed, never negative. A record dated in the
// future is a clock or data problem, and reporting "-3 days old" on a card
// would put that confusion in front of a person rather than absorbing it.
func daysBetween(from, to int64) decimal.Decimal {
	if to <= from {
		return decimal.Zero
	}

	return decimal.NewFromInt((to - from) / secondsPerDay)
}
