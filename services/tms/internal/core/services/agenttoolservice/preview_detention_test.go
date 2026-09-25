package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func previewedCharge() *detention.DetentionOccurrence {
	return &detention.DetentionOccurrence{
		ID:                 pulid.MustNew("dto_"),
		CustomerID:         pulid.MustNew("cus_"),
		Status:             detention.OccurrenceStatusPending,
		RequiresApproval:   true,
		BillableAmount:     decimal.NewFromInt(425),
		Currency:           "USD",
		NotificationStatus: detention.NotificationStatusPending,
		PolicySnapshot:     &detention.PolicySnapshot{Currency: "USD"},
		LocationName:       "Memphis DC",
		ShipmentProNumber:  "SHP-1001",
		CustomerName:       "Acme Foods",
		Version:            4,
	}
}

func findChange(
	t *testing.T,
	preview *agent.ToolPreview,
	operation agent.PreviewOperation,
) *agent.RecordChange {
	t.Helper()

	for i := range preview.Changes {
		if preview.Changes[i].Operation == operation {
			return &preview.Changes[i]
		}
	}
	require.Failf(t, "no change", "the preview has no %s change", operation)

	return nil
}

func findField(t *testing.T, change *agent.RecordChange, path string) agent.PreviewFieldChange {
	t.Helper()

	for _, field := range change.Fields {
		if field.Path == path {
			return field
		}
	}
	require.Failf(t, "no field", "the change has no %q field", path)

	return agent.PreviewFieldChange{}
}

func TestWaiveDetentionPreview_ShowsTheRevenueGivenUp(t *testing.T) {
	t.Parallel()

	before := previewedCharge()
	sibling := previewedCharge()
	sibling.ID = pulid.MustNew("dto_")
	sibling.LocationName = "Dallas DC"
	sibling.BillableAmount = decimal.NewFromInt(100)
	sibling.Status = detention.OccurrenceStatusApproved
	after := *before
	require.NoError(t, after.Waive(detention.WaiverReasonWeather, "Ice storm", pulid.MustNew("usr_"), 10))

	det := &fakeDetention{change: &detentionservice.OccurrenceChange{
		Before:   before,
		After:    &after,
		Shipment: []*detention.DetentionOccurrence{before, sibling},
	}}
	tool := newWaiveDetentionTool(det)
	params := approvedParams(map[string]any{
		"occurrenceId": before.ID.String(),
		"reason":       "Weather",
		"note":         "Ice storm",
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, det.waived, "a preview must not waive")
	assert.Contains(t, preview.Summary, "425.00 USD")
	assert.Contains(t, preview.Summary, "at Memphis DC on SHP-1001")

	change := findChange(t, preview, agent.PreviewOperationUpdate)
	assert.Equal(t, permission.ResourceDetentionPolicy, change.Resource)
	assert.Equal(t, before.ID, change.EntityID)
	assert.Equal(t, int64(4), *change.Version)
	status := findField(t, change, "status")
	assert.Equal(t, "Pending", status.Before)
	assert.Equal(t, "Waived", status.After)
	assert.True(t, findField(t, change, "waivedAt").Volatile)

	require.NotNil(t, change.Money)
	assert.Equal(t, "USD", change.Money.Currency)
	require.Len(t, change.Money.Lines, 2)
	assert.True(t, change.Money.TotalBefore.Decimal.Equal(decimal.NewFromInt(525)))
	assert.True(t, change.Money.TotalAfter.Decimal.Equal(decimal.NewFromInt(100)))
	assert.True(t, change.Money.Delta.Decimal.Equal(decimal.NewFromInt(-425)))

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, *det.previewedWaive, *det.waived,
		"the preview and the write must make the same waiver")
}

func TestSendDetentionNoticePreview_ShowsTheNoticeAsSent(t *testing.T) {
	t.Parallel()

	before := previewedCharge()
	after := *before
	sentAt := int64(1_767_230_000)
	after.NotificationStatus = detention.NotificationStatusSent
	after.NoticeSentAt = &sentAt
	versionID := pulid.MustNew("dtv_")

	det := &fakeDetention{notice: &detentionservice.NoticePreview{
		Before:     before,
		After:      &after,
		Kind:       detention.NoticeKindStarted,
		Recipients: []string{"ap@acme.test"},
		Content: detentionservice.NoticeContent{
			Subject:   "Detention charges started",
			Text:      "Detention began at 09:00.",
			VersionID: &versionID,
		},
		Attachment: "detention-notice-SHP-1001.pdf",
		Sender: &serviceports.EmailSender{
			Email:      "notices@carrier.test",
			Name:       "Carrier Billing",
			Suppressed: []string{"ap@acme.test"},
		},
	}}
	tool := newSendDetentionNoticeTool(det)
	params := executeParams(map[string]any{"occurrenceId": before.ID.String()})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, det.noticed, "a preview must not send")

	send := findChange(t, preview, agent.PreviewOperationSend)
	require.NotNil(t, send.Message)
	assert.Equal(t, agent.MessageChannelEmail, send.Message.Channel)
	assert.Equal(t, "Carrier Billing <notices@carrier.test>", send.Message.From)
	assert.Equal(t, []string{"ap@acme.test"}, send.Message.To)
	assert.Equal(t, "Detention charges started", send.Message.Subject)
	assert.Equal(t, "Detention began at 09:00.", send.Message.Body)
	assert.Equal(t, []string{"detention-notice-SHP-1001.pdf"}, send.Message.Attachments)
	assert.Equal(t, versionID, send.Message.TemplateVersionID)
	assert.Equal(t, permission.ResourceCustomer, send.Resource)
	assert.Equal(t, "Acme Foods", send.Label)

	status := findChange(t, preview, agent.PreviewOperationUpdate)
	assert.Equal(t, "Sent", findField(t, status, "notificationStatus").After)
	require.Len(t, preview.Warnings, 1)
	assert.Equal(t, agent.PreviewWarningWouldFail, preview.Warnings[0].Code)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, *det.previewedNotice, *det.noticed)
}
