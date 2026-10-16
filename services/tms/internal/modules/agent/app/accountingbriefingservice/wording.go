package accountingbriefingservice

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/accountingbrief"
	"github.com/emoss08/trenova/internal/modules/agent/ports/accountingbriefports"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	dsoComparisonWeeks = 4
	maxClausesPerLine  = 3
)

type clause = []accountingbriefports.AccountingBriefingSegment

func text(value string) accountingbriefports.AccountingBriefingSegment {
	return accountingbriefports.AccountingBriefingSegment{Text: value}
}

func linked(value string, link accountingbrief.Link) accountingbriefports.AccountingBriefingSegment {
	return accountingbriefports.AccountingBriefingSegment{Text: value, Link: &link}
}

func dollars(minor int64) string {
	return money.FormatWholeDollars(money.DecimalFromMinor(minor))
}

func tenths(value int64) string {
	return decimal.New(value, -1).StringFixed(1)
}

func invoices(count int) string {
	return stringutils.CountNoun(count, "invoice", "invoices")
}

func payments(count int) string {
	return stringutils.CountNoun(count, "payment", "payments")
}

func customers(count int) string {
	return stringutils.CountNoun(count, "customer", "customers")
}

func days(count int) string {
	return stringutils.CountNoun(count, "day", "days")
}

func verb(singular, plural string, count int) string {
	return stringutils.Pluralize(singular, plural, count)
}

func sentence(clauses []clause) []accountingbriefports.AccountingBriefingSegment {
	segments := make([]accountingbriefports.AccountingBriefingSegment, 0, len(clauses)*2+1)
	for i, c := range clauses {
		switch {
		case i == 0:
		case i == len(clauses)-1:
			segments = append(segments, text(" and "))
		default:
			segments = append(segments, text(", "))
		}
		segments = append(segments, c...)
	}
	if len(clauses) > 0 {
		segments = append(segments, text(". "))
	}
	return segments
}

func Deterministic(f accountingbrief.Facts) []accountingbriefports.AccountingBriefingSegment {
	if !f.HasReceivables() {
		return []accountingbriefports.AccountingBriefingSegment{
			text("Nothing is owed to you and no cash is waiting to be applied."),
		}
	}

	today := firstClauses(todayClauses(f))
	if len(today) > 0 {
		today[0][0].Text = capitalize(today[0][0].Text)
	}
	ahead := firstClauses(aheadClauses(f))
	if len(ahead) > 0 {
		ahead[0][0].Text = capitalize(ahead[0][0].Text)
	}

	segments := make([]accountingbriefports.AccountingBriefingSegment, 0, 24)
	if len(today) == 0 {
		segments = append(segments, text(quietLine(f)+" "))
	} else {
		segments = append(segments, sentence(today)...)
	}
	segments = append(segments, sentence(ahead)...)

	last := len(segments) - 1
	segments[last].Text = strings.TrimRight(segments[last].Text, " ")

	return segments
}

func quietLine(f accountingbrief.Facts) string {
	if f.OverdueInvoices == 0 {
		return "Nothing is past due, so nobody needs a call today."
	}
	return "Every past-due customer has been contacted in the last " +
		days(accountingbrief.UncontactedDays) +
		", no promise to pay has lapsed and nothing is about to pass " +
		days(accountingbrief.PastDueThresholdDays) + " past due."
}

func todayClauses(f accountingbrief.Facts) []clause {
	clauses := make([]clause, 0, 4)

	if f.BrokenPromises > 0 {
		if _, name, ok := f.CustomerFor(accountingbrief.LinkBrokenPromise); ok {
			clauses = append(clauses, clause{linked(
				name+" missed a promise to pay "+dollars(f.BrokenPromiseMinor),
				accountingbrief.LinkBrokenPromise,
			)})
		} else {
			clauses = append(clauses, clause{linked(
				stringutils.CountNoun(f.BrokenPromises, "promise", "promises")+" to pay "+
					dollars(f.BrokenPromiseMinor)+" "+verb("has", "have", f.BrokenPromises)+
					" lapsed unpaid",
				accountingbrief.LinkBrokenPromise,
			)})
		}
	}
	if f.CrossingNinetyInvoices > 0 {
		clauses = append(clauses, clause{linked(
			invoices(f.CrossingNinetyInvoices)+" worth "+dollars(f.CrossingNinetyMinor)+" "+
				verb("passes", "pass", f.CrossingNinetyInvoices)+" "+
				days(accountingbrief.PastDueThresholdDays)+" past due this week",
			accountingbrief.LinkCrossingNinety,
		)})
	}
	if f.UncontactedAccounts > 0 {
		clauses = append(clauses, clause{linked(
			customers(f.UncontactedAccounts)+" owing "+dollars(f.UncontactedOverdueMinor)+
				" past due "+verb("has", "have", f.UncontactedAccounts)+
				" had no collections contact in "+days(accountingbrief.UncontactedDays),
			accountingbrief.LinkUncontacted,
		)})
	}
	if f.NothingCollected() {
		clauses = append(clauses, clause{linked(
			"nothing has been collected in the last "+days(accountingbrief.CollectionWindowDays)+
				" against "+dollars(f.BilledWindowMinor)+" billed",
			accountingbrief.LinkNoCollections,
		)})
	}
	if f.DraftInvoices > 0 {
		line := invoices(f.DraftInvoices) + " worth " + dollars(f.DraftMinor) + " " +
			verb("is", "are", f.DraftInvoices) + " still in draft"
		if f.OldestDraftDays > 0 {
			line += ", the oldest " + days(f.OldestDraftDays) + " old"
		}
		clauses = append(clauses, clause{linked(line, accountingbrief.LinkDrafts)})
	}
	if f.FollowUpsDue > 0 {
		clauses = append(clauses, clause{linked(
			stringutils.CountNoun(f.FollowUpsDue, "follow-up", "follow-ups")+" "+
				verb("is", "are", f.FollowUpsDue)+" due",
			accountingbrief.LinkFollowUpsDue,
		)})
	}

	return clauses
}

func aheadClauses(f accountingbrief.Facts) []clause {
	clauses := make([]clause, 0, 6)

	if f.PromisesDue > 0 {
		clauses = append(clauses, clause{linked(
			stringutils.CountNoun(f.PromisesDue, "promise", "promises")+" to pay "+
				dollars(f.PromisesDueMinor)+" "+verb("comes", "come", f.PromisesDue)+
				" due in the next "+days(accountingbrief.PromiseLookaheadDays),
			accountingbrief.LinkPromisesDue,
		)})
	}
	if f.DueSoonInvoices > 0 {
		clauses = append(clauses, clause{linked(
			dollars(f.DueSoonMinor)+" across "+invoices(f.DueSoonInvoices)+
				" falls due in the next "+days(accountingbrief.DueSoonDays),
			accountingbrief.LinkDueSoon,
		)})
	}
	if f.DueThisMonthWorthNaming() {
		line := dollars(f.DueThisMonthMinor) + " across " + invoices(f.DueThisMonthInvoices) +
			" falls due in the next " + days(accountingbrief.DueThisMonthDays)
		if f.NextDueInDays > 0 {
			line += ", the first in " + days(f.NextDueInDays)
		}
		clauses = append(clauses, clause{linked(line, accountingbrief.LinkDueThisMonth)})
	}
	if f.Concentrated() {
		clauses = append(clauses, clause{
			linked(f.TopCustomerName, accountingbrief.LinkTopCustomer),
			text(" holds " + strconv.Itoa(f.TopCustomerSharePercent()) + "% of everything past due"),
		})
	}
	if f.DSOMoved() {
		direction, change := "rose", f.DSODeltaTenths
		if change < 0 {
			direction, change = "fell", -change
		}
		clauses = append(clauses, clause{text(
			"days sales outstanding " + direction + " " + tenths(change) + " days in " +
				strconv.Itoa(dsoComparisonWeeks) + " weeks, to " + tenths(f.DSOTenths),
		)})
	}
	if f.ShortPayRemainders > 0 {
		clauses = append(clauses, clause{linked(
			stringutils.CountNoun(f.ShortPayRemainders, "short payment", "short payments")+" "+
				verb("leaves", "leave", f.ShortPayRemainders)+" "+dollars(f.ShortPayRemainderMinor)+
				" open in balances under "+dollars(accountingbrief.ShortPayRemainderMaxMinor)+
				" that could be written off",
			accountingbrief.LinkShortPayRemainders,
		)})
	}
	if f.UnappliedCashMinor > 0 {
		clauses = append(clauses, clause{linked(
			dollars(f.UnappliedCashMinor)+" of received cash is still unapplied",
			accountingbrief.LinkUnappliedCash,
		)})
	}

	return clauses
}

func firstClauses(clauses []clause) []clause {
	return clauses[:min(len(clauses), maxClausesPerLine)]
}

func capitalize(value string) string {
	if value == "" || value[0] < 'a' || value[0] > 'z' {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func supportedValues(f accountingbrief.Facts) []decimal.Decimal {
	minors := []int64{
		f.OpenMinor,
		f.OverdueMinor,
		f.PastNinetyMinor,
		f.DisputedMinor,
		f.UnappliedCashMinor,
		f.CollectedTodayMinor,
		f.TopCustomerOverdueMinor,
		f.CrossingNinetyMinor,
		f.DueSoonMinor,
		f.ShortPayRemainderMinor,
		f.BrokenPromiseMinor,
		f.PromisesDueMinor,
		f.UncontactedOverdueMinor,
		f.DraftMinor,
		f.BilledWindowMinor,
		f.DueThisMonthMinor,
		accountingbrief.ShortPayRemainderMaxMinor,
	}
	counts := []int{
		f.OpenInvoices,
		f.OverdueInvoices,
		f.DisputedInvoices,
		f.UnappliedPayments,
		f.PaymentsToday,
		f.CEIPercent,
		f.CrossingNinetyInvoices,
		f.DueSoonInvoices,
		f.ShortPayRemainders,
		f.BrokenPromises,
		f.PromisesDue,
		f.UncontactedAccounts,
		f.FollowUpsDue,
		f.TopCustomerSharePercent(),
		f.DraftInvoices,
		f.OldestDraftDays,
		f.DueThisMonthInvoices,
		f.NextDueInDays,
		accountingbrief.DueThisMonthDays,
		accountingbrief.PastDueThresholdDays,
		accountingbrief.UncontactedDays,
		accountingbrief.DueSoonDays,
		accountingbrief.PromiseLookaheadDays,
		dsoComparisonWeeks,
		accountingbrief.CollectionWindowDays,
	}

	out := make([]decimal.Decimal, 0, len(minors)+len(counts)+2)
	for _, minor := range minors {
		out = append(out, money.DecimalFromMinor(minor))
	}
	for _, count := range counts {
		out = append(out, decimal.NewFromInt(int64(count)))
	}
	out = append(out, decimal.New(f.DSOTenths, -1))
	if f.HasDSOBaseline() {
		out = append(out, decimal.New(f.DSODeltaTenths, -1).Abs())
	}

	return out
}
