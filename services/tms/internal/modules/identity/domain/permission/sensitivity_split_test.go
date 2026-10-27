package permission

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The fields of a medical, drug and alcohol or injury record that say what
// happened to a person stay Restricted, while what scheduling and the posted
// log need is Internal. An agent at the default data access, and a person
// whose role reaches only Internal, is shown the second and never the first.
func TestMedicalRecordsKeepWhatHappenedToThePersonRestricted(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	cases := map[Resource]sensitivitySplit{
		ResourceWorkerDOTTest: {
			internal: []string{
				"testType", "isDot", "reason", "status", "scheduledAt", "collectedAt",
				"resultAt", "drawEntryId",
			},
			restricted: []string{
				"result", "substance", "alcoholConcentration", "collectorName",
				"collectionSite", "specimenId", "labName", "mroName", "mroVerifiedAt",
				"notes", "documentId",
			},
		},
		ResourceWorkerLeave: {
			internal: []string{
				"leaveType", "status", "frequency", "fmlaDesignated", "startsAt", "endsAt",
				"certificationStatus", "certificationDueAt", "usedOn", "hours",
				"countsAgainstEntitlement",
			},
			restricted: []string{
				"reason", "notes", "militaryCaregiver", "eligibilityHoursWorked", "documentId",
			},
		},
		ResourceQualification: {
			internal: []string{
				"employerName", "employerDotNumber", "employedFrom", "employedTo", "status",
				"method", "requestedAt", "followUpCount", "hadAccidents", "accidentCount",
			},
			restricted: []string{
				"contactName", "contactPhone", "contactEmail", "hadDrugAlcoholViolations",
				"findings", "notes", "documentId",
			},
		},
		ResourceWorkerCredential: {
			internal: []string{
				"workerId", "credentialTypeId", "status", "issuingAuthority", "issuedAt",
				"expiresAt", "health", "daysUntilExpiry", "documentId", "verifiedAt",
			},
			restricted: []string{"notes", "archiveReason"},
		},
		ResourceWorkerInjury: {
			internal: []string{
				"caseNumber", "caseYear", "classification", "illnessType", "status",
				"recordable", "occurredAt", "daysAway", "daysRestricted", "privacyCase",
				"logName", "claimStatus",
			},
			restricted: []string{
				"treatment", "description", "bodyPart", "harmfulAgent", "location",
				"claimNumber", "claimCarrier", "notes", "documentId",
			},
		},
	}
	assertSplit(t, registry, cases)
}

type sensitivitySplit struct {
	internal   []string
	restricted []string
}

func assertSplit(t *testing.T, registry *Registry, cases map[Resource]sensitivitySplit) {
	t.Helper()

	for resource, tc := range cases {
		for _, field := range tc.internal {
			assert.Equal(t, SensitivityInternal,
				registry.GetFieldSensitivity(resource.String(), field), "%s.%s", resource, field)
		}
		for _, field := range tc.restricted {
			assert.Equal(t, SensitivityRestricted,
				registry.GetFieldSensitivity(resource.String(), field), "%s.%s", resource, field)
		}
	}
}

// An invoice's, an adjustment's and a dispute's state is Internal, so an
// agent at the default data access can say where a receivable stands; the
// money and the words written about it stay Restricted.
func TestReceivablesKeepTheMoneyAndTheNotesRestricted(t *testing.T) {
	t.Parallel()

	assertSplit(t, NewRegistry(), map[Resource]sensitivitySplit{
		ResourceInvoice: {
			internal: []string{
				"scope", "shipmentCount", "voidedAt", "ediSendStatus", "chargeCode",
				"accessorialChargeId", "executionError", "supportingDocumentIds", "kind",
				"approvalStatus", "lines",
			},
			restricted: []string{
				"totalAmount", "appliedAmount", "openBalance", "memo", "voidReason",
				"lastEdiError", "reason", "policyReason", "creditAmount", "creditQuantity",
				"rebillQuantity",
			},
		},
		ResourceInvoiceDispute: {
			internal: []string{
				"invoiceId", "customerId", "status", "reasonCode", "openedAt", "resolvedAt",
				"resolution", "resolutionAdjustmentId",
			},
			restricted: []string{
				"disputedAmount", "disputedAmountMinor", "notes", "resolutionNotes",
			},
		},
		ResourceCustomerPayment: {
			internal: []string{"applications", "invoiceId", "paymentMethod", "referenceNumber"},
			restricted: []string{
				"amountMinor", "appliedAmountMinor", "unappliedAmountMinor", "shortPayAmountMinor",
				"memo",
			},
		},
	})
}

// What a factor funded, holds and charges stays Restricted, as an invoice's
// amounts are, while where each factoring record stands is Internal, so an
// agent at the default data access works the factoring queue without being
// shown the money.
func TestFactoringRecordsKeepTheMoneyRestricted(t *testing.T) {
	t.Parallel()

	assertSplit(t, NewRegistry(), map[Resource]sensitivitySplit{
		ResourceFactoringAgreement: {
			internal: []string{
				"status", "effectiveFrom", "effectiveTo", "recourse", "accountingTreatment",
				"fundingDays", "submissionTime", "chargebackWindowDays", "daysFrom", "daysTo",
			},
			restricted: []string{
				"advanceRateBp", "reserveRateBp", "minimumFeeMinor", "rateBp", "flatMinor",
				"notes",
			},
		},
		ResourceFactoringNoa: {
			internal: []string{
				"status", "sentAt", "sentTo", "acknowledgedAt", "expiresAt", "releasedAt",
				"releaseReason",
			},
			restricted: []string{"notes"},
		},
		ResourceFactoringSubmission: {
			internal: []string{
				"number", "status", "invoiceCount", "transportTarget", "lastError", "sentAt",
				"invoiceNumber", "fundedOn", "rejectionReason", "chargebackReason",
			},
			restricted: []string{
				"faceMinor", "expectedAdvanceMinor", "expectedFeeMinor", "expectedReserveMinor",
				"advanceMinor", "feeMinor", "reserveMinor",
			},
		},
		ResourceFactoringRemittance: {
			internal: []string{
				"status", "fileName", "lineCount", "matchedCount", "exceptionCount",
				"matchStatus", "invoiceNumber", "debtorName", "score", "suggestions",
			},
			restricted: []string{
				"totalCashMinor", "faceMinor", "advanceMinor", "feeMinor", "reserveMinor",
				"cashMinor", "cells",
			},
		},
		ResourceFactoringReserve: {
			internal: []string{"factoringCompanyId", "entryCount", "lastOccurredOn"},
			restricted: []string{
				"reserveMinor", "liabilityMinor", "advancedMinor", "releasedMinor",
				"chargedBackMinor", "feeMinor",
			},
		},
		ResourceInvoice: {
			internal: []string{
				"verdict", "failedChecks", "readySince", "evaluatedAt", "checks", "message",
				"fix",
			},
			restricted: []string{"openBalanceMinor", "totalAmountMinor"},
		},
	})
}
