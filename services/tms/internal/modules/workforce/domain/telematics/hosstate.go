package telematics

import (
	"context"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type WorkerHOSState struct {
	keel.Table `keel:"worker_hos_states"`

	OrganizationID          pulid.ID   `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID          pulid.ID   `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	WorkerID                pulid.ID   `json:"workerId" keel:"worker_id,pk"`
	Provider                string     `json:"provider" keel:"provider"`
	ProviderDriverID        string     `json:"providerDriverId" keel:"provider_driver_id"`
	DutyStatus              DutyStatus `json:"dutyStatus" keel:"duty_status,nullzero"`
	DriveRemainingMs        int64      `json:"driveRemainingMs" keel:"drive_remaining_ms"`
	ShiftRemainingMs        int64      `json:"shiftRemainingMs" keel:"shift_remaining_ms"`
	CycleRemainingMs        int64      `json:"cycleRemainingMs" keel:"cycle_remaining_ms"`
	CycleTomorrowMs         int64      `json:"cycleTomorrowMs" keel:"cycle_tomorrow_ms"`
	BreakRemainingMs        int64      `json:"breakRemainingMs" keel:"break_remaining_ms"`
	CycleStartedAt          *int64     `json:"cycleStartedAt" keel:"cycle_started_at"`
	ShiftDrivingViolationMs int64      `json:"shiftDrivingViolationMs" keel:"shift_driving_violation_ms"`
	CycleViolationMs        int64      `json:"cycleViolationMs" keel:"cycle_violation_ms"`
	CurrentVehicleID        string     `json:"currentVehicleId" keel:"current_vehicle_id,nullzero"`
	RulesetCycle            string     `json:"rulesetCycle" keel:"ruleset_cycle,nullzero"`
	RulesetShift            string     `json:"rulesetShift" keel:"ruleset_shift,nullzero"`
	RulesetRestart          string     `json:"rulesetRestart" keel:"ruleset_restart,nullzero"`
	RulesetBreak            string     `json:"rulesetBreak" keel:"ruleset_break,nullzero"`
	RulesetJurisdiction     string     `json:"rulesetJurisdiction" keel:"ruleset_jurisdiction,nullzero"`
	CurrentTractorID        pulid.ID   `json:"currentTractorId" keel:"current_tractor_id,nullzero"`
	RecordedAt              int64      `json:"recordedAt" keel:"recorded_at"`
	ReceivedAt              int64      `json:"receivedAt" keel:"received_at"`

	Worker *worker.Worker `json:"worker,omitempty" keel:"belongs:worker_id"`
}

func (s *WorkerHOSState) BeforeCreate(_ context.Context) error {
	if s.Provider == "" {
		s.Provider = "Samsara"
	}

	return nil
}

func (s *WorkerHOSState) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&s.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&s.Provider,
			validation.Required.Error("Provider is required"),
			validation.Length(1, maxProviderLength).
				Error("Provider cannot be longer than 32 characters"),
		),
		validation.Field(&s.ProviderDriverID,
			validation.Required.Error("Provider driver id is required"),
		),
		validation.Field(&s.DutyStatus,
			domainvalidation.ValidEnum[DutyStatus]("Duty status is invalid"),
		),
		// Dispatch decides whether a driver can legally take a load from these
		// clocks, so a negative remainder would offer hours that do not exist.
		validation.Field(&s.DriveRemainingMs,
			validation.Min(int64(0)).Error("Drive time remaining cannot be negative"),
		),
		validation.Field(&s.ShiftRemainingMs,
			validation.Min(int64(0)).Error("Shift time remaining cannot be negative"),
		),
		validation.Field(&s.CycleRemainingMs,
			validation.Min(int64(0)).Error("Cycle time remaining cannot be negative"),
		),
		validation.Field(&s.CycleTomorrowMs,
			validation.Min(int64(0)).Error("Cycle time tomorrow cannot be negative"),
		),
		validation.Field(&s.BreakRemainingMs,
			validation.Min(int64(0)).Error("Break time remaining cannot be negative"),
		),
		validation.Field(&s.ShiftDrivingViolationMs,
			validation.Min(int64(0)).Error("Shift driving violation cannot be negative"),
		),
		validation.Field(&s.CycleViolationMs,
			validation.Min(int64(0)).Error("Cycle violation cannot be negative"),
		),
		validation.Field(&s.RecordedAt,
			validation.Required.Error("Recorded at is required"),
			validation.Min(int64(1)).Error("Recorded at must be a valid timestamp"),
		),
		validation.Field(&s.ReceivedAt,
			validation.Required.Error("Received at is required"),
			validation.Min(int64(1)).Error("Received at must be a valid timestamp"),
		),
	))
}
