// Package detectors holds the rules that find something worth saying.
//
// Each one reads aggregate rows through a repository port, applies its own
// thresholds, and returns findings with exact numbers. None of them knows that
// narration exists. They are pure enough to test against a fake repository,
// which matters: the thresholds are the judgement calls in this feature, and a
// judgement call nobody can test is a guess.
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

// OnTimeDeclineKey is the stable identifier stored on every insight this
// detector produces.
const OnTimeDeclineKey = "ontime-decline"

// On-time thresholds.
//
// The volume floor is what separates a signal from an anecdote. A customer with
// four deliveries can go from 100% to 75% because one truck hit traffic, and
// putting that on a home screen as a decline teaches people to ignore the panel.
// Twelve stops in each period is small enough to catch a real regional customer
// and large enough that one bad day cannot produce the finding on its own.
var (
	minStopsPerPeriod        = int64(12)
	onTimeWarningDropPoints  = decimal.NewFromInt(5)
	onTimeCriticalDropPoints = decimal.NewFromInt(12)
)

// OnTimeDecline reports customers whose delivery performance has fallen against
// their own recent record.
//
// The comparison is a customer against themselves rather than against a fleet
// average, because a customer with tight appointment windows in dense cities
// will always look worse than one taking drop trailers, and telling an operation
// that is not news. A customer who was at 95% last month and is at 84% this
// month is news.
type OnTimeDecline struct {
	metrics repositories.InsightMetricsRepository
}

func NewOnTimeDecline(metrics repositories.InsightMetricsRepository) *OnTimeDecline {
	return &OnTimeDecline{metrics: metrics}
}

func (d *OnTimeDecline) Key() string                     { return OnTimeDeclineKey }
func (d *OnTimeDecline) Category() insight.Category      { return insight.CategoryServiceQuality }
func (d *OnTimeDecline) Resource() permission.Resource   { return permission.ResourceShipment }
func (d *OnTimeDecline) Operation() permission.Operation { return permission.OpRead }

func (d *OnTimeDecline) Detect(
	ctx context.Context,
	params detector.Params,
) ([]detector.Finding, error) {
	rows, err := d.metrics.CustomerOnTimeComparison(ctx, repositories.InsightWindowRequest{
		TenantInfo:  params.TenantInfo,
		WindowStart: params.WindowStart,
		WindowEnd:   params.WindowEnd,
	})
	if err != nil {
		return nil, fmt.Errorf("read on-time comparison: %w", err)
	}

	findings := make([]detector.Finding, 0, len(rows))
	for _, row := range rows {
		if finding, ok := d.findingFor(row, params); ok {
			findings = append(findings, finding)
		}
	}

	return findings, nil
}

func (d *OnTimeDecline) findingFor(
	row repositories.CustomerOnTimeRow,
	params detector.Params,
) (detector.Finding, bool) {
	// Both periods need enough volume. A customer who shipped nothing last month
	// has not declined, they have started, and comparing against an empty period
	// produces a 100-point drop out of nowhere.
	if row.CurrentTotal < minStopsPerPeriod || row.PriorTotal < minStopsPerPeriod {
		return detector.Finding{}, false
	}

	current := percentage(row.CurrentOnTime, row.CurrentTotal)
	prior := percentage(row.PriorOnTime, row.PriorTotal)
	drop := prior.Sub(current)

	if drop.LessThan(onTimeWarningDropPoints) {
		return detector.Finding{}, false
	}

	lateStops := row.CurrentTotal - row.CurrentOnTime

	return detector.Finding{
		DedupeKey: detector.DedupeKey(OnTimeDeclineKey, row.CustomerID.String()),
		Subject:   row.CustomerName,
		Headline: fmt.Sprintf(
			"On-time delivery for %s is %s%%, down from %s%% the previous %d days",
			row.CustomerName,
			current.StringFixed(1),
			prior.StringFixed(1),
			params.WindowDays(),
		),
		Severity: detector.SeverityFor(drop, onTimeWarningDropPoints, onTimeCriticalDropPoints),
		Metrics: []insight.Metric{
			detector.WithBaseline(
				detector.Percent(
					"onTimePercent",
					"On-time delivery",
					current,
					insight.DirectionLowerIsWorse,
				),
				prior,
				fmt.Sprintf("previous %d days", params.WindowDays()),
			),
			detector.Count(
				"lateStops",
				"Late deliveries",
				lateStops,
				insight.DirectionHigherIsWorse,
			),
			detector.Count(
				"deliveryStops",
				"Deliveries in period",
				row.CurrentTotal,
				insight.DirectionNeutral,
			),
		},
		Links: []insight.Link{
			detector.FilteredLink(
				"Shipments for this customer",
				detector.RouteShipments,
				int(row.CurrentTotal),
				detector.FieldFilter{
					Field:    "customerId",
					Operator: detector.OpEq,
					Value:    row.CustomerID.String(),
				},
			),
			detector.FilteredLink(
				"Service failures",
				detector.RouteServiceFailures,
				0,
				detector.FieldFilter{
					Field:    "customerId",
					Operator: detector.OpEq,
					Value:    row.CustomerID.String(),
				},
			),
		},
	}, true
}

// percentage computes a share as a percentage, guarding the empty denominator
// that a group with no activity would otherwise produce.
func percentage(part, total int64) decimal.Decimal {
	if total == 0 {
		return decimal.Zero
	}

	return decimal.NewFromInt(part).
		Div(decimal.NewFromInt(total)).
		Mul(decimal.NewFromInt(100)).
		Round(1)
}
