package dispatchfinding

type Severity string

const (
	SeverityBlock = Severity("Block")
	SeverityWarn  = Severity("Warn")
	SeverityInfo  = Severity("Info")
)

const (
	CodeDriverUnderage        = "driver.underage"
	CodeLicenseExpired        = "driver.license_expired"
	CodeMedicalCardExpired    = "driver.medical_card_expired"
	CodePhysicalOverdue       = "driver.physical_overdue"
	CodePreEmploymentDrugTest = "driver.pre_employment_drug_test"
	CodeDrugAlcoholProhibited = "driver.drug_alcohol_prohibited"
	CodeDrugAlcoholPending    = "driver.drug_alcohol_pending"
	CodeDrugAlcoholUnknown    = "driver.drug_alcohol_unknown"
	CodeClearinghouseOverdue  = "driver.clearinghouse_query_overdue"
	CodeMVROverdue            = "driver.mvr_overdue"
	//nolint:gosec // G101: an eligibility finding code; "passed" is not a password
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
	CodeWindowPassed       = "move.window_passed"
	CodeDeadheadExcessive  = "move.deadhead_excessive"
	CodeLocationUnknown    = "move.location_unknown"
	CodeDriverTypeMismatch = "move.driver_type_mismatch"
)

type Finding struct {
	Code       string   `json:"code"`
	Severity   Severity `json:"severity"`
	Field      string   `json:"field"`
	Message    string   `json:"message"`
	Regulation string   `json:"regulation,omitempty"`
}
