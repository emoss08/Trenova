package triagequeuetools

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func queueStateOf(message *inboundmessage.InboundMessage) *inboundmessageports.QueueState {
	return &inboundmessageports.QueueState{
		ID:          message.ID,
		Status:      message.Status,
		FromAddress: message.FromAddress,
		Subject:     message.Subject,
	}
}

func onePlannedChange(
	message *inboundmessage.InboundMessage,
	change func(*inboundmessageports.QueueState),
) *inboundmessageservice.QueuePlan {
	before := queueStateOf(message)
	after := *before
	change(&after)

	return &inboundmessageservice.QueuePlan{
		Changes: []inboundmessageservice.QueueChange{{Before: before, After: &after}},
	}
}

var triageToolCases = []agenttoolstest.TriageToolCase{
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newAssignInboundMessagesTool(f)
		},
		Name:     "inbox_assign_inbound_message",
		Required: []string{"messageIds"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpAssign, Egress: agent.EgressInternal,
		MaxTier: agent.TierActWithApproval,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newReopenInboundMessagesTool(f)
		},
		Name:     "inbox_reopen_inbound_message",
		Required: []string{"messageIds"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpUpdate, Egress: agent.EgressInternal,
		MaxTier: agent.TierActWithApproval,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newSnoozeInboundMessagesTool(f)
		},
		Name:     "inbox_snooze_inbound_message",
		Required: []string{"messageIds"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpUpdate, Egress: agent.EgressInternal,
		MaxTier: agent.TierActWithApproval,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newWakeInboundMessageTool(f)
		},
		Name:     "inbox_wake_inbound_message",
		Required: []string{"messageId"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpUpdate, Egress: agent.EgressInternal,
		MaxTier: agent.TierAutoExecute,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newWaitForInboundReplyTool(f)
		},
		Name:     "inbox_await_reply",
		Required: []string{"messageId", "until"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpUpdate, Egress: agent.EgressInternal,
		MaxTier: agent.TierActWithApproval,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newUnlinkInboundMessageTool(f)
		},
		Name:     "inbox_unlink_inbound_message",
		Required: []string{"messageId", "record"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpUpdate, Egress: agent.EgressInternal,
		MaxTier: agent.TierAutoExecute, Condition: true,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newHandOffTriageTool(f)
		},
		Name:     "inbox_assign_queue",
		Required: []string{"toUserId"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpAssign, Egress: agent.EgressInternal,
		MaxTier: agent.TierActWithApproval,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newBlockInboundSenderTool(f)
		},
		Name:     "inbox_block_sender",
		Required: []string{"messageId"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpUpdate, Egress: agent.EgressInternal,
		MaxTier: agent.TierActWithApproval,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newUnblockInboundSenderTool(f, f)
		},
		Name:     "inbox_unblock_sender",
		Required: []string{"messageId"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpUpdate, Egress: agent.EgressInternal,
		MaxTier: agent.TierActWithApproval,
	},
}

func TestTriageTools_DeclareTheirSchemaAndPolicy(t *testing.T) {
	t.Parallel()

	agenttoolstest.CheckTriageToolSchemaAndPolicy(t, triageToolCases)
}

func TestTriageTools_RunOnlyAsFarAsTheMailboxAllows(t *testing.T) {
	t.Parallel()

	agenttoolstest.CheckTriageToolMailboxCondition(t, triageToolCases)
}

func TestTriageTools_NameTheMessageTheyChange(t *testing.T) {
	t.Parallel()

	agenttoolstest.CheckTriageToolTargets(t, triageToolCases)
}

/*
The queue tools stay inside the organization. Blocking a sender silently drops
an outsider's mail, so it is asked first whatever the desk has earned.
*/
func TestTriageTools_KeepTheirCaps(t *testing.T) {
	t.Parallel()

	policies := agenttoolstest.TriageToolPolicies(agenttoolstest.NewFakeTriage(), triageToolCases)
	for name, policy := range policies {
		assert.Falsef(t, policy.HasEgress(agent.EgressExternalRecipient),
			"%s must not send anything outside the organization", name)
	}

	block := policies["inbox_block_sender"]
	assert.Equal(t, agent.TierActWithApproval, agenttoolpolicy.Promotable(block),
		"blocking a sender is always asked first")
}

func TestAssignInboundMessages_GivesTheSelectionToTheOwner(t *testing.T) {
	t.Parallel()

	f := agenttoolstest.NewFakeTriage()
	owner := pulid.MustNew("usr_")
	f.QueuePlan = onePlannedChange(f.Message, func(s *inboundmessageports.QueueState) {
		s.OwnerID = owner
	})
	tool := newAssignInboundMessagesTool(f)
	params := agenttoolstest.DeskParams(map[string]any{
		"messageIds": agenttoolstest.IdList([]pulid.ID{f.Message.ID}),
		"ownerId":    owner.String(),
	})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Empty(t, f.Written, "a preview writes nothing")
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceInboundMessage, change.Resource)
	field := agenttoolstest.FieldByPath(t, change, "ownerId")
	require.NotNil(t, field.AfterRef)
	assert.Equal(t, permission.ResourceUser, field.AfterRef.Resource)
	assert.Equal(t, owner, field.AfterRef.ID)

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "assigned", result.Action)
	require.Len(t, f.Written, 1)
	req := f.Written[0].(*inboundmessageservice.AssignRequest)
	assert.Equal(t, params.OrganizationID, req.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, req.TenantInfo.BuID)
	assert.Equal(t, params.Actor.UserID, req.ActorID)
	assert.Equal(t, []pulid.ID{f.Message.ID}, req.Selection.IDs)
	assert.Nil(t, req.Selection.Folder, "an agent never selects a whole folder")
	assert.Equal(t, owner, req.OwnerID)
	assert.True(t, req.CanAssignOthers, "the policy's assign permission is what lets it")
}

func TestQueueTools_WarnWhenNothingWouldChange(t *testing.T) {
	t.Parallel()

	f := agenttoolstest.NewFakeTriage()
	f.QueuePlan = &inboundmessageservice.QueuePlan{Unchanged: 1}
	preview, err := newReopenInboundMessagesTool(f).(agenttoolports.ToolPreviewer).Preview(
		t.Context(),
		agenttoolstest.DeskParams(map[string]any{
			"messageIds": agenttoolstest.IdList([]pulid.ID{f.Message.ID}),
		}),
	)
	require.NoError(t, err)
	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}

func TestQueueTools_RefuseASelectionPastTheBound(t *testing.T) {
	t.Parallel()

	f := agenttoolstest.NewFakeTriage()
	err := newReopenInboundMessagesTool(f).(agenttoolports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.DeskParams(map[string]any{
			"messageIds": agenttoolstest.ManyIDs(maxTriageSelection + 1),
		}),
	)
	require.Error(t, err)
	assert.Empty(t, f.Planned)
}

func TestSnoozeInboundMessages_TakesATimeOrAnEventButNotBoth(t *testing.T) {
	t.Parallel()

	f := agenttoolstest.NewFakeTriage()
	f.QueuePlan = onePlannedChange(f.Message, func(s *inboundmessageports.QueueState) {
		s.Parked = inboundmessage.ParkingSnoozed
	})
	tool := newSnoozeInboundMessagesTool(f)
	ids := agenttoolstest.IdList([]pulid.ID{f.Message.ID})

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageIds": ids,
		"until":      "2026-11-02T08:00",
	})))
	timed := f.Written[0].(*inboundmessageservice.SnoozeRequest)
	assert.Equal(t, time.Date(2026, time.November, 2, 8, 0, 0, 0, time.UTC).Unix(), timed.Until)
	assert.Empty(t, timed.Event)

	documentType := pulid.MustNew("dt_")
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageIds":     ids,
		"event":          "DocumentAttached",
		"documentTypeId": documentType.String(),
	})))
	evented := f.Written[1].(*inboundmessageservice.SnoozeRequest)
	assert.Equal(t, inboundmessage.SnoozeEventDocumentAttached, evented.Event)
	assert.Equal(t, documentType, evented.DocumentTypeID)
	assert.Zero(t, evented.Until)

	for name, params := range map[string]map[string]any{
		"neither": {"messageIds": ids},
		"both":    {"messageIds": ids, "until": "2026-11-02T08:00", "event": "InvoicePaid"},
		"unknown": {"messageIds": ids, "event": "Lunch"},
	} {
		assert.Errorf(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(params)), name)
	}
	assert.Len(t, f.Written, 2)
}

func TestWakeInboundMessage_CallsTheWakeThatFitsHowItIsParked(t *testing.T) {
	t.Parallel()

	f := agenttoolstest.NewFakeTriage()
	f.QueuePlan = onePlannedChange(f.Message, func(s *inboundmessageports.QueueState) {
		s.Parked = ""
	})
	tool := newWakeInboundMessageTool(f)
	params := agenttoolstest.DeskParams(map[string]any{"messageId": f.Message.ID.String()})

	f.Message.Parked = inboundmessage.ParkingSnoozed
	require.NoError(t, tool.Execute(t.Context(), params))
	_, woke := f.Written[0].(*inboundmessageservice.MessageActionRequest)
	assert.True(t, woke, "a snoozed message is woken")

	f.Message.Parked = inboundmessage.ParkingAwaitingReply
	require.NoError(t, tool.Execute(t.Context(), params))
	_, stopped := f.Written[1].(agenttoolstest.StopWaitingCall)
	assert.True(t, stopped, "a message waiting on a reply stops waiting")

	f.Message.Parked = ""
	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
	require.ErrorIs(t, tool.Execute(t.Context(), params), errInboundNotParked)
	assert.Len(t, f.Written, 2)
}

func TestWaitForInboundReply_ParksUntilTheTimeGiven(t *testing.T) {
	t.Parallel()

	f := agenttoolstest.NewFakeTriage()
	tool := newWaitForInboundReplyTool(f)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageId": f.Message.ID.String(),
		"until":     "2026-11-04T17:30",
	})))
	req := f.Written[0].(*inboundmessageservice.WaitForReplyRequest)
	assert.Equal(t, f.Message.ID, req.MessageID)
	assert.Equal(t, time.Date(2026, time.November, 4, 17, 30, 0, 0, time.UTC).Unix(), req.Until)

	assert.Error(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageId": f.Message.ID.String(),
		"until":     "2026-11-04T17:30:00Z",
	})), "a time with an offset is refused")
}

func TestUnlinkInboundMessage_RemovesTheLinkNamed(t *testing.T) {
	t.Parallel()

	f := agenttoolstest.NewFakeTriage()
	tool := newUnlinkInboundMessageTool(f)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageId": f.Message.ID.String(),
		"record":    "Customer",
	})))
	req := f.Written[0].(*inboundmessageservice.UnlinkRequest)
	assert.Equal(t, inboundmessageservice.LinkedCustomer, req.Record)

	assert.Error(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageId": f.Message.ID.String(),
		"record":    "Mailbox",
	})))
}

func TestHandOffTriage_SaysWhenTheQueueIsLargerThanThePreview(t *testing.T) {
	t.Parallel()

	f := agenttoolstest.NewFakeTriage()
	to := pulid.MustNew("usr_")
	f.QueuePlan = onePlannedChange(f.Message, func(s *inboundmessageports.QueueState) {
		s.OwnerID = to
	})
	f.QueuePlan.More = true
	tool := newHandOffTriageTool(f)
	params := agenttoolstest.DeskParams(map[string]any{"toUserId": to.String()})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.True(t, preview.Partial)
	assert.Contains(t, preview.Summary, "at least")
	assert.Empty(t, f.Written)

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, 3, result.Total)
	assert.Equal(t, to, f.Written[0].(*inboundmessageservice.HandOffRequest).ToUserID)
}

func TestBlockInboundSender_PreviewsTheSourceAndTheMessage(t *testing.T) {
	t.Parallel()

	f := agenttoolstest.NewFakeTriage()
	after := *f.Message
	after.Status = inboundmessage.StatusIgnored
	f.SenderPlan = &triageworkservice.SenderPlan{
		Message: f.Message,
		After:   &after,
		Mailbox: f.Message.Mailbox,
		Sender:  "dana@shipper.example",
		Blocked: []string{"dana@shipper.example"},
		Changed: true,
	}
	tool := newBlockInboundSenderTool(f)
	params := agenttoolstest.DeskParams(map[string]any{"messageId": f.Message.ID.String()})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Empty(t, f.Written)
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, permission.ResourceInboundMailbox, preview.Changes[0].Resource)
	assert.Equal(t, "Ignored",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 1), "status").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	req := f.Written[0].(*triageworkservice.BlockSenderRequest)
	assert.Equal(t, f.Message.ID, req.MessageID)
	assert.Equal(t, params.Actor.UserID, req.UserID)
}

func TestUnblockInboundSender_NamesTheSenderByTheirMessage(t *testing.T) {
	t.Parallel()

	f := agenttoolstest.NewFakeTriage()
	f.SenderPlan = &triageworkservice.SenderPlan{
		Mailbox: f.Message.Mailbox,
		Sender:  "dana@shipper.example",
	}
	tool := newUnblockInboundSenderTool(f, f)
	params := agenttoolstest.DeskParams(map[string]any{"messageId": f.Message.ID.String()})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
	require.ErrorIs(t, tool.Execute(t.Context(), params), errSenderNotBlocked)
	assert.Empty(t, f.Written, "a sender who is not blocked is not unblocked")

	f.SenderPlan.Mailbox.BlockedSenders = []string{"dana@shipper.example"}
	f.SenderPlan.Changed = true
	require.NoError(t, tool.Execute(t.Context(), params))
	planned := f.Planned[len(f.Planned)-1].(*triageworkservice.UnblockSenderRequest)
	assert.Equal(t, f.Message.MailboxID, planned.MailboxID)
	assert.Equal(t, "dana@shipper.example", planned.Sender, "the sender as a block records it")
	require.Len(t, f.Written, 1)
	assert.Equal(t, planned, f.Written[0])
}
