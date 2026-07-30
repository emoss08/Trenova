package reportjobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

const (
	deliverMaxAttempts = 5

	deliveredEventType    = "report_run_delivered"
	deliveryFailedEvent   = "report_delivery_email_failed"
	deliveryDedupeWindow  = 24 * time.Hour
	deliverySource        = "reportjobs.DeliverScheduledRun"
	deliveryChannelEmail  = "email"
	deliveryChannelInApp  = "notification"
	deliveryOutcomeOK     = "delivered"
	deliveryOutcomeFailed = "failed"

	alertSkipConditionUnmet = "condition_not_met"
	alertSkipAlreadyFiring  = "already_alerting"

	dataKeyRunID             = "runId"
	dataKeyStatus            = "status"
	dataKeyScheduleID        = "scheduleId"
	dataKeyFormat            = "format"
	dataKeyRowCount          = "rowCount"
	dataKeyTruncated         = "truncated"
	dataKeyReportName        = "reportName"
	dataKeyByteSize          = "byteSize"
	dataKeyArtifactExpiresAt = "artifactExpiresAt"
)

func (a *Activities) DeliverScheduledRunActivity(
	ctx context.Context,
	payload *DeliverRunPayload,
) (*DeliverRunResult, error) {
	tenant := pagination.TenantInfo{
		OrgID: payload.OrganizationID,
		BuID:  payload.BusinessUnitID,
	}

	run, err := a.runRepo.GetByID(ctx, &repositories.GetReportRunRequest{
		TenantInfo: tenant,
		RunID:      payload.RunID,
	})
	if err != nil {
		return nil, err
	}

	if run.Status != report.RunStatusSucceeded || run.ArtifactKey == "" || run.ScheduleID.IsNil() {
		return &DeliverRunResult{Skipped: true}, nil
	}

	schedule, err := a.scheduleRepo.GetByID(ctx, &repositories.GetReportScheduleRequest{
		TenantInfo: tenant,
		ScheduleID: run.ScheduleID,
	})
	if err != nil {
		var notFound *errortypes.NotFoundError
		if errors.As(err, &notFound) {
			return &DeliverRunResult{Skipped: true}, nil
		}
		return nil, err
	}

	if !schedule.Delivery.HasEmail() && !schedule.Delivery.HasNotify() {
		return &DeliverRunResult{Skipped: true}, nil
	}

	if deliver, reason := a.evaluateAlert(ctx, run, schedule, payload.Digest); !deliver {
		return &DeliverRunResult{Skipped: true, AlertSkipReason: reason}, nil
	}

	runnerTenant := pagination.TenantInfo{
		OrgID:  run.OrganizationID,
		BuID:   run.BusinessUnitID,
		UserID: run.RequestedByID,
	}
	_, title, _ := a.runDisplayMetadata(ctx, run, runnerTenant)

	result := &DeliverRunResult{}

	if schedule.Delivery.HasEmail() {
		if err = a.deliverRunEmail(ctx, run, schedule, title, payload.Digest, result); err != nil {
			return nil, err
		}
	}

	if schedule.Delivery.HasNotify() {
		a.deliverRunNotifications(ctx, run, schedule, title, result)
	}

	a.auditDelivery(run, schedule, result)

	return result, nil
}

// digestTotal reads one column's grand total out of the captured digest.
// A missing column or absent total returns nil rather than zero: the measure
// did not appear, and treating that as zero would fire "revenue below target"
// on a report that simply lost the column.
func digestTotal(digest *services.ReportDigest, columnID string) *float64 {
	if digest.IsEmpty() || len(digest.Totals) == 0 {
		return nil
	}
	for i := range digest.Columns {
		if digest.Columns[i].ID != columnID {
			continue
		}
		if i >= len(digest.Totals) {
			return nil
		}
		return numericTotal(digest.Totals[i])
	}
	return nil
}

func numericTotal(value any) *float64 {
	switch typed := value.(type) {
	case decimal.Decimal:
		converted, _ := typed.Float64()
		return &converted
	case int64:
		converted := float64(typed)
		return &converted
	case float64:
		return &typed
	default:
		return nil
	}
}

// evaluateAlert decides whether a run's result is worth sending. A schedule
// without an alert always delivers, which keeps every existing schedule
// behaving exactly as before.
//
// The firing flag is what makes an alert usable day to day: a daily exception
// report that stays broken for a week should mail once, not seven times, and it
// should mail again the next time the condition clears and trips.
func (a *Activities) evaluateAlert(
	ctx context.Context,
	run *report.ReportRun,
	schedule *report.ReportSchedule,
	digest *services.ReportDigest,
) (deliver bool, reason string) {
	alert := schedule.Alert
	if alert == nil {
		return true, ""
	}

	wasFiring := schedule.AlertFiring
	matched := alert.Matches(run.RowCount)
	if alert.TargetsMeasure() {
		matched = alert.MatchesValue(digestTotal(digest, alert.ColumnID))
	}

	// Only the transition is written. A failure to persist is logged rather than
	// raised: the delivery decision for this run stands on its own result, and
	// losing the flag costs at most one duplicate email, never a missed alert.
	if matched != wasFiring {
		schedule.AlertFiring = matched
		if _, err := a.scheduleRepo.Update(ctx, schedule); err != nil {
			a.l.Warn("failed to persist report alert state",
				zap.String("scheduleId", schedule.ID.String()),
				zap.Bool("firing", matched),
				zap.Error(err))
		}
	}

	switch {
	case !matched:
		return false, alertSkipConditionUnmet
	case alert.SuppressWhileFiring && wasFiring:
		return false, alertSkipAlreadyFiring
	default:
		return true, ""
	}
}

func (a *Activities) deliverRunEmail(
	ctx context.Context,
	run *report.ReportRun,
	schedule *report.ReportSchedule,
	title string,
	digest *services.ReportDigest,
	result *DeliverRunResult,
) error {
	if a.email == nil {
		a.recordEmailFailure(ctx, run, schedule, title, result,
			"Email delivery is not configured on this instance")
		return nil
	}

	attach, attachTooLarge := a.attachmentPlan(run, schedule, title)
	tenantInfo := pagination.TenantInfo{
		OrgID:  run.OrganizationID,
		BuID:   run.BusinessUnitID,
		UserID: schedule.RunAsID,
	}

	// A styling problem must not cost a scheduled delivery, so a failed render
	// falls back to the shipped default. If even that fails the schedule owner is
	// told, because the alternative is a report nobody knows never arrived.
	rendered, err := a.templates.RenderMessage(ctx, &services.RenderMessageRequest{
		TenantInfo: tenantInfo,
		Kind:       documenttemplate.KindReportDeliveryEmail,
		Data: a.deliveryEmailContext(&deliveryEmailContent{
			run:            run,
			schedule:       schedule,
			title:          title,
			digest:         digest,
			attached:       attach != nil,
			attachTooLarge: attachTooLarge,
		}),
		ReferenceID:       run.ID,
		UserID:            schedule.RunAsID,
		FallbackToBuiltIn: true,
	})
	if err != nil {
		// Deliberately not returned: a template that does not compile fails
		// identically on every attempt, so retrying only delays the failure
		// reaching the owner who has to fix it.
		a.recordEmailFailure(ctx, run, schedule, title, result,
			"The report delivery email template could not be rendered: "+err.Error())
		return nil //nolint:nilerr // recorded for the schedule owner; retrying cannot help
	}

	req := &services.SendEmailRequest{
		TenantInfo:     tenantInfo,
		Purpose:        email.PurposeReporting,
		To:             schedule.Delivery.EmailRecipients,
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: "report-run-" + run.ID.String(),
	}
	if attach != nil {
		req.Attachments = []services.EmailAttachment{*attach}
	}

	if _, err = a.email.Send(ctx, req); err != nil {
		if a.isRetryableEmailError(ctx, err) {
			return err
		}
		a.recordEmailFailure(ctx, run, schedule, title, result, err.Error())
		return nil
	}

	result.EmailedRecipients = len(schedule.Delivery.EmailRecipients)
	result.EmailAttached = attach != nil
	if a.metrics != nil {
		a.metrics.RecordDelivery(deliveryChannelEmail, deliveryOutcomeOK)
	}
	return nil
}

// isRetryableEmailError keeps transient provider/database failures on the
// activity retry path while surfacing configuration and validation problems
// (missing profile, bad recipients) to the schedule owner immediately.
func (a *Activities) isRetryableEmailError(ctx context.Context, err error) bool {
	if activity.GetInfo(ctx).Attempt >= deliverMaxAttempts {
		return false
	}
	if errors.Is(err, services.ErrNonRetryableEmailSend) {
		return false
	}
	var (
		businessErr   *errortypes.BusinessError
		validationErr *errortypes.Error
		multiErr      *errortypes.MultiError
		authzErr      *errortypes.AuthorizationError
		notFoundErr   *errortypes.NotFoundError
	)
	if errors.As(err, &businessErr) || errors.As(err, &validationErr) ||
		errors.As(err, &multiErr) || errors.As(err, &authzErr) ||
		errors.As(err, &notFoundErr) {
		return false
	}
	return true
}

func (a *Activities) recordEmailFailure(
	ctx context.Context,
	run *report.ReportRun,
	schedule *report.ReportSchedule,
	title string,
	result *DeliverRunResult,
	reason string,
) {
	result.EmailError = reason
	if a.metrics != nil {
		a.metrics.RecordDelivery(deliveryChannelEmail, deliveryOutcomeFailed)
	}
	a.l.Error("scheduled report email delivery failed",
		zap.String("runId", run.ID.String()),
		zap.String("scheduleId", schedule.ID.String()),
		zap.String("reason", reason))

	correlationID := run.ID.String()
	if exists, err := a.notification.ExistsRecent(ctx,
		repositories.ExistsRecentNotificationRequest{
			OrganizationID: run.OrganizationID,
			BusinessUnitID: run.BusinessUnitID,
			EventType:      deliveryFailedEvent,
			CorrelationID:  correlationID,
			Since:          timeutils.NowUnix() - int64(deliveryDedupeWindow.Seconds()),
		}); err == nil && exists {
		return
	}

	failureWording, ok := a.renderNotification(
		ctx,
		pagination.TenantInfo{
			OrgID:  run.OrganizationID,
			BuID:   run.BusinessUnitID,
			UserID: schedule.RunAsID,
		},
		documenttemplate.KindNotificationReportDeliveryFailed,
		run.ID,
		&documenttemplate.ReportNotificationContext{
			ReportName: title,
			Format:     string(run.Format),
			Reason:     strings.TrimSuffix(strings.TrimSpace(reason), "."),
		},
	)
	if !ok {
		return
	}

	if _, err := a.notification.Create(ctx, &notification.Notification{
		OrganizationID: run.OrganizationID,
		BusinessUnitID: &run.BusinessUnitID,
		TargetUserID:   &schedule.RunAsID,
		Channel:        notification.ChannelUser,
		EventType:      deliveryFailedEvent,
		Priority:       notification.PriorityHigh,
		Title:          failureWording.Title,
		Message:        failureWording.Message,
		Data: map[string]any{
			dataKeyRunID:      run.ID.String(),
			dataKeyScheduleID: schedule.ID.String(),
			dataKeyFormat:     string(run.Format),
			dataKeyReportName: title,
		},
		CorrelationID: &correlationID,
		Source:        deliverySource,
	}); err != nil {
		a.l.Warn("failed to notify schedule owner of email delivery failure",
			zap.String("runId", run.ID.String()), zap.Error(err))
	}
}

func deliveryNotificationData(
	run *report.ReportRun,
	schedule *report.ReportSchedule,
	title string,
) map[string]any {
	data := map[string]any{
		dataKeyRunID:      run.ID.String(),
		dataKeyScheduleID: schedule.ID.String(),
		dataKeyFormat:     string(run.Format),
		dataKeyRowCount:   run.RowCount,
		dataKeyTruncated:  run.Truncated,
		dataKeyReportName: title,
		dataKeyByteSize:   run.ByteSize,
		dataKeyStatus:     string(run.Status),
	}
	if run.ArtifactExpiresAt > 0 {
		data[dataKeyArtifactExpiresAt] = run.ArtifactExpiresAt
	}
	return data
}

func (a *Activities) attachmentPlan(
	run *report.ReportRun,
	schedule *report.ReportSchedule,
	title string,
) (attach *services.EmailAttachment, tooLarge bool) {
	if !schedule.Delivery.EmailAttach {
		return nil, false
	}
	if run.ByteSize <= 0 || run.ByteSize > a.cfg.GetEmailMaxAttachmentBytes() {
		return nil, true
	}
	return &services.EmailAttachment{
		FileName:    deliveryFileName(run, title),
		ContentType: run.Format.ContentType(),
		ObjectKey:   run.ArtifactKey,
		SizeBytes:   run.ByteSize,
	}, false
}

func deliveryFileName(run *report.ReportRun, title string) string {
	stamp := time.Unix(run.CreatedAt, 0).UTC().Format("2006-01-02")
	return fmt.Sprintf(
		"%s %s.%s",
		fileutils.SanitizeDisplayFilename(title, "report", 120),
		stamp,
		run.Format.Extension(),
	)
}

func (a *Activities) deliverRunNotifications(
	ctx context.Context,
	run *report.ReportRun,
	schedule *report.ReportSchedule,
	title string,
	result *DeliverRunResult,
) {
	targets := make([]pulid.ID, 0, len(schedule.Delivery.NotifyUserIDs))
	for _, userID := range sliceutils.Dedupe(schedule.Delivery.NotifyUserIDs) {
		// The schedule owner already receives the standard run-completed
		// notification from finalization.
		if userID != schedule.RunAsID {
			targets = append(targets, userID)
		}
	}
	if len(targets) == 0 {
		return
	}

	users, err := a.userRepo.GetByIDs(ctx, repositories.GetUsersByIDsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: run.OrganizationID,
			BuID:  run.BusinessUnitID,
		},
		UserIDs: targets,
	})
	if err != nil {
		a.l.Warn("failed to resolve in-app delivery recipients",
			zap.String("runId", run.ID.String()), zap.Error(err))
		if a.metrics != nil {
			a.metrics.RecordDelivery(deliveryChannelInApp, deliveryOutcomeFailed)
		}
		return
	}

	// One render for the whole recipient list: it is the same run and the same
	// wording, and rendering per user would resolve the template once per person.
	wording, ok := a.renderNotification(
		ctx,
		pagination.TenantInfo{
			OrgID:  run.OrganizationID,
			BuID:   run.BusinessUnitID,
			UserID: schedule.RunAsID,
		},
		documenttemplate.KindNotificationReportDelivered,
		run.ID,
		deliveredNotificationContext(run, title),
	)
	if !ok {
		return
	}

	since := timeutils.NowUnix() - int64(deliveryDedupeWindow.Seconds())
	for _, user := range users {
		userID := user.ID
		correlationID := run.ID.String() + ":" + userID.String()

		if exists, existsErr := a.notification.ExistsRecent(ctx,
			repositories.ExistsRecentNotificationRequest{
				OrganizationID: run.OrganizationID,
				BusinessUnitID: run.BusinessUnitID,
				EventType:      deliveredEventType,
				CorrelationID:  correlationID,
				Since:          since,
			}); existsErr == nil && exists {
			continue
		}

		if _, createErr := a.notification.Create(ctx, &notification.Notification{
			OrganizationID: run.OrganizationID,
			BusinessUnitID: &run.BusinessUnitID,
			TargetUserID:   &userID,
			Channel:        notification.ChannelUser,
			EventType:      deliveredEventType,
			Priority:       notification.PriorityMedium,
			Title:          wording.Title,
			Message:        wording.Message,
			Data:           deliveryNotificationData(run, schedule, title),
			CorrelationID:  &correlationID,
			Source:         deliverySource,
		}); createErr != nil {
			a.l.Warn("failed to create scheduled report delivery notification",
				zap.String("runId", run.ID.String()),
				zap.String("userId", userID.String()),
				zap.Error(createErr))
			if a.metrics != nil {
				a.metrics.RecordDelivery(deliveryChannelInApp, deliveryOutcomeFailed)
			}
			continue
		}

		result.NotifiedUsers++
		if a.metrics != nil {
			a.metrics.RecordDelivery(deliveryChannelInApp, deliveryOutcomeOK)
		}
	}
}

func (a *Activities) auditDelivery(
	run *report.ReportRun,
	schedule *report.ReportSchedule,
	result *DeliverRunResult,
) {
	if err := a.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceReport,
		ResourceID:     run.ID.String(),
		Operation:      permission.OpExport,
		UserID:         schedule.RunAsID,
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
		CurrentState: map[string]any{
			"event":             "scheduled_delivery",
			dataKeyScheduleID:   schedule.ID.String(),
			dataKeyFormat:       string(run.Format),
			"emailRecipients":   schedule.Delivery.EmailRecipients,
			"emailedRecipients": result.EmailedRecipients,
			"emailAttached":     result.EmailAttached,
			"emailError":        result.EmailError,
			"notifiedUsers":     result.NotifiedUsers,
		},
	}); err != nil {
		a.l.Warn("failed to audit scheduled report delivery",
			zap.String("runId", run.ID.String()), zap.Error(err))
	}
}

func formatInTimezone(unix int64, timezone string) string {
	return time.Unix(unix, 0).In(timezoneOrUTC(timezone)).Format("Jan 2, 2006 at 3:04 PM MST")
}

// timezoneOrUTC resolves a schedule's zone, falling back rather than failing —
// a delivery is not worth losing over an unparseable timezone.
func timezoneOrUTC(timezone string) *time.Location {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}
