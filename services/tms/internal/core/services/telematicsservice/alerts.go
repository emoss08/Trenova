package telematicsservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	lowDriveThresholdMs   = hourMs
	hosAlertDedupeSeconds = int64(6 * 3600)
)

func (s *Service) notifyCriticalHOSTransitions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	next []*telematics.WorkerHOSState,
) error {
	if s.notifications == nil || len(next) == 0 {
		return nil
	}

	previous, err := s.repo.ListWorkerHOSStates(ctx, &repositories.ListWorkerHOSStatesRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	previousByWorker := make(map[pulid.ID]*telematics.WorkerHOSState, len(previous))
	for _, state := range previous {
		previousByWorker[state.WorkerID] = state
	}

	names, err := s.workerNames(ctx, tenantInfo)
	if err != nil {
		return err
	}

	for _, state := range next {
		prev := previousByWorker[state.WorkerID]
		name := names[state.WorkerID]
		if name == "" {
			name = state.WorkerID.String()
		}

		if state.ShiftDrivingViolationMs > 0 && (prev == nil || prev.ShiftDrivingViolationMs <= 0) {
			s.sendHOSAlert(
				ctx,
				tenantInfo,
				state.WorkerID,
				"shift-driving-violation",
				"HOS violation: shift driving limit",
				fmt.Sprintf(
					"%s has exceeded the shift driving limit. Remove them from dispatch until the clock resets.",
					name,
				),
				notification.PriorityCritical,
			)
		}
		if state.CycleViolationMs > 0 && (prev == nil || prev.CycleViolationMs <= 0) {
			s.sendHOSAlert(
				ctx,
				tenantInfo,
				state.WorkerID,
				"cycle-violation",
				"HOS violation: cycle limit",
				fmt.Sprintf(
					"%s has exceeded the cycle on-duty limit. Remove them from dispatch until the cycle resets.",
					name,
				),
				notification.PriorityCritical,
			)
		}
		if isOnDutyStatus(state.DutyStatus) &&
			state.DriveRemainingMs < lowDriveThresholdMs &&
			(prev == nil || prev.DriveRemainingMs >= lowDriveThresholdMs) {
			s.sendHOSAlert(ctx, tenantInfo, state.WorkerID, "low-drive-time",
				"Driver low on drive time",
				fmt.Sprintf("%s has less than 1 hour of drive time remaining.", name),
				notification.PriorityHigh)
		}
	}
	return nil
}

func isOnDutyStatus(status telematics.DutyStatus) bool {
	return status == telematics.DutyStatusDriving || status == telematics.DutyStatusOnDuty
}

func (s *Service) workerNames(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (map[pulid.ID]string, error) {
	mappings, err := s.repo.ListWorkerMappings(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	names := make(map[pulid.ID]string, len(mappings))
	for _, mapping := range mappings {
		names[mapping.WorkerID] = strings.TrimSpace(
			mapping.FirstName + " " + mapping.LastName,
		)
	}
	return names, nil
}

func (s *Service) sendHOSAlert(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	kind string,
	title string,
	message string,
	priority notification.Priority,
) {
	correlation := fmt.Sprintf("hos-%s-%s", kind, workerID)
	exists, err := s.notifications.ExistsRecent(
		ctx,
		repositories.ExistsRecentNotificationRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			EventType:      "hos_alert",
			CorrelationID:  correlation,
			Since:          timeutils.NowUnix() - hosAlertDedupeSeconds,
		},
	)
	if err != nil || exists {
		return
	}

	buID := tenantInfo.BuID
	correlationID := correlation
	entity := &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		EventType:      "hos_alert",
		Priority:       priority,
		Channel:        notification.ChannelGlobal,
		Title:          title,
		Message:        message,
		Data: map[string]any{
			"link": "/dispatch/workers?panelType=edit&panelEntityId=" + workerID.String() + "&tab=hos",
		},
		RelatedEntities: map[string]any{
			"workerId": workerID.String(),
		},
		CorrelationID: &correlationID,
		Source:        "telematics",
	}
	if _, createErr := s.notifications.Create(ctx, entity); createErr != nil {
		s.l.Warn("failed to create HOS alert notification")
	}
}
