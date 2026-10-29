package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticerender"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticeservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
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
		PolicySnapshot:     &detentionpolicy.PolicySnapshot{Currency: "USD"},
		LocationName:       "Memphis DC",
		ShipmentProNumber:  "SHP-1001",
		CustomerName:       "Acme Foods",
		Version:            4,
	}
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
	require.NoError(
		t,
		after.Waive(detention.WaiverReasonWeather, "Ice storm", pulid.MustNew("usr_"), 10),
	)

	det := &agenttoolstest.FakeDetention{Change: &detentionreview.OccurrenceChange{
		Before:   before,
		After:    &after,
		Shipment: []*detention.DetentionOccurrence{before, sibling},
	}}
	tool := operationstools.NewWaiveDetentionTool(det)
	params := agenttoolstest.ApprovedParams(map[string]any{
		"occurrenceId": before.ID.String(),
		"reason":       "Weather",
		"note":         "Ice storm",
	})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, det.Waived, "a preview must not waive")
	assert.Contains(t, preview.Summary, "425.00 USD")
	assert.Contains(t, preview.Summary, "at Memphis DC on SHP-1001")

	change := agenttoolstest.FindChange(t, preview, agentpreview.PreviewOperationUpdate)
	assert.Equal(t, permission.ResourceDetention, change.Resource)
	assert.Equal(t, before.ID, change.EntityID)
	assert.Equal(t, int64(4), *change.Version)
	status := agenttoolstest.FindField(t, change, "status")
	assert.Equal(t, "Pending", status.Before)
	assert.Equal(t, "Waived", status.After)
	assert.True(t, agenttoolstest.FindField(t, change, "waivedAt").Volatile)

	require.NotNil(t, change.Money)
	assert.Equal(t, "USD", change.Money.Currency)
	require.Len(t, change.Money.Lines, 2)
	assert.True(t, change.Money.TotalBefore.Decimal.Equal(decimal.NewFromInt(525)))
	assert.True(t, change.Money.TotalAfter.Decimal.Equal(decimal.NewFromInt(100)))
	assert.True(t, change.Money.Delta.Decimal.Equal(decimal.NewFromInt(-425)))

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, *det.PreviewedWaive, *det.Waived,
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

	det := &agenttoolstest.FakeDetention{Notice: &detentionnoticeservice.NoticePreview{
		Before:     before,
		After:      &after,
		Kind:       detentionnotice.NoticeKindStarted,
		Recipients: []string{"ap@acme.test"},
		Content: detentionnoticerender.NoticeContent{
			Subject:   "Detention charges started",
			Text:      "Detention began at 09:00.",
			VersionID: &versionID,
		},
		Attachment: "detention-notice-SHP-1001.pdf",
		Sender: &emailports.EmailSender{
			Email:      "notices@carrier.test",
			Name:       "Carrier Billing",
			Suppressed: []string{"ap@acme.test"},
			Refused:    true,
		},
	}}
	tool := operationstools.NewSendDetentionNoticeTool(det)
	params := agenttoolstest.ExecuteParams(map[string]any{"occurrenceId": before.ID.String()})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, det.Noticed, "a preview must not send")

	send := agenttoolstest.FindChange(t, preview, agentpreview.PreviewOperationSend)
	require.NotNil(t, send.Message)
	assert.Equal(t, agentpreview.MessageChannelEmail, send.Message.Channel)
	assert.Equal(t, "Carrier Billing <notices@carrier.test>", send.Message.From)
	assert.Equal(t, []string{"ap@acme.test"}, send.Message.To)
	assert.Equal(t, "Detention charges started", send.Message.Subject)
	assert.Equal(t, "Detention began at 09:00.", send.Message.Body)
	assert.Equal(t, []string{"detention-notice-SHP-1001.pdf"}, send.Message.Attachments)
	assert.Equal(t, versionID, send.Message.TemplateVersionID)
	assert.Equal(t, permission.ResourceCustomer, send.Resource)
	assert.Equal(t, "Acme Foods", send.Label)

	status := agenttoolstest.FindChange(t, preview, agentpreview.PreviewOperationUpdate)
	assert.Equal(t, "Sent", agenttoolstest.FindField(t, status, "notificationStatus").After)
	require.Len(t, preview.Warnings, 1)
	assert.Equal(t, agentpreview.PreviewWarningWouldFail, preview.Warnings[0].Code)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, *det.PreviewedNotice, *det.Noticed)
}
