package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSenders struct {
	suppressed []string
}

func (f fakeSenders) ResolveSender(
	context.Context,
	*serviceports.SendEmailRequest,
) (*serviceports.EmailSender, error) {
	return &serviceports.EmailSender{
		Email:      "ops@carrier.example",
		Name:       "Carrier Operations",
		Suppressed: f.suppressed,
	}, nil
}

// assertMessageIsTheSend holds a preview's message to what the write then
// handed the mailer.
func assertMessageIsTheSend(
	t *testing.T,
	message *agent.MessagePreview,
	sent *serviceports.SendEmailRequest,
) {
	t.Helper()

	require.NotNil(t, message)
	require.NotNil(t, sent)
	assert.Equal(t, agent.MessageChannelEmail, message.Channel)
	assert.Equal(t, sent.To, message.To)
	assert.Equal(t, sent.Subject, message.Subject)
	assert.Equal(t, sent.Text, message.Body)
	assert.Equal(t, "Carrier Operations <ops@carrier.example>", message.From)
}

func TestEmailCustomerPreview_IsTheEmailExecuteSends(t *testing.T) {
	t.Parallel()

	tool, mailer, _, comments, _ := customerEmailFixture()
	tool.deps.senders = fakeSenders{}
	params := executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "S12345 running about an hour late",
		"body":       "The truck is held in traffic; we now expect 3:30 PM.",
	})
	params.IdempotencyKey = "idem-1"

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, mailer.sent, "a preview must not send")
	require.Nil(t, comments.created, "a preview must not record")
	assert.Empty(t, preview.Warnings)

	send := findChange(t, preview, agent.PreviewOperationSend)
	assert.Equal(t, permission.ResourceCustomer, send.Resource)
	assert.Equal(t, "Acme Freight", send.Label)
	recorded := findChange(t, preview, agent.PreviewOperationCreate)
	assert.Equal(t, permission.ResourceShipmentComment, recorded.Resource)

	require.NoError(t, tool.Execute(t.Context(), params))
	assertMessageIsTheSend(t, send.Message, mailer.sent)
	assert.Equal(t, []string{"ops@acme.example", "ap@acme.example"}, send.Message.To)
	require.NotNil(t, comments.created)
	assert.Equal(t, comments.created.Comment, findField(t, recorded, "comment").After)
}

func TestEmailCustomerPreview_SaysNothingGoesToACustomerAlreadyTold(t *testing.T) {
	t.Parallel()

	tool, mailer, _, comments, renderer := customerEmailFixture()
	comments.existing = []*shipment.ShipmentComment{{
		CreatedAt: timeutils.NowUnix() - 600,
		Type:      shipment.CommentTypeCustomerUpdate,
		Metadata:  map[string]any{"source": "agent", "tool": "email_customer"},
	}}
	params := executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "S12345 has departed",
		"body":       "The truck left the yard.",
	})
	params.IdempotencyKey = "idem-2"

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Empty(t, preview.Changes)
	require.Len(t, preview.Warnings, 1)
	assert.Equal(t, agent.PreviewWarningAlreadyToldCustomer, preview.Warnings[0].Code)
	assert.Nil(t, renderer.req, "nothing is rendered for a send that will not happen")

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrCustomerAlreadyTold)
	assert.Nil(t, mailer.sent)
}

func TestEmailCustomerPreview_WarnsOfSuppressedRecipientsAndSensitiveText(t *testing.T) {
	t.Parallel()

	tool, _, _, _, _ := customerEmailFixture()
	tool.deps.senders = fakeSenders{suppressed: []string{"ap@acme.example"}}
	params := executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "Remittance details",
		"body":       "Please remit to account 123456789 0001234567.",
	})
	params.IdempotencyKey = "idem-3"

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)

	codes := make([]agent.PreviewWarningCode, 0, len(preview.Warnings))
	for _, warning := range preview.Warnings {
		codes = append(codes, warning.Code)
	}
	assert.ElementsMatch(t, []agent.PreviewWarningCode{
		agent.PreviewWarningWouldFail,
		agent.PreviewWarningSensitiveContent,
	}, codes)
}

func TestRequestMissingDocsPreview_IsTheEmailExecuteSends(t *testing.T) {
	t.Parallel()

	mailer := &fakeMailer{}
	tool := &requestMissingDocsTool{
		email:     mailer,
		templates: &fakeRenderer{},
		senders:   fakeSenders{},
	}
	params := executeParams(map[string]any{
		"profileId":          pulid.MustNew("emp_").String(),
		"to":                 []any{"pod@consignee.example"},
		"subject":            "Missing POD for SHP-1001",
		"body":               "We still need the signed delivery receipt.",
		"customerName":       "Halstead Grocery Group",
		"shipmentProNumber":  "SHP-1001",
		"requestedDocuments": []any{"Signed POD"},
	})
	params.IdempotencyKey = "idem-4"

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, mailer.sent, "a preview must not send")
	assert.Contains(t, preview.Summary, "Signed POD")
	assert.Contains(t, preview.Summary, "SHP-1001")

	send := findChange(t, preview, agent.PreviewOperationSend)
	assert.Equal(t, "Halstead Grocery Group", send.Label)

	require.NoError(t, tool.Execute(t.Context(), params))
	assertMessageIsTheSend(t, send.Message, mailer.sent)
}

func TestReplyToInboundMessagePreview_IsTheReplyExecuteSends(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{message: inboundMessage()}
	mailer := &fakeMailer{}
	tool := &replyToInboundMessageTool{
		inbox: desk,
		deps:  inboundReplier{email: mailer, templates: &fakeRenderer{}, senders: fakeSenders{}},
	}
	params := deskParams(map[string]any{
		"messageId": desk.message.ID.String(),
		"profileId": pulid.MustNew("emp_").String(),
		"body":      "It delivers tomorrow before noon.",
	})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, mailer.sent, "a preview must not send")
	require.Nil(t, desk.reviewed, "a preview must not settle the message")
	assert.Equal(t, inboundmessage.StatusInReview, desk.message.Status,
		"a preview must not touch the message it read")

	send := findChange(t, preview, agent.PreviewOperationSend)
	assert.Equal(t, []string{"dana@shipper.example"}, send.Message.To)
	settled := findChange(t, preview, agent.PreviewOperationUpdate)
	assert.Equal(t, "Actioned", findField(t, settled, "status").After)
	note := findField(t, settled, "reviewNote")

	require.NoError(t, tool.Execute(t.Context(), params))
	assertMessageIsTheSend(t, send.Message, mailer.sent)
	require.NotNil(t, desk.reviewed)
	assert.Equal(t, desk.reviewed.Note, note.After)
}
