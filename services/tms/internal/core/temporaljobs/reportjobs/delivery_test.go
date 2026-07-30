package reportjobs

import (
	"html/template"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deliveryTestActivities(cfg *config.ReportingConfig) *Activities {
	return &Activities{cfg: cfg}
}

func deliveryTestRun() *report.ReportRun {
	return &report.ReportRun{
		ID:                pulid.ID("rrun_test"),
		OrganizationID:    pulid.ID("org_test"),
		BusinessUnitID:    pulid.ID("bu_test"),
		ScheduleID:        pulid.ID("rsch_test"),
		Status:            report.RunStatusSucceeded,
		Format:            report.FormatXLSX,
		ArtifactKey:       "reports/org_test/rrun_test/1/report.xlsx",
		RowCount:          1200,
		ByteSize:          4096,
		CreatedAt:         1752680000,
		CompletedAt:       1752680060,
		ArtifactExpiresAt: 1753284860,
	}
}

func deliveryTestSchedule(delivery *report.ScheduleDelivery) *report.ReportSchedule {
	return &report.ReportSchedule{
		ID:       pulid.ID("rsch_test"),
		RunAsID:  pulid.ID("usr_owner"),
		Timezone: "America/Denver",
		Delivery: delivery,
	}
}

func TestAttachmentPlan(t *testing.T) {
	a := deliveryTestActivities(&config.ReportingConfig{EmailMaxAttachmentBytes: 8192})

	t.Run("disabled", func(t *testing.T) {
		schedule := deliveryTestSchedule(&report.ScheduleDelivery{
			EmailRecipients: []string{"ops@example.com"},
		})
		attach, tooLarge := a.attachmentPlan(deliveryTestRun(), schedule, "Fleet Utilization")
		assert.Nil(t, attach)
		assert.False(t, tooLarge)
	})

	t.Run("within limit", func(t *testing.T) {
		schedule := deliveryTestSchedule(&report.ScheduleDelivery{
			EmailRecipients: []string{"ops@example.com"},
			EmailAttach:     true,
		})
		attach, tooLarge := a.attachmentPlan(deliveryTestRun(), schedule, "Fleet Utilization")
		require.NotNil(t, attach)
		assert.False(t, tooLarge)
		assert.Equal(t, "reports/org_test/rrun_test/1/report.xlsx", attach.ObjectKey)
		assert.Equal(t, int64(4096), attach.SizeBytes)
		assert.Equal(t, report.FormatXLSX.ContentType(), attach.ContentType)
		assert.True(t, strings.HasPrefix(attach.FileName, "Fleet Utilization "))
		assert.True(t, strings.HasSuffix(attach.FileName, ".xlsx"))
	})

	t.Run("over limit", func(t *testing.T) {
		schedule := deliveryTestSchedule(&report.ScheduleDelivery{
			EmailRecipients: []string{"ops@example.com"},
			EmailAttach:     true,
		})
		run := deliveryTestRun()
		run.ByteSize = 10_000
		attach, tooLarge := a.attachmentPlan(run, schedule, "Fleet Utilization")
		assert.Nil(t, attach)
		assert.True(t, tooLarge)
	})
}

func TestDeliveryFileNameSanitizesTitle(t *testing.T) {
	name := deliveryFileName(deliveryTestRun(), "Q3: Fleet/Utilization?")
	assert.NotContains(t, name, "/")
	assert.NotContains(t, name, ":")
	assert.NotContains(t, name, "?")
	assert.True(t, strings.HasSuffix(name, ".xlsx"))
}

// The message the recipient acts on is now the template's, so what this asserts
// is the context: every fact the wording depends on has to be there and be right,
// because a template author cannot compute the ones that are missing.
func TestDeliveryEmailContext(t *testing.T) {
	t.Run("attached, with a download link", func(t *testing.T) {
		a := deliveryTestActivities(&config.ReportingConfig{
			DeliveryLinkBaseURL: "https://app.example.com/",
		})
		schedule := deliveryTestSchedule(&report.ScheduleDelivery{
			EmailRecipients: []string{"ops@example.com"},
			EmailAttach:     true,
		})

		out := a.deliveryEmailContext(&deliveryEmailContent{
			run:      deliveryTestRun(),
			schedule: schedule,
			title:    "Fleet <Utilization>",
			attached: true,
		})

		// The title is carried verbatim; escaping it here would double-escape it
		// once html/template does its job.
		assert.Equal(t, "Fleet <Utilization>", out.Title)
		assert.Equal(t, "xlsx", out.Format)
		assert.Equal(t, int64(1200), out.RowCount)
		assert.NotEmpty(t, out.FileSize)
		assert.Contains(t, out.CompletedAt, "MDT", "the schedule's zone, not UTC")
		assert.True(t, out.Attached)
		assert.Contains(t, out.AttachedNote, "attached to this email")
		assert.Equal(
			t,
			template.URL("https://app.example.com/reports/runs"),
			out.DownloadURL,
		)
		assert.Contains(t, out.ExpiresAt, "MDT")
	})

	t.Run("too large to attach, and nowhere to link to", func(t *testing.T) {
		a := deliveryTestActivities(&config.ReportingConfig{EmailMaxAttachmentBytes: 8192})
		schedule := deliveryTestSchedule(&report.ScheduleDelivery{
			EmailRecipients: []string{"ops@example.com"},
			EmailAttach:     true,
		})

		out := a.deliveryEmailContext(&deliveryEmailContent{
			run:            deliveryTestRun(),
			schedule:       schedule,
			title:          "Fleet Utilization",
			attachTooLarge: true,
		})

		assert.False(t, out.Attached)
		assert.Contains(t, out.AttachedNote, "exceeds the")
		assert.Contains(t, out.AttachedNote, "attachment limit")
		assert.Empty(t, out.DownloadURL,
			"an unconfigured base URL must not become a link to nowhere")
	})

	t.Run("neither attached nor too large says where the file is", func(t *testing.T) {
		a := deliveryTestActivities(&config.ReportingConfig{})
		schedule := deliveryTestSchedule(&report.ScheduleDelivery{
			EmailRecipients: []string{"ops@example.com"},
		})

		out := a.deliveryEmailContext(&deliveryEmailContent{
			run:      deliveryTestRun(),
			schedule: schedule,
			title:    "Fleet Utilization",
		})

		assert.Contains(t, out.AttachedNote, "run history")
	})

	t.Run("a truncated run says so", func(t *testing.T) {
		a := deliveryTestActivities(&config.ReportingConfig{})
		schedule := deliveryTestSchedule(&report.ScheduleDelivery{
			EmailRecipients: []string{"ops@example.com"},
		})
		run := deliveryTestRun()
		run.Truncated = true

		out := a.deliveryEmailContext(&deliveryEmailContent{
			run:      run,
			schedule: schedule,
			title:    "Fleet Utilization",
		})

		assert.True(t, out.Truncated,
			"the template warns on this flag; losing it hides that rows were dropped")
	})
}

func TestFormatInTimezone(t *testing.T) {
	assert.Contains(t, formatInTimezone(1752680060, "America/Denver"), "MDT")
	assert.Contains(t, formatInTimezone(1752680060, "not-a-zone"), "UTC")
}
