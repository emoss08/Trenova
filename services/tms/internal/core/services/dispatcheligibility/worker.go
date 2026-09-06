package dispatcheligibility

import (
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	minDriverAgeInterstate  = 21
	hazmatCertValidityYears = 5
	mvrCheckValidityMonths  = 12
)

const (
	regDriverQualification = "49 CFR 391.11(b)(1)"
	regLicense             = "49 CFR 391.11(b)(5)"
	regMedicalCert         = "49 CFR 391.45"
	regDrugAndAlcohol      = "49 CFR 382.301(a)"
	//nolint:gosec // G101: a regulation citation, not a credential
	regDrugAndAlcoholProhibition = "49 CFR 382.501"
	regClearinghouse             = "49 CFR 382.701(b)"
	regMVR                       = "49 CFR 391.25(c)(2)"
	regHazmatEndorsement         = "49 CFR 383.93"
)

type WorkerComplianceInput struct {
	Worker               *worker.Worker
	Control              *dispatchcontrol.DispatchControl
	HasHazmatCommodities bool
}

// EvaluateWorkerCompliance runs the FMCSA driver-qualification rule set against a
// worker. The check order is significant: the assignment write path renders these
// findings into a MultiError and its existing tests assert that ordering.
func EvaluateWorkerCompliance(in WorkerComplianceInput) *Evaluation {
	eval := NewEvaluation(4)

	if in.Worker == nil || in.Worker.Profile == nil || in.Control == nil {
		return eval
	}

	severity := enforcementSeverity(in.Control.ComplianceEnforcementLevel)

	evaluateDriverQualification(in, severity, eval)
	evaluateMedicalCert(in, severity, eval)
	evaluateDrugAndAlcohol(in, severity, eval)
	evaluateMVR(in, severity, eval)
	evaluateHazmatEndorsement(in, severity, eval)

	return eval
}

func evaluateDriverQualification(
	in WorkerComplianceInput,
	severity Severity,
	eval *Evaluation,
) {
	if !in.Control.EnforceDriverQualificationCompliance {
		return
	}

	profile := in.Worker.Profile

	if !timeutils.IsAtLeastAge(profile.DOB, minDriverAgeInterstate) {
		eval.Add(Finding{
			Code:       CodeDriverUnderage,
			Severity:   severity,
			Field:      "dob",
			Message:    "Driver must be at least 21 years old for interstate commerce (49 CFR 391.11(b)(1))",
			Regulation: regDriverQualification,
		})
	}

	if timeutils.IsExpired(profile.LicenseExpiry) {
		eval.Add(Finding{
			Code:       CodeLicenseExpired,
			Severity:   severity,
			Field:      "licenseExpiry",
			Message:    "Commercial driver's license is expired (49 CFR 391.11(b)(5))",
			Regulation: regLicense,
		})
	}
}

func evaluateMedicalCert(in WorkerComplianceInput, severity Severity, eval *Evaluation) {
	if !in.Control.EnforceMedicalCertCompliance {
		return
	}

	profile := in.Worker.Profile

	if profile.MedicalCardExpiry != nil && timeutils.IsExpired(*profile.MedicalCardExpiry) {
		eval.Add(Finding{
			Code:       CodeMedicalCardExpired,
			Severity:   severity,
			Field:      "medicalCardExpiry",
			Message:    "Medical certificate is expired (49 CFR 391.45)",
			Regulation: regMedicalCert,
		})
	}

	if profile.PhysicalDueDate != nil && timeutils.IsOverdue(*profile.PhysicalDueDate) {
		eval.Add(Finding{
			Code:       CodePhysicalOverdue,
			Severity:   severity,
			Field:      "physicalDueDate",
			Message:    "Physical examination is overdue (49 CFR 391.45)",
			Regulation: regMedicalCert,
		})
	}
}

// evaluateDrugAndAlcohol reads the standing the testing programme keeps on the
// profile rather than re-deriving it here: the record lives in
// worker_dot_tests, worker_dot_violations and worker_clearinghouse_queries, and
// the profile column is the cache those three write.
//
// Only a standing prohibition blocks. A driver with nothing on file, or one
// waiting on a laboratory, is a gap in the paperwork rather than a finding
// against the driver, and grounding a fleet over paperwork is not what
// 49 CFR 382 asks for — so those warn at every enforcement level.
func evaluateDrugAndAlcohol(in WorkerComplianceInput, severity Severity, eval *Evaluation) {
	if !in.Control.EnforceDrugAndAlcoholCompliance {
		return
	}

	profile := in.Worker.Profile

	switch profile.DrugAlcoholStatus.Normalized() {
	case worker.DrugAlcoholProhibited:
		eval.Add(Finding{
			Code:       CodeDrugAlcoholProhibited,
			Severity:   SeverityBlock,
			Field:      "drugAlcoholStatus",
			Message:    drugAlcoholProhibitedMessage(profile.ReturnToDutyStatus),
			Regulation: regDrugAndAlcoholProhibition,
		})
	case worker.DrugAlcoholPending:
		eval.Add(Finding{
			Code:       CodeDrugAlcoholPending,
			Severity:   SeverityWarn,
			Field:      "drugAlcoholStatus",
			Message:    "A drug or alcohol test is awaiting a result",
			Regulation: regDrugAndAlcohol,
		})
	case worker.DrugAlcoholUnknown:
		eval.Add(Finding{
			Code:     CodeDrugAlcoholUnknown,
			Severity: SeverityWarn,
			Field:    "drugAlcoholStatus",
			Message: "No pre-employment drug test is on file " +
				"(49 CFR 382.301(a))",
			Regulation: regDrugAndAlcohol,
		})
	case worker.DrugAlcoholClear:
	}

	if profile.NextClearinghouseQueryDue != nil &&
		timeutils.IsOverdue(*profile.NextClearinghouseQueryDue) {
		eval.Add(Finding{
			Code:       CodeClearinghouseOverdue,
			Severity:   severity,
			Field:      "nextClearinghouseQueryDue",
			Message:    "The annual Clearinghouse query is overdue (49 CFR 382.701(b))",
			Regulation: regClearinghouse,
		})
	}
}

// drugAlcoholProhibitedMessage names the step that would clear the prohibition,
// because "prohibited" on its own tells a dispatcher nothing they can act on.
func drugAlcoholProhibitedMessage(rtd worker.ReturnToDutyStatus) string {
	switch rtd {
	case worker.ReturnToDutySAPEvaluation:
		return "Prohibited from safety-sensitive duty: awaiting the SAP evaluation " +
			"(49 CFR 382.501)"
	case worker.ReturnToDutyRTDTestRequired:
		return "Prohibited from safety-sensitive duty: the return-to-duty test has not " +
			"been passed (49 CFR 382.309)"
	default:
		return "Prohibited from safety-sensitive duty by the drug and alcohol record " +
			"(49 CFR 382.501)"
	}
}

func evaluateMVR(in WorkerComplianceInput, severity Severity, eval *Evaluation) {
	if !in.Control.EnforceDriverQualificationCompliance {
		return
	}

	profile := in.Worker.Profile

	if profile.LastMVRCheck > 0 &&
		!timeutils.IsWithinMonths(profile.LastMVRCheck, mvrCheckValidityMonths) {
		eval.Add(Finding{
			Code:       CodeMVROverdue,
			Severity:   severity,
			Field:      "lastMvrCheck",
			Message:    "Annual MVR check is overdue (49 CFR 391.25(c)(2))",
			Regulation: regMVR,
		})
	}

	if profile.MVRDueDate != nil && timeutils.IsOverdue(*profile.MVRDueDate) {
		eval.Add(Finding{
			Code:       CodeMVRDueDatePassed,
			Severity:   severity,
			Field:      "mvrDueDate",
			Message:    "MVR due date has passed (49 CFR 391.25(c)(2))",
			Regulation: regMVR,
		})
	}
}

func evaluateHazmatEndorsement(in WorkerComplianceInput, severity Severity, eval *Evaluation) {
	if !in.Control.EnforceHazmatCompliance || !in.HasHazmatCommodities {
		return
	}

	profile := in.Worker.Profile

	if !profile.Endorsement.RequiresHazmatExpiry() {
		eval.Add(Finding{
			Code:     CodeHazmatEndorsementMissing,
			Severity: severity,
			Field:    "endorsement",
			Message: "Shipment contains hazardous materials — worker requires an H or X " +
				"endorsement (49 CFR 383.93)",
			Regulation: regHazmatEndorsement,
		})
		return
	}

	if profile.HazmatExpiry == nil || *profile.HazmatExpiry <= 0 {
		eval.Add(Finding{
			Code:       CodeHazmatExpiryMissing,
			Severity:   severity,
			Field:      "hazmatExpiry",
			Message:    "Hazmat expiry date is required for H or X endorsement (49 CFR 383.93)",
			Regulation: regHazmatEndorsement,
		})
		return
	}

	if timeutils.IsExpired(*profile.HazmatExpiry) {
		eval.Add(Finding{
			Code:       CodeHazmatExpired,
			Severity:   severity,
			Field:      "hazmatExpiry",
			Message:    "Hazmat endorsement is expired (49 CFR 383.93)",
			Regulation: regHazmatEndorsement,
		})
	}

	maxAllowed := timeutils.MaxAllowedUnix(timeutils.NowUnix(), hazmatCertValidityYears)
	if *profile.HazmatExpiry > maxAllowed {
		eval.Add(Finding{
			Code:       CodeHazmatExceedsValidity,
			Severity:   severity,
			Field:      "hazmatExpiry",
			Message:    "Hazmat endorsement exceeds maximum validity period of 5 years (49 CFR 383.93)",
			Regulation: regHazmatEndorsement,
		})
	}
}
