package telematicsmapping

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/telematicsservice"
)

func MapWorkerHOSState(state *telematics.WorkerHOSState) *gqlmodel.WorkerHosState {
	out := &gqlmodel.WorkerHosState{
		WorkerID:                state.WorkerID.String(),
		Provider:                state.Provider,
		ProviderDriverID:        state.ProviderDriverID,
		DriveRemainingMs:        int(state.DriveRemainingMs),
		ShiftRemainingMs:        int(state.ShiftRemainingMs),
		CycleRemainingMs:        int(state.CycleRemainingMs),
		CycleTomorrowMs:         int(state.CycleTomorrowMs),
		BreakRemainingMs:        int(state.BreakRemainingMs),
		ShiftDrivingViolationMs: int(state.ShiftDrivingViolationMs),
		CycleViolationMs:        int(state.CycleViolationMs),
		RecordedAt:              int(state.RecordedAt),
	}
	limits := telematicsservice.LimitsForRuleset(
		state.RulesetCycle,
		state.RulesetShift,
		state.RulesetJurisdiction,
	)
	out.DriveLimitMs = int(limits.DriveMs)
	out.ShiftLimitMs = int(limits.ShiftMs)
	out.CycleLimitMs = int(limits.CycleMs)
	out.BreakLimitMs = int(limits.BreakMs)
	if state.RulesetCycle != "" {
		rulesetCycle := state.RulesetCycle
		out.RulesetCycle = &rulesetCycle
	}
	if state.RulesetShift != "" {
		rulesetShift := state.RulesetShift
		out.RulesetShift = &rulesetShift
	}
	if state.RulesetJurisdiction != "" {
		rulesetJurisdiction := state.RulesetJurisdiction
		out.RulesetJurisdiction = &rulesetJurisdiction
	}
	if state.DutyStatus != "" {
		dutyStatus := string(state.DutyStatus)
		out.DutyStatus = &dutyStatus
	}
	if state.CycleStartedAt != nil {
		startedAt := int(*state.CycleStartedAt)
		out.CycleStartedAt = &startedAt
	}
	if state.CurrentVehicleID != "" {
		vehicleID := state.CurrentVehicleID
		out.CurrentVehicleID = &vehicleID
	}
	if !state.CurrentTractorID.IsNil() {
		tractorID := state.CurrentTractorID.String()
		out.CurrentTractorID = &tractorID
	}
	if state.Worker != nil {
		out.WorkerName = WorkerDisplayName(state.Worker)
	}
	return out
}

func MapWorkerHOSViolation(violation *telematics.WorkerHOSViolation) *gqlmodel.WorkerHosViolation {
	out := &gqlmodel.WorkerHosViolation{
		WorkerID:         violation.WorkerID.String(),
		ViolationType:    violation.ViolationType,
		DurationMs:       int(violation.DurationMs),
		ViolationStartAt: int(violation.ViolationStartAt),
		DetectedAt:       int(violation.DetectedAt),
	}
	if violation.Description != "" {
		description := violation.Description
		out.Description = &description
	}
	if violation.DayStartAt != nil {
		dayStart := int(*violation.DayStartAt)
		out.DayStartAt = &dayStart
	}
	if violation.DayEndAt != nil {
		dayEnd := int(*violation.DayEndAt)
		out.DayEndAt = &dayEnd
	}
	return out
}

func MapWorkerHOSDailyLog(day *telematicsservice.WorkerHOSDailyLog) *gqlmodel.WorkerHosDailyLog {
	out := &gqlmodel.WorkerHosDailyLog{
		StartAt:                      int(day.StartAt),
		EndAt:                        int(day.EndAt),
		DriveDistanceMeters:          int(day.DriveDistanceMeters),
		ActiveDurationMs:             int(day.ActiveDurationMs),
		DriveDurationMs:              int(day.DriveDurationMs),
		OnDutyDurationMs:             int(day.OnDutyDurationMs),
		OffDutyDurationMs:            int(day.OffDutyDurationMs),
		SleeperBerthDurationMs:       int(day.SleeperBerthDurationMs),
		PersonalConveyanceDurationMs: int(day.PersonalConveyanceDurationMs),
		YardMoveDurationMs:           int(day.YardMoveDurationMs),
		IsCertified:                  day.IsCertified,
		VehicleNames:                 day.VehicleNames,
	}
	if day.CertifiedAt != nil {
		certifiedAt := int(*day.CertifiedAt)
		out.CertifiedAt = &certifiedAt
	}
	if day.ShippingDocs != "" {
		shippingDocs := day.ShippingDocs
		out.ShippingDocs = &shippingDocs
	}
	return out
}

func WorkerDisplayName(wrk *worker.Worker) string {
	return strings.TrimSpace(wrk.FullName())
}
