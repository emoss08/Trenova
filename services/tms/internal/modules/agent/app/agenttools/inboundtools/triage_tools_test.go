package inboundtools

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func triageMessage() *inboundmessage.InboundMessage {
	mailboxID := pulid.MustNew("imbx_")

	return &inboundmessage.InboundMessage{
		ID:          pulid.MustNew("imsg_"),
		MailboxID:   mailboxID,
		Status:      inboundmessage.StatusInReview,
		FromAddress: "Dana@Shipper.example",
		Subject:     "Where is 88213?",
		Version:     3,
		Mailbox: &inboundmessage.Mailbox{
			ID:      mailboxID,
			Address: "loads@carrier.example",
		},
	}
}

func newFakeTriage() *fakeTriage {
	return &fakeTriage{message: triageMessage()}
}

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

type triageToolCase struct {
	build     func(f *fakeTriage) agentports.AgentTool
	name      string
	required  []string
	resource  permission.Resource
	operation permission.Operation
	egress    agent.EgressClass
	maxTier   agent.AutonomyTier
	condition bool
}

var triageToolCases = []triageToolCase{
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newAssignInboundMessagesTool(f)
		},
		name:     "inbox_assign_inbound_message",
		required: []string{"messageIds"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpAssign, egress: agent.EgressInternal,
		maxTier: agent.TierActWithApproval,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newReopenInboundMessagesTool(f)
		},
		name:     "inbox_reopen_inbound_message",
		required: []string{"messageIds"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpUpdate, egress: agent.EgressInternal,
		maxTier: agent.TierActWithApproval,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newSnoozeInboundMessagesTool(f)
		},
		name:     "inbox_snooze_inbound_message",
		required: []string{"messageIds"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpUpdate, egress: agent.EgressInternal,
		maxTier: agent.TierActWithApproval,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newWakeInboundMessageTool(f)
		},
		name:     "inbox_wake_inbound_message",
		required: []string{"messageId"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpUpdate, egress: agent.EgressInternal,
		maxTier: agent.TierAutoExecute,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newWaitForInboundReplyTool(f)
		},
		name:     "inbox_await_reply",
		required: []string{"messageId", "until"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpUpdate, egress: agent.EgressInternal,
		maxTier: agent.TierActWithApproval,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newUnlinkInboundMessageTool(f)
		},
		name:     "inbox_unlink_inbound_message",
		required: []string{"messageId", "record"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpUpdate, egress: agent.EgressInternal,
		maxTier: agent.TierAutoExecute, condition: true,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newHandOffTriageTool(f)
		},
		name:     "inbox_assign_queue",
		required: []string{"toUserId"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpAssign, egress: agent.EgressInternal,
		maxTier: agent.TierActWithApproval,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newBlockInboundSenderTool(f)
		},
		name:     "inbox_block_sender",
		required: []string{"messageId"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpUpdate, egress: agent.EgressInternal,
		maxTier: agent.TierActWithApproval,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newUnblockInboundSenderTool(f, f)
		},
		name:     "inbox_unblock_sender",
		required: []string{"messageId"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpUpdate, egress: agent.EgressInternal,
		maxTier: agent.TierActWithApproval,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newDraftInboundReplyTool(f, f)
		},
		name:     "inbox_draft_reply",
		required: []string{"messageId"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpUpdate, egress: agent.EgressInternal,
		maxTier: agent.TierAutoExecute, condition: true,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newDiscardInboundReplyDraftTool(f)
		},
		name:     "inbox_discard_reply_draft",
		required: []string{"messageId"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpUpdate, egress: agent.EgressInternal,
		maxTier: agent.TierActWithApproval,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newDraftInboundLoadTool(f, f)
		},
		name:     "inbox_draft_load",
		required: []string{"messageId"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpUpdate, egress: agent.EgressInternal,
		maxTier: agent.TierAutoExecute, condition: true,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newCreateInboundLoadTool(f)
		},
		name:     "inbox_create_load",
		required: []string{"messageId", "fields"}, resource: permission.ResourceShipment,
		operation: permission.OpCreate, egress: agent.EgressInternal,
		maxTier: agent.TierActWithApproval,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newDiscardInboundLoadTool(f, f)
		},
		name:     "inbox_discard_load",
		required: []string{"messageId"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpUpdate, egress: agent.EgressInternal,
		maxTier: agent.TierAutoExecute, condition: true,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newFileInboundAttachmentTool(f, f)
		},
		name:     "inbox_file_attachment",
		required: []string{"messageId", "attachmentId", "shipmentId"},
		resource: permission.ResourceDocument, operation: permission.OpUpdate,
		egress: agent.EgressInternal, maxTier: agent.TierAutoExecute, condition: true,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newUnfileInboundAttachmentTool(f, f)
		},
		name:     "inbox_unfile_attachment",
		required: []string{"messageId", "attachmentId"}, resource: permission.ResourceDocument,
		operation: permission.OpUpdate, egress: agent.EgressInternal,
		maxTier: agent.TierActWithApproval,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newLogInboundExceptionTool(f, f)
		},
		name:     "inbox_record_discrepancy",
		required: []string{"messageId", "kind", "quantity", "unit"},
		resource: permission.ResourceShipmentHold, operation: permission.OpCreate,
		egress: agent.EgressInternal, maxTier: agent.TierAutoExecute, condition: true,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newSettleDetentionDisputeTool(f, f)
		},
		name:     "detention_resolve_dispute",
		required: []string{"messageId", "decision"}, resource: permission.ResourceDetention,
		operation: permission.OpUpdate, egress: agent.EgressMoney,
		maxTier: agent.TierPropose,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newResolveInboundConflictTool(fakeConflictDesk{f})
		},
		name:     "inbox_resolve_conflict",
		required: []string{"messageId", "conflictId", "resolution"},
		resource: permission.ResourceInboundMessage, operation: permission.OpUpdate,
		egress: agent.EgressInternal, maxTier: agent.TierActWithApproval,
	},
	{
		build: func(f *fakeTriage) agentports.AgentTool {
			return newReopenInboundConflictTool(f, fakeConflictDesk{f})
		},
		name:     "inbox_reopen_conflict",
		required: []string{"messageId", "conflictId"}, resource: permission.ResourceInboundMessage,
		operation: permission.OpUpdate, egress: agent.EgressInternal,
		maxTier: agent.TierAutoExecute, condition: true,
	},
}

func TestTriageTools_DeclareTheirSchemaAndPolicy(t *testing.T) {
	t.Parallel()

	for _, tc := range triageToolCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tool := tc.build(newFakeTriage())
			require.Equal(t, tc.name, tool.Name())

			schema := tool.ParamSchema()
			assert.Equal(t, false, schema[toolschema.KeyAdditionalProperties])
			assert.ElementsMatch(t, tc.required, schema[toolschema.KeyRequired])
			properties, _ := schema[toolschema.KeyProperties].(map[string]any)
			for _, name := range tc.required {
				assert.Containsf(t, properties, name, "%s is required but not declared", name)
			}

			policy := tool.Policy()
			assert.Equal(t, tc.name, policy.Name)
			assert.Equal(t, agent.ToolKindAction, policy.Kind)
			assert.Equal(t, tc.resource, policy.Resource)
			assert.Equal(t, tc.operation, policy.Operation)
			assert.Equal(t, []agent.EgressClass{tc.egress}, policy.Egress)
			assert.Equal(t, tc.maxTier, policy.MaxTier)
			assert.False(t, policy.DefaultTier.Above(policy.MaxTier))
			assert.Equal(t, tc.condition, policy.Condition != nil)
			assert.NotEmpty(t, policy.Rationale)
			assert.NotEmpty(t, policy.Artifact, "a tool that reports its result names its record")

			_, previews := tool.(agentports.ToolPreviewer)
			assert.True(t, previews, "every write previews what it would do")
			_, reports := tool.(agentports.ToolResultReporter)
			assert.True(t, reports)
		})
	}
}

/*
The queue tools stay inside the organization. Blocking a sender silently drops
an outsider's mail, so it is asked first whatever the desk has earned; settling
a disputed detention charge gives up or keeps revenue, so a person always
decides; and drafting a reply only drafts: nothing in the set sends.
*/
func TestTriageTools_KeepTheirCaps(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	policies := make(map[string]agentports.ToolPolicy, len(triageToolCases))
	for _, tc := range triageToolCases {
		tool := tc.build(f)
		policies[tool.Name()] = tool.Policy()
	}

	for name, policy := range policies {
		assert.Falsef(t, policy.HasEgress(agent.EgressExternalRecipient),
			"%s must not send anything outside the organization", name)
	}

	block := policies["inbox_block_sender"]
	assert.Equal(t, agent.TierActWithApproval, agenttoolpolicy.Promotable(block),
		"blocking a sender is always asked first")

	settle := policies["detention_resolve_dispute"]
	assert.Equal(t, agent.TierPropose, agenttoolpolicy.Promotable(settle))
	assert.True(t, settle.HeldWhenTainted(), "a decision that moves money waits once tainted")

	draft := policies["inbox_draft_reply"]
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, draft.Egress,
		"a draft is only seen inside the organization")

	create := policies["inbox_create_load"]
	assert.Equal(t, agent.TierActWithApproval, agenttoolpolicy.Promotable(create),
		"a load is never created unattended")
	assert.True(t, create.Idempotent)
}

func TestTriageTools_RunOnlyAsFarAsTheMailboxAllows(t *testing.T) {
	t.Parallel()

	for _, tc := range triageToolCases {
		if !tc.condition {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newFakeTriage()
			f.message.Status = inboundmessage.StatusClassified
			f.message.Mailbox.ReviewPolicy = inboundmessage.ReviewAutoHandle
			condition := tc.build(f).Policy().Condition
			params := agenttoolstest.DeskParams(map[string]any{"messageId": f.message.ID.String()})

			assert.Equal(t, agent.TierAutoExecute, condition.Limit(t.Context(), params))

			f.message.Status = inboundmessage.StatusInReview
			assert.Equal(t, agent.TierPropose, condition.Limit(t.Context(), params),
				"a message held for review is a proposal")
		})
	}
}

func TestTriageTools_NameTheMessageTheyChange(t *testing.T) {
	t.Parallel()

	for _, tc := range triageToolCases {
		tool := tc.build(newFakeTriage())
		properties, _ := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
		if _, single := properties["messageId"]; !single {
			continue
		}
		id := pulid.MustNew("imsg_")
		target, ok := tool.(agentports.TargetedTool).
			Target(map[string]any{"messageId": id.String()})
		require.Truef(t, ok, "%s names no target", tc.name)
		assert.Equal(t, agentports.ToolTarget{Resource: permission.ResourceInboundMessage, ID: id},
			target, tc.name)
	}
}

func TestAssignInboundMessages_GivesTheSelectionToTheOwner(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	owner := pulid.MustNew("usr_")
	f.queuePlan = onePlannedChange(f.message, func(s *inboundmessageports.QueueState) {
		s.OwnerID = owner
	})
	tool := newAssignInboundMessagesTool(f)
	params := agenttoolstest.DeskParams(map[string]any{
		"messageIds": agenttoolstest.IdList([]pulid.ID{f.message.ID}),
		"ownerId":    owner.String(),
	})

	preview, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Empty(t, f.written, "a preview writes nothing")
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceInboundMessage, change.Resource)
	field := agenttoolstest.FieldByPath(t, change, "ownerId")
	require.NotNil(t, field.AfterRef)
	assert.Equal(t, permission.ResourceUser, field.AfterRef.Resource)
	assert.Equal(t, owner, field.AfterRef.ID)

	result, err := tool.(agentports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, "assigned", result.Action)
	require.Len(t, f.written, 1)
	req := f.written[0].(*inboundmessageservice.AssignRequest)
	assert.Equal(t, params.OrganizationID, req.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, req.TenantInfo.BuID)
	assert.Equal(t, params.Actor.UserID, req.ActorID)
	assert.Equal(t, []pulid.ID{f.message.ID}, req.Selection.IDs)
	assert.Nil(t, req.Selection.Folder, "an agent never selects a whole folder")
	assert.Equal(t, owner, req.OwnerID)
	assert.True(t, req.CanAssignOthers, "the policy's assign permission is what lets it")
}

func TestQueueTools_WarnWhenNothingWouldChange(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	f.queuePlan = &inboundmessageservice.QueuePlan{Unchanged: 1}
	preview, err := newReopenInboundMessagesTool(f).(agentports.ToolPreviewer).Preview(
		t.Context(),
		agenttoolstest.DeskParams(map[string]any{
			"messageIds": agenttoolstest.IdList([]pulid.ID{f.message.ID}),
		}),
	)
	require.NoError(t, err)
	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestQueueTools_RefuseASelectionPastTheBound(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	err := newReopenInboundMessagesTool(f).(agentports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.DeskParams(map[string]any{
			"messageIds": agenttoolstest.ManyIDs(maxTriageSelection + 1),
		}),
	)
	require.Error(t, err)
	assert.Empty(t, f.planned)
}

func TestSnoozeInboundMessages_TakesATimeOrAnEventButNotBoth(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	f.queuePlan = onePlannedChange(f.message, func(s *inboundmessageports.QueueState) {
		s.Parked = inboundmessage.ParkingSnoozed
	})
	tool := newSnoozeInboundMessagesTool(f)
	ids := agenttoolstest.IdList([]pulid.ID{f.message.ID})

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageIds": ids,
		"until":      "2026-11-02T08:00",
	})))
	timed := f.written[0].(*inboundmessageservice.SnoozeRequest)
	assert.Equal(t, time.Date(2026, time.November, 2, 8, 0, 0, 0, time.UTC).Unix(), timed.Until)
	assert.Empty(t, timed.Event)

	documentType := pulid.MustNew("dt_")
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageIds":     ids,
		"event":          "DocumentAttached",
		"documentTypeId": documentType.String(),
	})))
	evented := f.written[1].(*inboundmessageservice.SnoozeRequest)
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
	assert.Len(t, f.written, 2)
}

func TestWakeInboundMessage_CallsTheWakeThatFitsHowItIsParked(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	f.queuePlan = onePlannedChange(f.message, func(s *inboundmessageports.QueueState) {
		s.Parked = ""
	})
	tool := newWakeInboundMessageTool(f)
	params := agenttoolstest.DeskParams(map[string]any{"messageId": f.message.ID.String()})

	f.message.Parked = inboundmessage.ParkingSnoozed
	require.NoError(t, tool.Execute(t.Context(), params))
	_, woke := f.written[0].(*inboundmessageservice.MessageActionRequest)
	assert.True(t, woke, "a snoozed message is woken")

	f.message.Parked = inboundmessage.ParkingAwaitingReply
	require.NoError(t, tool.Execute(t.Context(), params))
	_, stopped := f.written[1].(stopWaitingCall)
	assert.True(t, stopped, "a message waiting on a reply stops waiting")

	f.message.Parked = ""
	preview, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
	require.ErrorIs(t, tool.Execute(t.Context(), params), errInboundNotParked)
	assert.Len(t, f.written, 2)
}

func TestWaitForInboundReply_ParksUntilTheTimeGiven(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	tool := newWaitForInboundReplyTool(f)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageId": f.message.ID.String(),
		"until":     "2026-11-04T17:30",
	})))
	req := f.written[0].(*inboundmessageservice.WaitForReplyRequest)
	assert.Equal(t, f.message.ID, req.MessageID)
	assert.Equal(t, time.Date(2026, time.November, 4, 17, 30, 0, 0, time.UTC).Unix(), req.Until)

	assert.Error(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageId": f.message.ID.String(),
		"until":     "2026-11-04T17:30:00Z",
	})), "a time with an offset is refused")
}

func TestUnlinkInboundMessage_RemovesTheLinkNamed(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	tool := newUnlinkInboundMessageTool(f)

	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageId": f.message.ID.String(),
		"record":    "Customer",
	})))
	req := f.written[0].(*inboundmessageservice.UnlinkRequest)
	assert.Equal(t, inboundmessageservice.LinkedCustomer, req.Record)

	assert.Error(t, tool.Execute(t.Context(), agenttoolstest.DeskParams(map[string]any{
		"messageId": f.message.ID.String(),
		"record":    "Mailbox",
	})))
}

func TestHandOffTriage_SaysWhenTheQueueIsLargerThanThePreview(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	to := pulid.MustNew("usr_")
	f.queuePlan = onePlannedChange(f.message, func(s *inboundmessageports.QueueState) {
		s.OwnerID = to
	})
	f.queuePlan.More = true
	tool := newHandOffTriageTool(f)
	params := agenttoolstest.DeskParams(map[string]any{"toUserId": to.String()})

	preview, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.True(t, preview.Partial)
	assert.Contains(t, preview.Summary, "at least")
	assert.Empty(t, f.written)

	result, err := tool.(agentports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, 3, result.Total)
	assert.Equal(t, to, f.written[0].(*inboundmessageservice.HandOffRequest).ToUserID)
}

func TestBlockInboundSender_PreviewsTheSourceAndTheMessage(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	after := *f.message
	after.Status = inboundmessage.StatusIgnored
	f.senderPlan = &triageworkservice.SenderPlan{
		Message: f.message,
		After:   &after,
		Mailbox: f.message.Mailbox,
		Sender:  "dana@shipper.example",
		Blocked: []string{"dana@shipper.example"},
		Changed: true,
	}
	tool := newBlockInboundSenderTool(f)
	params := agenttoolstest.DeskParams(map[string]any{"messageId": f.message.ID.String()})

	preview, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Empty(t, f.written)
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, permission.ResourceInboundMailbox, preview.Changes[0].Resource)
	assert.Equal(t, "Ignored",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 1), "status").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	req := f.written[0].(*triageworkservice.BlockSenderRequest)
	assert.Equal(t, f.message.ID, req.MessageID)
	assert.Equal(t, params.Actor.UserID, req.UserID)
}

func TestUnblockInboundSender_NamesTheSenderByTheirMessage(t *testing.T) {
	t.Parallel()

	f := newFakeTriage()
	f.senderPlan = &triageworkservice.SenderPlan{
		Mailbox: f.message.Mailbox,
		Sender:  "dana@shipper.example",
	}
	tool := newUnblockInboundSenderTool(f, f)
	params := agenttoolstest.DeskParams(map[string]any{"messageId": f.message.ID.String()})

	preview, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
	require.ErrorIs(t, tool.Execute(t.Context(), params), errSenderNotBlocked)
	assert.Empty(t, f.written, "a sender who is not blocked is not unblocked")

	f.senderPlan.Mailbox.BlockedSenders = []string{"dana@shipper.example"}
	f.senderPlan.Changed = true
	require.NoError(t, tool.Execute(t.Context(), params))
	planned := f.planned[len(f.planned)-1].(*triageworkservice.UnblockSenderRequest)
	assert.Equal(t, f.message.MailboxID, planned.MailboxID)
	assert.Equal(t, "dana@shipper.example", planned.Sender, "the sender as a block records it")
	require.Len(t, f.written, 1)
	assert.Equal(t, planned, f.written[0])
}
