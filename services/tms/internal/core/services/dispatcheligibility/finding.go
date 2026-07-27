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

func (s Severity) IsValid() bool {
	switch s {
	case SeverityBlock, SeverityWarn, SeverityInfo:
		return true
	default:
		return false
	}
}

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

// Finding is a single eligibility observation about pairing a worker (and optional
// equipment) with a shipment move. Findings are the shared currency between the
// assignment write path, which maps them onto a MultiError, and the dispatcher
// console, which renders them as badges and feeds them into candidate scoring.
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

func (e *Evaluation) HasFindings() bool {
	return e != nil && len(e.Findings) > 0
}

func (e *Evaluation) Count(severity Severity) int {
	if e == nil {
		return 0
	}
	count := 0
	for i := range e.Findings {
		if e.Findings[i].Severity == severity {
			count++
		}
	}
	return count
}

// AppendToMultiError writes the findings onto an existing MultiError under the given
// prefix. Callers on the assignment write path use this so the structured findings and
// the legacy field-level validation errors never drift apart.
//
// Info findings are deliberately skipped: they carry context the dispatcher console
// renders (stale telematics, missing HOS feed) but which has never been, and must not
// become, a validation error on the assignment write path.
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

// SeverityForEnforcement maps an organization's configured enforcement level onto the
// severity carried by regulatory findings: Block makes a finding a hard stop,
// Warning and Audit leave it advisory.
func SeverityForEnforcement(level dispatchcontrol.ComplianceEnforcementLevel) Severity {
	if level.ShouldBlock() {
		return SeverityBlock
	}
	return SeverityWarn
}

// ComplianceErrorCode is the single source of truth for turning an enforcement level
// into the error code a compliance violation reports with. Both the assignment and
// worker services route through it so the two surfaces cannot drift.
func ComplianceErrorCode(level dispatchcontrol.ComplianceEnforcementLevel) errortypes.ErrorCode {
	return severityErrorCode(SeverityForEnforcement(level))
}

func enforcementSeverity(level dispatchcontrol.ComplianceEnforcementLevel) Severity {
	return SeverityForEnforcement(level)
}
