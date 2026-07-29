package telematicsservice

import (
	"context"

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
	result *TenantSweepResult,
) error {
	workersByExternalID, err := s.workersByExternalID(ctx, tenantInfo)
	if err != nil {
		return err
	}
	if len(workersByExternalID) == 0 {
		return nil
	}

	now := timeutils.NowUnix()
	windowStart := now - hosLogLookbackSeconds
	providerType := string(provider.Type())

	workerIDs := make([]pulid.ID, 0, len(workersByExternalID))
	logs := make([]*telematics.WorkerHOSLog, 0, len(workersByExternalID)*16)
	failed := 0
	for externalID, workerID := range workersByExternalID {
		entries, listErr := provider.ListHOSLogs(ctx, externalID, windowStart, now)
		if listErr != nil {
			failed++
			s.l.Warn("failed to fetch hos logs for worker",
				zap.String("organizationId", tenantInfo.OrgID.String()),
				zap.String("workerId", workerID.String()),
				zap.Error(listErr))
			continue
		}

		workerIDs = append(workerIDs, workerID)
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

	if failed > 0 {
		s.l.Warn("hos log sweep skipped workers after provider errors",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Int("failed", failed))
	}
	return nil
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
