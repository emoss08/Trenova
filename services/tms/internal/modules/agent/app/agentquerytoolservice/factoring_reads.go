package agentquerytoolservice

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	toolListFactoringReadiness = "factoring_list_readiness"
	toolListFactoringNoas      = "factoring_list_noas"
	toolListFactoringSubs      = "factoring_list_submissions"
	toolListFactoringRemits    = "factoring_list_remittances"
	toolListFactoringAgreement = "factoring_list_agreements"

	paramFactoringCompany = "factoringCompanyId"
	paramFactoringInvoice = "invoiceId"
	paramFactoringVerdict = "verdict"
	fieldFactoringChecks  = "checks"
	fieldFactoringPassed  = "passed"
	fieldMatchStatus      = "matchStatus"
	fieldExpiresAt        = "expiresAt"
	fieldEffectiveFrom    = "effectiveFrom"
	fieldEffectiveTo      = "effectiveTo"
	fieldSubmissionTime   = "submissionTime"
	fieldRemittanceCells  = "cells"
	keyFactoringSummary   = "summary"
	keyFactoringPath      = "path"

	factoringQueuePath   = "/billing/factoring"
	factoringSubsPath    = factoringQueuePath + "/submissions/"
	factoringRemitsPath  = factoringQueuePath + "/remittances/"
	factoringMinorAmount = "Amounts are in minor units and can be withheld at your access."

	factoringCompanySource = "invoice_get, factoring_list_readiness or factoring_list_noas"
	factoringCompanyWhat   = "The factor"
	factoringCustomerWhat  = "The customer"
	factoringCustomerFrom  = "customer_list or invoice_get"
)

// factoringCompanyReference is the factor a factoring read is narrowed to or
// keyed by.
var factoringCompanyReference = idReference{
	resource: permission.ResourceFactoringCompany,
	what:     factoringCompanyWhat,
	source:   factoringCompanySource,
}

// factoringReadSpecs are the factoring lists, each read through the
// operation the Desk reads it with.
func factoringReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{
		readinessListSpec(), noasListSpec(), submissionsListSpec(), remittancesListSpec(),
		agreementsListSpec(),
	}
}

// readinessListSpec reads the factoring queue: factored customers' invoices
// by readiness, with the queue's totals.
func readinessListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         toolListFactoringReadiness,
		entityPlural: "invoices",
		summary: "List factored customers' invoices by readiness, newest first, with the " +
			"queue's totals. summary totals what is ready, not ready, submitted, ready for " +
			"more than a day without being submitted, funded, reserve held and chargebacks " +
			"due. Ready is the default; ask for NotReady to see what is holding invoices " +
			"back, each with its failed checks, or Submitted for what is on a schedule. Pass " +
			"Ready invoices to factoring_create_submission, one factor per call, and an " +
			"invoice to factoring_get_readiness for every check. " + factoringMinorAmount,
		searchTerms: []string{"factoring queue", "ready to factor", "factor"},
		queryNote: "Matched only against the verdict; narrow by customerId or " +
			"factoringCompanyId instead.",
		notes: map[string]string{
			paramFactoringVerdict: "Ready unless you name others; NotReady rows carry " +
				"their failed checks.",
		},
		defaults: map[string]any{
			paramFactoringVerdict: []any{string(factoring.VerdictReady)},
		},
		references: map[string]idReference{
			paramFactoringCompany: factoringCompanyReference,
			paramCustomerID: {
				resource: permission.ResourceCustomer,
				what:     factoringCustomerWhat,
				source:   factoringCustomerFrom,
			},
		},
		resource: permission.ResourceInvoice,
		shape:    readinessChecks,
		alongside: []alongsideSpec{{
			key:       keyFactoringSummary,
			operation: "FactoringSummary",
			what:      "queue totals",
		}},
	}
}

// noasListSpec reads the notices of assignment.
func noasListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         toolListFactoringNoas,
		entityPlural: "notices of assignment",
		summary: "List notices of assignment, the letters telling customers to pay a " +
			"factor. Each says who it is for, whether it was sent, acknowledged with a " +
			"signed copy, or released, and when it expires. effective is true only for an " +
			"acknowledged, unexpired notice; an invoice is not ready to factor without " +
			"one. Pass id and version to factoring_send_noa, " +
			"factoring_record_noa_acknowledgement or factoring_release_noa.",
		searchTerms: []string{"notice of assignment", "noa", "factoring letter"},
		queryNote:   "Words to find in the address the notice was sent to.",
		references: map[string]idReference{
			paramFactoringCompany: factoringCompanyReference,
			paramCustomerID: {
				resource: permission.ResourceCustomer,
				what:     factoringCustomerWhat,
				source:   factoringCustomerFrom,
			},
		},
		resource: permission.ResourceFactoringNoa,
		shape:    noaEffective,
	}
}

// submissionsListSpec reads the schedules of accounts sent to factors.
func submissionsListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         toolListFactoringSubs,
		entityPlural: "factoring schedules",
		summary: "List the schedules of accounts sent to factors, newest first. Each has " +
			"its number, factor, status, invoice count, expected funding, where it went and " +
			"a failed one's last error. Pass a schedule's id to factoring_get_submission for its " +
			"invoices; a failed one's id and version go to factoring_retry_submission or " +
			"factoring_cancel_submission, and a sent one's to factoring_record_funding. " +
			factoringMinorAmount,
		searchTerms: []string{"factoring schedule", "schedule of accounts", "submitted to factor"},
		queryNote:   "Words to find in the schedule's number.",
		references: map[string]idReference{
			paramFactoringCompany: factoringCompanyReference,
		},
		resource: permission.ResourceFactoringSubmission,
		shape:    pathTo(factoringSubsPath),
	}
}

// remittancesListSpec reads the factors' remittances.
func remittancesListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         toolListFactoringRemits,
		entityPlural: "factor remittances",
		summary: "List factor remittances, newest first, with how far each has been read " +
			"and booked. Each has its factor, file, status (Pending, NeedsMapping, Parsed, " +
			"Failed, PartiallyApplied, Applied or Discarded), line, matched, exception and " +
			"applied counts, and why it failed if it did. Pass one's id to " +
			"factoring_get_remittance for its lines. " + factoringMinorAmount,
		searchTerms: []string{"factor remittance", "funding report", "remittance exceptions"},
		queryNote:   "Words to find in the remittance's file name.",
		references: map[string]idReference{
			paramFactoringCompany: factoringCompanyReference,
		},
		resource: permission.ResourceFactoringRemittance,
		shape:    pathTo(factoringRemitsPath),
	}
}

// agreementsListSpec reads one factor's agreements.
func agreementsListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         toolListFactoringAgreement,
		entityPlural: "factoring agreements",
		summary: "List a factor's agreements, newest first, marking the one in force " +
			"(inForce). Each gives the advance and reserve rates in basis points, fee tiers " +
			"by days outstanding, minimum fee, recourse and chargeback window, accounting " +
			"treatment, funding days, the daily submission time, and the documents the " +
			"factor needs with every invoice. A factor with no agreement in force has no " +
			"invoices ready to factor. The rates and fees can be withheld at your access.",
		searchTerms: []string{"factoring agreement", "advance rate", "factoring fee", "recourse"},
		references: map[string]idReference{
			paramFactoringCompany: factoringCompanyReference,
		},
		resource: permission.ResourceFactoringAgreement,
		shape:    agreementInForce,
	}
}

// factoringGetSpecs are the factoring gets, each read through the operation
// the Desk reads it with.
func factoringGetSpecs() []getSpec {
	return []getSpec{
		{
			name:   "factoring_get_readiness",
			entity: "invoice's factoring readiness",
			summary: "Check now whether a posted invoice is ready to sell to its factor, and " +
				"why not. Every check comes back with what clears it: SetUpAgreement, " +
				"SendNoa, RecordNoaAcknowledgement, ResolveDispute, RequestDocument (with the " +
				"document type and shipment) or ReviewShipment. An Excluded verdict means the " +
				"invoice is not something to factor: a draft, voided, paid, a credit memo, " +
				"kept out by factoring_set_override, or for a customer that is not factored.",
			searchTerms: []string{"factor", "factoring", "ready to factor", "notice of assignment",
				"noa"},
			resource: permission.ResourceInvoice,
			references: map[string]idReference{paramFactoringInvoice: {
				resource: permission.ResourceInvoice,
				what:     "The invoice",
				source:   "invoice_list, invoice_get or factoring_list_readiness",
			}},
			shape: func(record map[string]any) { record[keyFactoringPath] = factoringQueuePath },
		},
		{
			name:   "factoring_get_submission",
			entity: "factoring schedule",
			summary: "Read one schedule of accounts with every invoice on it and why it " +
				"failed, if it did. Each invoice (items) has its item id, amount, expected " +
				"advance, fee and reserve, its packet once built, and once the factor decided " +
				"it, what it funded (the advance, fee and reserve it holds) or why it " +
				"rejected it, and whether its reserve was released or it was charged back; " +
				"the schedule has its status, where it went, attempts and last error. Use " +
				"its version and an invoice's item id with factoring_record_funding, " +
				"factoring_release_reserve, factoring_record_chargeback or " +
				"factoring_reverse_chargeback, and with factoring_retry_submission or " +
				"factoring_cancel_submission for a failed one. " + factoringMinorAmount,
			searchTerms: []string{"factoring schedule", "schedule of accounts", "failed schedule"},
			resource:    permission.ResourceFactoringSubmission,
			paramName:   "submissionId",
			idSource:    "from factoring_list_submissions",
			shape:       recordPath(factoringSubsPath),
		},
		{
			name:   "factoring_get_remittance",
			entity: "factor remittance",
			summary: "Read one factor remittance with its lines and how each was matched. " +
				"Each line has its row, kind, invoice number, amounts and date as the factor " +
				"printed them, its match status, the schedule item it is matched to, and up " +
				"to three ranked suggestions with why; an unreadable line has its cells. Use " +
				"its version and a line id with factoring_resolve_remittance_line, and its " +
				"version with factoring_apply_remittance or factoring_discard_remittance. " +
				factoringMinorAmount,
			searchTerms: []string{"factor remittance", "remittance lines", "remittance exceptions"},
			resource:    permission.ResourceFactoringRemittance,
			paramName:   "remittanceId",
			idSource:    "from factoring_list_remittances",
			notes: map[string]string{
				"matchStatuses": "Leave it out for every line.",
			},
			shape: remittanceLines,
		},
		{
			name:   "factoring_get_reserve_balance",
			entity: "factoring reserve balance",
			summary: "Read what a factor holds and has paid, from the factoring entries booked " +
				"to the ledger. It shows the reserve it still holds, what is owed it under " +
				"secured borrowing, the advances it paid, the reserves it released, what was " +
				"paid back for chargebacks net of reversals, the fees booked, and the day of " +
				"the last movement. Use it before factoring_release_reserve to see what the " +
				"factor should remit. " + factoringMinorAmount,
			searchTerms: []string{"factoring reserve", "reserve balance", "due from factor",
				"factor fees"},
			resource: permission.ResourceFactoringReserve,
			references: map[string]idReference{paramFactoringCompany: {
				resource: permission.ResourceFactoringCompany,
				what:     factoringCompanyWhat,
				source:   "factoring_list_submissions or factoring_list_agreements",
			}},
		},
	}
}

// readinessChecks keeps a NotReady invoice's failed checks, which say what
// holds it back, and drops every other invoice's.
func readinessChecks(row map[string]any, _ map[string]struct{}, _ clock) {
	if stringOf(row[paramFactoringVerdict]) != string(factoring.VerdictNotReady) {
		delete(row, fieldFactoringChecks)
		return
	}
	checks, _ := row[fieldFactoringChecks].([]any)
	failed := make([]any, 0, len(checks))
	for _, check := range checks {
		object, _ := check.(map[string]any)
		if passed, _ := object[fieldFactoringPassed].(bool); !passed {
			failed = append(failed, check)
		}
	}
	row[fieldFactoringChecks] = failed
}

// noaEffective says whether a notice binds the customer now, as the domain
// decides it: acknowledged and not expired.
func noaEffective(row map[string]any, _ map[string]struct{}, clk clock) {
	expires, ends := typeutils.IntOf(row[fieldExpiresAt])
	row["effective"] = stringOf(row[accountingStatusField]) ==
		string(factoring.NoaStatusAcknowledged) &&
		(!ends || clk.Instant() < int64(expires))
}

// agreementInForce marks the agreement that covers invoices now, as the
// domain decides it, and writes the daily submission time as a clock.
func agreementInForce(row map[string]any, _ map[string]struct{}, clk clock) {
	now := clk.Instant()
	from, started := typeutils.IntOf(row[fieldEffectiveFrom])
	to, ends := typeutils.IntOf(row[fieldEffectiveTo])
	row["inForce"] = stringOf(row[accountingStatusField]) ==
		string(factoring.AgreementStatusActive) &&
		started && now >= int64(from) && (!ends || now < int64(to))
	if minutes, ok := typeutils.IntOf(row[fieldSubmissionTime]); ok {
		row[fieldSubmissionTime] = fmt.Sprintf("%02d:%02d", minutes/minutesPerHour,
			minutes%minutesPerHour)
	}
}

// pathTo names the page each row of a list opens on.
func pathTo(prefix string) func(map[string]any, map[string]struct{}, clock) {
	return func(row map[string]any, _ map[string]struct{}, _ clock) {
		if id := stringOf(row[recordIDField]); id != "" {
			row[keyFactoringPath] = prefix + id
		}
	}
}

// recordPath names the page a record opens on.
func recordPath(prefix string) func(map[string]any) {
	open := pathTo(prefix)

	return func(record map[string]any) { open(record, nil, clock{}) }
}

// remittanceLines keeps an unreadable line's cells, which say what the
// factor printed, and drops every other line's, and names the remittance's
// page.
func remittanceLines(record map[string]any) {
	recordPath(factoringRemitsPath)(record)
	lines, _ := record["lines"].([]any)
	for _, line := range lines {
		object, _ := line.(map[string]any)
		if stringOf(object[fieldMatchStatus]) != string(factoring.RemittanceLineError) {
			delete(object, fieldRemittanceCells)
		}
	}
}
