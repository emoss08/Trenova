package agenttoolservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
)

var _ serviceports.ToolPreviewer = (*scheduleReportTool)(nil)

func (t *scheduleReportTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	planned, err := t.schedules.PreviewSchedule(ctx, request)
	if err != nil {
		return nil, err
	}
	schedule, definition := planned.Schedule, planned.Definition

	created, err := toolpreview.Create(
		toolpreview.Record{
			Resource: permission.ResourceReport,
			ID:       definition.ID,
			Label:    "Schedule for " + definition.Name,
		},
		schedule,
		toolpreview.Only(
			previewFieldDefinitionID,
			"cronExpression",
			"timezone",
			"formats",
			"enabled",
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			previewFieldDefinitionID: permission.ResourceReport,
		}),
	)
	if err != nil {
		return nil, err
	}

	var recipients []string
	if schedule.Delivery != nil {
		recipients = schedule.Delivery.EmailRecipients
	}
	send := toolpreview.Send(
		toolpreview.Record{
			Resource: permission.ResourceReport,
			ID:       definition.ID,
			Label:    definition.Name,
		},
		&agent.MessagePreview{
			Channel: agent.MessageChannelEmail,
			To:      recipients,
			Subject: definition.Name,
			Body:    scheduledReportBody(schedule, definition),
			Cadence: scheduleCadence(schedule),
		},
	)

	return toolpreview.Build(
		fmt.Sprintf("Would email the %s report to %s on the schedule %s, first on %s.",
			definition.Name,
			strings.Join(recipients, ", "),
			scheduleCadence(schedule),
			time.Unix(schedule.NextRunAt, 0).UTC().Format("2006-01-02 15:04 MST")),
		created,
		send,
	), nil
}

func scheduledReportBody(
	schedule *report.ReportSchedule,
	definition *report.ReportDefinition,
) string {
	formats := strings.ToUpper(strings.Join(schedule.Formats, ", "))
	if schedule.Delivery != nil && schedule.Delivery.EmailAttach {
		return fmt.Sprintf("The latest %s report, attached as %s.", definition.Name, formats)
	}

	return fmt.Sprintf("A link to the latest %s report, as %s.", definition.Name, formats)
}

func scheduleCadence(schedule *report.ReportSchedule) string {
	return fmt.Sprintf("%q (%s)", schedule.CronExpression, schedule.Timezone)
}
