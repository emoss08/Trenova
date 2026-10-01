package resolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/telematicsservice"
)

func mapVehiclePosition(position *telematics.VehiclePosition) *gqlmodel.VehiclePosition {
	out := &gqlmodel.VehiclePosition{
		TractorID:         position.TractorID.String(),
		Provider:          position.Provider,
		ProviderVehicleID: position.ProviderVehicleID,
		Latitude:          position.Latitude,
		Longitude:         position.Longitude,
		HeadingDegrees:    position.HeadingDegrees,
		SpeedMph:          position.SpeedMph,
		RecordedAt:        int(position.RecordedAt),
		ReceivedAt:        int(position.ReceivedAt),
	}
	if position.EngineState != "" {
		engineState := string(position.EngineState)
		out.EngineState = &engineState
	}
	if position.FuelPercent != nil {
		fuel := *position.FuelPercent
		out.FuelPercent = &fuel
	}
	if position.OdometerMeters != nil {
		odometer := int(*position.OdometerMeters)
		out.OdometerMeters = &odometer
	}
	if position.FormattedLocation != "" {
		formatted := position.FormattedLocation
		out.FormattedLocation = &formatted
	}
	if position.Tractor != nil {
		out.TractorCode = position.Tractor.Code
		if position.Tractor.PrimaryWorker != nil {
			workerID := position.Tractor.PrimaryWorkerID.String()
			out.PrimaryWorkerID = &workerID
			if name := workerDisplayName(position.Tractor.PrimaryWorker); name != "" {
				out.PrimaryWorkerName = &name
			}
		}
	}
	return out
}

func mapWorkerHOSState(state *telematics.WorkerHOSState) *gqlmodel.WorkerHosState {
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
		out.WorkerName = workerDisplayName(state.Worker)
	}
	return out
}

func mapWorkerHOSViolation(violation *telematics.WorkerHOSViolation) *gqlmodel.WorkerHosViolation {
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

func mapTelematicsStatus(status *telematicsservice.Status) *gqlmodel.TelematicsStatus {
	out := &gqlmodel.TelematicsStatus{
		Provider:          status.Provider,
		Enabled:           status.Enabled,
		Configured:        status.Configured,
		WebhookConfigured: status.WebhookConfigured,
		FailureCount:      status.FailureCount,
		MappedTractors:    status.MappedTractors,
		TotalTractors:     status.TotalTractors,
		MappedWorkers:     status.MappedWorkers,
	}
	if status.LastPolledAt > 0 {
		polledAt := int(status.LastPolledAt)
		out.LastPolledAt = &polledAt
	}
	if status.LastSuccessAt > 0 {
		successAt := int(status.LastSuccessAt)
		out.LastSuccessAt = &successAt
	}
	if status.LastError != "" {
		lastError := status.LastError
		out.LastError = &lastError
	}
	return out
}

func mapWorkerHOSLogEntry(entry *telematicsservice.WorkerHOSLogEntry) *gqlmodel.WorkerHosLogEntry {
	out := &gqlmodel.WorkerHosLogEntry{
		HosStatusType: entry.HosStatusType,
		LogStartAt:    int(entry.LogStartAt),
		Codrivers:     entry.Codrivers,
	}
	if entry.LogEndAt != nil {
		endAt := int(*entry.LogEndAt)
		out.LogEndAt = &endAt
	}
	if entry.Remark != "" {
		remark := entry.Remark
		out.Remark = &remark
	}
	if entry.VehicleID != "" {
		vehicleID := entry.VehicleID
		out.VehicleID = &vehicleID
	}
	if entry.VehicleName != "" {
		vehicleName := entry.VehicleName
		out.VehicleName = &vehicleName
	}
	if entry.Latitude != nil {
		latitude := *entry.Latitude
		out.Latitude = &latitude
	}
	if entry.Longitude != nil {
		longitude := *entry.Longitude
		out.Longitude = &longitude
	}
	return out
}

func mapWorkerHOSDailyLog(day *telematicsservice.WorkerHOSDailyLog) *gqlmodel.WorkerHosDailyLog {
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

func mapVehicleInspection(
	record *telematicsservice.VehicleInspectionRecord,
) *gqlmodel.VehicleInspection {
	out := &gqlmodel.VehicleInspection{
		ID:                    record.ID.String(),
		Provider:              record.Provider,
		InspectionType:        record.InspectionType,
		SafetyStatus:          record.SafetyStatus,
		StartedAt:             int(record.StartedAt),
		EndedAt:               int(record.EndedAt),
		Signed:                record.Signed,
		DefectCount:           record.DefectCount,
		UnresolvedDefectCount: record.UnresolvedDefectCount,
	}
	if !record.TractorID.IsNil() {
		tractorID := record.TractorID.String()
		out.TractorID = &tractorID
	}
	if !record.WorkerID.IsNil() {
		workerID := record.WorkerID.String()
		out.WorkerID = &workerID
	}
	if record.WorkerName != "" {
		workerName := record.WorkerName
		out.WorkerName = &workerName
	}
	if record.OdometerMeters != nil {
		odometer := int(*record.OdometerMeters)
		out.OdometerMeters = &odometer
	}
	if record.Location != "" {
		location := record.Location
		out.Location = &location
	}
	if len(record.Defects) > 0 {
		out.Defects = record.Defects
	}
	return out
}

func mapWorkerFormSubmission(
	submission *telematicsservice.WorkerFormSubmission,
) *gqlmodel.WorkerFormSubmission {
	out := &gqlmodel.WorkerFormSubmission{
		ID:           submission.ID,
		TemplateID:   submission.TemplateID,
		TemplateName: submission.TemplateName,
		SubmittedAt:  int(submission.SubmittedAt),
		Fields:       make([]*gqlmodel.FormSubmissionField, 0, len(submission.Fields)),
	}
	for i := range submission.Fields {
		field := &submission.Fields[i]
		out.Fields = append(out.Fields, &gqlmodel.FormSubmissionField{
			Label: field.Label,
			Value: field.Value,
		})
	}
	return out
}

func mapHOSCertificationSummary(
	summary *telematicsservice.HOSCertificationSummary,
) *gqlmodel.HosCertificationSummary {
	return &gqlmodel.HosCertificationSummary{
		WorkerID:        summary.WorkerID.String(),
		WorkerName:      summary.WorkerName,
		UncertifiedDays: summary.UncertifiedDays,
		TotalDays:       summary.TotalDays,
	}
}

func mapDriverFeasibility(result *telematicsservice.DriverFeasibility) *gqlmodel.DriverFeasibility {
	out := &gqlmodel.DriverFeasibility{
		WorkerID:         result.WorkerID,
		WorkerName:       result.WorkerName,
		DriveRemainingMs: int(result.DriveRemainingMs),
		ShiftRemainingMs: int(result.ShiftRemainingMs),
		CycleRemainingMs: int(result.CycleRemainingMs),
		EstimatedDriveMs: int(result.EstimatedDriveMs),
		Verdict:          result.Verdict,
		Reasons:          result.Reasons,
		RecordedAt:       int(result.RecordedAt),
	}
	if result.DutyStatus != "" {
		dutyStatus := string(result.DutyStatus)
		out.DutyStatus = &dutyStatus
	}
	if result.DeadheadMiles != nil {
		deadhead := *result.DeadheadMiles
		out.DeadheadMiles = &deadhead
	}
	if result.TractorID != "" {
		tractorID := result.TractorID
		out.TractorID = &tractorID
	}
	if result.TractorCode != "" {
		tractorCode := result.TractorCode
		out.TractorCode = &tractorCode
	}
	return out
}

func workerDisplayName(wrk *worker.Worker) string {
	return strings.TrimSpace(wrk.FullName())
}
