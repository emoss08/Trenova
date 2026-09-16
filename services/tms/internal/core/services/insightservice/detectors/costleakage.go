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

const (
	// UnbilledDetentionKey identifies the detention-not-billed detector.
	UnbilledDetentionKey = "unbilled-detention"
	// EmptyMilesKey identifies the empty-mile concentration detector.
	EmptyMilesKey = "empty-miles"
)

// Detention thresholds. These are money because detention is money: the dwell
// itself is only interesting insofar as it was billable and went unbilled.
var (
	detentionWarningAmount  = decimal.NewFromInt(1_000)
	detentionCriticalAmount = decimal.NewFromInt(7_500)
	// minDetentionOccurrences keeps a single bad afternoon at one consignee from
	// becoming a standing card about that location.
	minDetentionOccurrences = int64(3)
)

// UnbilledDetention reports detention that accrued and never became a charge.
//
// This is the most common quiet loss in a truckload operation: the driver sat,
// the clock ran, the occurrence was recorded, and nobody raised the accessorial
// because the notice deadline passed or somebody waived it to keep a customer
// happy. Grouping by location rather than customer is deliberate — the fix is
// usually a scheduling or facility problem at one door, and the same shipper can
// be fine at three sites and impossible at a fourth.
type UnbilledDetention struct {
	metrics repositories.InsightMetricsRepository
}

func NewUnbilledDetention(metrics repositories.InsightMetricsRepository) *UnbilledDetention {
	return &UnbilledDetention{metrics: metrics}
}

func (d *UnbilledDetention) Key() string                   { return UnbilledDetentionKey }
func (d *UnbilledDetention) Category() insight.Category    { return insight.CategoryCostLeakage }
func (d *UnbilledDetention) Resource() permission.Resource { return permission.ResourceShipment }
func (d *UnbilledDetention) Operation() permission.Operation {
	return permission.OpRead
}

func (d *UnbilledDetention) Detect(
	ctx context.Context,
	params detector.Params,
) ([]detector.Finding, error) {
	rows, err := d.metrics.UnbilledDetention(ctx, repositories.InsightWindowRequest{
		TenantInfo:  params.TenantInfo,
		WindowStart: params.WindowStart,
		WindowEnd:   params.WindowEnd,
	})
	if err != nil {
		return nil, fmt.Errorf("read unbilled detention: %w", err)
	}

	findings := make([]detector.Finding, 0, len(rows))
	for _, row := range rows {
		if finding, ok := d.findingFor(row, params); ok {
			findings = append(findings, finding)
		}
	}

	return findings, nil
}

func (d *UnbilledDetention) findingFor(
	row repositories.UnbilledDetentionRow,
	params detector.Params,
) (detector.Finding, bool) {
	if row.OccurrenceCount < minDetentionOccurrences {
		return detector.Finding{}, false
	}

	amount, err := decimal.NewFromString(row.UnbilledAmount)
	if err != nil || amount.LessThan(detentionWarningAmount) {
		return detector.Finding{}, false
	}

	dwellHours := decimal.NewFromInt(row.TotalDwellMinutes).Div(decimal.NewFromInt(60)).Round(1)

	return detector.Finding{
		DedupeKey: detector.DedupeKey(UnbilledDetentionKey, row.LocationID.String()),
		Subject:   row.LocationName,
		Headline: fmt.Sprintf(
			"%s in detention accrued at %s over %d stops and was never billed",
			amount.StringFixed(2),
			row.LocationName,
			row.OccurrenceCount,
		),
		Severity: detector.SeverityFor(amount, detentionWarningAmount, detentionCriticalAmount),
		Metrics: []insight.Metric{
			detector.Money(
				"unbilledDetention",
				"Detention not billed",
				amount,
				insight.DirectionHigherIsWorse,
			),
			detector.Count(
				"detentionStops",
				"Stops that went into detention",
				row.OccurrenceCount,
				insight.DirectionHigherIsWorse,
			),
			detector.Hours(
				"dwellHours",
				"Total time at the dock",
				dwellHours,
				insight.DirectionHigherIsWorse,
			),
		},
		Links: []insight.Link{
			detector.FilteredLink(
				"Shipments through this location",
				detector.RouteShipments,
				int(row.OccurrenceCount),
			),
		},
	}, true
}

// Empty-mile thresholds, in percentage points above the organization's own
// average rather than an absolute ratio.
//
// An absolute threshold would be wrong for every operation but one. A dedicated
// fleet running out-and-back lanes lives at 8% empty and a brokered spot
// operation at 25%, and neither is a problem in itself. A customer running ten
// points worse than the rest of that same operation's book is the thing worth
// looking at.
var (
	emptyMileWarningSpread  = decimal.NewFromInt(8)
	emptyMileCriticalSpread = decimal.NewFromInt(18)
	// minMovesForEmptyMiles is the volume floor. Two moves can be 60% empty by
	// accident of geography.
	minMovesForEmptyMiles = int64(10)
)

// EmptyMiles reports customers whose freight runs materially emptier than the
// rest of the book.
type EmptyMiles struct {
	metrics repositories.InsightMetricsRepository
}

func NewEmptyMiles(metrics repositories.InsightMetricsRepository) *EmptyMiles {
	return &EmptyMiles{metrics: metrics}
}

func (d *EmptyMiles) Key() string                     { return EmptyMilesKey }
func (d *EmptyMiles) Category() insight.Category      { return insight.CategoryCostLeakage }
func (d *EmptyMiles) Resource() permission.Resource   { return permission.ResourceShipment }
func (d *EmptyMiles) Operation() permission.Operation { return permission.OpRead }

func (d *EmptyMiles) Detect(
	ctx context.Context,
	params detector.Params,
) ([]detector.Finding, error) {
	rows, err := d.metrics.CustomerEmptyMiles(ctx, repositories.InsightWindowRequest{
		TenantInfo:  params.TenantInfo,
		WindowStart: params.WindowStart,
		WindowEnd:   params.WindowEnd,
	})
	if err != nil {
		return nil, fmt.Errorf("read empty miles: %w", err)
	}

	findings := make([]detector.Finding, 0, len(rows))
	for _, row := range rows {
		if finding, ok := d.findingFor(row, params); ok {
			findings = append(findings, finding)
		}
	}

	return findings, nil
}

func (d *EmptyMiles) findingFor(
	row repositories.CustomerEmptyMilesRow,
	params detector.Params,
) (detector.Finding, bool) {
	if row.MoveCount < minMovesForEmptyMiles {
		return detector.Finding{}, false
	}

	customerShare, ok := share(row.EmptyMiles, row.TotalMiles)
	if !ok {
		return detector.Finding{}, false
	}

	orgShare, ok := share(row.OrgEmptyMiles, row.OrgTotalMiles)
	if !ok {
		return detector.Finding{}, false
	}

	spread := customerShare.Sub(orgShare)
	if spread.LessThan(emptyMileWarningSpread) {
		return detector.Finding{}, false
	}

	emptyMiles, err := decimal.NewFromString(row.EmptyMiles)
	if err != nil {
		return detector.Finding{}, false
	}

	return detector.Finding{
		DedupeKey: detector.DedupeKey(EmptyMilesKey, row.CustomerID.String()),
		Subject:   row.CustomerName,
		Headline: fmt.Sprintf(
			"%s runs %s%% empty against a fleet average of %s%%",
			row.CustomerName,
			customerShare.StringFixed(1),
			orgShare.StringFixed(1),
		),
		Severity: detector.SeverityFor(spread, emptyMileWarningSpread, emptyMileCriticalSpread),
		Metrics: []insight.Metric{
			detector.WithBaseline(
				detector.Percent(
					"emptyMilePercent",
					"Empty miles",
					customerShare,
					insight.DirectionHigherIsWorse,
				),
				orgShare,
				"fleet average",
			),
			detector.Miles(
				"emptyMiles",
				"Empty miles run",
				emptyMiles.Round(0),
				insight.DirectionHigherIsWorse,
			),
			detector.Count(
				"moves",
				"Moves in period",
				row.MoveCount,
				insight.DirectionNeutral,
			),
		},
		Links: []insight.Link{
			detector.FilteredLink(
				"Shipments for this customer",
				detector.RouteShipments,
				int(row.MoveCount),
				detector.FieldFilter{
					Field:    "customerId",
					Operator: detector.OpEq,
					Value:    row.CustomerID.String(),
				},
			),
		},
	}, true
}

// share computes one distance as a percentage of another, from the string forms
// the repository returns. It reports false rather than zero when the numbers
// cannot be read or the denominator is empty, so a parse failure never becomes a
// confident-looking 0%.
func share(part, total string) (decimal.Decimal, bool) {
	partValue, err := decimal.NewFromString(part)
	if err != nil {
		return decimal.Zero, false
	}

	totalValue, err := decimal.NewFromString(total)
	if err != nil || totalValue.IsZero() {
		return decimal.Zero, false
	}

	return partValue.Div(totalValue).Mul(decimal.NewFromInt(100)).Round(1), true
}
