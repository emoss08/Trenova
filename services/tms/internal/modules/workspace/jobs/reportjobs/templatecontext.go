package reportjobs

import (
	"context"
	"fmt"
	"html/template"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// deliveryEmailContent groups what the message needs; the parameter list had
// outgrown a readable signature.
type deliveryEmailContent struct {
	run            *report.ReportRun
	schedule       *report.ReportSchedule
	title          string
	digest         *reportports.ReportDigest
	attached       bool
	attachTooLarge bool
}

// deliveryEmailContext maps a finished run onto the variables the report
// delivery template declares.
//
// The layout, the wording, and the digest table used to be two strings.Builders
// in delivery.go — one for text, one for inline-styled HTML — which meant an
// organization could not change a word of a message their own people receive on a
// schedule. The numbers still come from the column display contracts, so the
// email, the grid, and the exports still agree cell for cell.
func (a *Activities) deliveryEmailContext(
	ctx context.Context,
	content *deliveryEmailContent,
) (documenttemplate.ReportDeliveryEmailContext, error) {
	run, schedule := content.run, content.schedule
	loc := timezoneOrUTC(schedule.Timezone)

	digest, err := digestContext(
		ctx,
		content.digest,
		loc,
		i18n.FromContext(ctx).String(),
		run.RowCount,
	)
	if err != nil {
		return documenttemplate.ReportDeliveryEmailContext{}, err
	}

	out := documenttemplate.ReportDeliveryEmailContext{
		Title:        content.title,
		Format:       string(run.Format),
		CompletedAt:  formatInTimezone(run.CompletedAt, schedule.Timezone),
		RowCount:     run.RowCount,
		FileSize:     fileutils.HumanizeBytes(run.ByteSize),
		Truncated:    run.Truncated,
		Attached:     content.attached,
		AttachedNote: a.attachmentNote(content),
		Digest:       digest,
	}

	if schedule.ReportDefinition != nil {
		out.Description = schedule.ReportDefinition.Description
	}
	if schedule.RunAs != nil {
		out.RequestedBy = schedule.RunAs.Name
	}
	if run.ArtifactExpiresAt > 0 {
		out.ExpiresAt = formatInTimezone(run.ArtifactExpiresAt, schedule.Timezone)
	}

	// The link is built from configured deployment settings, never from anything
	// a user typed, which is why the context types it as a template.URL. An
	// unconfigured base URL leaves it empty and the template drops the button
	// rather than linking nowhere.
	if base := a.cfg.GetDeliveryLinkBaseURL(); base != "" {
		//nolint:gosec // G203: configured base URL, not user input.
		out.DownloadURL = template.URL(base + "/reports/runs")
	}

	return out, nil
}

// attachmentNote says where the file is, which is the one sentence a recipient
// actually acts on.
//
// A reader who is told to look for an attachment that was dropped for size will
// go looking for it, so the size limit is named rather than implied.
func (a *Activities) attachmentNote(content *deliveryEmailContent) string {
	switch {
	case content.attached:
		return "The report file is attached to this email."
	case content.attachTooLarge:
		return fmt.Sprintf(
			"The report file exceeds the %s attachment limit, so it was not attached.",
			fileutils.HumanizeBytes(a.cfg.GetEmailMaxAttachmentBytes()),
		)
	default:
		return "The report file is available from the run history in Trenova."
	}
}

// notificationWording is a rendered notification: a title and a message.
type notificationWording struct {
	Title   string
	Message string
}

// renderNotification renders one report notification through the organization's
// template.
//
// It falls back to the built-in, like every notification: an owner who is never
// told their schedule was disabled will find out from a customer instead. A
// failure that even the built-in cannot survive returns false and the caller skips
// the notification rather than writing one with an empty title.
func (a *Activities) renderNotification(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	kind documenttemplate.Kind,
	referenceID pulid.ID,
	data *documenttemplate.ReportNotificationContext,
) (notificationWording, bool) {
	if a.templates == nil {
		a.l.Warn("template rendering is not configured; skipping report notification",
			zap.String("kind", string(kind)))
		return notificationWording{}, false
	}

	rendered, err := a.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo:        tenantInfo,
		Kind:              kind,
		Data:              data,
		ReferenceID:       referenceID,
		FallbackToBuiltIn: true,
	})
	if err != nil {
		a.l.Warn("failed to render report notification",
			zap.String("kind", string(kind)), zap.Error(err))
		return notificationWording{}, false
	}

	return notificationWording{Title: rendered.Subject, Message: rendered.Text}, true
}

// runNotificationContext describes a finished run to its notification template.
func runNotificationContext(
	run *report.ReportRun,
	name string,
) *documenttemplate.ReportNotificationContext {
	out := &documenttemplate.ReportNotificationContext{
		ReportName: name,
		Format:     string(run.Format),
		RowCount:   run.RowCount,
		FileSize:   fileutils.HumanizeBytes(run.ByteSize),
		Truncated:  run.Truncated,
	}
	if run.Error != nil {
		out.Reason = run.Error.Message
	}

	return out
}

// deliveredNotificationContext describes a delivered scheduled report.
//
// The artifact expiry is included because a recipient who reads the notification
// a week later needs to know the download is gone rather than broken.
func deliveredNotificationContext(
	run *report.ReportRun,
	title string,
) *documenttemplate.ReportNotificationContext {
	out := &documenttemplate.ReportNotificationContext{
		ReportName: title,
		Format:     string(run.Format),
		RowCount:   run.RowCount,
		FileSize:   fileutils.HumanizeBytes(run.ByteSize),
		Truncated:  run.Truncated,
	}
	if run.ArtifactExpiresAt > 0 {
		out.ExpiresAt = timeutils.FormatUnixDateIn(run.ArtifactExpiresAt, "")
	}

	return out
}
