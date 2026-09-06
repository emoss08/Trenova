package compliancejobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

const (
	eventDriverTrainingDue = "dash.training_due"
	eventTrainingOverdue   = "training_overdue"
	eventTrainingExpired   = "training_expired"
	trainingLink           = "/hr/workers?tab=training"
)

// trainingDueSteps are the days-before-due marks a driver is nudged at; the
// due day and every day past it fire under the weekly dedupe window.
var trainingDueSteps = []int64{7, 3, 1}

type trainingSweepState struct {
	now            int64
	remindersByOrg map[pagination.TenantInfo]bool
	// touchedWorkers are the workers the sweep walked past. Their roster
	// roll-up is refreshed at the end, because a course going overdue at
	// midnight moves a worker's standing with nobody writing anything.
	touchedWorkers map[pulid.ID]pagination.TenantInfo
	result         *TrainingReminderSweepResult
}

func (a *Activities) TrainingReminderSweepActivity(
	ctx context.Context,
) (*TrainingReminderSweepResult, error) {
	state := &trainingSweepState{
		now:            timeutils.NowUnix(),
		remindersByOrg: make(map[pagination.TenantInfo]bool),
		touchedWorkers: make(map[pulid.ID]pagination.TenantInfo),
		result:         new(TrainingReminderSweepResult),
	}

	if err := a.walkTraining(ctx, state, a.trainingRepo.ListDue, a.sweepDueTraining); err != nil {
		return nil, err
	}
	if err := a.walkTraining(ctx, state, a.trainingRepo.ListExpiring, a.sweepExpiringTraining); err != nil {
		return nil, err
	}

	a.refreshTrainingRollups(ctx, state)

	return state.result, nil
}

// refreshTrainingRollups repairs the roster cache for every worker the sweep
// walked past. Best-effort per worker: one worker's cache failing must not
// abandon the rest of the sweep.
func (a *Activities) refreshTrainingRollups(ctx context.Context, state *trainingSweepState) {
	for workerID, tenantInfo := range state.touchedWorkers {
		if _, err := a.training.RefreshRollup(ctx, tenantInfo, workerID); err != nil {
			a.logger.Warn("failed to refresh training rollup after sweep",
				zap.String("workerId", workerID.String()),
				zap.Error(err))
		}
	}
}

type trainingLister func(
	ctx context.Context,
	req *repositories.ListTrainingRemindersRequest,
) ([]*worker.WorkerTrainingRecord, error)

func (a *Activities) walkTraining(
	ctx context.Context,
	state *trainingSweepState,
	list trainingLister,
	handle func(context.Context, *trainingSweepState, *worker.WorkerTrainingRecord) error,
) error {
	var after pulid.ID
	for {
		page, err := list(ctx, &repositories.ListTrainingRemindersRequest{
			HorizonDays: sweepHorizonDays,
			GraceDays:   sweepGraceDays,
			AfterID:     after,
			Limit:       sweepPageSize,
		})
		if err != nil {
			return err
		}
		for _, record := range page {
			state.result.RecordsChecked++
			state.touchedWorkers[record.WorkerID] = pagination.TenantInfo{OrgID: record.OrganizationID, BuID: record.BusinessUnitID}
			if err = handle(ctx, state, record); err != nil {
				state.result.Failed++
				a.logger.Error("training sweep failed",
					zap.String("recordId", record.ID.String()),
					zap.String("workerId", record.WorkerID.String()),
					zap.Error(err))
			}
		}
		activity.RecordHeartbeat(ctx, state.result.RecordsChecked)
		if len(page) < sweepPageSize {
			return nil
		}
		after = page[len(page)-1].ID
	}
}

func (a *Activities) sweepDueTraining(
	ctx context.Context,
	state *trainingSweepState,
	record *worker.WorkerTrainingRecord,
) error {
	if record.Worker == nil || record.Course == nil || record.DueAt == nil {
		return nil
	}
	tenantInfo := pagination.TenantInfo{OrgID: record.OrganizationID, BuID: record.BusinessUnitID}
	daysLeft := worker.DaysUntil(*record.DueAt, state.now)
	if trainingStep(daysLeft) < 0 {
		return nil
	}

	remind, err := a.trainingRemindersEnabled(ctx, tenantInfo, state.remindersByOrg)
	if err != nil {
		return err
	}
	if remind && !record.Worker.UserID.IsNil() {
		sent, notifyErr := a.notifyDriverTraining(ctx, tenantInfo, record, daysLeft, false)
		if notifyErr != nil {
			return notifyErr
		}
		if sent {
			state.result.DriverNotifications++
		}
	}

	if daysLeft < 0 && record.Course.IsRequired {
		sent, alertErr := a.alertTraining(ctx, tenantInfo, record, daysLeft, eventTrainingOverdue)
		if alertErr != nil {
			return alertErr
		}
		if sent {
			state.result.ComplianceAlerts++
		}
	}
	return nil
}

func (a *Activities) sweepExpiringTraining(
	ctx context.Context,
	state *trainingSweepState,
	record *worker.WorkerTrainingRecord,
) error {
	if record.Worker == nil || record.Course == nil || record.ExpiresAt == nil {
		return nil
	}
	tenantInfo := pagination.TenantInfo{OrgID: record.OrganizationID, BuID: record.BusinessUnitID}
	daysLeft := worker.DaysUntil(*record.ExpiresAt, state.now)

	if daysLeft < 0 {
		if _, err := a.training.MarkExpired(ctx, record); err != nil {
			return err
		}
		state.result.Expired++
		if record.Course.AppliesTo(record.Worker) {
			if _, err := a.training.AssignRequired(ctx, tenantInfo, record.WorkerID, record.AssignedByID); err != nil {
				return err
			}
			sent, alertErr := a.alertTraining(ctx, tenantInfo, record, daysLeft, eventTrainingExpired)
			if alertErr != nil {
				return alertErr
			}
			if sent {
				state.result.ComplianceAlerts++
			}
		}
		return nil
	}

	if reminderStep(daysLeft) < 0 {
		return nil
	}
	remind, err := a.trainingRemindersEnabled(ctx, tenantInfo, state.remindersByOrg)
	if err != nil || !remind || record.Worker.UserID.IsNil() {
		return err
	}
	sent, notifyErr := a.notifyDriverTraining(ctx, tenantInfo, record, daysLeft, true)
	if notifyErr != nil {
		return notifyErr
	}
	if sent {
		state.result.DriverNotifications++
	}
	return nil
}

// trainingStep returns the reminder mark a days-until-due value sits on, or
// -1 between marks. The due day and every day after are mark 0.
func trainingStep(daysLeft int64) int64 {
	if daysLeft <= 0 {
		return 0
	}
	for _, step := range trainingDueSteps {
		if daysLeft == step {
			return step
		}
	}
	return -1
}

func (a *Activities) trainingRemindersEnabled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	cache map[pagination.TenantInfo]bool,
) (bool, error) {
	if enabled, ok := cache[tenantInfo]; ok {
		return enabled, nil
	}
	control, err := a.dashControlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return false, err
	}
	// A bundling cadence collects this course into the digest instead, so
	// sending one here as well would tell the driver twice.
	enabled := control.SendCredentialReminders && !control.DriverDigestCadence.Bundles()
	cache[tenantInfo] = enabled
	return enabled, nil
}

func trainingDate(value *int64) string {
	if value == nil {
		return ""
	}
	return timeutils.FormatUnixDateIn(*value, "")
}

func (a *Activities) notifyDriverTraining(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	record *worker.WorkerTrainingRecord,
	daysLeft int64,
	renewal bool,
) (bool, error) {
	correlation := fmt.Sprintf("training-driver-%s", record.ID)
	exists, err := a.alreadySent(ctx, tenantInfo, eventDriverTrainingDue, correlation)
	if err != nil || exists {
		return false, err
	}

	dueAt := record.DueAt
	if renewal {
		dueAt = record.ExpiresAt
	}
	a.driverNotify.NotifyWithCorrelation(ctx, &drivernotificationservice.DriverNotification{
		TenantInfo: tenantInfo,
		WorkerID:   record.WorkerID,
		EventType:  eventDriverTrainingDue,
		Priority:   notification.PriorityHigh,
		Context: documenttemplate.DriverNotificationContext{
			CourseName: record.Course.Name,
			DueInDays:  int(daysLeft),
			DueAt:      trainingDate(dueAt),
			Renewal:    renewal,
		},
		Link: "/dash/profile",
	}, correlation)
	return true, nil
}

func (a *Activities) alertTraining(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	record *worker.WorkerTrainingRecord,
	daysLeft int64,
	eventType string,
) (bool, error) {
	correlation := fmt.Sprintf("training-compliance-%s-%s", record.ID, eventType)
	exists, err := a.alreadySent(ctx, tenantInfo, eventType, correlation)
	if err != nil || exists {
		return false, err
	}

	buID := tenantInfo.BuID
	correlationID := correlation
	name := record.Worker.FirstName + " " + record.Worker.LastName
	entity := &notification.Notification{
		OrganizationID:  tenantInfo.OrgID,
		BusinessUnitID:  &buID,
		EventType:       eventType,
		Channel:         notification.ChannelGlobal,
		Priority:        notification.PriorityHigh,
		Data:            map[string]any{"link": trainingLink, "workerId": record.WorkerID.String()},
		RelatedEntities: map[string]any{"workerId": record.WorkerID.String(), "trainingRecordId": record.ID.String()},
		CorrelationID:   &correlationID,
		Source:          "compliance_sweep",
	}
	if eventType == eventTrainingExpired {
		entity.Title = "Driver training lapsed"
		entity.Message = fmt.Sprintf(
			"%s's %s certification expired %d day(s) ago. A renewal has been assigned.",
			name,
			record.Course.Name,
			-daysLeft,
		)
	} else {
		entity.Title = "Required training overdue"
		entity.Message = fmt.Sprintf(
			"%s has not completed %s, which was due %d day(s) ago on %s.",
			name,
			record.Course.Name,
			-daysLeft,
			trainingDate(record.DueAt),
		)
	}
	if _, err = a.notifications.Create(ctx, entity); err != nil {
		return false, err
	}
	return true, nil
}
