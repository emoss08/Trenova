package telematics

import (
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type WorkerHOSState struct {
	bun.BaseModel `bun:"table:worker_hos_states,alias:whs" json:"-"`

	OrganizationID          pulid.ID   `json:"organizationId"          bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID          pulid.ID   `json:"businessUnitId"          bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	WorkerID                pulid.ID   `json:"workerId"                bun:"worker_id,pk,type:VARCHAR(100),notnull"`
	Provider                string     `json:"provider"                bun:"provider,type:VARCHAR(32),notnull,default:'Samsara'"`
	ProviderDriverID        string     `json:"providerDriverId"        bun:"provider_driver_id,type:TEXT,notnull"`
	DutyStatus              DutyStatus `json:"dutyStatus"              bun:"duty_status,type:VARCHAR(32),nullzero"`
	DriveRemainingMs        int64      `json:"driveRemainingMs"        bun:"drive_remaining_ms,type:BIGINT,notnull,default:0"`
	ShiftRemainingMs        int64      `json:"shiftRemainingMs"        bun:"shift_remaining_ms,type:BIGINT,notnull,default:0"`
	CycleRemainingMs        int64      `json:"cycleRemainingMs"        bun:"cycle_remaining_ms,type:BIGINT,notnull,default:0"`
	CycleTomorrowMs         int64      `json:"cycleTomorrowMs"         bun:"cycle_tomorrow_ms,type:BIGINT,notnull,default:0"`
	BreakRemainingMs        int64      `json:"breakRemainingMs"        bun:"break_remaining_ms,type:BIGINT,notnull,default:0"`
	CycleStartedAt          *int64     `json:"cycleStartedAt"          bun:"cycle_started_at,type:BIGINT,nullzero"`
	ShiftDrivingViolationMs int64      `json:"shiftDrivingViolationMs" bun:"shift_driving_violation_ms,type:BIGINT,notnull,default:0"`
	CycleViolationMs        int64      `json:"cycleViolationMs"        bun:"cycle_violation_ms,type:BIGINT,notnull,default:0"`
	CurrentVehicleID        string     `json:"currentVehicleId"        bun:"current_vehicle_id,type:TEXT,nullzero"`
	RulesetCycle            string     `json:"rulesetCycle"            bun:"ruleset_cycle,type:VARCHAR(64),nullzero"`
	RulesetShift            string     `json:"rulesetShift"            bun:"ruleset_shift,type:VARCHAR(64),nullzero"`
	RulesetRestart          string     `json:"rulesetRestart"          bun:"ruleset_restart,type:VARCHAR(64),nullzero"`
	RulesetBreak            string     `json:"rulesetBreak"            bun:"ruleset_break,type:VARCHAR(64),nullzero"`
	RulesetJurisdiction     string     `json:"rulesetJurisdiction"     bun:"ruleset_jurisdiction,type:VARCHAR(16),nullzero"`
	CurrentTractorID        pulid.ID   `json:"currentTractorId"        bun:"current_tractor_id,type:VARCHAR(100),nullzero"`
	RecordedAt              int64      `json:"recordedAt"              bun:"recorded_at,type:BIGINT,notnull"`
	ReceivedAt              int64      `json:"receivedAt"              bun:"received_at,type:BIGINT,notnull"`

	Worker *worker.Worker `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}
