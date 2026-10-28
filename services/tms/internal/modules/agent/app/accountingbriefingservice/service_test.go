package accountingbriefingservice

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/accountingbrief"
	"github.com/emoss08/trenova/internal/modules/agent/ports/accountingbriefports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	acmeID = pulid.ID("cus_acme")
	day    = int64(secondsPerDay)
)

var startedAt = time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)

func joined(segments []accountingbriefports.AccountingBriefingSegment) string {
	var b strings.Builder
	for _, segment := range segments {
		b.WriteString(segment.Text)
	}
	return b.String()
}

func linkOf(segment accountingbriefports.AccountingBriefingSegment) accountingbrief.Link {
	if segment.Link == nil {
		return ""
	}
	return *segment.Link
}

func link(value accountingbrief.Link) *accountingbrief.Link { return &value }

func busyBook() accountingbrief.Facts {
	return accountingbrief.Facts{
		OpenMinor:                 4_825_000,
		OpenInvoices:              32,
		OverdueMinor:              1_240_000,
		OverdueInvoices:           9,
		PastNinetyMinor:           310_000,
		DisputedMinor:             180_000,
		DisputedInvoices:          2,
		UnappliedCashMinor:        250_000,
		UnappliedPayments:         1,
		CollectedTodayMinor:       675_000,
		PaymentsToday:             3,
		DSOTenths:                 423,
		DSODeltaTenths:            31,
		CEIPercent:                81,
		TopCustomerID:             acmeID,
		TopCustomerName:           "Acme Freight",
		TopCustomerOverdueMinor:   520_000,
		CrossingNinetyInvoices:    2,
		CrossingNinetyMinor:       420_000,
		DueSoonInvoices:           1,
		DueSoonMinor:              250_000,
		ShortPayRemainders:        1,
		ShortPayRemainderMinor:    3_500,
		BrokenPromises:            1,
		BrokenPromiseMinor:        400_000,
		BrokenPromiseCustomerID:   acmeID,
		BrokenPromiseCustomerName: "Acme Freight",
		PromisesDue:               1,
		PromisesDueMinor:          200_000,
		UncontactedAccounts:       2,
		UncontactedOverdueMinor:   150_000,
		FollowUpsDue:              3,
		DueThisMonthInvoices:      2,
		DueThisMonthMinor:         330_000,
		NextDueInDays:             3,
		DraftInvoices:             2,
		DraftMinor:                180_000,
		OldestDraftDays:           3,
		CollectedWindowMinor:      2_000_000,
		BilledWindowMinor:         3_000_000,
	}
}

func TestDeterministicWordingLeadsWithWhatTheDashboardDoesNotShow(t *testing.T) {
	t.Parallel()

	segments := Deterministic(busyBook())
	assert.Equal(
		t,
		"Acme Freight missed a promise to pay $4,000, 2 invoices worth $4,200 pass 90 days past "+
			"due this week and 2 customers owing $1,500 past due have had no collections contact "+
			"in 14 days. 1 promise to pay $2,000 comes due in the next 2 days, $2,500 across 1 "+
			"invoice falls due in the next 7 days and Acme Freight holds 42% of everything past due.",
		joined(segments),
	)
	assert.NotContains(
		t,
		joined(segments),
		"$48,250",
		"the open balance is on the dashboard already",
	)

	links := make([]accountingbrief.Link, 0, len(segments))
	for _, segment := range segments {
		if l := linkOf(segment); l != "" {
			links = append(links, l)
		}
	}
	assert.Equal(t, []accountingbrief.Link{
		accountingbrief.LinkBrokenPromise,
		accountingbrief.LinkCrossingNinety,
		accountingbrief.LinkUncontacted,
		accountingbrief.LinkPromisesDue,
		accountingbrief.LinkDueSoon,
		accountingbrief.LinkTopCustomer,
	}, links)
}

func TestDeterministicWordingNamesSeveralBrokenPromisesWithoutACustomer(t *testing.T) {
	t.Parallel()

	segments := Deterministic(accountingbrief.Facts{
		OpenMinor:                 900_000,
		OpenInvoices:              4,
		OverdueMinor:              600_000,
		OverdueInvoices:           3,
		BrokenPromises:            2,
		BrokenPromiseMinor:        450_000,
		BrokenPromiseCustomerID:   acmeID,
		BrokenPromiseCustomerName: "Acme Freight",
	})
	assert.Equal(t, "2 promises to pay $4,500 have lapsed unpaid.", joined(segments))
}

func TestDeterministicWordingForAQuietBook(t *testing.T) {
	t.Parallel()

	current := Deterministic(accountingbrief.Facts{
		OpenMinor:       4_606_200,
		OpenInvoices:    35,
		DSOTenths:       910,
		DSODeltaTenths:  910,
		DueSoonInvoices: 4,
		DueSoonMinor:    1_210_000,
	})
	assert.Equal(
		t,
		"Nothing is past due, so nobody needs a call today. "+
			"$12,100 across 4 invoices falls due in the next 7 days.",
		joined(current),
	)

	worked := Deterministic(accountingbrief.Facts{
		OpenMinor:       900_000,
		OpenInvoices:    4,
		OverdueMinor:    100_000,
		OverdueInvoices: 1,
		DSOTenths:       185,
		DSODeltaTenths:  -42,
	})
	assert.Equal(
		t,
		"Every past-due customer has been contacted in the last 14 days, no promise to pay has "+
			"lapsed and nothing is about to pass 90 days past due. "+
			"Days sales outstanding fell 4.2 days in 4 weeks, to 18.5.",
		joined(worked),
	)
}

func TestDeterministicWordingForANewBookNamesWhatToDoBeforeAnythingIsLate(t *testing.T) {
	t.Parallel()

	facts := accountingbrief.Facts{
		OpenMinor:            4_606_216,
		OpenInvoices:         35,
		DSOTenths:            910,
		DSODeltaTenths:       910,
		DueThisMonthInvoices: 35,
		DueThisMonthMinor:    4_606_216,
		NextDueInDays:        23,
		DraftInvoices:        7,
		DraftMinor:           912_300,
		OldestDraftDays:      6,
		BilledWindowMinor:    4_606_216,
	}
	segments := Deterministic(facts)
	assert.Equal(
		t,
		"Nothing has been collected in the last 91 days against $46,062 billed and 7 invoices "+
			"worth $9,123 are still in draft, the oldest 6 days old. $46,062 across 35 invoices "+
			"falls due in the next 30 days, the first in 23 days.",
		joined(segments),
	)
	assert.Equal(t, []accountingbrief.Action{
		{Link: accountingbrief.LinkDrafts, Count: 7, AmountMinor: 912_300},
		{Link: accountingbrief.LinkNoCollections, AmountMinor: 4_606_216},
		{Link: accountingbrief.LinkDueThisMonth, Count: 35, AmountMinor: 4_606_216},
	}, facts.Actions())

	_, ok := AcceptNarration([]accountingbriefports.AccountingBriefingSegment{
		{Text: "7 invoices worth $9,123", Link: link(accountingbrief.LinkDrafts)},
		{Text: " are still drafts after 6 days; "},
		{
			Text: "the first of $46.1k falls due in 23 days",
			Link: link(accountingbrief.LinkDueThisMonth),
		},
		{Text: "."},
	}, facts, facts.Links())
	assert.True(t, ok)
}

func TestDeterministicWordingForAnEmptyBook(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"Nothing is owed to you and no cash is waiting to be applied.",
		joined(Deterministic(accountingbrief.Facts{})),
	)
}

func TestAcceptNarration(t *testing.T) {
	t.Parallel()

	facts := busyBook()
	allowed := facts.Links()

	accepted, ok := AcceptNarration([]accountingbriefports.AccountingBriefingSegment{
		{Text: "Acme Freight missed a $4k promise", Link: link(accountingbrief.LinkBrokenPromise)},
		{Text: ", and "},
		{
			Text: "2 invoices worth $4,200 cross 90 days this week",
			Link: link(accountingbrief.LinkCrossingNinety),
		},
		{Text: "; a call now keeps them out of the 90-day bucket."},
	}, facts, allowed)
	require.True(t, ok)
	assert.Len(t, accepted, 4)

	cases := map[string][]accountingbriefports.AccountingBriefingSegment{
		"an invented figure": {
			{
				Text: "$90,000 is about to pass 90 days",
				Link: link(accountingbrief.LinkCrossingNinety),
			},
		},
		"a link outside the allowed set": {
			{Text: "nothing is refunded", Link: link(accountingbrief.Link("Refunds"))},
		},
		"a missed promise naming someone else": {
			{Text: "Globex missed a promise", Link: link(accountingbrief.LinkBrokenPromise)},
		},
		"a leading customer naming someone else": {
			{Text: "Globex holds 42% of it", Link: link(accountingbrief.LinkTopCustomer)},
		},
		"no wording at all": {},
		"only whitespace":   {{Text: "   "}},
		"too long": {
			{Text: strings.Repeat("Collections are on track. ", 30)},
		},
	}
	for name, segments := range cases {
		_, ok := AcceptNarration(segments, facts, allowed)
		assert.False(t, ok, name)
	}
}

func TestNarrationCannotCiteAChangeWithoutABaseline(t *testing.T) {
	t.Parallel()

	facts := accountingbrief.Facts{
		OpenMinor:      4_606_200,
		OpenInvoices:   35,
		DSOTenths:      450,
		DSODeltaTenths: 125,
	}
	_, ok := AcceptNarration([]accountingbriefports.AccountingBriefingSegment{
		{Text: "DSO is 45.0 days, up 12.5 over four weeks."},
	}, facts, facts.Links())
	assert.True(t, ok, "a change from a real baseline may be cited")

	facts.DSODeltaTenths = 575
	require.False(t, facts.HasDSOBaseline())
	_, ok = AcceptNarration([]accountingbriefports.AccountingBriefingSegment{
		{Text: "DSO is 45.0 days, up 57.5 over four weeks."},
	}, facts, facts.Links())
	assert.False(t, ok, "a change nothing computed is rejected")
}

func TestFactsReadTheLedgerAndTheWorklist(t *testing.T) {
	t.Parallel()

	f := newFixture(stubCompletion{err: llmports.ErrNoProviderConfigured})
	facts, err := f.service.Facts(t.Context(), tenantInfo(), "America/Chicago")
	require.NoError(t, err)

	want := busyBook()
	want.Timezone = "America/Chicago"
	assert.Equal(t, want, *facts)
	assert.Equal(t, collectionports.CollectorScopeAll, f.collections.scope)
	assert.Equal(t, worklistLimit, f.collections.limit)
	assert.Equal(t, startedAt.Unix(), f.receivables.openItemsAsOf)

	f.receivables.top = []*customerledgerports.ARTopOverdueCustomer{{
		CustomerID:   acmeID,
		CustomerName: "Acme Freight",
	}}
	facts, err = f.service.Facts(t.Context(), tenantInfo(), "America/Chicago")
	require.NoError(t, err)
	assert.False(t, facts.HasTopCustomer(), "a customer with nothing past due leads nothing")
	assert.Empty(t, facts.TopCustomerName)
}

func TestFactsFailWhenTheLedgerCannotBeRead(t *testing.T) {
	t.Parallel()

	f := newFixture(stubCompletion{err: llmports.ErrNoProviderConfigured})
	f.receivables.err = errors.New("ledger unavailable")

	_, err := f.service.Briefing(t.Context(), tenantInfo())
	require.Error(t, err)
	assert.Empty(t, f.briefs.rows, "nothing is stored from figures that could not be read")
}

func TestFactsFailWhenTheWorklistCannotBeRead(t *testing.T) {
	t.Parallel()

	f := newFixture(stubCompletion{err: llmports.ErrNoProviderConfigured})
	f.collections.err = errors.New("worklist unavailable")

	_, err := f.service.Briefing(t.Context(), tenantInfo())
	require.Error(t, err)
	assert.Empty(t, f.briefs.rows)
}

func TestBriefingFallsBackWithoutAModel(t *testing.T) {
	t.Parallel()

	f := newFixture(stubCompletion{err: llmports.ErrNoProviderConfigured})
	briefing, err := f.service.Briefing(t.Context(), tenantInfo())
	require.NoError(t, err)

	assert.False(t, briefing.Narrated)
	assert.Equal(t, joined(Deterministic(busyBook())), joined(briefing.Segments))
	assert.Equal(t, 1, briefing.Generation)
	require.Len(t, f.briefs.rows, 1)
	assert.Equal(t, accountingbrief.TriggerOnDemand, f.briefs.rows[0].Trigger)
	assert.Len(t, f.briefs.rows[0].BriefDate, len("2006-01-02"))

	for _, segment := range briefing.Segments {
		switch linkOf(segment) {
		case accountingbrief.LinkBrokenPromise, accountingbrief.LinkTopCustomer:
			require.NotNil(t, segment.CustomerID, segment.Text)
			assert.Equal(t, acmeID, *segment.CustomerID)
		default:
			assert.Nil(t, segment.CustomerID, segment.Text)
		}
	}

	customer := acmeID
	assert.Equal(t, []accountingbriefports.AccountingBriefingAction{
		{
			Link:         accountingbrief.LinkBrokenPromise,
			Count:        1,
			AmountMinor:  400_000,
			CustomerID:   &customer,
			CustomerName: "Acme Freight",
		},
		{Link: accountingbrief.LinkCrossingNinety, Count: 2, AmountMinor: 420_000},
		{Link: accountingbrief.LinkUncontacted, Count: 2, AmountMinor: 150_000},
		{Link: accountingbrief.LinkDrafts, Count: 2, AmountMinor: 180_000},
		{Link: accountingbrief.LinkFollowUpsDue, Count: 3},
		{Link: accountingbrief.LinkPromisesDue, Count: 1, AmountMinor: 200_000},
		{Link: accountingbrief.LinkDueSoon, Count: 1, AmountMinor: 250_000},
		{Link: accountingbrief.LinkShortPayRemainders, Count: 1, AmountMinor: 3_500},
		{Link: accountingbrief.LinkUnappliedCash, Count: 1, AmountMinor: 250_000},
	}, briefing.Actions)
}

func TestBriefingUsesGuardedNarration(t *testing.T) {
	t.Parallel()

	honest := `{"segments":[` +
		`{"text":"Acme Freight missed a $4k promise","link":"BrokenPromise"},` +
		`{"text":", and ","link":""},` +
		`{"text":"2 invoices worth $4,200 cross 90 days this week","link":"CrossingNinety"},` +
		`{"text":".","link":""}]}`
	f := newFixture(stubCompletion{text: honest})
	briefing, err := f.service.Briefing(t.Context(), tenantInfo())
	require.NoError(t, err)
	assert.True(t, briefing.Narrated)
	assert.Equal(
		t,
		"Acme Freight missed a $4k promise, and 2 invoices worth $4,200 cross 90 days this week.",
		joined(briefing.Segments),
	)
	require.NotNil(t, briefing.Segments[0].CustomerID)
	assert.Equal(t, acmeID, *briefing.Segments[0].CustomerID)
	assert.Equal(t, "test-model", f.briefs.rows[0].ModelIdentifier)

	lying := `{"segments":[{"text":"$90,000 is past due","link":"Overdue"}]}`
	f = newFixture(stubCompletion{text: lying})
	briefing, err = f.service.Briefing(t.Context(), tenantInfo())
	require.NoError(t, err)
	assert.False(t, briefing.Narrated)
	assert.Equal(t, joined(Deterministic(busyBook())), joined(briefing.Segments))
}

func TestBriefingIsWrittenOnceAndThenRead(t *testing.T) {
	t.Parallel()

	calls := 0
	f := newFixture(stubCompletion{err: llmports.ErrNoProviderConfigured, calls: &calls})
	first, err := f.service.Briefing(t.Context(), tenantInfo())
	require.NoError(t, err)
	reads := f.receivables.reads

	f.clock.advance(10 * time.Minute)
	f.receivables.overview.OverdueMinor = 1_000_000
	second, err := f.service.Briefing(t.Context(), tenantInfo())
	require.NoError(t, err)

	assert.Equal(t, first.GeneratedAt, second.GeneratedAt)
	assert.Equal(t, reads, f.receivables.reads, "the book is not looked at again inside the window")
	assert.Len(t, f.briefs.rows, 1)
	assert.Equal(t, 1, calls)
}

func TestBriefingIsRewrittenWhenTheFindingsMove(t *testing.T) {
	t.Parallel()

	f := newFixture(stubCompletion{err: llmports.ErrNoProviderConfigured})
	_, err := f.service.Briefing(t.Context(), tenantInfo())
	require.NoError(t, err)

	f.clock.advance(refreshAfter + time.Minute)
	f.receivables.overview.OverdueMinor = 1_000_000
	briefing, err := f.service.Briefing(t.Context(), tenantInfo())
	require.NoError(t, err)

	assert.Equal(t, 2, briefing.Generation)
	assert.Contains(t, joined(briefing.Segments), "Acme Freight holds 52% of everything past due")
	require.Len(t, f.briefs.rows, 2)
	assert.Equal(t, accountingbrief.TriggerRefreshed, f.briefs.rows[1].Trigger)
}

func TestBriefingIsKeptAndVerifiedWhenTheFindingsHold(t *testing.T) {
	t.Parallel()

	f := newFixture(stubCompletion{err: llmports.ErrNoProviderConfigured})
	first, err := f.service.Briefing(t.Context(), tenantInfo())
	require.NoError(t, err)

	f.clock.advance(refreshAfter + time.Minute)
	second, err := f.service.Briefing(t.Context(), tenantInfo())
	require.NoError(t, err)

	assert.Equal(t, first.Generation, second.Generation)
	require.Len(t, f.briefs.rows, 1)
	assert.Equal(t, f.clock.now().Unix(), f.briefs.rows[0].VerifiedAt)

	reads := f.receivables.reads
	f.clock.advance(10 * time.Minute)
	_, err = f.service.Briefing(t.Context(), tenantInfo())
	require.NoError(t, err)
	assert.Equal(t, reads, f.receivables.reads, "a verified brief starts a new window")
}

func TestBriefingStopsRewritingAfterTheDaysLimit(t *testing.T) {
	t.Parallel()

	f := newFixture(stubCompletion{err: llmports.ErrNoProviderConfigured})
	generation := 0
	for range 20 {
		f.clock.advance(refreshAfter + time.Minute)
		f.receivables.overview.OverdueMinor += 10_000
		briefing, err := f.service.Briefing(t.Context(), tenantInfo())
		require.NoError(t, err)
		generation = briefing.Generation
	}

	assert.Less(t, generation, 20)
	assert.False(t, accountingbrief.CanRefresh(generation))
	assert.Len(t, f.briefs.rows, generation)
}

func TestWriteBriefForTheMorning(t *testing.T) {
	t.Parallel()

	f := newFixture(stubCompletion{err: llmports.ErrNoProviderConfigured})
	req := &accountingbriefports.WriteAccountingBriefRequest{
		TenantInfo: tenantInfo(),
		Trigger:    accountingbrief.TriggerScheduled,
	}

	first, err := f.service.WriteBrief(t.Context(), req)
	require.NoError(t, err)
	second, err := f.service.WriteBrief(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID, "the morning is written once")
	assert.Equal(t, accountingbrief.TriggerScheduled, first.Trigger)
	assert.Len(t, f.briefs.rows, 1)

	briefing, err := f.service.Briefing(t.Context(), tenantInfo())
	require.NoError(t, err)
	assert.Equal(t, 1, briefing.Generation, "reading the morning's brief does not rewrite it")
}

func TestContextSetsTheFindingsApartFromTheHeadlineFigures(t *testing.T) {
	t.Parallel()

	ctx := buildContext(busyBook(), Deterministic(busyBook()))
	require.Len(t, ctx.Sections, 3)
	assert.Equal(t, []string{findingsSection, headlineSection, plainWordingSection},
		[]string{ctx.Sections[0].Title, ctx.Sections[1].Title, ctx.Sections[2].Title})
	assert.Contains(
		t,
		ctx.Sections[0].Content,
		"Missed promises to pay: 1 promise to pay lapsed unpaid, $4,000 still owed, by Acme Freight (link BrokenPromise)",
	)
	assert.Contains(t, ctx.Sections[0].Content,
		"Change in days sales outstanding over the last 4 weeks: +3.1 days, now 42.3")
	assert.NotContains(t, ctx.Sections[0].Content, "Open receivables")
	assert.Contains(t, ctx.Sections[1].Content, "Open receivables: $48,250 across 32 invoices")
	for _, section := range ctx.Sections {
		assert.False(t, section.Trusted, section.Title)
	}

	noBaseline := buildContext(accountingbrief.Facts{
		OpenMinor:      4_606_200,
		OpenInvoices:   35,
		DSOTenths:      910,
		DSODeltaTenths: 910,
	}, nil)
	assert.Contains(t, noBaseline.Sections[0].Content, "No findings")
	assert.NotContains(t, noBaseline.Sections[0].Content, "Change in days sales outstanding")
}

func TestOutputSchemaAsksForSegmentsLinkedToAllowedLinks(t *testing.T) {
	t.Parallel()

	schema := outputSchema([]accountingbrief.Link{
		accountingbrief.LinkBrokenPromise,
		accountingbrief.LinkTopCustomer,
	})
	properties := schema["properties"].(map[string]any)
	segments := properties["segments"].(map[string]any)
	segment := segments["items"].(map[string]any)
	fields := segment["properties"].(map[string]any)
	linkField := fields["link"].(map[string]any)

	assert.Equal(t, []string{"", "BrokenPromise", "TopCustomer"}, linkField["enum"])
}

func tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1", UserID: "usr_1"}
}

type clock struct{ at time.Time }

func (c *clock) now() time.Time { return c.at }

func (c *clock) advance(d time.Duration) { c.at = c.at.Add(d) }

type stubReceivables struct {
	overview      customerledgerports.ARBalanceOverview
	kpis          customerledgerports.ARDashboardKPIs
	stats         customerledgerports.ARPaymentStats
	top           []*customerledgerports.ARTopOverdueCustomer
	openItems     []*customerledgerports.AROpenItem
	openItemsAsOf int64
	drafts        customerledgerports.ARDraftInvoices
	collected     customerledgerports.ARCollectionTotals
	cei           float64
	err           error
	reads         int
}

func (s *stubReceivables) GetDraftInvoices(
	context.Context,
	customerledgerports.GetARAnalyticsRequest,
) (*customerledgerports.ARDraftInvoices, error) {
	drafts := s.drafts
	return &drafts, nil
}

func (s *stubReceivables) ListOpenItems(
	_ context.Context,
	_ pagination.TenantInfo,
	customerID pulid.ID,
	asOfDate int64,
) ([]*customerledgerports.AROpenItem, error) {
	if !customerID.IsNil() {
		return nil, errors.New("the brief reads every customer's open items")
	}
	s.openItemsAsOf = asOfDate
	return s.openItems, nil
}

func (s *stubReceivables) GetDashboardKPIs(
	context.Context,
	customerledgerports.GetARAnalyticsRequest,
) (*customerledgerports.ARDashboardKPIs, error) {
	s.reads++
	if s.err != nil {
		return nil, s.err
	}
	kpis := s.kpis
	overview := s.overview
	kpis.Overview = &overview
	totals := s.collected
	kpis.Collection = &customerledgerports.ARCollectionPerformance{CEI: s.cei, Totals: &totals}
	return &kpis, nil
}

func (s *stubReceivables) GetPaymentStats(
	context.Context,
	customerledgerports.GetARAnalyticsRequest,
) (*customerledgerports.ARPaymentStats, error) {
	stats := s.stats
	return &stats, nil
}

func (s *stubReceivables) ListTopOverdueCustomers(
	_ context.Context,
	req customerledgerports.ListARTopOverdueCustomersRequest,
) ([]*customerledgerports.ARTopOverdueCustomer, error) {
	if req.Limit != topCustomerLimit {
		return nil, errors.New("the brief names one customer")
	}
	return s.top, nil
}

func newReceivables(now int64) *stubReceivables {
	book := busyBook()
	return &stubReceivables{
		overview: customerledgerports.ARBalanceOverview{
			TotalOpenMinor:       book.OpenMinor,
			OpenInvoiceCount:     book.OpenInvoices,
			OverdueMinor:         book.OverdueMinor,
			OverdueInvoiceCount:  book.OverdueInvoices,
			DisputedOpenMinor:    book.DisputedMinor,
			DisputedInvoiceCount: book.DisputedInvoices,
			UnappliedCashMinor:   book.UnappliedCashMinor,
			Buckets: customerledgerports.ARAgingBucketTotals{
				DaysOver90Minor: book.PastNinetyMinor,
			},
		},
		kpis: customerledgerports.ARDashboardKPIs{
			CurrentDSODays: 42.31,
			DSODeltaDays:   3.06,
		},
		cei: 80.6,
		stats: customerledgerports.ARPaymentStats{
			PostedTodayMinor:      book.CollectedTodayMinor,
			PostedTodayCount:      book.PaymentsToday,
			UnappliedCashMinor:    book.UnappliedCashMinor,
			UnappliedPaymentCount: book.UnappliedPayments,
		},
		top: []*customerledgerports.ARTopOverdueCustomer{{
			CustomerID:   acmeID,
			CustomerName: " Acme Freight ",
			OverdueMinor: book.TopCustomerOverdueMinor,
		}},
		drafts: customerledgerports.ARDraftInvoices{
			Count:           book.DraftInvoices,
			AmountMinor:     book.DraftMinor,
			OldestCreatedAt: now - 3*day - 3600,
		},
		collected: customerledgerports.ARCollectionTotals{
			CollectedMinor:   book.CollectedWindowMinor,
			CreditSalesMinor: book.BilledWindowMinor,
		},
		openItems: []*customerledgerports.AROpenItem{
			{OpenAmountMinor: 300_000, DaysPastDue: 85},
			{OpenAmountMinor: 120_000, DaysPastDue: 89},
			{OpenAmountMinor: 50_000, DaysPastDue: 90},
			{OpenAmountMinor: 0, DaysPastDue: 86},
			{OpenAmountMinor: 250_000, DueDate: now + 3*day},
			{OpenAmountMinor: 80_000, DueDate: now + 10*day},
			{OpenAmountMinor: 3_500, DaysPastDue: 40, HasShortPay: true},
			{OpenAmountMinor: 9_000, DaysPastDue: 12, HasShortPay: true},
			nil,
		},
	}
}

type stubCollections struct {
	collectionports.Service
	page  *collectionports.WorklistPage
	err   error
	scope collectionports.CollectorScope
	limit int
}

func (s *stubCollections) Worklist(
	_ context.Context,
	req *collectionports.WorklistRequest,
) (*collectionports.WorklistPage, error) {
	s.scope, s.limit = req.Scope, req.Limit
	if s.err != nil {
		return nil, s.err
	}
	return s.page, nil
}

func newCollections(now int64) *stubCollections {
	lastWeek := now - 2*day
	longAgo := now - 20*day
	return &stubCollections{page: &collectionports.WorklistPage{
		Summary: collectionports.WorklistSummary{FollowUpsDue: 3},
		Rows: []*collectionports.WorklistRow{
			{
				CustomerID:     acmeID,
				CustomerName:   " Acme Freight ",
				OverdueMinor:   520_000,
				LastActivityAt: &lastWeek,
				Promise: &collectionports.Promise{
					AmountMinor:    500_000,
					PaidSinceMinor: 100_000,
					Status:         collection.PromiseBroken,
				},
			},
			{
				CustomerID:   "cus_globex",
				CustomerName: "Globex",
				OverdueMinor: 150_000,
				Promise: &collectionports.Promise{
					AmountMinor: 200_000,
					PromisedOn:  now + day,
					Status:      collection.PromiseOpen,
				},
			},
			{
				CustomerID:     "cus_initech",
				CustomerName:   "Initech",
				OverdueMinor:   90_000,
				LastActivityAt: &longAgo,
			},
			{CustomerID: "cus_umbrella", CustomerName: "Umbrella", OverdueMinor: 60_000},
			{
				CustomerID:   "cus_hooli",
				CustomerName: "Hooli",
				Promise: &collectionports.Promise{
					AmountMinor:    40_000,
					PaidSinceMinor: 40_000,
					Status:         collection.PromiseKept,
				},
			},
			{
				CustomerID:     "cus_stark",
				CustomerName:   "Stark",
				OverdueMinor:   30_000,
				LastActivityAt: &lastWeek,
				Promise: &collectionports.Promise{
					AmountMinor: 30_000,
					PromisedOn:  now + 5*day,
					Status:      collection.PromiseOpen,
				},
			},
			nil,
		},
	}}
}

type stubOrgs struct {
	tenantports.OrganizationRepository
}

func (stubOrgs) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: "America/Chicago"}, nil
}

type memoryBriefs struct {
	rows []*accountingbrief.Brief
}

func (m *memoryBriefs) Latest(
	_ context.Context,
	req *accountingbriefports.GetLatestAccountingBriefRequest,
) (*accountingbrief.Brief, error) {
	var latest *accountingbrief.Brief
	for _, row := range m.rows {
		if row.BriefDate != req.BriefDate || row.OrganizationID != req.TenantInfo.OrgID {
			continue
		}
		if latest == nil || row.Generation > latest.Generation {
			latest = row
		}
	}
	if latest == nil {
		return latest, nil
	}
	copied := *latest
	return &copied, nil
}

func (m *memoryBriefs) Insert(
	ctx context.Context,
	brief *accountingbrief.Brief,
) (*accountingbrief.Brief, error) {
	for _, row := range m.rows {
		if row.BriefDate == brief.BriefDate && row.OrganizationID == brief.OrganizationID &&
			row.Generation == brief.Generation {
			return m.Latest(ctx, &accountingbriefports.GetLatestAccountingBriefRequest{
				TenantInfo: pagination.TenantInfo{OrgID: brief.OrganizationID},
				BriefDate:  brief.BriefDate,
			})
		}
	}
	stored := *brief
	stored.ID = pulid.MustNew("acb_")
	m.rows = append(m.rows, &stored)
	copied := stored
	return &copied, nil
}

func (m *memoryBriefs) MarkVerified(
	_ context.Context,
	req *accountingbriefports.MarkAccountingBriefVerifiedRequest,
) error {
	for _, row := range m.rows {
		if row.ID == req.ID && row.OrganizationID == req.TenantInfo.OrgID &&
			row.VerifiedAt < req.VerifiedAt {
			row.VerifiedAt = req.VerifiedAt
		}
	}
	return nil
}

func (m *memoryBriefs) DeleteBefore(
	context.Context,
	*accountingbriefports.DeleteAccountingBriefsBeforeRequest,
) (int, error) {
	return 0, nil
}

type stubCompletion struct {
	llmports.CompletionService
	text  string
	err   error
	calls *int
}

func (s stubCompletion) CompleteStructured(
	_ context.Context,
	req *llmports.StructuredCompletionRequest,
) (*llmports.StructuredCompletionResult, error) {
	if s.calls != nil {
		*s.calls++
	}
	if s.err != nil {
		return nil, s.err
	}
	if req.SchemaName != schemaName {
		return nil, errors.New("no wording in this test")
	}
	return &llmports.StructuredCompletionResult{Text: s.text, ModelIdentifier: "test-model"}, nil
}

type fixture struct {
	service     *Service
	receivables *stubReceivables
	collections *stubCollections
	briefs      *memoryBriefs
	clock       *clock
}

func newFixture(completion stubCompletion) *fixture {
	c := &clock{at: startedAt}
	receivables := newReceivables(startedAt.Unix())
	collections := newCollections(startedAt.Unix())
	briefs := &memoryBriefs{}

	return &fixture{
		service: NewWithDependencies(&Dependencies{
			Receivables:   receivables,
			Collections:   collections,
			Briefs:        briefs,
			Organizations: stubOrgs{},
			Completion:    completion,
			Logger:        zap.NewNop(),
			Now:           c.now,
		}),
		receivables: receivables,
		collections: collections,
		briefs:      briefs,
		clock:       c,
	}
}
