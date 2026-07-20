package reportjobs

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

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

	runnerTenant := pagination.TenantInfo{
		OrgID:  run.OrganizationID,
		BuID:   run.BusinessUnitID,
		UserID: run.RequestedByID,
	}
	_, title, _ := a.runDisplayMetadata(ctx, run, runnerTenant)

	result := &DeliverRunResult{}

	if schedule.Delivery.HasEmail() {
		if err = a.deliverRunEmail(ctx, run, schedule, title, result); err != nil {
			return nil, err
		}
	}

	if schedule.Delivery.HasNotify() {
		a.deliverRunNotifications(ctx, run, schedule, title, result)
	}

	a.auditDelivery(run, schedule, result)

	return result, nil
}

func (a *Activities) deliverRunEmail(
	ctx context.Context,
	run *report.ReportRun,
	schedule *report.ReportSchedule,
	title string,
	result *DeliverRunResult,
) error {
	if a.email == nil {
		a.recordEmailFailure(ctx, run, schedule, title, result,
			"Email delivery is not configured on this instance")
		return nil
	}

	attach, attachTooLarge := a.attachmentPlan(run, schedule, title)
	subject := fmt.Sprintf(
		"Scheduled report: %s (%s)", title, strings.ToUpper(string(run.Format)),
	)
	text, htmlBody := a.deliveryEmailBody(run, schedule, title, attach != nil, attachTooLarge)

	req := &services.SendEmailRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  run.OrganizationID,
			BuID:   run.BusinessUnitID,
			UserID: schedule.RunAsID,
		},
		Purpose:        email.PurposeReporting,
		To:             schedule.Delivery.EmailRecipients,
		Subject:        subject,
		HTML:           htmlBody,
		Text:           text,
		IdempotencyKey: "report-run-" + run.ID.String(),
	}
	if attach != nil {
		req.Attachments = []services.EmailAttachment{*attach}
	}

	if _, err := a.email.Send(ctx, req); err != nil {
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

	if _, err := a.notification.Create(ctx, &notification.Notification{
		OrganizationID: run.OrganizationID,
		BusinessUnitID: &run.BusinessUnitID,
		TargetUserID:   &schedule.RunAsID,
		Channel:        notification.ChannelUser,
		EventType:      deliveryFailedEvent,
		Priority:       notification.PriorityHigh,
		Title:          "Scheduled report email failed",
		Message: fmt.Sprintf(
			"The scheduled report could not be emailed to its recipients: %s. "+
				"The report is still available to download from the run history.",
			strings.TrimSuffix(strings.TrimSpace(reason), "."),
		),
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
			Title:          "Scheduled report ready: " + title,
			Message: fmt.Sprintf(
				"The scheduled report %q (%s) is ready to download from the report run history.",
				title, strings.ToUpper(string(run.Format)),
			),
			Data:          deliveryNotificationData(run, schedule, title),
			CorrelationID: &correlationID,
			Source:        deliverySource,
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

func (a *Activities) deliveryEmailBody(
	run *report.ReportRun,
	schedule *report.ReportSchedule,
	title string,
	attached, attachTooLarge bool,
) (text, htmlBody string) {
	generatedAt := formatInTimezone(run.CompletedAt, schedule.Timezone)
	linkURL := a.cfg.GetDeliveryLinkBaseURL()
	if linkURL != "" {
		linkURL += "/reports/runs"
	}

	facts := []string{
		"Format: " + strings.ToUpper(string(run.Format)),
		fmt.Sprintf("Rows: %d", run.RowCount),
		"Size: " + fileutils.HumanizeBytes(run.ByteSize),
		"Generated: " + generatedAt,
	}
	if run.ArtifactExpiresAt > 0 {
		facts = append(facts,
			"Available until: "+formatInTimezone(run.ArtifactExpiresAt, schedule.Timezone))
	}

	var notes []string
	if run.Truncated {
		notes = append(
			notes,
			"The result exceeded the row limit and was truncated — narrow the report's filters to capture everything.",
		)
	}
	switch {
	case attached:
		notes = append(notes, "The report file is attached to this email.")
	case attachTooLarge:
		notes = append(notes, fmt.Sprintf(
			"The report file exceeds the %s attachment limit, so it was not attached.",
			fileutils.HumanizeBytes(a.cfg.GetEmailMaxAttachmentBytes())))
	}
	if linkURL != "" {
		notes = append(notes, "Download it any time from the report run history: "+linkURL)
	} else {
		notes = append(notes,
			"Download it any time from Reports → Run history in Trenova.")
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Your scheduled report %q is ready.\n\n", title)
	sb.WriteString(strings.Join(facts, "\n"))
	sb.WriteString("\n\n")
	sb.WriteString(strings.Join(notes, "\n"))
	text = sb.String()

	var hb strings.Builder
	hb.WriteString(
		`<div style="font-family:-apple-system,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;max-width:560px;margin:0 auto;color:#1a1a1a;">`,
	)
	fmt.Fprintf(&hb,
		`<h2 style="font-size:18px;font-weight:600;margin:24px 0 4px;">%s</h2>`,
		html.EscapeString(title))
	hb.WriteString(`<p style="margin:0 0 16px;color:#666;font-size:13px;">Scheduled report</p>`)
	hb.WriteString(`<table style="border-collapse:collapse;font-size:13px;margin:0 0 16px;">`)
	for _, fact := range facts {
		label, value, _ := strings.Cut(fact, ": ")
		fmt.Fprintf(
			&hb,
			`<tr><td style="padding:4px 16px 4px 0;color:#666;">%s</td><td style="padding:4px 0;">%s</td></tr>`,
			html.EscapeString(label),
			html.EscapeString(value),
		)
	}
	hb.WriteString(`</table>`)
	for _, note := range notes {
		fmt.Fprintf(&hb,
			`<p style="margin:0 0 8px;font-size:13px;color:#444;">%s</p>`,
			html.EscapeString(note))
	}
	if linkURL != "" {
		fmt.Fprintf(
			&hb,
			`<p style="margin:24px 0;"><a href="%s" style="background:#1a1a1a;color:#fff;text-decoration:none;padding:10px 20px;border-radius:6px;font-size:13px;font-weight:500;display:inline-block;">Open run history</a></p>`,
			html.EscapeString(linkURL),
		)
	}
	hb.WriteString(`</div>`)
	htmlBody = hb.String()

	return text, htmlBody
}

func formatInTimezone(unix int64, timezone string) string {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	return time.Unix(unix, 0).In(loc).Format("Jan 2, 2006 at 3:04 PM MST")
}
