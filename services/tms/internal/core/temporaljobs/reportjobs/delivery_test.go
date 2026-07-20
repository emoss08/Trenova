package reportjobs

import (
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

func TestDeliveryEmailBody(t *testing.T) {
	t.Run("with link and attachment", func(t *testing.T) {
		a := deliveryTestActivities(&config.ReportingConfig{
			DeliveryLinkBaseURL: "https://app.example.com/",
		})
		schedule := deliveryTestSchedule(&report.ScheduleDelivery{
			EmailRecipients: []string{"ops@example.com"},
			EmailAttach:     true,
		})

		text, htmlBody := a.deliveryEmailBody(
			deliveryTestRun(), schedule, "Fleet <Utilization>", true, false,
		)

		assert.Contains(t, text, `"Fleet <Utilization>" is ready`)
		assert.Contains(t, text, "Format: XLSX")
		assert.Contains(t, text, "Rows: 1200")
		assert.Contains(t, text, "attached to this email")
		assert.Contains(t, text, "https://app.example.com/reports/runs")

		assert.Contains(t, htmlBody, "Fleet &lt;Utilization&gt;")
		assert.NotContains(t, htmlBody, "Fleet <Utilization>")
		assert.Contains(t, htmlBody, `href="https://app.example.com/reports/runs"`)
	})

	t.Run("attachment too large without link", func(t *testing.T) {
		a := deliveryTestActivities(&config.ReportingConfig{})
		schedule := deliveryTestSchedule(&report.ScheduleDelivery{
			EmailRecipients: []string{"ops@example.com"},
			EmailAttach:     true,
		})

		text, htmlBody := a.deliveryEmailBody(
			deliveryTestRun(), schedule, "Fleet Utilization", false, true,
		)

		assert.Contains(t, text, "exceeds the")
		assert.Contains(t, text, "attachment limit")
		assert.Contains(t, text, "Reports → Run history")
		assert.NotContains(t, htmlBody, "href=")
	})

	t.Run("truncated run carries a warning", func(t *testing.T) {
		a := deliveryTestActivities(&config.ReportingConfig{})
		schedule := deliveryTestSchedule(&report.ScheduleDelivery{
			EmailRecipients: []string{"ops@example.com"},
		})
		run := deliveryTestRun()
		run.Truncated = true

		text, _ := a.deliveryEmailBody(run, schedule, "Fleet Utilization", false, false)
		assert.Contains(t, text, "truncated")
	})
}

func TestFormatInTimezone(t *testing.T) {
	assert.Contains(t, formatInTimezone(1752680060, "America/Denver"), "MDT")
	assert.Contains(t, formatInTimezone(1752680060, "not-a-zone"), "UTC")
}
