package detentionintelligence

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	rollup *detentionports.IntelligenceRollup
	zone   string
	got    *detentionports.IntelligenceRequest
}

func (f *fakeRepo) Intelligence(
	_ context.Context,
	req *detentionports.IntelligenceRequest,
) (*detentionports.IntelligenceRollup, error) {
	f.got = req
	return f.rollup, nil
}

func (f *fakeRepo) OrganizationTimezone(context.Context, pagination.TenantInfo) (string, error) {
	return f.zone, nil
}

const (
	day  = int64(86_400)
	zone = "America/Chicago"
)

var chicago = mustZone(zone)

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// now is mid-afternoon on 15 March 2030 in Chicago, after clocks went forward on the 10th.
var now = time.Date(2030, time.March, 15, 15, 0, 0, 0, chicago)

func midnight(year int, month time.Month, d int) int64 {
	return time.Date(year, month, d, 0, 0, 0, 0, chicago).Unix()
}

// from is the first day of the 90-day window ending on now's day.
var from = midnight(2029, time.December, 16)

func newService(repo *fakeRepo) *Service {
	if repo.zone == "" {
		repo.zone = zone
	}
	return New(Params{Repo: repo, Now: func() int64 { return now.Unix() }})
}

func money(v string) decimal.Decimal { return decimal.RequireFromString(v) }

func nullMoney(v string) decimal.NullDecimal { return decimal.NewNullDecimal(money(v)) }

func facility(
	id pulid.ID,
	name string,
	dock, open, early, late int64,
) detentionports.FacilityIntelligence {
	return detentionports.FacilityIntelligence{
		LocationID:         id,
		LocationName:       name,
		StopCount:          100,
		DetainedCount:      60,
		PastFreeMinutes:    dock + open + early + late,
		DockMinutes:        dock,
		OpenMinutes:        open,
		EarlyMinutes:       early,
		LateMinutes:        late,
		MedianDwellMinutes: 150,
		MedianFreeMinutes:  120,
		BilledAmount:       money("1000"),
		DriverPayAmount:    nullMoney("400"),
		NetMargin:          nullMoney("600"),
		WaivedAmount:       money("0"),
	}
}

func rollup() *detentionports.IntelligenceRollup {
	ridgeway := pulid.MustNew("loc_")
	meridian := pulid.MustNew("loc_")
	small := []detentionports.FacilityIntelligence{
		facility(ridgeway, "Ridgeway DC", 14_000, 0, 2_000, 1_000),
		facility(meridian, "Meridian Paper Mill", 6_000, 600, 300, 300),
		facility(pulid.MustNew("loc_"), "Bluff Creek", 600, 300, 600, 200),
		facility(pulid.MustNew("loc_"), "Northgate", 200, 100, 400, 100),
		facility(pulid.MustNew("loc_"), "Eastline", 100, 50, 100, 50),
	}
	small[1].MedianDwellMinutes = 221
	small[1].MedianFreeMinutes = 60

	return &detentionports.IntelligenceRollup{
		Totals: detentionports.IntelligenceTotals{
			StopCount:       500,
			DetainedCount:   300,
			PastFreeMinutes: 27_000,
			DockMinutes:     20_900,
			OpenMinutes:     1_050,
			EarlyMinutes:    3_400,
			LateMinutes:     1_650,
			BilledAmount:    money("100000"),
			DriverPayAmount: nullMoney("60000"),
			NetMargin:       nullMoney("40000"),
		},
		Previous:   detentionports.IntelligenceTotals{StopCount: 450, PastFreeMinutes: 24_000},
		Facilities: small,
		PreviousFacilities: []detentionports.FacilityMinutes{
			{LocationID: ridgeway, PastFreeMinutes: 16_000},
		},
		Customers: []detentionports.CustomerIntelligence{
			{
				CustomerID:      pulid.MustNew("cus_"),
				CustomerName:    "Hartwell Foods",
				StopCount:       84,
				DriverPayAmount: nullMoney("12660"),
				NetMargin:       nullMoney("-3240"),
			},
		},
		Recovery: []detentionports.RecoveryBucket{
			{
				Bucket:          detentionports.RecoveryBilled,
				OccurrenceCount: 200,
				GrossAmount:     money("130000"),
				BillableAmount:  money("120000"),
				InvoicedAmount:  money("110000"),
				CollectedAmount: money("90000"),
			},
			{
				Bucket:          detentionports.RecoveryNotice,
				OccurrenceCount: 23,
				GrossAmount:     money("9860"),
			},
			{
				Bucket:          detentionports.RecoveryWaived,
				OccurrenceCount: 81,
				GrossAmount:     money("22140"),
			},
		},
		Splits: detentionports.Splits{
			Pickup:      detentionports.SplitFigure{StopCount: 250, DetainedCount: 130},
			Delivery:    detentionports.SplitFigure{StopCount: 250, DetainedCount: 170},
			Appointment: detentionports.SplitFigure{StopCount: 450, DetainedCount: 290},
			Open:        detentionports.SplitFigure{StopCount: 50, DetainedCount: 10},
			Live:        detentionports.SplitFigure{StopCount: 450, DetainedCount: 290},
			Drop:        detentionports.SplitFigure{StopCount: 50, DetainedCount: 10},
		},
		PolicyUses: []detentionports.PolicyUse{
			{LocationID: meridian, PolicyName: "Standard", FirstClockStart: from + day},
			{LocationID: meridian, PolicyName: "One hour", FirstClockStart: from + 60*day},
			{LocationID: ridgeway, PolicyName: "Standard", FirstClockStart: from},
		},
	}
}

func build(t *testing.T, visible protectedfield.GroupSet) (*Intelligence, *fakeRepo) {
	t.Helper()
	repo := &fakeRepo{rollup: rollup()}
	result, err := newService(repo).Build(context.Background(), &Request{
		Days:    90,
		Visible: visible,
	})
	require.NoError(t, err)
	return result, repo
}

func text(segments []Segment) string {
	var out strings.Builder
	for _, segment := range segments {
		out.WriteString(segment.Text)
	}
	return out.String()
}

func TestBuild_ReadsWholeDaysOfTheOrganizationsZoneAndTheWindowBefore(t *testing.T) {
	result, repo := build(t, protectedfield.GroupSet(0).With(protectedfield.Margins))

	assert.Equal(t, detentionports.IntelligenceWindow{
		From:       midnight(2029, time.December, 16),
		To:         midnight(2030, time.March, 16) - 1,
		MonthsFrom: midnight(2030, time.January, 1),
		MonthsEnd:  midnight(2030, time.March, 1),
	}, repo.got.Current, "January and February are whole months inside the window")
	assert.Equal(t, detentionports.IntelligenceWindow{
		From:       midnight(2029, time.September, 17),
		To:         midnight(2029, time.December, 16) - 1,
		MonthsFrom: midnight(2029, time.October, 1),
		MonthsEnd:  midnight(2029, time.December, 1),
	}, repo.got.Previous, "the window before has the same 90 days")
	assert.Equal(t, zone, repo.got.Timezone)
	assert.Equal(t, FacilityLimit, repo.got.Limit)
	assert.Equal(t, 90, result.Days)
	assert.Equal(t, "2030-03-15", result.Through)
	assert.Equal(t, zone, result.Timezone)
	assert.Equal(t, repo.got.Current.From, result.From)
	assert.Equal(t, repo.got.Previous.To, result.PreviousTo)
}

func TestBuild_ReadsTheDaysAFacilityReadNames(t *testing.T) {
	repo := &fakeRepo{rollup: rollup()}
	_, err := newService(repo).Build(context.Background(), &Request{Days: 7, Through: "2030-03-12"})
	require.NoError(t, err)

	assert.Equal(t, midnight(2030, time.March, 6), repo.got.Current.From)
	assert.Equal(t, midnight(2030, time.March, 13)-1, repo.got.Current.To)
	assert.Equal(t, 7*day-3600, repo.got.Current.To+1-repo.got.Current.From,
		"the week the clocks went forward is an hour short")
	assert.Equal(t, repo.got.Current.To+1, repo.got.Current.MonthsFrom,
		"a window inside one month reads no month")
	assert.Equal(t, repo.got.Current.MonthsFrom, repo.got.Current.MonthsEnd)
}

func TestBuild_ReadsAWindowOfWholeMonthsFromMonthsAlone(t *testing.T) {
	repo := &fakeRepo{rollup: rollup()}
	_, err := newService(repo).Build(context.Background(), &Request{Days: 59, Through: "2030-02-28"})
	require.NoError(t, err)

	assert.Equal(t, detentionports.IntelligenceWindow{
		From:       midnight(2030, time.January, 1),
		To:         midnight(2030, time.March, 1) - 1,
		MonthsFrom: midnight(2030, time.January, 1),
		MonthsEnd:  midnight(2030, time.March, 1),
	}, repo.got.Current)
}

func TestBuild_RefusesAWindowItCannotRead(t *testing.T) {
	service := newService(&fakeRepo{rollup: rollup()})
	cases := map[string]*Request{
		"no days":            {Days: 0},
		"longer than a year": {Days: MaxDays + 1},
		"ends after today":   {Days: 30, Through: "2030-03-16"},
		"not a day":          {Days: 30, Through: "15/03/2030"},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := service.Build(context.Background(), req)
			require.Error(t, err)
		})
	}
}

func TestBuild_CountsInUTCWhenTheOrganizationsZoneIsUnknown(t *testing.T) {
	for _, unknown := range []string{"Mars/Olympus", "Local"} {
		repo := &fakeRepo{rollup: rollup(), zone: unknown}
		result, err := newService(repo).Build(context.Background(), &Request{Days: 1})
		require.NoError(t, err)
		assert.Equal(t, "UTC", repo.got.Timezone)
		assert.Equal(t, "UTC", result.Timezone)
		assert.Equal(t, time.Date(2030, time.March, 15, 0, 0, 0, 0, time.UTC).Unix(),
			repo.got.Current.From)
	}
}

func TestBuild_WithholdsMarginsAndTheStepsBuiltOnThem(t *testing.T) {
	result, _ := build(t, 0)

	assert.False(t, result.Totals.NetMargin.Valid)
	assert.False(t, result.Totals.DriverPayAmount.Valid)
	assert.False(t, result.Facilities[0].NetMargin.Valid)
	assert.False(t, result.Customers[0].NetMargin.Valid)
	for _, step := range result.NextSteps {
		assert.NotEqual(t, StepCustomerTerms, step.Kind,
			"a reader without margins is never told which customer loses money")
	}
}

func TestBuild_ProposesTheCustomerWhoseTermsLoseMoneyToAReaderWithMargins(t *testing.T) {
	result, _ := build(t, protectedfield.GroupSet(0).With(protectedfield.Margins))

	var found *NextStep
	for idx := range result.NextSteps {
		if result.NextSteps[idx].Kind == StepCustomerTerms {
			found = &result.NextSteps[idx]
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "Free time for Hartwell Foods costs more than it bills", text(found.Title))
	assert.Equal(t, "-3240", found.ImpactAmount.String())
}

func TestBuild_RanksAtMostFourStepsByWhatEachRecovers(t *testing.T) {
	result, _ := build(t, protectedfield.GroupSet(0).With(protectedfield.Margins))

	require.Len(t, result.NextSteps, maxNextSteps)
	first := result.NextSteps[0]
	assert.Equal(
		t,
		StepShipperReview,
		first.Kind,
		"14,000 on-time minutes at one dock outweigh the rest",
	)
	assert.Equal(t, "Take the scorecard to Ridgeway DC", text(first.Title))
	require.NotNil(t, first.ImpactMinutes)
	assert.Equal(t, int64(14_000), *first.ImpactMinutes)
	assert.Equal(t, TargetFacility, first.Title[1].Target)
}

func TestBuild_FollowsTheMoneyInAFixedOrder(t *testing.T) {
	result, _ := build(t, protectedfield.GroupSet(0).With(protectedfield.Margins))

	r := result.Recovery
	assert.Equal(t, "162000", r.Accrued.String())
	assert.Equal(t, "120000", r.Billed.String())
	assert.Equal(t, "110000", r.Invoiced.String())
	assert.Equal(t, "90000", r.Collected.String())
	buckets := make([]string, 0, len(r.Steps))
	for _, step := range r.Steps {
		buckets = append(buckets, step.Bucket)
	}
	assert.Equal(
		t,
		[]string{"Late", "Notice", "Waived", "Disputed", "Pending", "Other", "Capped"},
		buckets,
	)
	assert.Equal(t, "10000", r.Steps[len(r.Steps)-1].Amount.String(),
		"caps and minimums are what billed charges accrued but did not bill")
}

func TestBuild_ReportsAPolicyChangeOnlyWhereAFacilityMovedPolicy(t *testing.T) {
	result, _ := build(t, 0)

	require.Len(t, result.PolicyChanges, 1)
	assert.Equal(t, "One hour", result.PolicyChanges[0].PolicyName)
	assert.Equal(t, "Meridian Paper Mill", result.PolicyChanges[0].LocationName)
	assert.Equal(t, from+60*day, result.PolicyChanges[0].ChangedAt)
}

func TestBuild_KeepsEachFacilitysMinutesFromTheWindowBefore(t *testing.T) {
	result, _ := build(t, 0)

	require.NotNil(t, result.Facilities[0].PreviousPastFreeMinutes)
	assert.Equal(t, int64(16_000), *result.Facilities[0].PreviousPastFreeMinutes)
	assert.Nil(t, result.Facilities[1].PreviousPastFreeMinutes)
}

func paragraphs(summary [][]Segment) []string {
	out := make([]string, 0, len(summary))
	for _, paragraph := range summary {
		out = append(out, text(paragraph))
	}
	return out
}

func TestSummarize_BriefsTheWaitTheScheduleTheMoneyAndWhatMoved(t *testing.T) {
	result, _ := build(t, protectedfield.GroupSet(0).With(protectedfield.Margins))

	assert.Equal(t, []string{
		"Trucks waited 450 h past free time in the last 90 days, 50 h more than the 90 days before. " +
			"2 of 5 facilities account for 90% of it. " +
			"On 77% of the hours the truck was on time, so the wait was the dock's. " +
			"19% is ours to fix: trucks that arrived early and waited for their slot, or late and forfeited their free time.",
		"At 5 facilities the median stop runs past the free time granted, so detention is built into the schedule; " +
			"Meridian Paper Mill grants 1h for a 3h 41m stop.",
		"Of $162,000 the clock accrued, $120,000 was billed and $90,000 has been collected. " +
			"Another $9,860 never reached an invoice because notice went out late. " +
			"Hartwell Foods costs $3,240 more in driver pay than it is billed.",
		"Ridgeway DC moved most: 17 h more past free time than the window before.",
	}, paragraphs(result.Summary))
}

func TestSummarize_NamesNoLosingCustomerToAReaderWithoutMargins(t *testing.T) {
	result, _ := build(t, 0)

	for _, paragraph := range paragraphs(result.Summary) {
		assert.NotContains(t, paragraph, "Hartwell Foods")
		assert.NotContains(t, paragraph, "driver pay than it is billed")
	}
}

func TestSummarize_PointsEachNamedFacilityAtItsScorecard(t *testing.T) {
	result, _ := build(t, protectedfield.GroupSet(0).With(protectedfield.Margins))

	var named []string
	for _, paragraph := range result.Summary {
		for _, segment := range paragraph {
			if segment.Target == TargetFacility {
				named = append(named, segment.Text)
			}
		}
	}
	assert.Equal(t, []string{"Meridian Paper Mill", "Ridgeway DC"}, named)
}

func TestSummarize_SaysSoWhenNothingSettled(t *testing.T) {
	repo := &fakeRepo{rollup: &detentionports.IntelligenceRollup{}}
	result, err := newService(repo).Build(context.Background(), &Request{Days: 90})
	require.NoError(t, err)

	assert.Equal(
		t,
		[]string{"No stop settled detention in the last 90 days."},
		paragraphs(result.Summary),
	)
	assert.Empty(t, result.NextSteps)
}

func TestSummarize_GroupsTheDigitsOfLargeHours(t *testing.T) {
	repo := &fakeRepo{rollup: &detentionports.IntelligenceRollup{
		Totals: detentionports.IntelligenceTotals{
			StopCount:       4_000,
			DetainedCount:   1_800,
			PastFreeMinutes: 90_000,
			DockMinutes:     90_000,
		},
		Previous: detentionports.IntelligenceTotals{StopCount: 3_000, PastFreeMinutes: 6_000},
	}}
	result, err := newService(repo).Build(context.Background(), &Request{Days: 90})
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(
		paragraphs(result.Summary)[0],
		"Trucks waited 1,500 h past free time in the last 90 days, 1,400 h more than the 90 days before.",
	), paragraphs(result.Summary)[0])
}

func TestBuild_AddsTheDaysUpByWeekWithQuietWeeksAtZero(t *testing.T) {
	// The window runs from Sunday 16 December 2029, so its first week starts Monday the 10th.
	firstMonday := time.Date(2029, time.December, 10, 0, 0, 0, 0, time.UTC).Unix()
	week := 7 * day
	repo := &fakeRepo{rollup: &detentionports.IntelligenceRollup{
		Days: []detentionports.DayFigure{
			{
				Day:             time.Date(2030, time.January, 15, 0, 0, 0, 0, chicago),
				StopCount:       8,
				DetainedCount:   3,
				PastFreeMinutes: 400,
			},
			{
				Day:             time.Date(2030, time.January, 18, 0, 0, 0, 0, chicago),
				StopCount:       4,
				DetainedCount:   1,
				PastFreeMinutes: 200,
			},
		},
	}}

	result, err := newService(repo).Build(context.Background(), &Request{Days: 90})
	require.NoError(t, err)

	require.Len(t, result.Weeks, 14)
	for i, bucket := range result.Weeks {
		assert.Equal(t, firstMonday+int64(i)*week, bucket.WeekStart)
	}
	assert.Equal(t, detentionports.WeekBucket{WeekStart: firstMonday + 4*week}, result.Weeks[4])
	assert.Equal(t, detentionports.WeekBucket{
		WeekStart:       firstMonday + 5*week,
		StopCount:       12,
		DetainedCount:   4,
		PastFreeMinutes: 600,
	}, result.Weeks[5], "Tuesday and Friday of the week of 14 January add up in it")
}
