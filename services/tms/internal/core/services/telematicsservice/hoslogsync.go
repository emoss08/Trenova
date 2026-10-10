package telematicsservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// hosLogLookbackSeconds covers one day more than the longest 8-day cycle window, so
// every sweep re-reads the full span the projection engine reasons over and remains
// self-healing against ELD edits and late certifications.
const hosLogLookbackSeconds = int64(9 * 86400)

func (s *Service) syncHOSLogs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider services.TelematicsProvider,
	drivers *sweepDrivers,
	result *TenantSweepResult,
) error {
	if len(drivers.externalIDs) == 0 {
		return nil
	}

	now := timeutils.NowUnix()
	windowStart := now - hosLogLookbackSeconds
	providerType := string(provider.Type())

	logsByDriver, listErr := provider.ListHOSLogs(ctx, drivers.externalIDs, windowStart, now)
	failedDrivers, ok := partialDriverFailures(listErr)
	if !ok {
		s.l.Warn("failed to fetch hos logs for workers",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Int("failed", len(drivers.externalIDs)),
			zap.Error(listErr))
		return nil
	}

	workerIDs := make([]pulid.ID, 0, max(len(drivers.externalIDs)-len(failedDrivers), 0))
	logs := make([]*telematics.WorkerHOSLog, 0, len(drivers.externalIDs)*16)
	for _, externalID := range drivers.externalIDs {
		if _, failed := failedDrivers[externalID]; failed {
			continue
		}

		workerID := drivers.workersByExternalID[externalID]
		workerIDs = append(workerIDs, workerID)
		entries := logsByDriver[externalID]
		for i := range entries {
			entry := &entries[i]
			if entry.LogStartAt <= 0 {
				continue
			}
			logs = append(logs, &telematics.WorkerHOSLog{
				OrganizationID:    tenantInfo.OrgID,
				BusinessUnitID:    tenantInfo.BuID,
				WorkerID:          workerID,
				LogStartAt:        entry.LogStartAt,
				DutyStatus:        normalizeDutyStatus(entry.HosStatusType),
				LogEndAt:          entry.LogEndAt,
				Remark:            entry.Remark,
				Provider:          providerType,
				ProviderVehicleID: entry.VehicleID,
				ReceivedAt:        now,
			})
		}
	}

	if len(failedDrivers) > 0 {
		s.l.Warn("hos log sweep skipped workers after provider errors",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Int("failed", len(failedDrivers)),
			zap.Error(listErr))
	}

	if len(workerIDs) == 0 {
		return nil
	}

	synced, err := s.repo.SyncWorkerHOSLogs(ctx, &repositories.SyncWorkerHOSLogsRequest{
		TenantInfo:  tenantInfo,
		WorkerIDs:   workerIDs,
		WindowStart: windowStart,
		Logs:        logs,
	})
	if err != nil {
		return err
	}
	result.HOSLogsUpserted = synced
	return nil
}

func partialDriverFailures(err error) (map[string]struct{}, bool) {
	if err == nil {
		return nil, true
	}
	batchErr, ok := errors.AsType[*services.ProviderDriverBatchError](err)
	if !ok {
		return nil, false
	}
	return batchErr.FailedDriverSet(), true
}

// normalizeDutyStatus keeps unknown provider statuses in the timeline as on-duty time
// rather than dropping them: a gap would read as rest and could credit a driver with a
// reset they never took.
func normalizeDutyStatus(raw string) telematics.DutyStatus {
	status := telematics.DutyStatus(raw)
	if status.IsValid() {
		return status
	}
	return telematics.DutyStatusOnDuty
}
