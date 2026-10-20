package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTriageAssignees struct {
	people   []*inboundmessageports.AssigneeView
	settings *triage.Settings
	tenant   pagination.TenantInfo
}

func (f *fakeTriageAssignees) Assignees(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*inboundmessageports.AssigneeView, error) {
	f.tenant = tenantInfo

	return f.people, nil
}

func (f *fakeTriageAssignees) Settings(
	context.Context,
	pagination.TenantInfo,
) (*triage.Settings, error) {
	return f.settings, nil
}

type fakeReplyDrafts struct {
	drafts map[pulid.ID]*triage.Reply
	asked  *inboundmessageports.InboundRepliesByMessagesRequest
}

func (f *fakeReplyDrafts) DraftsByMessageIDs(
	_ context.Context,
	req inboundmessageports.InboundRepliesByMessagesRequest,
) (map[pulid.ID]*triage.Reply, error) {
	f.asked = &req

	return f.drafts, nil
}

func TestListTriageAssignees_NamesWhoCanWorkTheQueueAndWhoIsOnCall(t *testing.T) {
	t.Parallel()

	onCall := pulid.MustNew("usr_")
	other := pulid.MustNew("usr_")
	reader := &fakeTriageAssignees{
		people: []*inboundmessageports.AssigneeView{
			{UserID: onCall, Name: "Marcus Lee", Waiting: 4},
			{UserID: other, Name: "Dana Ruiz"},
		},
		settings: &triage.Settings{OnCallUserID: onCall, EndOfDayMinute: 17*60 + 30},
	}
	tool := newListTriageAssigneesTool(reader)
	params := testParams(map[string]any{})

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	view := result.(triageAssigneesView)
	assert.Equal(t, params.OrganizationID, reader.tenant.OrgID)
	assert.Equal(t, onCall.String(), view.OnCallUserID)
	assert.Equal(t, "17:30", view.EndOfDay)
	require.Len(t, view.People, 2)
	assert.True(t, view.People[0].OnCall)
	assert.Equal(t, 4, view.People[0].Waiting)
	assert.False(t, view.People[1].OnCall)
	assert.Equal(t, "Dana Ruiz", view.People[1].Name)

	policy := tool.Policy()
	assert.Equal(t, agent.ToolKindQuery, policy.Kind)
	assert.Equal(t, permission.ResourceInboundMessage, policy.Resource)
	assert.Equal(t, []agent.EgressClass{agent.EgressNone}, policy.Egress)
}

/*
inbox_get_inbound_message is where the Triage tools' ids come from, so it shows a
message's place in the queue: its owner, why it is parked, the drafts on it,
each conflict with its id, and where each attachment is filed.
*/
func TestGetInboundMessage_ShowsWhereTheMessageStandsInTriage(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	shipmentID := pulid.MustNew("shp_")
	message := &inboundmessage.InboundMessage{
		ID:            pulid.MustNew("imsg_"),
		Status:        inboundmessage.StatusInReview,
		OwnerID:       owner,
		DueAt:         1_791_003_600,
		Parked:        inboundmessage.ParkingAwaitingReply,
		AwaitingUntil: 1_791_086_400,
		LoadDraft: &inboundmessage.LoadDraft{
			Source: "tender.pdf",
			Fields: []inboundmessage.LoadDraftField{
				{Key: inboundmessage.LoadFieldOrigin, Value: "Columbus, OH"},
				{Key: inboundmessage.LoadFieldWeight},
			},
		},
		Conflicts: []inboundmessage.Conflict{{
			ID:   "dup:imsga_1",
			Kind: inboundmessage.ConflictDuplicateDocument,
		}},
		ActionState: inboundmessage.ActionState{
			StopsDecision:   inboundmessage.StopsDecisionStandBy,
			SenderBlockedAt: 1_791_000_000,
		},
		Attachments: []*inboundmessage.InboundAttachment{{
			ID:              pulid.MustNew("imsga_"),
			FileName:        "pod.pdf",
			PageCount:       3,
			FiledShipmentID: shipmentID,
			FiledAs:         "Proof of Delivery",
		}},
	}
	drafts := &fakeReplyDrafts{drafts: map[pulid.ID]*triage.Reply{
		message.ID: {ToAddress: "dana@shipper.example", Subject: "Re: Load", Body: "Draft body"},
	}}
	params := testParams(map[string]any{"messageId": message.ID.String()})

	result, err := newGetInboundMessageTool(
		&fakeInboundMessages{message: message},
		drafts,
	).Query(t.Context(), params)
	require.NoError(t, err)

	view := result.(inboundMessageView)
	assert.Equal(t, owner.String(), view.OwnerID)
	assert.NotEmpty(t, view.DueAt)
	require.NotNil(t, view.Parked)
	assert.Equal(t, "AwaitingReply", view.Parked.As)
	assert.NotEmpty(t, view.Parked.Until)
	require.NotNil(t, view.LoadDraft)
	assert.Equal(t,
		[]inboundLoadFieldView{{Key: "Origin", Value: "Columbus, OH"}},
		view.LoadDraft.Fields,
		"an empty field is left out",
	)
	require.Len(t, view.Conflicts, 1)
	assert.Equal(t, "dup:imsga_1", view.Conflicts[0].ID)
	assert.Equal(t, "StandBy", view.StopsDecision)
	assert.True(t, view.SenderBlocked)
	require.Len(t, view.Attachments, 1)
	assert.Equal(t, shipmentID.String(), view.Attachments[0].FiledShipmentID)
	assert.Equal(t, 3, view.Attachments[0].PageCount)
	require.NotNil(t, view.ReplyDraft)
	assert.Equal(t, "Draft body", view.ReplyDraft.Body)
	require.NotNil(t, drafts.asked)
	assert.Equal(t, []pulid.ID{message.ID}, drafts.asked.MessageIDs)
	assert.Equal(t, params.OrganizationID, drafts.asked.TenantInfo.OrgID)
}
