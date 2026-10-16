package accountingbrief

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeBoundsWhatIsStored(t *testing.T) {
	t.Parallel()

	unknown := Link("Refunds")
	overdue := LinkOverdue
	segments := make([]Segment, 0, MaxSegments+4)
	segments = append(segments,
		Segment{Text: strings.Repeat("a", MaxSegmentLength+50), Link: &overdue},
		Segment{Text: "b", Link: &unknown},
	)
	for range MaxSegments + 2 {
		segments = append(segments, Segment{Text: "c"})
	}

	brief := &Brief{Segments: segments, GeneratedAt: 100}
	brief.Normalize()

	require.Len(t, brief.Segments, MaxSegments)
	assert.Len(t, []rune(brief.Segments[0].Text), MaxSegmentLength)
	require.NotNil(t, brief.Segments[0].Link)
	assert.Equal(t, LinkOverdue, *brief.Segments[0].Link)
	assert.Nil(t, brief.Segments[1].Link, "a link nothing renders is dropped")
	assert.Equal(t, int64(100), brief.VerifiedAt, "a new brief holds as of when it was written")

	empty := &Brief{}
	empty.Normalize()
	assert.NotNil(t, empty.Segments)
}

func TestValidateRequiresADayAndWording(t *testing.T) {
	t.Parallel()

	brief := &Brief{
		OrganizationID: pulid.ID("org_1"),
		BusinessUnitID: pulid.ID("bu_1"),
		BriefDate:      "2026-10-16",
		Generation:     1,
		Trigger:        TriggerOnDemand,
		Segments:       []Segment{{Text: "Nothing is owed."}},
		GeneratedAt:    100,
	}
	multiErr := errortypes.NewMultiError()
	brief.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())

	invalid := *brief
	invalid.BriefDate = "Oct 16"
	invalid.Trigger = Trigger("Hourly")
	invalid.Segments = nil
	invalid.Generation = -1
	multiErr = errortypes.NewMultiError()
	invalid.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())
	assert.Len(t, multiErr.Errors, 4)
}

func TestLinksOfferOnlyFindingsThatHoldSomething(t *testing.T) {
	t.Parallel()

	assert.Empty(t, Facts{}.Links())
	assert.Equal(t, []Link{LinkOpenInvoices}, Facts{OpenMinor: 90_000, OpenInvoices: 1}.Links())

	facts := Facts{
		OpenInvoices:            3,
		OverdueInvoices:         2,
		PastNinetyMinor:         1,
		DisputedInvoices:        1,
		UnappliedCashMinor:      1,
		TopCustomerID:           pulid.ID("cus_1"),
		TopCustomerName:         "Acme Freight",
		TopCustomerOverdueMinor: 1,
		CrossingNinetyInvoices:  1,
		DueSoonInvoices:         1,
		BrokenPromises:          1,
		PromisesDue:             1,
		UncontactedAccounts:     1,
		FollowUpsDue:            1,
		ShortPayRemainders:      1,
		DraftInvoices:           1,
		BilledWindowMinor:       1,
		DueThisMonthInvoices:    1,
	}
	want := make([]Link, 0, len(Links()))
	for _, l := range Links() {
		if l != LinkDueThisMonth {
			want = append(want, l)
		}
	}
	assert.Equal(t, want, facts.Links(), "a week's dues already name what falls due this month")

	facts.DueSoonInvoices = 0
	assert.Contains(t, facts.Links(), LinkDueThisMonth)

	facts.TopCustomerName = ""
	assert.NotContains(t, facts.Links(), LinkTopCustomer)

	facts.CollectedWindowMinor = 1
	assert.NotContains(t, facts.Links(), LinkNoCollections)
}

func TestNothingCollectedNeedsBillingAndNoCash(t *testing.T) {
	t.Parallel()

	assert.False(t, Facts{}.NothingCollected(), "a book with no billing has nothing to collect")
	assert.True(t, Facts{BilledWindowMinor: 100}.NothingCollected())
	assert.False(t, Facts{BilledWindowMinor: 100, CollectedWindowMinor: 1}.NothingCollected())
	assert.False(t, Facts{BilledWindowMinor: 100, CollectedTodayMinor: 1}.NothingCollected())
}

func TestSameIgnoresTheTimezone(t *testing.T) {
	t.Parallel()

	a := Facts{OpenMinor: 100, Timezone: "America/Chicago"}
	assert.True(t, a.Same(Facts{OpenMinor: 100, Timezone: "UTC"}))
	assert.False(t, a.Same(Facts{OpenMinor: 101, Timezone: "America/Chicago"}))
	assert.False(t, a.Same(Facts{OpenMinor: 100, BrokenPromises: 1}), "a new finding moves the brief")
}

func TestCanRefreshIsBoundedPerDay(t *testing.T) {
	t.Parallel()

	assert.True(t, CanRefresh(1))
	assert.False(t, CanRefresh(maxGenerationsPerDay))
}

func TestActionsPutWhatNeedsAPersonTodayFirst(t *testing.T) {
	t.Parallel()

	facts := Facts{
		OpenMinor:                 500_000,
		OpenInvoices:              6,
		OverdueMinor:              200_000,
		OverdueInvoices:           2,
		UnappliedCashMinor:        10_000,
		UnappliedPayments:         2,
		CrossingNinetyInvoices:    1,
		CrossingNinetyMinor:       60_000,
		DueSoonInvoices:           3,
		DueSoonMinor:              90_000,
		ShortPayRemainders:        2,
		ShortPayRemainderMinor:    4_000,
		BrokenPromises:            1,
		BrokenPromiseMinor:        70_000,
		BrokenPromiseCustomerID:   pulid.ID("cus_1"),
		BrokenPromiseCustomerName: "Acme Freight",
		PromisesDue:               1,
		PromisesDueMinor:          20_000,
		UncontactedAccounts:       2,
		UncontactedOverdueMinor:   80_000,
		FollowUpsDue:              4,
	}
	assert.Equal(t, []Action{
		{
			Link:         LinkBrokenPromise,
			Count:        1,
			AmountMinor:  70_000,
			CustomerID:   pulid.ID("cus_1"),
			CustomerName: "Acme Freight",
		},
		{Link: LinkCrossingNinety, Count: 1, AmountMinor: 60_000},
		{Link: LinkUncontacted, Count: 2, AmountMinor: 80_000},
		{Link: LinkFollowUpsDue, Count: 4},
		{Link: LinkPromisesDue, Count: 1, AmountMinor: 20_000},
		{Link: LinkDueSoon, Count: 3, AmountMinor: 90_000},
		{Link: LinkShortPayRemainders, Count: 2, AmountMinor: 4_000},
		{Link: LinkUnappliedCash, Count: 2, AmountMinor: 10_000},
	}, facts.Actions())
}

func TestActionsLeaveOutTheHeadlineFigures(t *testing.T) {
	t.Parallel()

	assert.Empty(t, Facts{OpenMinor: 4_606_200, OpenInvoices: 35, OverdueInvoices: 3}.Actions(),
		"an open or past-due balance alone is on the dashboard already")
}

func TestABrokenPromiseNamesItsCustomerOnlyWhenItIsTheOnlyOne(t *testing.T) {
	t.Parallel()

	facts := Facts{
		BrokenPromises:            2,
		BrokenPromiseCustomerID:   pulid.ID("cus_1"),
		BrokenPromiseCustomerName: "Acme Freight",
	}
	_, _, ok := facts.CustomerFor(LinkBrokenPromise)
	assert.False(t, ok)

	facts.BrokenPromises = 1
	id, name, ok := facts.CustomerFor(LinkBrokenPromise)
	require.True(t, ok)
	assert.Equal(t, pulid.ID("cus_1"), id)
	assert.Equal(t, "Acme Freight", name)
}

func TestConcentrationNeedsAMeaningfulShare(t *testing.T) {
	t.Parallel()

	facts := Facts{
		OverdueMinor:            1_000_000,
		TopCustomerID:           pulid.ID("cus_1"),
		TopCustomerName:         "Acme Freight",
		TopCustomerOverdueMinor: 290_000,
	}
	assert.Equal(t, 29, facts.TopCustomerSharePercent())
	assert.False(t, facts.Concentrated())

	facts.TopCustomerOverdueMinor = 300_000
	assert.True(t, facts.Concentrated())
	assert.Equal(t, 0, Facts{OverdueMinor: 0}.TopCustomerSharePercent())
}

func TestDSOMovesOnlyFromARealBaseline(t *testing.T) {
	t.Parallel()

	assert.False(t, Facts{DSOTenths: 910, DSODeltaTenths: 910}.HasDSOBaseline())
	assert.False(t, Facts{}.HasDSOBaseline())
	assert.True(t, Facts{DSOTenths: 423, DSODeltaTenths: 31}.HasDSOBaseline())

	assert.False(t, Facts{DSOTenths: 910, DSODeltaTenths: 910}.DSOMoved())
	assert.False(t, Facts{DSOTenths: 423, DSODeltaTenths: 29}.DSOMoved())
	assert.True(t, Facts{DSOTenths: 423, DSODeltaTenths: 31}.DSOMoved())
	assert.True(t, Facts{DSOTenths: 185, DSODeltaTenths: -42}.DSOMoved())
}
