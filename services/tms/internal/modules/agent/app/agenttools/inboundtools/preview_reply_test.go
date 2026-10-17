package inboundtools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplyToInboundMessagePreview_IsTheReplyExecuteSends(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{message: inboundMessage()}
	mailer := &agenttoolstest.FakeMailer{}
	tool := &ReplyToInboundMessageTool{
		inbox: desk,
		deps: inboundReplier{
			email:     mailer,
			templates: &agenttoolstest.FakeRenderer{},
			senders:   agenttoolstest.FakeSenders{},
		},
	}
	params := agenttoolstest.DeskParams(map[string]any{
		"messageId": desk.message.ID.String(),
		"profileId": pulid.MustNew("emp_").String(),
		"body":      "It delivers tomorrow before noon.",
	})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, mailer.Sent, "a preview must not send")
	require.Nil(t, desk.reviewed, "a preview must not settle the message")
	assert.Equal(t, inboundmessage.StatusInReview, desk.message.Status,
		"a preview must not touch the message it read")

	send := agenttoolstest.FindChange(t, preview, agent.PreviewOperationSend)
	assert.Equal(t, []string{"dana@shipper.example"}, send.Message.To)
	settled := agenttoolstest.FindChange(t, preview, agent.PreviewOperationUpdate)
	assert.Equal(t, "Actioned", agenttoolstest.FindField(t, settled, "status").After)
	note := agenttoolstest.FindField(t, settled, "reviewNote")

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.AssertMessageIsTheSend(t, send.Message, mailer.Sent)
	require.NotNil(t, desk.reviewed)
	assert.Equal(t, desk.reviewed.Note, note.After)
}
