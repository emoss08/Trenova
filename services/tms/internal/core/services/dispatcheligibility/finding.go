package dispatcheligibility

import (
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type Severity string

const (
	SeverityBlock = Severity("Block")
	SeverityWarn  = Severity("Warn")
	SeverityInfo  = Severity("Info")
)

const (
	CodeDriverUnderage           = "driver.underage"
	CodeLicenseExpired           = "driver.license_expired"
	CodeMedicalCardExpired       = "driver.medical_card_expired"
	CodePhysicalOverdue          = "driver.physical_overdue"
	CodePreEmploymentDrugTest    = "driver.pre_employment_drug_test"
	CodeMVROverdue               = "driver.mvr_overdue"
	CodeMVRDueDatePassed         = "driver.mvr_due_date_passed"
	CodeHazmatEndorsementMissing = "driver.hazmat_endorsement_missing"
	CodeHazmatExpiryMissing      = "driver.hazmat_expiry_missing"
	CodeHazmatExpired            = "driver.hazmat_expired"
	CodeHazmatExceedsValidity    = "driver.hazmat_exceeds_validity"

	CodeHOSShiftDrivingViolation = "hos.shift_driving_violation"
	CodeHOSCycleViolation        = "hos.cycle_violation"
	CodeHOSNoDriveTime           = "hos.no_drive_time"
	CodeHOSNoShiftTime           = "hos.no_shift_time"
	CodeHOSNoCycleTime           = "hos.no_cycle_time"
	CodeHOSBreakRequired         = "hos.break_required"
	CodeHOSStale                 = "hos.stale"
	CodeHOSMissing               = "hos.missing"
	CodeHOSInsufficientForTrip   = "hos.insufficient_for_trip"
	CodeHOSTightForTrip          = "hos.tight_for_trip"

	CodeWorkerInactive        = "availability.worker_inactive"
	CodeWorkerNotAssignable   = "availability.worker_not_assignable"
	CodeWorkerNotForDispatch  = "availability.worker_unavailable_for_dispatch"
	CodeWorkerBlocked         = "availability.worker_blocked"
	CodeWorkerOnPTO           = "availability.worker_on_pto"
	CodeWorkerOverlappingMove = "availability.worker_overlapping_move"
	CodeWorkerNoTractor       = "availability.worker_no_tractor"

	CodeTractorUnavailable   = "equipment.tractor_unavailable"
	CodeTrailerUnavailable   = "equipment.trailer_unavailable"
	CodeTractorTypeMismatch  = "equipment.tractor_type_mismatch"
	CodeTrailerTypeMismatch  = "equipment.trailer_type_mismatch"
	CodeTractorInUse         = "equipment.tractor_in_use"
	CodeTrailerInUse         = "equipment.trailer_in_use"
	CodeFleetMismatch        = "continuity.fleet_mismatch"
	CodeTrailerDiscontinuity = "continuity.trailer_discontinuity"

	CodeMoveNotAssignable  = "move.not_assignable"
	CodeShipmentOnHold     = "move.shipment_on_hold"
	CodeAppointmentAtRisk  = "move.appointment_at_risk"
	CodeAppointmentMissed  = "move.appointment_missed"
	CodeDeadheadExcessive  = "move.deadhead_excessive"
	CodeDriverTypeMismatch = "move.driver_type_mismatch"
)

type Finding struct {
	Code       string   `json:"code"`
	Severity   Severity `json:"severity"`
	Field      string   `json:"field"`
	Message    string   `json:"message"`
	Regulation string   `json:"regulation,omitempty"`
}

type Evaluation struct {
	Findings []Finding `json:"findings"`
}

func NewEvaluation(capacity int) *Evaluation {
	return &Evaluation{Findings: make([]Finding, 0, capacity)}
}

func (e *Evaluation) Add(f Finding) {
	e.Findings = append(e.Findings, f)
}

func (e *Evaluation) Merge(other *Evaluation) {
	if other == nil || len(other.Findings) == 0 {
		return
	}
	e.Findings = append(e.Findings, other.Findings...)
}

func (e *Evaluation) Blocked() bool {
	if e == nil {
		return false
	}
	for i := range e.Findings {
		if e.Findings[i].Severity == SeverityBlock {
			return true
		}
	}
	return false
}

func (e *Evaluation) AppendToMultiError(multiErr *errortypes.MultiError, prefix string) {
	if e == nil || multiErr == nil || len(e.Findings) == 0 {
		return
	}

	target := multiErr
	if prefix != "" {
		target = multiErr.WithPrefix(prefix)
	}

	for i := range e.Findings {
		if e.Findings[i].Severity == SeverityInfo {
			continue
		}
		target.Add(
			e.Findings[i].Field,
			severityErrorCode(e.Findings[i].Severity),
			e.Findings[i].Message,
		)
	}
}

func (e *Evaluation) ToMultiError(prefix string) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	e.AppendToMultiError(multiErr, prefix)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func severityErrorCode(severity Severity) errortypes.ErrorCode {
	if severity == SeverityBlock {
		return errortypes.ErrComplianceViolation
	}
	return errortypes.ErrInvalid
}

func SeverityForEnforcement(level dispatchcontrol.ComplianceEnforcementLevel) Severity {
	if level.ShouldBlock() {
		return SeverityBlock
	}
	return SeverityWarn
}

func ComplianceErrorCode(level dispatchcontrol.ComplianceEnforcementLevel) errortypes.ErrorCode {
	return severityErrorCode(SeverityForEnforcement(level))
}

func enforcementSeverity(level dispatchcontrol.ComplianceEnforcementLevel) Severity {
	return SeverityForEnforcement(level)
}
