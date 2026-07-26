package telematicsservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
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
			s.sendHOSAlert(ctx, tenantInfo, state.WorkerID, hosAlert{
				Kind:     "shift-driving-violation",
				Title:    "HOS violation: shift driving limit",
				Priority: notification.PriorityCritical,
				Message: fmt.Sprintf(
					"%s has exceeded the shift driving limit. Remove them from dispatch until the clock resets.",
					name,
				),
				DriverTitle: "You've hit your shift driving limit",
				DriverMessage: "You've exceeded the shift driving limit. Stop driving and " +
					"contact dispatch — you can't be dispatched until your clock resets.",
			})
		}
		if state.CycleViolationMs > 0 && (prev == nil || prev.CycleViolationMs <= 0) {
			s.sendHOSAlert(ctx, tenantInfo, state.WorkerID, hosAlert{
				Kind:     "cycle-violation",
				Title:    "HOS violation: cycle limit",
				Priority: notification.PriorityCritical,
				Message: fmt.Sprintf(
					"%s has exceeded the cycle on-duty limit. Remove them from dispatch until the cycle resets.",
					name,
				),
				DriverTitle: "You've hit your cycle limit",
				DriverMessage: "You've exceeded the cycle on-duty limit. Contact dispatch — " +
					"you can't be dispatched until your cycle resets.",
			})
		}
		if isOnDutyStatus(state.DutyStatus) &&
			state.DriveRemainingMs < lowDriveThresholdMs &&
			(prev == nil || prev.DriveRemainingMs >= lowDriveThresholdMs) {
			s.sendHOSAlert(ctx, tenantInfo, state.WorkerID, hosAlert{
				Kind:          "low-drive-time",
				Title:         "Driver low on drive time",
				Priority:      notification.PriorityHigh,
				Message:       fmt.Sprintf("%s has less than 1 hour of drive time remaining.", name),
				DriverTitle:   "Less than 1 hour of drive time",
				DriverMessage: "You have less than 1 hour of drive time left. Plan your next stop now.",
			})
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

// hosAlert carries both sides of an HOS event: the dispatcher-facing copy that
// goes out organization-wide, and the second-person copy delivered only to the
// affected driver's portal user.
type hosAlert struct {
	Kind          string
	Title         string
	Message       string
	DriverTitle   string
	DriverMessage string
	Priority      notification.Priority
}

func (s *Service) sendHOSAlert(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	alert hosAlert,
) {
	correlation := fmt.Sprintf("hos-%s-%s", alert.Kind, workerID)
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
		Priority:       alert.Priority,
		Channel:        notification.ChannelGlobal,
		Title:          alert.Title,
		Message:        alert.Message,
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

	s.notifyDriverOfHOSAlert(ctx, tenantInfo, workerID, alert, correlation)
}

// notifyDriverOfHOSAlert delivers the driver's own copy of an HOS event. It
// shares the dispatcher alert's dedupe gate, so a driver is notified at most
// once per kind per dedupe window.
func (s *Service) notifyDriverOfHOSAlert(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	alert hosAlert,
	correlation string,
) {
	if s.driverNotify == nil || alert.DriverTitle == "" {
		return
	}

	s.driverNotify.NotifyWithCorrelation(ctx, &drivernotificationservice.DriverNotification{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
		EventType:  "dash.hos_alert",
		Priority:   alert.Priority,
		Title:      alert.DriverTitle,
		Message:    alert.DriverMessage,
		Link:       "/dash/hos",
		RelatedEntities: map[string]any{
			"workerId": workerID.String(),
		},
	}, "dash-"+correlation)
}
