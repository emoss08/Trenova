package telematicsresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
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
			if name := base.WorkerDisplayName(position.Tractor.PrimaryWorker); name != "" {
				out.PrimaryWorkerName = &name
			}
		}
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
