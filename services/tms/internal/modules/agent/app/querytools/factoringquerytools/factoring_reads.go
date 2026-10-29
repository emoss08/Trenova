package factoringquerytools

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/filtercatalog"
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
	fieldSubmissionTime   = "submissionTime"
	fieldRemittanceCells  = "cells"
	keyFactoringSummary   = "summary"
	entityInvoices        = "invoices"
	keyFactoringPath      = "path"
	factoringScheduleTerm = "factoring schedule"
	factorRemittanceTerm  = "factor remittance"

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
var factoringCompanyReference = readspec.Reference{
	Resource: permission.ResourceFactoringCompany,
	What:     factoringCompanyWhat,
	Source:   factoringCompanySource,
}

// ReadSpecs are the factoring lists, each read through the
// operation the Desk reads it with.
func ReadSpecs() []readspec.List {
	return []readspec.List{
		readinessListSpec(), noasListSpec(), submissionsListSpec(), remittancesListSpec(),
		agreementsListSpec(),
	}
}

// readinessListSpec reads the factoring queue: factored customers' invoices
// by readiness, with the queue's totals.
func readinessListSpec() readspec.List {
	return readspec.List{
		Name:         toolListFactoringReadiness,
		EntityPlural: entityInvoices,
		Summary: "List factored customers' invoices by readiness, newest first, with the " +
			"queue's totals. summary totals what is ready, not ready, submitted, ready for " +
			"more than a day without being submitted, funded, reserve held and chargebacks " +
			"due. Ready is the default; ask for NotReady to see what is holding invoices " +
			"back, each with its failed checks, or Submitted for what is on a schedule. Pass " +
			"Ready invoices to factoring_create_submission, one factor per call, and an " +
			"invoice to factoring_get_readiness for every check. " + factoringMinorAmount,
		SearchTerms: []string{"factoring queue", "ready to factor", "factor"},
		QueryNote: "Matched only against the verdict; narrow by customerId or " +
			"factoringCompanyId instead.",
		Notes: map[string]string{
			paramFactoringVerdict: "Ready unless you name others; NotReady rows carry " +
				"their failed checks.",
		},
		Defaults: map[string]any{
			paramFactoringVerdict: []any{string(factoringreadiness.VerdictReady)},
		},
		References: map[string]readspec.Reference{
			paramFactoringCompany: factoringCompanyReference,
			querytool.ParamCustomerID: {
				Resource: permission.ResourceCustomer,
				What:     factoringCustomerWhat,
				Source:   factoringCustomerFrom,
			},
		},
		Resource: permission.ResourceInvoice,
		Shape:    readinessChecks,
		Alongside: []readspec.Alongside{{
			Key:       keyFactoringSummary,
			Operation: "FactoringSummary",
			What:      "queue totals",
		}},
	}
}

// noasListSpec reads the notices of assignment.
func noasListSpec() readspec.List {
	return readspec.List{
		Name:         toolListFactoringNoas,
		EntityPlural: "notices of assignment",
		Summary: "List notices of assignment, the letters telling customers to pay a " +
			"factor. Each says who it is for, whether it was sent, acknowledged with a " +
			"signed copy, or released, and when it expires. effective is true only for an " +
			"acknowledged, unexpired notice; an invoice is not ready to factor without " +
			"one. Pass id and version to factoring_send_noa, " +
			"factoring_record_noa_acknowledgement or factoring_release_noa.",
		SearchTerms: []string{"notice of assignment", "noa", "factoring letter"},
		QueryNote:   "Words to find in the address the notice was sent to.",
		References: map[string]readspec.Reference{
			paramFactoringCompany: factoringCompanyReference,
			querytool.ParamCustomerID: {
				Resource: permission.ResourceCustomer,
				What:     factoringCustomerWhat,
				Source:   factoringCustomerFrom,
			},
		},
		Resource: permission.ResourceFactoringNoa,
		Shape:    noaEffective,
	}
}

// submissionsListSpec reads the schedules of accounts sent to factors.
func submissionsListSpec() readspec.List {
	return readspec.List{
		Name:         toolListFactoringSubs,
		EntityPlural: "factoring schedules",
		Summary: "List the schedules of accounts sent to factors, newest first. Each has " +
			"its number, factor, status, invoice count, expected funding, where it went and " +
			"a failed one's last error. Pass a schedule's id to factoring_get_submission for its " +
			"invoices; a failed one's id and version go to factoring_retry_submission or " +
			"factoring_cancel_submission, and a sent one's to factoring_record_funding. " +
			factoringMinorAmount,
		SearchTerms: []string{factoringScheduleTerm, "schedule of accounts", "submitted to factor"},
		QueryNote:   "Words to find in the schedule's number.",
		References: map[string]readspec.Reference{
			paramFactoringCompany: factoringCompanyReference,
		},
		Resource: permission.ResourceFactoringSubmission,
		Shape:    pathTo(factoringSubsPath),
	}
}

// remittancesListSpec reads the factors' remittances.
func remittancesListSpec() readspec.List {
	return readspec.List{
		Name:         toolListFactoringRemits,
		EntityPlural: "factor remittances",
		Summary: "List factor remittances, newest first, with how far each has been read " +
			"and booked. Each has its factor, file, status (Pending, NeedsMapping, Parsed, " +
			"Failed, PartiallyApplied, Applied or Discarded), line, matched, exception and " +
			"applied counts, and why it failed if it did. Pass one's id to " +
			"factoring_get_remittance for its lines. " + factoringMinorAmount,
		SearchTerms: []string{factorRemittanceTerm, "funding report", "remittance exceptions"},
		QueryNote:   "Words to find in the remittance's file name.",
		References: map[string]readspec.Reference{
			paramFactoringCompany: factoringCompanyReference,
		},
		Resource: permission.ResourceFactoringRemittance,
		Shape:    pathTo(factoringRemitsPath),
	}
}

// agreementsListSpec reads one factor's agreements.
func agreementsListSpec() readspec.List {
	return readspec.List{
		Name:         toolListFactoringAgreement,
		EntityPlural: "factoring agreements",
		Summary: "List a factor's agreements, newest first, marking the one in force " +
			"(inForce). Each gives the advance and reserve rates in basis points, fee tiers " +
			"by days outstanding, minimum fee, recourse and chargeback window, accounting " +
			"treatment, funding days, the daily submission time, and the documents the " +
			"factor needs with every invoice. A factor with no agreement in force has no " +
			"invoices ready to factor. The rates and fees can be withheld at your access.",
		SearchTerms: []string{"factoring agreement", "advance rate", "factoring fee", "recourse"},
		References: map[string]readspec.Reference{
			paramFactoringCompany: factoringCompanyReference,
		},
		Resource: permission.ResourceFactoringAgreement,
		Shape:    agreementInForce,
	}
}

// GetSpecs are the factoring gets, each read through the operation
// the Desk reads it with.
func GetSpecs() []readspec.Get {
	return []readspec.Get{
		{
			Name:   "factoring_get_readiness",
			Entity: "invoice's factoring readiness",
			Summary: "Check now whether a posted invoice is ready to sell to its factor, and " +
				"why not. Every check comes back with what clears it: SetUpAgreement, " +
				"SendNoa, RecordNoaAcknowledgement, ResolveDispute, RequestDocument (with the " +
				"document type and shipment) or ReviewShipment. An Excluded verdict means the " +
				"invoice is not something to factor: a draft, voided, paid, a credit memo, " +
				"kept out by factoring_set_override, or for a customer that is not factored.",
			SearchTerms: []string{"factor", "factoring", "ready to factor", "notice of assignment",
				"noa"},
			Resource:  permission.ResourceInvoice,
			ParamName: paramFactoringInvoice,
			References: map[string]readspec.Reference{paramFactoringInvoice: {
				Resource: permission.ResourceInvoice,
				What:     "The invoice",
				Source:   "invoice_list, invoice_get or factoring_list_readiness",
			}},
			Shape: func(record map[string]any, _ map[string]struct{}, _ filtercatalog.Clock) {
				record[keyFactoringPath] = factoringQueuePath
			},
		},
		{
			Name:   "factoring_get_submission",
			Entity: factoringScheduleTerm,
			Summary: "Read one schedule of accounts with every invoice on it and why it " +
				"failed, if it did. Each invoice (items) has its item id, amount, expected " +
				"advance, fee and reserve, its packet once built, and once the factor decided " +
				"it, what it funded (the advance, fee and reserve it holds) or why it " +
				"rejected it, and whether its reserve was released or it was charged back; " +
				"the schedule has its status, where it went, attempts and last error. Use " +
				"its version and an invoice's item id with factoring_record_funding, " +
				"factoring_release_reserve, factoring_record_chargeback or " +
				"factoring_reverse_chargeback, and with factoring_retry_submission or " +
				"factoring_cancel_submission for a failed one. " + factoringMinorAmount,
			SearchTerms: []string{factoringScheduleTerm, "schedule of accounts", "failed schedule"},
			Resource:    permission.ResourceFactoringSubmission,
			ParamName:   "submissionId",
			IDSource:    "from factoring_list_submissions",
			Shape:       pathTo(factoringSubsPath),
		},
		{
			Name:   "factoring_get_remittance",
			Entity: factorRemittanceTerm,
			Summary: "Read one factor remittance with its lines and how each was matched. " +
				"Each line has its row, kind, invoice number, amounts and date as the factor " +
				"printed them, its match status, the schedule item it is matched to, and up " +
				"to three ranked suggestions with why; an unreadable line has its cells. Use " +
				"its version and a line id with factoring_resolve_remittance_line, and its " +
				"version with factoring_apply_remittance or factoring_discard_remittance. " +
				factoringMinorAmount,
			SearchTerms: []string{
				factorRemittanceTerm,
				"remittance lines",
				"remittance exceptions",
			},
			Resource:  permission.ResourceFactoringRemittance,
			ParamName: "remittanceId",
			IDSource:  "from factoring_list_remittances",
			Notes: map[string]string{
				"matchStatuses": "Leave it out for every line.",
			},
			Shape: remittanceLines,
		},
		{
			Name:   "factoring_get_reserve_balance",
			Entity: "factoring reserve balance",
			Summary: "Read what a factor holds and has paid, from the factoring entries booked " +
				"to the ledger. It shows the reserve it still holds, what is owed it under " +
				"secured borrowing, the advances it paid, the reserves it released, what was " +
				"paid back for chargebacks net of reversals, the fees booked, and the day of " +
				"the last movement. Use it before factoring_release_reserve to see what the " +
				"factor should remit. " + factoringMinorAmount,
			SearchTerms: []string{"factoring reserve", "reserve balance", "due from factor",
				"factor fees"},
			Resource:  permission.ResourceFactoringReserve,
			ParamName: paramFactoringCompany,
			References: map[string]readspec.Reference{paramFactoringCompany: {
				Resource: permission.ResourceFactoringCompany,
				What:     factoringCompanyWhat,
				Source:   "factoring_list_submissions or factoring_list_agreements",
			}},
		},
	}
}

// readinessChecks keeps a NotReady invoice's failed checks, which say what
// holds it back, and drops every other invoice's.
func readinessChecks(row map[string]any, _ map[string]struct{}, _ filtercatalog.Clock) {
	if querytool.StringOf(
		row[paramFactoringVerdict],
	) != string(
		factoringreadiness.VerdictNotReady,
	) {
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
func noaEffective(row map[string]any, _ map[string]struct{}, clk filtercatalog.Clock) {
	expires, ends := typeutils.IntOf(row[fieldExpiresAt])
	row["effective"] = querytool.StringOf(row[querytool.AccountingStatusField]) ==
		string(factoring.NoaStatusAcknowledged) &&
		(!ends || clk.Instant() < int64(expires))
}

// agreementInForce marks the agreement that covers invoices now, as the
// domain decides it, and writes the daily submission time as a clock.
func agreementInForce(row map[string]any, _ map[string]struct{}, clk filtercatalog.Clock) {
	now := clk.Instant()
	from, started := typeutils.IntOf(row[querytool.FieldEffectiveFrom])
	to, ends := typeutils.IntOf(row[querytool.FieldEffectiveTo])
	row["inForce"] = querytool.StringOf(row[querytool.AccountingStatusField]) ==
		string(factoring.AgreementStatusActive) &&
		started && now >= int64(from) && (!ends || now < int64(to))
	if minutes, ok := typeutils.IntOf(row[fieldSubmissionTime]); ok {
		row[fieldSubmissionTime] = fmt.Sprintf("%02d:%02d", minutes/querytool.MinutesPerHour,
			minutes%querytool.MinutesPerHour)
	}
}

// pathTo names the page each row of a list opens on.
func pathTo(prefix string) func(map[string]any, map[string]struct{}, filtercatalog.Clock) {
	return func(row map[string]any, _ map[string]struct{}, _ filtercatalog.Clock) {
		if id := querytool.StringOf(row[querytool.RecordIDField]); id != "" {
			row[keyFactoringPath] = prefix + id
		}
	}
}

// remittanceLines keeps an unreadable line's cells, which say what the
// factor printed, and drops every other line's, and names the remittance's
// page.
func remittanceLines(record map[string]any, withheld map[string]struct{}, clk filtercatalog.Clock) {
	pathTo(factoringRemitsPath)(record, withheld, clk)
	lines, _ := record["lines"].([]any)
	for _, line := range lines {
		object, _ := line.(map[string]any)
		if querytool.StringOf(
			object[fieldMatchStatus],
		) != string(
			factoringremittance.RemittanceLineError,
		) {
			delete(object, fieldRemittanceCells)
		}
	}
}
