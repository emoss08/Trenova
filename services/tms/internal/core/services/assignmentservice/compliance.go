package assignmentservice

import (
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	minDriverAgeInterstate  = 21
	hazmatCertValidityYears = 5
	mvrCheckValidityMonths  = 12
)

func runWorkerComplianceChecks(
	w *worker.Worker,
	dc *dispatchcontrol.DispatchControl,
	hasHazmatCommodities bool,
	prefix string,
	multiErr *errortypes.MultiError,
) {
	checkDriverQualification(w, dc, prefix, multiErr)
	checkMedicalCert(w, dc, prefix, multiErr)
	checkDrugAndAlcohol(w, dc, prefix, multiErr)
	checkMVR(w, dc, prefix, multiErr)
	checkHazmatEndorsement(w, dc, hasHazmatCommodities, prefix, multiErr)
}

func checkDriverQualification(
	w *worker.Worker,
	dc *dispatchcontrol.DispatchControl,
	prefix string,
	multiErr *errortypes.MultiError,
) {
	if !dc.EnforceDriverQualificationCompliance {
		return
	}

	if w.Profile == nil {
		return
	}

	errCode := getComplianceErrorCode(dc.ComplianceEnforcementLevel)

	if !timeutils.IsAtLeastAge(w.Profile.DOB, minDriverAgeInterstate) {
		multiErr.WithPrefix(prefix).Add(
			"dob",
			errCode,
			"Driver must be at least 21 years old for interstate commerce (49 CFR 391.11(b)(1))",
		)
	}

	if timeutils.IsExpired(w.Profile.LicenseExpiry) {
		multiErr.WithPrefix(prefix).Add(
			"licenseExpiry",
			errCode,
			"Commercial driver's license is expired (49 CFR 391.11(b)(5))",
		)
	}
}

func checkMedicalCert(
	w *worker.Worker,
	dc *dispatchcontrol.DispatchControl,
	prefix string,
	multiErr *errortypes.MultiError,
) {
	if !dc.EnforceMedicalCertCompliance {
		return
	}

	if w.Profile == nil {
		return
	}

	errCode := getComplianceErrorCode(dc.ComplianceEnforcementLevel)

	if w.Profile.MedicalCardExpiry != nil && timeutils.IsExpired(*w.Profile.MedicalCardExpiry) {
		multiErr.WithPrefix(prefix).Add(
			"medicalCardExpiry",
			errCode,
			"Medical certificate is expired (49 CFR 391.45)",
		)
	}

	if w.Profile.PhysicalDueDate != nil && timeutils.IsOverdue(*w.Profile.PhysicalDueDate) {
		multiErr.WithPrefix(prefix).Add(
			"physicalDueDate",
			errCode,
			"Physical examination is overdue (49 CFR 391.45)",
		)
	}
}

func checkDrugAndAlcohol(
	w *worker.Worker,
	dc *dispatchcontrol.DispatchControl,
	prefix string,
	multiErr *errortypes.MultiError,
) {
	if !dc.EnforceDrugAndAlcoholCompliance {
		return
	}

	if w.Profile == nil {
		return
	}

	if w.Profile.LastDrugTest > 0 && w.Profile.LastDrugTest <= w.Profile.HireDate {
		errCode := getComplianceErrorCode(dc.ComplianceEnforcementLevel)
		multiErr.WithPrefix(prefix).Add(
			"lastDrugTest",
			errCode,
			"Pre-employment drug test is required before hire date (49 CFR 382.301(a))",
		)
	}
}

func checkMVR(
	w *worker.Worker,
	dc *dispatchcontrol.DispatchControl,
	prefix string,
	multiErr *errortypes.MultiError,
) {
	if !dc.EnforceHOSCompliance {
		return
	}

	if w.Profile == nil {
		return
	}

	errCode := getComplianceErrorCode(dc.ComplianceEnforcementLevel)

	if w.Profile.LastMVRCheck > 0 &&
		!timeutils.IsWithinMonths(w.Profile.LastMVRCheck, mvrCheckValidityMonths) {
		multiErr.WithPrefix(prefix).Add(
			"lastMvrCheck",
			errCode,
			"Annual MVR check is overdue (49 CFR 391.25(c)(2))",
		)
	}

	if w.Profile.MVRDueDate != nil && timeutils.IsOverdue(*w.Profile.MVRDueDate) {
		multiErr.WithPrefix(prefix).Add(
			"mvrDueDate",
			errCode,
			"MVR due date has passed (49 CFR 391.25(c)(2))",
		)
	}
}

func checkHazmatEndorsement(
	w *worker.Worker,
	dc *dispatchcontrol.DispatchControl,
	hasHazmatCommodities bool,
	prefix string,
	multiErr *errortypes.MultiError,
) {
	if !dc.EnforceHazmatCompliance || !hasHazmatCommodities {
		return
	}

	if w.Profile == nil {
		return
	}

	errCode := getComplianceErrorCode(dc.ComplianceEnforcementLevel)

	if !w.Profile.Endorsement.RequiresHazmatExpiry() {
		multiErr.WithPrefix(prefix).Add(
			"endorsement",
			errCode,
			"Shipment contains hazardous materials — worker requires an H or X endorsement (49 CFR 383.93)",
		)
		return
	}

	if w.Profile.HazmatExpiry == nil || *w.Profile.HazmatExpiry <= 0 {
		multiErr.WithPrefix(prefix).Add(
			"hazmatExpiry",
			errCode,
			"Hazmat expiry date is required for H or X endorsement (49 CFR 383.93)",
		)
		return
	}

	if timeutils.IsExpired(*w.Profile.HazmatExpiry) {
		multiErr.WithPrefix(prefix).Add(
			"hazmatExpiry",
			errCode,
			"Hazmat endorsement is expired (49 CFR 383.93)",
		)
	}

	maxAllowed := timeutils.MaxAllowedUnix(timeutils.NowUnix(), hazmatCertValidityYears)
	if *w.Profile.HazmatExpiry > maxAllowed {
		multiErr.WithPrefix(prefix).Add(
			"hazmatExpiry",
			errCode,
			"Hazmat endorsement exceeds maximum validity period of 5 years (49 CFR 383.93)",
		)
	}
}

func getComplianceErrorCode(level dispatchcontrol.ComplianceEnforcementLevel) errortypes.ErrorCode {
	if level.ShouldBlock() {
		return errortypes.ErrComplianceViolation
	}
	return errortypes.ErrInvalid
}
