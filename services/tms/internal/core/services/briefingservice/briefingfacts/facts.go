// Package briefingfacts gathers the morning's numbers.
//
// Everything a briefing says is computed here, from the same repositories
// the pages read, and handed to the writer as figures with labels. Nothing
// downstream may introduce a number: the writer turns these into sentences
// and a guard rejects any figure that is not one of them. That is the whole
// reason the package exists separately from the service that narrates.
//
// Every source is optional and every failure is survivable. A deployment
// without a dispatch console, or a tenant whose billing tables are mid
// migration, still gets a briefing — one section short, and the section
// that could not be read says so rather than reporting a zero, because a
// zero and an unreadable source mean opposite things on a morning page.
package briefingfacts

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	// credentialHorizonDays is how far ahead the compliance line looks. A
	// month is long enough to book a physical and short enough that the
	// number means something this morning.
	credentialHorizonDays = 30
	// credentialPageLimit bounds the walk behind the compliance count. A
	// fleet of 300 trucks never approaches it; a bigger one reports the
	// bound rather than loading every row into a morning page.
	credentialPageLimit = 500
	// boardWindowHours is how far forward the coverage figures look.
	boardWindowHours = 24
)

// Request is one organization's day, at the instant the sweep started.
type Request struct {
	TenantInfo pagination.TenantInfo
	// Now is the instant the whole sweep was anchored to, so every role's
	// briefing for a day describes exactly the same window.
	Now int64
	// DayStart and DayEnd bound the organization's local day.
	DayStart int64
	DayEnd   int64
	// Timezone is the organization's, used only to say which day this is.
	Timezone string
}

// Facts is the day in numbers, keyed for the prompt and for the guard.
// Values are decimals rather than strings so the guard can compare them;
// the rendered strings live on the sections.
type Facts struct {
	// Values is every figure the writer may cite, by key.
	Values map[string]decimal.Decimal
	// Sections is the page as computed, each with deterministic wording.
	Sections []briefing.Section
	// Unavailable names the sources that could not be read. A section over
	// an unreadable source is left out rather than reported as zero.
	Unavailable []string
}

// Sources are the repositories the facts are drawn from. Every one is
// optional: a nil source is a section the briefing does without.
type Sources struct {
	Watchtower  repositories.WatchtowerRepository
	Decisions   services.AgentDecisionQueueService
	Board       repositories.DispatchConsoleRepository
	Credentials repositories.WorkerCredentialRepository
	Billing     repositories.BillingQueueRepository
	Payments    repositories.CustomerPaymentRepository
	Detention   repositories.DetentionOccurrenceRepository
}

// Builder gathers the facts. It holds only repositories, so it is safe to
// share and cheap to construct.
type Builder struct {
	sources Sources
}

func NewBuilder(sources Sources) *Builder {
	return &Builder{sources: sources}
}

// Build gathers everything once for an organization. The sections come
// back in reading order; a role's briefing keeps the ones that role opens.
func (b *Builder) Build(ctx context.Context, req Request) *Facts {
	facts := &Facts{
		Values:      make(map[string]decimal.Decimal, 24),
		Sections:    make([]briefing.Section, 0, 6),
		Unavailable: make([]string, 0, 6),
	}

	b.attention(ctx, req, facts)
	b.decisions(ctx, req, facts)
	b.coverage(ctx, req, facts)
	b.compliance(ctx, req, facts)
	b.billing(ctx, req, facts)
	b.cash(ctx, req, facts)
	b.detention(ctx, req, facts)

	return facts
}

// SectionsFor keeps the blocks a role actually opens. Leadership reads the
// whole page; everybody else reads their own work plus what is waiting on
// them, because a page whose first half is somebody else's job stops being
// read after a week.
func SectionsFor(role briefing.RoleKey, sections []briefing.Section) []briefing.Section {
	wanted := roleSections(role)
	if wanted == nil {
		return sections
	}

	kept := make([]briefing.Section, 0, len(sections))
	for _, section := range sections {
		if _, ok := wanted[section.Key]; ok {
			kept = append(kept, section)
		}
	}

	return kept
}

func roleSections(role briefing.RoleKey) map[briefing.SectionKey]struct{} {
	switch role {
	case briefing.RoleDispatch:
		return set(
			briefing.SectionAttention,
			briefing.SectionCoverage,
			briefing.SectionExceptions,
			briefing.SectionDecisions,
		)
	case briefing.RoleBilling:
		return set(
			briefing.SectionAttention,
			briefing.SectionBilling,
			briefing.SectionCash,
			briefing.SectionDecisions,
		)
	case briefing.RoleCompliance:
		return set(
			briefing.SectionAttention,
			briefing.SectionCompliance,
			briefing.SectionExceptions,
			briefing.SectionDecisions,
		)
	case briefing.RoleLeadership, briefing.RoleGeneral:
		return nil
	default:
		return nil
	}
}

func set(keys ...briefing.SectionKey) map[briefing.SectionKey]struct{} {
	out := make(map[briefing.SectionKey]struct{}, len(keys))
	for _, key := range keys {
		out[key] = struct{}{}
	}

	return out
}

// attention reads the watchtower rather than each source it projects, so
// the briefing and the feed can never disagree about how much is open.
func (b *Builder) attention(ctx context.Context, req Request, facts *Facts) {
	if b.sources.Watchtower == nil {
		facts.miss("watchtower")

		return
	}

	counts, err := b.sources.Watchtower.Counts(ctx, repositories.CountWatchtowerItemsRequest{
		TenantInfo: req.TenantInfo,
		SeenAt:     req.DayStart,
	})
	if err != nil || counts == nil {
		facts.miss("watchtower")

		return
	}

	facts.set("watchtower.open", counts.Unresolved)
	facts.set("watchtower.critical", counts.Critical)

	items := make([]briefing.Item, 0, len(counts.ByKind))
	for _, kind := range watchtower.AllSourceKinds() {
		for _, row := range counts.ByKind {
			if row.SourceKind != kind || row.Count == 0 {
				continue
			}
			facts.set("watchtower.kind."+string(kind), row.Count)
			items = append(items, briefing.Item{
				Label: kind.Label(),
				Value: strconv.Itoa(row.Count),
				Path:  watchtowerPath + "?kinds=" + string(kind),
			})
		}
	}

	summary := "Nothing is waiting on the tower."
	if counts.Unresolved > 0 {
		summary = plural(counts.Unresolved, "item", "items") + " open"
		if counts.Critical > 0 {
			summary += ", " + strconv.Itoa(counts.Critical) + " critical"
		}
		summary += "."
	}

	facts.add(briefing.Section{
		Key:     briefing.SectionAttention,
		Title:   "Needs attention",
		Summary: summary,
		Items:   items,
		Path:    watchtowerPath,
	})
	facts.addExceptions(items, counts.Critical)
}

func (b *Builder) decisions(ctx context.Context, req Request, facts *Facts) {
	if b.sources.Decisions == nil {
		facts.miss("decisions")

		return
	}

	summary, err := b.sources.Decisions.Summary(ctx, req.TenantInfo, nil)
	if err != nil || summary == nil {
		facts.miss("decisions")

		return
	}

	facts.set("decisions.total", summary.Total)
	items := make([]briefing.Item, 0, len(summary.ByAgent))
	for _, agent := range summary.ByAgent {
		items = append(items, briefing.Item{
			Label: agent.AgentName,
			Value: strconv.Itoa(agent.Count),
			Path:  decisionsPath + "?agent=" + agent.AgentDefinitionID.String(),
		})
	}

	line := "No agent is waiting on a decision."
	if summary.Total > 0 {
		line = plural(summary.Total, "decision", "decisions") + " waiting"
		if summary.OldestAt != nil {
			days := timeutils.WholeDaysBetween(*summary.OldestAt, req.Now)
			if days >= 1 {
				facts.set("decisions.oldestDays", int(days))
				line += ", the oldest for " + plural(int(days), "day", "days")
			}
		}
		line += "."
	}

	facts.add(briefing.Section{
		Key:     briefing.SectionDecisions,
		Title:   "Waiting on you",
		Summary: line,
		Items:   items,
		Path:    decisionsPath,
	})
}

func (b *Builder) coverage(ctx context.Context, req Request, facts *Facts) {
	if b.sources.Board == nil {
		facts.miss("dispatch board")

		return
	}

	summary, err := b.sources.Board.GetBoardSummary(ctx, &repositories.DispatchBoardFilter{
		TenantInfo:     req.TenantInfo,
		WindowStart:    req.Now,
		WindowEnd:      req.Now + boardWindowHours*60*60,
		DayStartUnix:   req.DayStart,
		IncludeCovered: true,
	})
	if err != nil || summary == nil {
		facts.miss("dispatch board")

		return
	}

	facts.set("coverage.uncovered", summary.UncoveredMoves)
	facts.set("coverage.covered", summary.CoveredMoves)
	facts.set("coverage.late", summary.LateMoves)
	facts.set("coverage.atRisk", summary.AtRiskMoves)
	facts.set("coverage.availableDrivers", summary.AvailableDrivers)

	line := "Every move in the next day is covered."
	if summary.UncoveredMoves > 0 {
		line = plural(summary.UncoveredMoves, "move", "moves") + " still need a driver"
		if summary.AvailableDrivers > 0 {
			line += ", with " + plural(summary.AvailableDrivers, "driver", "drivers") + " available"
		}
		line += "."
	}

	facts.add(briefing.Section{
		Key:     briefing.SectionCoverage,
		Title:   "The next day",
		Summary: line,
		Items: []briefing.Item{
			{
				Label: "Uncovered moves",
				Value: strconv.Itoa(summary.UncoveredMoves),
				Path:  dispatchPath,
			},
			{Label: "Covered moves", Value: strconv.Itoa(summary.CoveredMoves), Path: dispatchPath},
			{Label: "Running late", Value: strconv.Itoa(summary.LateMoves), Path: dispatchPath},
			{Label: "At risk", Value: strconv.Itoa(summary.AtRiskMoves), Path: dispatchPath},
			{Label: "Drivers available", Value: strconv.Itoa(summary.AvailableDrivers)},
		},
		Path: dispatchPath,
	})
}

func (b *Builder) compliance(ctx context.Context, req Request, facts *Facts) {
	if b.sources.Credentials == nil {
		facts.miss("credentials")

		return
	}

	credentials, err := b.sources.Credentials.ListExpiring(
		ctx,
		&repositories.ListExpiringWorkerCredentialsRequest{
			TenantInfo:  req.TenantInfo,
			HorizonDays: credentialHorizonDays,
			AsOf:        req.DayStart,
			Limit:       credentialPageLimit,
		},
	)
	if err != nil {
		facts.miss("credentials")

		return
	}

	expiring := 0
	expired := 0
	for _, credential := range credentials {
		if credential == nil {
			continue
		}
		if credential.ExpiresAt != nil && *credential.ExpiresAt < req.DayStart {
			expired++

			continue
		}
		expiring++
	}
	facts.set("credentials.expiring", expiring)
	facts.set("credentials.expired", expired)

	line := "No credential expires in the next month."
	if expired > 0 || expiring > 0 {
		parts := make([]string, 0, 2)
		if expired > 0 {
			parts = append(
				parts,
				plural(expired, "credential", "credentials")+" already past expiry",
			)
		}
		if expiring > 0 {
			parts = append(
				parts,
				plural(expiring, "credential", "credentials")+" expiring within a month",
			)
		}
		line = strings.Join(parts, ", ") + "."
	}

	facts.add(briefing.Section{
		Key:     briefing.SectionCompliance,
		Title:   "Credentials",
		Summary: line,
		Items: []briefing.Item{
			{Label: "Past expiry", Value: strconv.Itoa(expired), Path: workersPath},
			{Label: "Expiring within a month", Value: strconv.Itoa(expiring), Path: workersPath},
		},
		Path: workersPath,
	})
}

func (b *Builder) billing(ctx context.Context, req Request, facts *Facts) {
	if b.sources.Billing == nil {
		facts.miss("billing queue")

		return
	}

	counts, err := b.sources.Billing.GetStatusCounts(
		ctx,
		&repositories.GetBillingQueueStatsRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		facts.miss("billing queue")

		return
	}

	items := make([]briefing.Item, 0, len(counts))
	total := 0
	// The statuses are sorted so two runs of the same morning lay the
	// block out the same way; a map's order would reshuffle it daily.
	statuses := make([]billingqueue.Status, 0, len(counts))
	for status := range counts {
		statuses = append(statuses, status)
	}
	sort.Slice(statuses, func(i, j int) bool { return statuses[i] < statuses[j] })
	for _, status := range statuses {
		count := counts[status]
		if count == 0 {
			continue
		}
		total += count
		facts.set("billing.status."+string(status), count)
		items = append(items, briefing.Item{
			Label: humanize(string(status)),
			Value: strconv.Itoa(count),
			Path:  billingPath + "?status=" + string(status),
		})
	}
	facts.set("billing.total", total)

	line := "The billing queue is clear."
	if total > 0 {
		line = plural(total, "item", "items") + " in the billing queue."
	}

	facts.add(briefing.Section{
		Key:     briefing.SectionBilling,
		Title:   "Billing queue",
		Summary: line,
		Items:   items,
		Path:    billingPath,
	})
}

func (b *Builder) cash(ctx context.Context, req Request, facts *Facts) {
	if b.sources.Payments == nil {
		facts.miss("payments")

		return
	}

	// Yesterday, in the organization's own day: a cash line that moved
	// with the server's midnight would report a different figure to two
	// offices of the same company.
	received, err := b.sources.Payments.SumReceived(ctx, repositories.SumPaymentsReceivedRequest{
		TenantInfo: req.TenantInfo,
		From:       req.DayStart - 24*60*60,
		To:         req.DayStart - 1,
	})
	if err != nil {
		facts.miss("payments")

		return
	}

	items := make([]briefing.Item, 0, len(received))
	count := 0
	for _, row := range received {
		if row == nil {
			continue
		}
		amount := decimal.NewFromInt(row.AmountMinor).Div(decimal.NewFromInt(100))
		facts.Values["cash."+strings.ToLower(row.CurrencyCode)] = amount
		count += row.Count
		items = append(items, briefing.Item{
			Label: "Received in " + row.CurrencyCode,
			Value: amount.StringFixed(2),
			Path:  paymentsPath,
		})
	}
	facts.set("cash.payments", count)

	line := "No payments were posted yesterday."
	if count > 0 {
		line = plural(count, "payment", "payments") + " posted yesterday."
	}

	facts.add(briefing.Section{
		Key:     briefing.SectionCash,
		Title:   "Cash received yesterday",
		Summary: line,
		Items:   items,
		Path:    paymentsPath,
	})
}

func (b *Builder) detention(ctx context.Context, req Request, facts *Facts) {
	if b.sources.Detention == nil {
		return
	}

	occurrences, err := b.sources.Detention.ListOpen(ctx, req.TenantInfo)
	if err != nil {
		facts.miss("detention")

		return
	}

	billable := 0
	minutes := 0
	for _, occurrence := range occurrences {
		if occurrence == nil {
			continue
		}
		if occurrence.BillableMinutes > 0 {
			billable++
			minutes += int(occurrence.BillableMinutes)
		}
	}
	facts.set("detention.open", len(occurrences))
	facts.set("detention.billable", billable)
	facts.set("detention.billableMinutes", minutes)
}

// addExceptions turns the tower's critical items into their own block, so
// a page that opens with counts still says plainly what is on fire.
func (f *Facts) addExceptions(items []briefing.Item, critical int) {
	if critical == 0 {
		return
	}

	f.add(briefing.Section{
		Key:     briefing.SectionExceptions,
		Title:   "Critical",
		Summary: plural(critical, "item", "items") + " on the tower is critical.",
		Items:   items,
		Path:    watchtowerPath + "?severities=Critical",
	})
}

func (f *Facts) add(section briefing.Section) {
	f.Sections = append(f.Sections, section)
}

func (f *Facts) set(key string, value int) {
	f.Values[key] = decimal.NewFromInt(int64(value))
}

func (f *Facts) miss(source string) {
	f.Unavailable = append(f.Unavailable, source)
}

// Supported is every figure the writer may cite, for the guard.
func (f *Facts) Supported() []decimal.Decimal {
	supported := make([]decimal.Decimal, 0, len(f.Values))
	for _, value := range f.Values {
		supported = append(supported, value)
	}

	return supported
}

// Map renders the facts for storage, so a reader can be shown the working
// and a rerun can be checked against what the day actually looked like.
func (f *Facts) Map() map[string]any {
	out := make(map[string]any, len(f.Values)+1)
	for key, value := range f.Values {
		out[key] = value.String()
	}
	if len(f.Unavailable) > 0 {
		out["unavailable"] = f.Unavailable
	}

	return out
}

func plural(count int, one, many string) string {
	word := many
	if count == 1 {
		word = one
	}

	return strconv.Itoa(count) + " " + word
}

func humanize(value string) string {
	var builder strings.Builder
	for index, r := range value {
		if index > 0 && r >= 'A' && r <= 'Z' {
			builder.WriteByte(' ')
		}
		builder.WriteRune(r)
	}

	return builder.String()
}
