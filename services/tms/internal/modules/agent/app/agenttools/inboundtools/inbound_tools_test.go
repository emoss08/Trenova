package inboundtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInboundDesk struct {
	message   *inboundmessage.InboundMessage
	linkErr   error
	checked   *inboundmessageservice.LinkRequest
	linked    *inboundmessageservice.LinkRequest
	reviewed  *inboundmessageservice.ReviewRequest
	requested *inboundmessageports.GetInboundMessageByIDRequest
}

func (f *fakeInboundDesk) GetByID(
	_ context.Context,
	req inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	f.requested = &req
	if f.message == nil {
		return nil, errortypes.NewNotFoundError("InboundMessage not found")
	}

	return f.message, nil
}

func (f *fakeInboundDesk) CheckLink(
	_ context.Context,
	req inboundmessageservice.LinkRequest,
) error {
	f.checked = &req
	return f.linkErr
}

func (f *fakeInboundDesk) Link(
	_ context.Context,
	req inboundmessageservice.LinkRequest,
) (*inboundmessage.InboundMessage, error) {
	f.linked = &req
	return f.message, f.linkErr
}

func (f *fakeInboundDesk) Review(
	_ context.Context,
	req inboundmessageservice.ReviewRequest,
) (*inboundmessage.InboundMessage, error) {
	f.reviewed = &req
	return f.message, nil
}

func inboundMessage() *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:          pulid.MustNew("imsg_"),
		Status:      inboundmessage.StatusInReview,
		FromAddress: "dana@shipper.example",
		Subject:     "Where is 88213?",
		MessageID:   "<abc@shipper.example>",
		References:  []string{"<first@shipper.example>"},
		Mailbox:     &inboundmessage.Mailbox{Address: "loads@carrier.example"},
	}
}

func TestLinkInboundMessage_ChecksBeforeProposingAndLinksInTheActorsTenant(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{message: inboundMessage()}
	tool := NewLinkInboundMessageTool(desk)
	shipmentID := pulid.MustNew("shp_")
	params := agenttoolstest.DeskParams(map[string]any{
		"messageId":  desk.message.ID.String(),
		"shipmentId": shipmentID.String(),
		"reason":     "The PRO in the subject is this load's.",
	})

	require.NoError(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	require.NotNil(t, desk.checked)
	assert.Equal(t, shipmentID, desk.checked.ShipmentID)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, desk.linked)
	assert.Equal(t, params.OrganizationID, desk.linked.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, desk.linked.TenantInfo.BuID)
	assert.Equal(t, shipmentID, desk.linked.ShipmentID)
	assert.True(t, desk.linked.CustomerID.IsNil())
}

func TestLinkInboundMessage_RefusesARecordTheServiceCannotFind(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{
		message: inboundMessage(),
		linkErr: errortypes.NewValidationError("shipmentId", errortypes.ErrInvalid, "No shipment"),
	}
	tool := NewLinkInboundMessageTool(desk)

	err := tool.(agenttoolports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.DeskParams(map[string]any{
			"messageId":  desk.message.ID.String(),
			"shipmentId": pulid.MustNew("shp_").String(),
			"reason":     "Guessing.",
		}),
	)

	require.Error(t, err)
}

func TestMarkInboundMessage_SettlesWithTheNote(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{message: inboundMessage()}
	params := agenttoolstest.DeskParams(map[string]any{
		"messageId": desk.message.ID.String(),
		"status":    "Ignored",
		"note":      "  A newsletter; nothing to do.  ",
	})

	require.NoError(t, NewMarkInboundMessageTool(desk).Execute(t.Context(), params))

	require.NotNil(t, desk.reviewed)
	assert.Equal(t, inboundmessage.StatusIgnored, desk.reviewed.Status)
	assert.Equal(t, "A newsletter; nothing to do.", desk.reviewed.Note)
	assert.Equal(t, params.OrganizationID, desk.reviewed.TenantInfo.OrgID)
}

func TestMarkInboundMessage_Refuses(t *testing.T) {
	t.Parallel()

	quarantined := inboundMessage()
	quarantined.Status = inboundmessage.StatusQuarantined

	cases := map[string]struct {
		message *inboundmessage.InboundMessage
		params  map[string]any
	}{
		"a status that is not a person's to choose": {
			message: inboundMessage(),
			params:  map[string]any{"status": "InReview", "note": "x"},
		},
		"no note": {
			message: inboundMessage(),
			params:  map[string]any{"status": "Actioned"},
		},
		"a quarantined message": {
			message: quarantined,
			params:  map[string]any{"status": "Ignored", "note": "spam"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			desk := &fakeInboundDesk{message: tc.message}
			tc.params["messageId"] = tc.message.ID.String()
			tool := NewMarkInboundMessageTool(desk)

			require.Error(
				t,
				tool.(agenttoolports.ToolValidator).Validate(
					t.Context(),
					agenttoolstest.DeskParams(tc.params),
				),
			)
			require.Error(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(tc.params)))
			assert.Nil(t, desk.reviewed)
		})
	}
}

func replyTool(
	desk *fakeInboundDesk,
	mailer *agenttoolstest.FakeMailer,
	renderer *agenttoolstest.FakeRenderer,
) agenttoolports.AgentTool {
	return &ReplyToInboundMessageTool{
		inbox: desk,
		deps:  inboundReplier{email: mailer, templates: renderer},
	}
}

/*
The recipient is the sender and nobody else. A model that could name the
address would be a model that could be talked, by the very message it is
answering, into sending the organization's data somewhere else.
*/
func TestReplyToInboundMessage_AnswersTheSenderInTheThread(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{message: inboundMessage()}
	mailer := &agenttoolstest.FakeMailer{}
	renderer := &agenttoolstest.FakeRenderer{}
	profileID := pulid.MustNew("emp_")
	params := agenttoolstest.DeskParams(map[string]any{
		"messageId": desk.message.ID.String(),
		"profileId": profileID.String(),
		"body":      "It delivers tomorrow before noon.",
	})

	require.NoError(t, replyTool(desk, mailer, renderer).Execute(t.Context(), params))

	require.NotNil(t, mailer.Sent)
	assert.Equal(t, []string{"dana@shipper.example"}, mailer.Sent.To)
	assert.Empty(t, mailer.Sent.CC)
	assert.Empty(t, mailer.Sent.BCC)
	assert.Equal(t, profileID, mailer.Sent.ProfileID)
	assert.Equal(t, "key-1", mailer.Sent.IdempotencyKey)
	assert.Equal(t, "<abc@shipper.example>", mailer.Sent.Headers["In-Reply-To"])
	assert.Equal(t,
		"<first@shipper.example> <abc@shipper.example>",
		mailer.Sent.Headers["References"],
	)

	require.NotNil(t, renderer.Req)
	assert.Equal(t, documenttemplate.KindAgentInboundReplyEmail, renderer.Req.Kind)
	data := renderer.Req.Data.(documenttemplate.AgentEmailContext)
	assert.Equal(t, "Re: Where is 88213?", data.AgentSubject)

	require.NotNil(t, desk.reviewed, "a reply settles the message")
	assert.Equal(t, inboundmessage.StatusActioned, desk.reviewed.Status)
	assert.Contains(t, desk.reviewed.Note, "dana@shipper.example")
}

func TestReplyToInboundMessage_LeavesTheThreadWhenTheSendersIDIsUnsafe(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{message: inboundMessage()}
	desk.message.MessageID = "abc@shipper.example>\r\nBcc: someone@else.example"
	mailer := &agenttoolstest.FakeMailer{}

	require.NoError(
		t,
		replyTool(
			desk,
			mailer,
			&agenttoolstest.FakeRenderer{},
		).Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
			"messageId": desk.message.ID.String(),
			"profileId": pulid.MustNew("emp_").String(),
			"body":      "Answer.",
		})),
	)

	require.NotNil(t, mailer.Sent)
	assert.Nil(t, mailer.Sent.Headers)
}

func TestReplyToInboundMessage_Refuses(t *testing.T) {
	t.Parallel()

	cases := map[string]func(*inboundmessage.InboundMessage){
		"a quarantined message": func(m *inboundmessage.InboundMessage) {
			m.Status = inboundmessage.StatusQuarantined
		},
		"an automated sender": func(m *inboundmessage.InboundMessage) {
			m.FromAddress = "no-reply@shipper.example"
		},
		"mail from our own mailbox": func(m *inboundmessage.InboundMessage) {
			m.FromAddress = "LOADS@carrier.example"
		},
		"a message with no sender": func(m *inboundmessage.InboundMessage) {
			m.FromAddress = ""
		},
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			desk := &fakeInboundDesk{message: inboundMessage()}
			mutate(desk.message)
			mailer := &agenttoolstest.FakeMailer{}
			tool := replyTool(desk, mailer, &agenttoolstest.FakeRenderer{})
			params := agenttoolstest.DeskParams(map[string]any{
				"messageId": desk.message.ID.String(),
				"profileId": pulid.MustNew("emp_").String(),
				"body":      "Answer.",
			})

			require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
			require.Error(t, tool.Execute(t.Context(), params))
			assert.Nil(t, mailer.Sent, "nothing may be sent")
			assert.Nil(t, desk.reviewed)
		})
	}
}

func TestReplyToInboundMessage_DoesNotSettleAMessageWhoseReplyFailed(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{message: inboundMessage()}
	mailer := &agenttoolstest.FakeMailer{Err: assert.AnError}

	err := replyTool(
		desk,
		mailer,
		&agenttoolstest.FakeRenderer{},
	).Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageId": desk.message.ID.String(),
		"profileId": pulid.MustNew("emp_").String(),
		"body":      "Answer.",
	}))

	require.ErrorIs(t, err, assert.AnError)
	assert.Nil(t, desk.reviewed)
}

/*
The mailbox is the only thing that grants an inbox desk autonomy. However well
the desk has done, a message on a mailbox that sends everything to a person, or
one that was held for review, or one somebody already settled, is a proposal.
*/
func TestInboundTools_AreOnlyAsAutonomousAsTheMailbox(t *testing.T) {
	t.Parallel()

	autoHandle := func(m *inboundmessage.InboundMessage) {
		m.Status = inboundmessage.StatusClassified
		m.Mailbox.ReviewPolicy = inboundmessage.ReviewAutoHandle
	}

	cases := []struct {
		name   string
		mutate func(*inboundmessage.InboundMessage)
		want   agent.AutonomyTier
	}{
		{name: "handled without review", mutate: autoHandle, want: agent.TierAutoExecute},
		{
			name: "a mailbox that reviews everything",
			mutate: func(m *inboundmessage.InboundMessage) {
				m.Status = inboundmessage.StatusClassified
				m.Mailbox.ReviewPolicy = inboundmessage.ReviewAlways
			},
			want: agent.TierPropose,
		},
		{
			name: "below the mailbox's confidence",
			mutate: func(m *inboundmessage.InboundMessage) {
				m.Status = inboundmessage.StatusClassified
				m.Mailbox.ReviewPolicy = inboundmessage.ReviewBelowConfidence
				m.Mailbox.MinConfidence = 0.9
				m.Confidence = 0.6
			},
			want: agent.TierPropose,
		},
		{
			name: "held for review",
			mutate: func(m *inboundmessage.InboundMessage) {
				autoHandle(m)
				m.Status = inboundmessage.StatusInReview
			},
			want: agent.TierPropose,
		},
		{
			name: "already settled",
			mutate: func(m *inboundmessage.InboundMessage) {
				autoHandle(m)
				m.Status = inboundmessage.StatusActioned
			},
			want: agent.TierPropose,
		},
		{
			name: "no mailbox to read the policy from",
			mutate: func(m *inboundmessage.InboundMessage) {
				autoHandle(m)
				m.Mailbox = nil
			},
			want: agent.TierPropose,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			desk := &fakeInboundDesk{message: inboundMessage()}
			tc.mutate(desk.message)
			params := agenttoolstest.DeskParams(
				map[string]any{"messageId": desk.message.ID.String()},
			)

			for _, tool := range []agenttoolports.AgentTool{
				NewLinkInboundMessageTool(desk),
				NewMarkInboundMessageTool(desk),
				replyTool(desk, &agenttoolstest.FakeMailer{}, &agenttoolstest.FakeRenderer{}),
			} {
				condition := tool.Policy().Condition
				require.NotNilf(t, condition, "%s reports no limit", tool.Name())
				assert.Equalf(t, tc.want, condition.Limit(t.Context(), params), tool.Name())
			}
		})
	}
}

func TestInboundTools_AMessageThatCannotBeReadIsAProposal(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{}
	condition := NewMarkInboundMessageTool(desk).Policy().Condition

	assert.Equal(
		t,
		agent.TierPropose,
		condition.Limit(t.Context(), agenttoolstest.DeskParams(map[string]any{
			"messageId": pulid.MustNew("imsg_").String(),
		})),
	)
}

// A load is never created unattended, whatever the desk has earned.
func TestCreateShipment_IsNeverCreatedUnattended(t *testing.T) {
	t.Parallel()

	policy := (&operationstools.CreateShipmentTool{}).Policy()
	assert.False(t, policy.MaxTier.Above(agent.TierActWithApproval))
	assert.Equal(t, agent.TierActWithApproval, agenttoolpolicy.Promotable(policy))
}
