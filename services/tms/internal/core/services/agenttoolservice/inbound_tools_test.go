package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
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
	requested *repositories.GetInboundMessageByIDRequest
}

func (f *fakeInboundDesk) GetByID(
	_ context.Context,
	req repositories.GetInboundMessageByIDRequest,
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
	tool := newLinkInboundMessageTool(desk)
	shipmentID := pulid.MustNew("shp_")
	params := deskParams(map[string]any{
		"messageId":  desk.message.ID.String(),
		"shipmentId": shipmentID.String(),
		"reason":     "The PRO in the subject is this load's.",
	})

	require.NoError(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
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
	tool := newLinkInboundMessageTool(desk)

	err := tool.(serviceports.ToolValidator).Validate(t.Context(), deskParams(map[string]any{
		"messageId":  desk.message.ID.String(),
		"shipmentId": pulid.MustNew("shp_").String(),
		"reason":     "Guessing.",
	}))

	require.Error(t, err)
}

func TestLinkInboundMessage_SimulationShowsWhatChanges(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{message: inboundMessage()}
	shipmentID := pulid.MustNew("shp_")
	simulation, err := newLinkInboundMessageTool(desk).(serviceports.ToolSimulator).Simulate(
		t.Context(),
		deskParams(map[string]any{
			"messageId":  desk.message.ID.String(),
			"shipmentId": shipmentID.String(),
			"reason":     "The PRO in the subject.",
		}),
	)
	require.NoError(t, err)

	require.NotEmpty(t, simulation.Changes)
	assert.Equal(t, "shipment", simulation.Changes[0].Field)
	assert.Equal(t, "none", simulation.Changes[0].From)
	assert.Equal(t, shipmentID.String(), simulation.Changes[0].To)
}

func TestMarkInboundMessage_SettlesWithTheNote(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{message: inboundMessage()}
	params := deskParams(map[string]any{
		"messageId": desk.message.ID.String(),
		"status":    "Ignored",
		"note":      "  A newsletter; nothing to do.  ",
	})

	require.NoError(t, newMarkInboundMessageTool(desk).Execute(t.Context(), params))

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
			tool := newMarkInboundMessageTool(desk)

			require.Error(
				t,
				tool.(serviceports.ToolValidator).Validate(t.Context(), deskParams(tc.params)),
			)
			require.Error(t, tool.Execute(t.Context(), deskParams(tc.params)))
			assert.Nil(t, desk.reviewed)
		})
	}
}

func replyTool(
	desk *fakeInboundDesk,
	mailer *fakeMailer,
	renderer *fakeRenderer,
) serviceports.AgentTool {
	return &replyToInboundMessageTool{
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
	mailer := &fakeMailer{}
	renderer := &fakeRenderer{}
	profileID := pulid.MustNew("emp_")
	params := deskParams(map[string]any{
		"messageId": desk.message.ID.String(),
		"profileId": profileID.String(),
		"body":      "It delivers tomorrow before noon.",
	})

	require.NoError(t, replyTool(desk, mailer, renderer).Execute(t.Context(), params))

	require.NotNil(t, mailer.sent)
	assert.Equal(t, []string{"dana@shipper.example"}, mailer.sent.To)
	assert.Empty(t, mailer.sent.CC)
	assert.Empty(t, mailer.sent.BCC)
	assert.Equal(t, profileID, mailer.sent.ProfileID)
	assert.Equal(t, "key-1", mailer.sent.IdempotencyKey)
	assert.Equal(t, "<abc@shipper.example>", mailer.sent.Headers["In-Reply-To"])
	assert.Equal(t,
		"<first@shipper.example> <abc@shipper.example>",
		mailer.sent.Headers["References"],
	)

	require.NotNil(t, renderer.req)
	assert.Equal(t, documenttemplate.KindAgentInboundReplyEmail, renderer.req.Kind)
	data := renderer.req.Data.(documenttemplate.AgentEmailContext)
	assert.Equal(t, "Re: Where is 88213?", data.AgentSubject)

	require.NotNil(t, desk.reviewed, "a reply settles the message")
	assert.Equal(t, inboundmessage.StatusActioned, desk.reviewed.Status)
	assert.Contains(t, desk.reviewed.Note, "dana@shipper.example")
}

func TestReplyToInboundMessage_LeavesTheThreadWhenTheSendersIDIsUnsafe(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{message: inboundMessage()}
	desk.message.MessageID = "abc@shipper.example>\r\nBcc: someone@else.example"
	mailer := &fakeMailer{}

	require.NoError(
		t,
		replyTool(desk, mailer, &fakeRenderer{}).Execute(t.Context(), deskParams(map[string]any{
			"messageId": desk.message.ID.String(),
			"profileId": pulid.MustNew("emp_").String(),
			"body":      "Answer.",
		})),
	)

	require.NotNil(t, mailer.sent)
	assert.Nil(t, mailer.sent.Headers)
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
			mailer := &fakeMailer{}
			tool := replyTool(desk, mailer, &fakeRenderer{})
			params := deskParams(map[string]any{
				"messageId": desk.message.ID.String(),
				"profileId": pulid.MustNew("emp_").String(),
				"body":      "Answer.",
			})

			require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
			require.Error(t, tool.Execute(t.Context(), params))
			assert.Nil(t, mailer.sent, "nothing may be sent")
			assert.Nil(t, desk.reviewed)
		})
	}
}

func TestReplyToInboundMessage_DoesNotSettleAMessageWhoseReplyFailed(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{message: inboundMessage()}
	mailer := &fakeMailer{err: assert.AnError}

	err := replyTool(desk, mailer, &fakeRenderer{}).Execute(t.Context(), deskParams(map[string]any{
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
			params := deskParams(map[string]any{"messageId": desk.message.ID.String()})

			for _, tool := range []serviceports.AgentTool{
				newLinkInboundMessageTool(desk),
				newMarkInboundMessageTool(desk),
				replyTool(desk, &fakeMailer{}, &fakeRenderer{}),
			} {
				limiter, ok := tool.(serviceports.ToolTierLimiter)
				require.Truef(t, ok, "%s reports no limit", tool.Name())
				assert.Equalf(t, tc.want, limiter.TierLimit(t.Context(), params), tool.Name())
			}
		})
	}
}

func TestInboundTools_AMessageThatCannotBeReadIsAProposal(t *testing.T) {
	t.Parallel()

	desk := &fakeInboundDesk{}
	limiter := newMarkInboundMessageTool(desk).(serviceports.ToolTierLimiter)

	assert.Equal(t, agent.TierPropose, limiter.TierLimit(t.Context(), deskParams(map[string]any{
		"messageId": pulid.MustNew("imsg_").String(),
	})))
}

// A load is never created unattended, whatever the desk has earned.
func TestCreateShipment_IsNeverCreatedUnattended(t *testing.T) {
	t.Parallel()

	var tool serviceports.AgentTool = &createShipmentTool{}
	limiter, ok := tool.(serviceports.ToolTierLimiter)
	require.True(t, ok)
	assert.False(
		t,
		limiter.TierLimit(t.Context(), deskParams(nil)).Above(agent.TierActWithApproval),
	)
}
