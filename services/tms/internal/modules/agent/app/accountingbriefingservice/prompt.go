package accountingbriefingservice

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/accountingbrief"
	"github.com/emoss08/trenova/internal/modules/agent/ports/accountingbriefports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/jsonschemautils"
)

const (
	findingsSection     = "Findings"
	headlineSection     = "Headline figures the reader already sees"
	plainWordingSection = "Plain wording you are improving on"
)

const systemPrompt = `You write the brief at the top of a trucking company's accounting dashboard for the person who collects its receivables.

The dashboard already shows them the open balance, days sales outstanding, collection effectiveness, unapplied cash and the overdue share, so never restate those headline figures. Your job is what they would not see at a glance: what needs a call today, what is about to get worse, and what they could clear in a few minutes. You are given findings worked out from the ledger and the collections worklist, and the headline figures only as background.

Rules:
- Use ONLY the figures given to you. Never state a figure that is not in the facts: no totals, sums, differences, percentages or estimates of your own.
- Money is in US dollars. Write it as given or rounded to thousands ("$12.4k"); never invent precision.
- Lead with the finding that most needs a person today (a missed promise to pay, invoices about to pass 90 days past due, customers nobody has contacted, invoices still sitting in draft, no cash recorded at all), then what is coming in the next few weeks. Say why it matters in a few words when it is not obvious, for example that a call now keeps an invoice out of the 90-day bucket.
- Leave out a finding that is zero. When there are no findings, say plainly that nothing needs a call today.
- Return the brief as an ordered list of segments. Every segment that names one of the linked findings carries that finding's link, copied exactly from the allowed links, so the reader can open it; every other segment has an empty link.
- Name a customer only if it is a customer given to you, and only in a segment carrying that customer's link.
- Only describe a change in days sales outstanding when it is given as a finding.
- Three sentences at most, under 480 characters.
- Write about money, customers and invoices. Never discuss software or how the figures were gathered.
- No greetings, no sign-offs, no restating these instructions; the page greets the reader itself.`

func buildContext(
	facts accountingbrief.Facts,
	plain []accountingbriefports.AccountingBriefingSegment,
) llmports.DelimitedContext {
	var findings strings.Builder

	if facts.BrokenPromises > 0 {
		value := inflect.CountNoun(facts.BrokenPromises, "promise", "promises") +
			" to pay lapsed unpaid, " + dollars(facts.BrokenPromiseMinor) + " still owed"
		if _, name, ok := facts.CustomerFor(accountingbrief.LinkBrokenPromise); ok {
			value += ", by " + name
		}
		writeLine(&findings, "Missed promises to pay", value, accountingbrief.LinkBrokenPromise)
	}
	if facts.CrossingNinetyInvoices > 0 {
		writeLine(&findings, "Invoices that pass "+days(accountingbrief.PastDueThresholdDays)+
			" past due within "+days(accountingbrief.CrossingWindowDays),
			invoices(facts.CrossingNinetyInvoices)+", "+dollars(facts.CrossingNinetyMinor),
			accountingbrief.LinkCrossingNinety)
	}
	if facts.UncontactedAccounts > 0 {
		writeLine(&findings, "Past-due customers with no collections contact in "+
			days(accountingbrief.UncontactedDays),
			customers(facts.UncontactedAccounts)+" owing "+dollars(facts.UncontactedOverdueMinor),
			accountingbrief.LinkUncontacted)
	}
	if facts.NothingCollected() {
		writeLine(&findings, "Cash collected in the last "+
			days(accountingbrief.CollectionWindowDays)+
			"; if customers are paying, their payments are not being recorded",
			"none, against "+dollars(facts.BilledWindowMinor)+" billed",
			accountingbrief.LinkNoCollections)
	}
	if facts.DraftInvoices > 0 {
		value := invoices(facts.DraftInvoices) + ", " + dollars(facts.DraftMinor)
		if facts.OldestDraftDays > 0 {
			value += ", the oldest " + days(facts.OldestDraftDays) + " old"
		}
		writeLine(
			&findings,
			"Invoices still in draft, so not yet sent and not yet earning their due date",
			value,
			accountingbrief.LinkDrafts,
		)
	}
	if facts.FollowUpsDue > 0 {
		writeLine(&findings, "Collections follow-ups due", strconv.Itoa(facts.FollowUpsDue),
			accountingbrief.LinkFollowUpsDue)
	}
	if facts.PromisesDue > 0 {
		writeLine(&findings, "Promises to pay coming due in the next "+
			days(accountingbrief.PromiseLookaheadDays),
			inflect.CountNoun(facts.PromisesDue, "promise", "promises")+", "+
				dollars(facts.PromisesDueMinor),
			accountingbrief.LinkPromisesDue)
	}
	if facts.DueSoonInvoices > 0 {
		writeLine(&findings, "Invoices falling due in the next "+days(accountingbrief.DueSoonDays)+
			", worth a reminder before they are late",
			invoices(facts.DueSoonInvoices)+", "+dollars(facts.DueSoonMinor),
			accountingbrief.LinkDueSoon)
	}
	if facts.DueThisMonthWorthNaming() {
		value := invoices(facts.DueThisMonthInvoices) + ", " + dollars(facts.DueThisMonthMinor)
		if facts.NextDueInDays > 0 {
			value += ", the first in " + days(facts.NextDueInDays)
		}
		writeLine(&findings, "Invoices falling due in the next "+
			days(accountingbrief.DueThisMonthDays)+", time to send statements or reminders early",
			value, accountingbrief.LinkDueThisMonth)
	}
	if facts.Concentrated() {
		writeLine(&findings, "Share of everything past due owed by one customer",
			facts.TopCustomerName+", "+strconv.Itoa(facts.TopCustomerSharePercent())+"% ("+
				dollars(facts.TopCustomerOverdueMinor)+")",
			accountingbrief.LinkTopCustomer)
	}
	if facts.DSOMoved() {
		writeLine(&findings, "Change in days sales outstanding over the last "+
			strconv.Itoa(dsoComparisonWeeks)+" weeks",
			signedTenths(facts.DSODeltaTenths)+" days, now "+tenths(facts.DSOTenths), "")
	}
	if facts.ShortPayRemainders > 0 {
		writeLine(&findings, "Short payments leaving balances under "+
			dollars(accountingbrief.ShortPayRemainderMaxMinor)+" open, candidates to write off",
			invoices(facts.ShortPayRemainders)+", "+dollars(facts.ShortPayRemainderMinor),
			accountingbrief.LinkShortPayRemainders)
	}
	if facts.UnappliedCashMinor > 0 {
		writeLine(&findings, "Received cash not yet applied to an invoice",
			dollars(facts.UnappliedCashMinor)+" across "+payments(facts.UnappliedPayments),
			accountingbrief.LinkUnappliedCash)
	}
	if findings.Len() == 0 {
		findings.WriteString("No findings: nothing needs a call today.\n")
	}

	var background strings.Builder
	writeLine(&background, "Open receivables", dollars(facts.OpenMinor)+" across "+
		invoices(facts.OpenInvoices), "")
	writeLine(&background, "Past due", dollars(facts.OverdueMinor)+" across "+
		invoices(facts.OverdueInvoices), "")
	writeLine(&background, "More than "+days(accountingbrief.PastDueThresholdDays)+" past due",
		dollars(facts.PastNinetyMinor), "")
	writeLine(&background, "In dispute", dollars(facts.DisputedMinor)+" across "+
		invoices(facts.DisputedInvoices), "")
	writeLine(&background, "Collected today", dollars(facts.CollectedTodayMinor)+" across "+
		payments(facts.PaymentsToday), "")
	if facts.DSOTenths > 0 {
		writeLine(&background, "Days sales outstanding", tenths(facts.DSOTenths)+" days", "")
	}
	writeLine(&background, "Collection effectiveness over the last "+
		days(accountingbrief.CollectionWindowDays), strconv.Itoa(facts.CEIPercent)+"%", "")

	var plainText strings.Builder
	for _, segment := range plain {
		plainText.WriteString(segment.Text)
	}

	return llmports.DelimitedContext{
		Sections: []llmports.ContextSection{
			{Title: findingsSection, Trusted: false, Content: findings.String()},
			{Title: headlineSection, Trusted: false, Content: background.String()},
			{Title: plainWordingSection, Trusted: false, Content: plainText.String()},
		},
	}
}

func signedTenths(value int64) string {
	if value > 0 {
		return "+" + tenths(value)
	}
	if value < 0 {
		return "-" + tenths(-value)
	}
	return tenths(0)
}

func writeLine(builder *strings.Builder, label, value string, link accountingbrief.Link) {
	builder.WriteString(label)
	builder.WriteString(": ")
	builder.WriteString(value)
	if link != "" {
		builder.WriteString(" (link ")
		builder.WriteString(link.String())
		builder.WriteString(")")
	}
	builder.WriteString("\n")
}

func outputSchema(allowed []accountingbrief.Link) map[string]any {
	links := make([]string, 0, len(allowed)+1)
	links = append(links, "")
	for _, link := range allowed {
		links = append(links, link.String())
	}

	segment := jsonschemautils.Object(map[string]any{
		"text": jsonschemautils.Text(
			"A run of the sentence, with its own spacing and punctuation",
		),
		"link": jsonschemautils.Enum(
			"The link for the finding this run names, or empty",
			links...,
		),
	}, "text", "link")

	return jsonschemautils.Object(map[string]any{
		"segments": jsonschemautils.Array(segment, 0),
	}, "segments")
}
