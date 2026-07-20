package reportjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const (
	maxDueSchedulesPerTick   = 200
	maxConsecutiveFailures   = 5
	scheduleSkipNotification = "report_schedule_skipped"
)

type DispatchDueSchedulesResult struct {
	Dispatched int `json:"dispatched"`
	Skipped    int `json:"skipped"`
	Disabled   int `json:"disabled"`
}

func (a *Activities) DispatchDueSchedulesActivity(
	ctx context.Context,
) (*DispatchDueSchedulesResult, error) {
	result := &DispatchDueSchedulesResult{}
	now := timeutils.NowUnix()

	due, err := a.scheduleRepo.ListDue(ctx, now, maxDueSchedulesPerTick)
	if err != nil {
		return nil, err
	}

	for _, schedule := range due {
		activity.RecordHeartbeat(ctx, schedule.ID.String())

		if err = a.dispatchSchedule(ctx, schedule, now, result); err != nil {
			a.l.Error("failed to dispatch report schedule",
				zap.String("scheduleId", schedule.ID.String()), zap.Error(err))
		}
	}

	return result, nil
}

func (a *Activities) dispatchSchedule(
	ctx context.Context,
	schedule *report.ReportSchedule,
	now int64,
	result *DispatchDueSchedulesResult,
) error {
	// Always advance the schedule first so a failing schedule can never spin
	// on every tick.
	nextRun, err := cronutils.NextRun(schedule.CronExpression, schedule.Timezone, now)
	if err != nil {
		schedule.Enabled = false
		result.Disabled++
		a.notifyScheduleOwner(ctx, schedule,
			"Report schedule disabled",
			"The schedule's cron expression is invalid and the schedule has been disabled.")
		_, updateErr := a.scheduleRepo.Update(ctx, schedule)
		return updateErr
	}
	schedule.NextRunAt = nextRun

	tenant := pagination.TenantInfo{
		OrgID:  schedule.OrganizationID,
		BuID:   schedule.BusinessUnitID,
		UserID: schedule.RunAsID,
	}

	definition, err := a.defRepo.GetByID(ctx, &repositories.GetReportDefinitionRequest{
		TenantInfo:   tenant,
		DefinitionID: schedule.DefinitionID,
	})
	if err != nil {
		result.Skipped++
		a.recordScheduleFailure(ctx, schedule,
			"The scheduled report definition could not be loaded.")
		// Persist the advanced NextRunAt (and failure streak) — otherwise the
		// schedule stays due and re-fires on every dispatch tick.
		if _, updateErr := a.scheduleRepo.Update(ctx, schedule); updateErr != nil {
			return updateErr
		}
		return err
	}

	if definition.Status != report.DefinitionStatusActive {
		result.Skipped++
		a.recordScheduleFailure(ctx, schedule, fmt.Sprintf(
			"The scheduled report %q is %s and was skipped.",
			definition.Name, definition.Status,
		))
		_, updateErr := a.scheduleRepo.Update(ctx, schedule)
		return updateErr
	}

	revisions, err := a.defRepo.ListRevisions(ctx, &repositories.ListReportRevisionsRequest{
		TenantInfo:   tenant,
		DefinitionID: definition.ID,
		Limit:        1,
	})
	if err != nil || len(revisions) == 0 {
		result.Skipped++
		_, updateErr := a.scheduleRepo.Update(ctx, schedule)
		if err == nil {
			err = fmt.Errorf("definition %s has no revisions", definition.ID)
		}
		if updateErr != nil {
			return updateErr
		}
		return err
	}

	a.dispatchScheduleRuns(ctx, schedule, definition, revisions[0], result)

	_, err = a.scheduleRepo.Update(ctx, schedule)
	return err
}

func (a *Activities) dispatchScheduleRuns(
	ctx context.Context,
	schedule *report.ReportSchedule,
	definition *report.ReportDefinition,
	revision *report.ReportDefinitionRevision,
	result *DispatchDueSchedulesResult,
) {
	for _, format := range schedule.Formats {
		run := &report.ReportRun{
			BusinessUnitID: schedule.BusinessUnitID,
			OrganizationID: schedule.OrganizationID,
			DefinitionID:   definition.ID,
			RevisionID:     revision.ID,
			ScheduleID:     schedule.ID,
			RequestedByID:  schedule.RunAsID,
			Trigger:        report.RunTriggerScheduled,
			Format:         report.Format(format),
			Status:         report.RunStatusQueued,
		}

		created, createErr := a.runRepo.Create(ctx, run)
		if createErr != nil {
			a.l.Error("failed to create scheduled report run",
				zap.String("scheduleId", schedule.ID.String()), zap.Error(createErr))
			continue
		}

		if _, startErr := a.workflows.StartWorkflow(ctx,
			client.StartWorkflowOptions{
				ID:                    "report-run/" + created.ID.String(),
				TaskQueue:             temporaltype.ReportTaskQueue,
				WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			},
			RunReportWorkflow,
			&RunReportPayload{
				RunID:          created.ID,
				OrganizationID: created.OrganizationID,
				BusinessUnitID: created.BusinessUnitID,
			},
		); startErr != nil {
			a.l.Error("failed to start scheduled report workflow",
				zap.String("runId", created.ID.String()), zap.Error(startErr))
			created.Status = report.RunStatusFailed
			created.Error = &report.RunError{
				Code:    "ENQUEUE_FAILED",
				Message: "The scheduled report could not be queued for generation",
			}
			if _, failErr := a.runRepo.Update(ctx, created); failErr != nil {
				a.l.Error("failed to mark scheduled run as failed", zap.Error(failErr))
			}
			continue
		}

		schedule.LastRunID = created.ID
		result.Dispatched++
	}
}

// recordScheduleFailure increments the failure streak, auto-disabling the
// schedule and notifying its owner once the streak reaches the limit.
func (a *Activities) recordScheduleFailure(
	ctx context.Context,
	schedule *report.ReportSchedule,
	reason string,
) {
	schedule.ConsecutiveFailures++
	if schedule.ConsecutiveFailures < maxConsecutiveFailures {
		a.notifyScheduleOwner(ctx, schedule, "Scheduled report skipped", reason)
		return
	}

	schedule.Enabled = false
	a.notifyScheduleOwner(ctx, schedule,
		"Report schedule disabled",
		fmt.Sprintf(
			"%s The schedule failed %d consecutive times and has been disabled.",
			reason, schedule.ConsecutiveFailures,
		))
}

func (a *Activities) notifyScheduleOwner(
	ctx context.Context,
	schedule *report.ReportSchedule,
	title, message string,
) {
	data := map[string]any{
		dataKeyScheduleID: schedule.ID.String(),
	}
	if def, err := a.defRepo.GetByID(ctx, &repositories.GetReportDefinitionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: schedule.OrganizationID,
			BuID:  schedule.BusinessUnitID,
		},
		DefinitionID: schedule.DefinitionID,
	}); err == nil {
		data[dataKeyReportName] = def.Name
	}

	if _, err := a.notification.Create(ctx, &notification.Notification{
		OrganizationID: schedule.OrganizationID,
		BusinessUnitID: &schedule.BusinessUnitID,
		TargetUserID:   &schedule.RunAsID,
		Channel:        notification.ChannelUser,
		EventType:      scheduleSkipNotification,
		Priority:       notification.PriorityHigh,
		Title:          title,
		Message:        message,
		Data:           data,
		Source:         "reportjobs.DispatchDueSchedules",
	}); err != nil {
		a.l.Warn("failed to notify schedule owner",
			zap.String("scheduleId", schedule.ID.String()), zap.Error(err))
	}
}

// recordScheduleRunOutcome feeds terminal run states back into the owning
// schedule: failures build the auto-disable streak, successes reset it.
func (a *Activities) recordScheduleRunOutcome(ctx context.Context, run *report.ReportRun) {
	if run.ScheduleID.IsNil() {
		return
	}

	schedule, err := a.scheduleRepo.GetByID(ctx, &repositories.GetReportScheduleRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: run.OrganizationID,
			BuID:  run.BusinessUnitID,
		},
		ScheduleID: run.ScheduleID,
	})
	if err != nil {
		a.l.Warn("failed to load schedule for run outcome",
			zap.String("runId", run.ID.String()), zap.Error(err))
		return
	}

	//nolint:exhaustive // only terminal success/failure feed the streak
	switch run.Status {
	case report.RunStatusSucceeded:
		if schedule.ConsecutiveFailures == 0 {
			return
		}
		schedule.ConsecutiveFailures = 0
	case report.RunStatusFailed:
		reason := "The scheduled report run failed."
		if run.Error != nil && run.Error.Message != "" {
			reason = run.Error.Message
		}
		a.recordScheduleFailure(ctx, schedule, reason)
	default:
		return
	}

	if _, err = a.scheduleRepo.Update(ctx, schedule); err != nil {
		a.l.Warn("failed to update schedule run outcome",
			zap.String("scheduleId", schedule.ID.String()), zap.Error(err))
	}
}
