package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/services/insightservice"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detector"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detectors"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	insightSeedHour = int64(3600)
	insightSeedDay  = 24 * insightSeedHour
	// insightSeedRefreshGap is the spacing between the history rows behind a
	// finding, matching the scheduled refresh cadence so a trend reads as one.
	insightSeedRefreshGap = 6 * insightSeedHour
)

type InsightSeed struct {
	seedhelpers.BaseSeed
}

// InsightSeed writes findings in every category and severity, with the
// superseded rows a refresh leaves behind so the detail view has a trend to
// draw, one dismissed finding so the dismissed filter is not empty, and one the
// system resolved. Some are narrated and attributed to the seeded local
// provider; the rest carry detector wording, so both marks appear.
//
// The numbers are invented rather than computed: this seed exists to show what
// the pages look like, and the scheduled refresh replaces every active row here
// with real findings the first time it runs.
//
// Depends on:
//   - Shipment: the customers the findings name
//   - Location: the doors the detention findings name
//   - AIProvider: the provider the narrated findings are attributed to
func NewInsightSeed() *InsightSeed {
	seed := &InsightSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"Insight",
		"1.0.0",
		"Seeds operational insights with history, a dismissal and a resolution",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(
		seedhelpers.SeedShipment,
		seedhelpers.SeedLocation,
		seedhelpers.SeedAIProvider,
	)
	return seed
}

type insightSeedRefs struct {
	org       *tenant.Organization
	admin     *tenant.User
	customers []*customer.Customer
	locations []*location.Location
	provider  *aiprovider.Provider
	now       int64
}

func (s *InsightSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			refs, err := s.loadRefs(ctx, tx, sc)
			if err != nil {
				return err
			}

			cols := buncolgen.InsightColumns
			count, err := tx.NewSelect().
				Model((*insight.Insight)(nil)).
				Where(cols.OrganizationID.Eq(), refs.org.ID).
				Where(cols.BusinessUnitID.Eq(), refs.org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("count existing insights: %w", err)
			}
			if count > 0 {
				return nil
			}

			for _, series := range s.series(refs) {
				if err = s.insertSeries(ctx, tx, sc, refs, series); err != nil {
					return fmt.Errorf("insert %s: %w", series.current.DedupeKey, err)
				}
			}

			return nil
		},
	)
}

func (s *InsightSeed) loadRefs(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
) (*insightSeedRefs, error) {
	org, err := sc.GetDefaultOrganization(ctx)
	if err != nil {
		return nil, err
	}
	admin, err := sc.GetUserByUsername(ctx, "admin")
	if err != nil {
		return nil, fmt.Errorf("get admin user: %w", err)
	}

	refs := &insightSeedRefs{org: org, admin: admin, now: timeutils.NowUnix()}

	customerCols := buncolgen.CustomerColumns
	refs.customers = make([]*customer.Customer, 0, 3)
	if err = tx.NewSelect().
		Model(&refs.customers).
		Where(customerCols.OrganizationID.Eq(), org.ID).
		Where(customerCols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Order(customerCols.Name.OrderAsc()).
		Limit(3).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load customers: %w", err)
	}
	if len(refs.customers) < 3 {
		return nil, fmt.Errorf("need three seeded customers: %w", seedhelpers.ErrEntityNotFound)
	}

	locationCols := buncolgen.LocationColumns
	refs.locations = make([]*location.Location, 0, 2)
	if err = tx.NewSelect().
		Model(&refs.locations).
		Where(locationCols.OrganizationID.Eq(), org.ID).
		Where(locationCols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Order(locationCols.Name.OrderAsc()).
		Limit(2).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load locations: %w", err)
	}
	if len(refs.locations) < 2 {
		return nil, fmt.Errorf("need two seeded locations: %w", seedhelpers.ErrEntityNotFound)
	}

	providerCols := buncolgen.ProviderColumns
	refs.provider = new(aiprovider.Provider)
	if err = tx.NewSelect().
		Model(refs.provider).
		Where(providerCols.OrganizationID.Eq(), org.ID).
		Where(providerCols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(providerCols.Name.Eq(), SeedAIProviderLocalName).
		Limit(1).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load seeded provider: %w", err)
	}

	return refs, nil
}

// insightSeries is one finding as it stands now plus the earlier readings a
// refresh superseded, oldest first. Each history entry is the metrics as they
// were; everything else is copied from the current row.
type insightSeries struct {
	current *insight.Insight
	history [][]insight.Metric
}

func (s *InsightSeed) insertSeries(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *insightSeedRefs,
	series insightSeries,
) error {
	for index, metrics := range series.history {
		age := int64(len(series.history)-index) * insightSeedRefreshGap
		past := *series.current
		past.ID = ""
		past.Status = insight.StatusSuperseded
		past.Metrics = metrics
		past.DetectedAt = series.current.DetectedAt - age
		past.StaleAt = past.DetectedAt + int64(insightservice.StaleAfter.Seconds())
		past.WindowStart = series.current.WindowStart - age
		past.WindowEnd = series.current.WindowEnd - age
		past.DismissedAt = nil
		past.DismissedByID = ""
		past.DismissReason = ""

		if err := s.insert(ctx, tx, sc, &past); err != nil {
			return err
		}
	}

	return s.insert(ctx, tx, sc, series.current)
}

func (s *InsightSeed) insert(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	entity *insight.Insight,
) error {
	if _, err := tx.NewInsert().Model(entity).Exec(ctx); err != nil {
		return err
	}

	return sc.TrackCreated(ctx, "insights", entity.ID, s.Name())
}

func (s *InsightSeed) base(refs *insightSeedRefs, detectedAgo int64) insight.Insight {
	detectedAt := refs.now - detectedAgo

	return insight.Insight{
		OrganizationID: refs.org.ID,
		BusinessUnitID: refs.org.BusinessUnitID,
		Status:         insight.StatusActive,
		WindowStart:    detectedAt - int64(insightservice.DefaultWindow.Seconds()),
		WindowEnd:      detectedAt,
		DetectedAt:     detectedAt,
		StaleAt:        detectedAt + int64(insightservice.StaleAfter.Seconds()),
	}
}

func (s *InsightSeed) narrated(entity *insight.Insight, refs *insightSeedRefs) *insight.Insight {
	entity.Narrated = true
	entity.ModelIdentifier = refs.provider.Model
	entity.ProviderID = refs.provider.ID

	return entity
}

func customerFilter(id pulid.ID) detector.FieldFilter {
	return detector.FieldFilter{Field: "customerId", Operator: detector.OpEq, Value: id.String()}
}

func (s *InsightSeed) series(refs *insightSeedRefs) []insightSeries {
	series := make([]insightSeries, 0, 8)
	series = append(series, s.onTimeSeries(refs)...)
	series = append(series, s.cashSeries(refs)...)
	series = append(series, s.costSeries(refs)...)
	series = append(series, s.complianceSeries(refs))

	return series
}

func (s *InsightSeed) onTimeSeries(refs *insightSeedRefs) []insightSeries {
	acme := refs.customers[0]
	peak := refs.customers[2]

	onTime := func(current, prior float64, stops, late int64) []insight.Metric {
		return []insight.Metric{
			detector.WithBaseline(
				detector.Percent("onTimePercent", "On-time delivery",
					decimal.NewFromFloat(current), insight.DirectionLowerIsWorse),
				decimal.NewFromFloat(prior), "prior 30 days",
			),
			detector.Count("stops", "Deliveries measured", stops, insight.DirectionNeutral),
			detector.Count("lateStops", "Late deliveries", late, insight.DirectionHigherIsWorse),
		}
	}

	critical := s.base(refs, 2*insightSeedHour)
	critical.DetectorKey = detectors.OnTimeDeclineKey
	critical.Category = insight.CategoryServiceQuality
	critical.Severity = insight.SeverityCritical
	critical.DedupeKey = detector.DedupeKey(detectors.OnTimeDeclineKey, acme.ID.String())
	critical.Subject = acme.Name
	critical.Headline = fmt.Sprintf("On-time delivery for %s has fallen 14 points to 79.6%%", acme.Name)
	critical.Narrative = fmt.Sprintf(
		"%s took 54 deliveries this month and 11 arrived late, against 4 of 51 in the "+
			"prior period. The slide began three weeks ago and has not recovered.",
		acme.Name,
	)
	critical.Recommendation = "Pull the late stops for this customer and check whether one " +
		"lane or one receiving window accounts for most of them before the next review call."
	critical.Metrics = onTime(79.6, 93.7, 54, 11)
	critical.Links = []insight.Link{
		detector.FilteredLink("Late deliveries", detector.RouteServiceFailures, 11, customerFilter(acme.ID)),
		detector.FilteredLink("Customer", detector.RouteCustomers, 0, detector.FieldFilter{
			Field: "id", Operator: detector.OpEq, Value: acme.ID.String(),
		}),
	}

	warning := s.base(refs, 2*insightSeedHour)
	warning.DetectorKey = detectors.OnTimeDeclineKey
	warning.Category = insight.CategoryServiceQuality
	warning.Severity = insight.SeverityWarning
	warning.DedupeKey = detector.DedupeKey(detectors.OnTimeDeclineKey, peak.ID.String())
	warning.Subject = peak.Name
	warning.Headline = fmt.Sprintf("On-time delivery for %s fell from 96.2%% to 89.5%%", peak.Name)
	warning.Metrics = onTime(89.5, 96.2, 38, 4)
	warning.Links = []insight.Link{
		detector.FilteredLink("Late deliveries", detector.RouteServiceFailures, 4, customerFilter(peak.ID)),
	}

	return []insightSeries{
		{
			current: s.narrated(&critical, refs),
			history: [][]insight.Metric{
				onTime(91.1, 93.7, 49, 4),
				onTime(86.4, 93.7, 51, 7),
				onTime(82.0, 93.7, 53, 9),
			},
		},
		{current: &warning, history: [][]insight.Metric{onTime(92.3, 96.2, 36, 3)}},
	}
}

func (s *InsightSeed) cashSeries(refs *insightSeedRefs) []insightSeries {
	global := refs.customers[1]
	acme := refs.customers[0]

	unbilled := func(amount float64, shipments, oldest int64) []insight.Metric {
		return []insight.Metric{
			detector.Money("unbilledAmount", "Delivered but unbilled",
				decimal.NewFromFloat(amount), insight.DirectionHigherIsWorse),
			detector.Count("unbilledShipments", "Shipments waiting", shipments, insight.DirectionHigherIsWorse),
			detector.Days("oldestAgeDays", "Oldest delivery",
				decimal.NewFromInt(oldest), insight.DirectionHigherIsWorse),
		}
	}
	links := func(id pulid.ID, count int) []insight.Link {
		return []insight.Link{
			detector.FilteredLink("Unbilled shipments", detector.RouteShipments, count,
				customerFilter(id),
				detector.FieldFilter{Field: "status", Operator: detector.OpEq, Value: "Completed"},
			),
			detector.FilteredLink("Billing queue", detector.RouteBillingQueue, 0),
		}
	}

	critical := s.base(refs, 2*insightSeedHour)
	critical.DetectorKey = detectors.UnbilledAgingKey
	critical.Category = insight.CategoryCashFlow
	critical.Severity = insight.SeverityCritical
	critical.DedupeKey = detector.DedupeKey(detectors.UnbilledAgingKey, global.ID.String())
	critical.Subject = global.Name
	critical.Headline = fmt.Sprintf("$31,480 delivered for %s is still unbilled, the oldest 23 days ago", global.Name)
	critical.Narrative = fmt.Sprintf(
		"Fourteen shipments for %s have been delivered and never invoiced. The amount has "+
			"grown every refresh for five days, which usually means one missing document "+
			"is holding a whole batch.",
		global.Name,
	)
	critical.Recommendation = "Open the billing queue filtered to this customer and clear the " +
		"oldest item first; the rest usually share its blocker."
	critical.Metrics = unbilled(31480, 14, 23)
	critical.Links = links(global.ID, 14)

	dismissedAt := refs.now - 4*insightSeedDay
	dismissed := s.base(refs, 4*insightSeedDay+insightSeedHour)
	dismissed.DetectorKey = detectors.UnbilledAgingKey
	dismissed.Category = insight.CategoryCashFlow
	dismissed.Severity = insight.SeverityWarning
	dismissed.Status = insight.StatusDismissed
	dismissed.DedupeKey = detector.DedupeKey(detectors.UnbilledAgingKey, acme.ID.String())
	dismissed.Subject = acme.Name
	dismissed.Headline = fmt.Sprintf("3 delivered shipments for %s worth $6,210 have not been billed", acme.Name)
	dismissed.Metrics = unbilled(6210, 3, 9)
	dismissed.Links = links(acme.ID, 3)
	dismissed.DismissedAt = &dismissedAt
	dismissed.DismissedByID = refs.admin.ID
	dismissed.DismissReason = "Consolidated monthly invoice — bills on the 1st by agreement"

	return []insightSeries{
		{
			current: s.narrated(&critical, refs),
			history: [][]insight.Metric{
				unbilled(12950, 6, 12),
				unbilled(18400, 8, 15),
				unbilled(24120, 11, 19),
				unbilled(27900, 12, 21),
			},
		},
		{current: &dismissed},
	}
}

func (s *InsightSeed) costSeries(refs *insightSeedRefs) []insightSeries {
	door := refs.locations[0]
	resolvedDoor := refs.locations[1]
	peak := refs.customers[2]

	detention := func(amount float64, occurrences int64, dwell float64) []insight.Metric {
		return []insight.Metric{
			detector.Money("unbilledDetention", "Detention never billed",
				decimal.NewFromFloat(amount), insight.DirectionHigherIsWorse),
			detector.Count("occurrences", "Occurrences", occurrences, insight.DirectionHigherIsWorse),
			detector.Hours("dwellHours", "Dwell past free time",
				decimal.NewFromFloat(dwell), insight.DirectionHigherIsWorse),
		}
	}

	warning := s.base(refs, 2*insightSeedHour)
	warning.DetectorKey = detectors.UnbilledDetentionKey
	warning.Category = insight.CategoryCostLeakage
	warning.Severity = insight.SeverityWarning
	warning.DedupeKey = detector.DedupeKey(detectors.UnbilledDetentionKey, door.ID.String())
	warning.Subject = door.Name
	warning.Headline = fmt.Sprintf("$4,350 of detention at %s was never billed across 7 visits", door.Name)
	warning.Narrative = fmt.Sprintf(
		"Drivers waited 29 hours past free time at %s this month and no accessorial was "+
			"raised for any of it. Every occurrence fell on a Monday or Friday.",
		door.Name,
	)
	warning.Recommendation = "Check whether the notice deadline is being missed for this " +
		"door, and whether the receiving schedule there needs a later appointment."
	warning.Metrics = detention(4350, 7, 29)
	warning.Links = []insight.Link{
		detector.FilteredLink("Shipments at this location", detector.RouteShipments, 7,
			detector.FieldFilter{Field: "destinationLocationId", Operator: detector.OpEq, Value: door.ID.String()},
		),
	}

	resolved := s.base(refs, 6*insightSeedHour+insightSeedRefreshGap)
	resolved.DetectorKey = detectors.UnbilledDetentionKey
	resolved.Category = insight.CategoryCostLeakage
	resolved.Severity = insight.SeverityInfo
	resolved.Status = insight.StatusResolved
	resolved.DedupeKey = detector.DedupeKey(detectors.UnbilledDetentionKey, resolvedDoor.ID.String())
	resolved.Subject = resolvedDoor.Name
	resolved.Headline = fmt.Sprintf("$1,120 of detention at %s was never billed across 3 visits", resolvedDoor.Name)
	resolved.Metrics = detention(1120, 3, 7.5)

	emptyMiles := func(share, orgShare, miles float64, moves int64) []insight.Metric {
		return []insight.Metric{
			detector.WithBaseline(
				detector.Percent("emptyShare", "Empty-mile share",
					decimal.NewFromFloat(share), insight.DirectionHigherIsWorse),
				decimal.NewFromFloat(orgShare), "fleet average",
			),
			detector.Miles("emptyMiles", "Empty miles",
				decimal.NewFromFloat(miles), insight.DirectionHigherIsWorse),
			detector.Count("moves", "Moves", moves, insight.DirectionNeutral),
		}
	}

	empty := s.base(refs, 2*insightSeedHour)
	empty.DetectorKey = detectors.EmptyMilesKey
	empty.Category = insight.CategoryCostLeakage
	empty.Severity = insight.SeverityInfo
	empty.DedupeKey = detector.DedupeKey(detectors.EmptyMilesKey, peak.ID.String())
	empty.Subject = peak.Name
	empty.Headline = fmt.Sprintf("%s runs 31.4%% empty miles against a fleet average of 18.9%%", peak.Name)
	empty.Metrics = emptyMiles(31.4, 18.9, 2860, 41)
	empty.Links = []insight.Link{
		detector.FilteredLink("Moves for this customer", detector.RouteShipments, 41, customerFilter(peak.ID)),
	}

	return []insightSeries{
		{
			current: s.narrated(&warning, refs),
			history: [][]insight.Metric{detention(2100, 4, 13), detention(3300, 6, 22)},
		},
		{current: &resolved},
		{current: &empty, history: [][]insight.Metric{emptyMiles(27.8, 19.2, 2210, 36)}},
	}
}

func (s *InsightSeed) complianceSeries(refs *insightSeedRefs) insightSeries {
	credentials := func(workers, expired int64, earliest int64) []insight.Metric {
		return []insight.Metric{
			detector.Count("workers", "Drivers affected", workers, insight.DirectionHigherIsWorse),
			detector.Count("alreadyExpired", "Already expired", expired, insight.DirectionHigherIsWorse),
			detector.Days("daysToEarliest", "Days to earliest expiry",
				decimal.NewFromInt(earliest), insight.DirectionLowerIsWorse),
		}
	}

	entity := s.base(refs, 2*insightSeedHour)
	entity.DetectorKey = detectors.CredentialExpiryKey
	entity.Category = insight.CategoryCompliance
	entity.Severity = insight.SeverityCritical
	entity.DedupeKey = detector.DedupeKey(detectors.CredentialExpiryKey, "MED_CARD")
	entity.Subject = "DOT Medical Card"
	entity.Headline = "5 drivers have a medical card expiring within 45 days and 1 has already lapsed"
	entity.Narrative = "Five medical cards fall due in the next six weeks and one driver is " +
		"already past the date, which takes that truck off the road today. Renewals " +
		"typically take three to four weeks once an exam is booked."
	entity.Recommendation = "Book the lapsed driver's exam first, then the two due inside " +
		"two weeks; the remaining three can be scheduled with the next safety meeting."
	entity.Metrics = credentials(5, 1, -2)
	entity.Links = []insight.Link{
		detector.FilteredLink("Drivers", detector.RouteWorkers, 5),
	}

	return insightSeries{
		current: s.narrated(&entity, refs),
		history: [][]insight.Metric{credentials(3, 0, 11), credentials(4, 0, 5)},
	}
}

func (s *InsightSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *InsightSeed) CanRollback() bool {
	return true
}
