package agentruntime

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	watchedShipment = "shp_01J9WATCHWATCHWATCHWATCH0"
	otherShipment   = "shp_01J9OTHEROTHEROTHEROTHER0"
)

type fakeChangeFeed struct {
	head    string
	changes []realtimeports.WatchedRecordChange
	asked   []*realtimeports.RecordChangesRequest
}

func (f *fakeChangeFeed) Head(context.Context, pulid.ID, pulid.ID) (string, error) {
	return f.head, nil
}

func (f *fakeChangeFeed) Since(
	_ context.Context,
	req *realtimeports.RecordChangesRequest,
) (*realtimeports.RecordChanges, error) {
	f.asked = append(f.asked, req)

	return &realtimeports.RecordChanges{Cursor: "0.2-0", Changes: f.changes}, nil
}

type clockedEffects struct {
	*localEffects

	now    int64
	events []kernelports.StreamEvent
}

func (fx *clockedEffects) Now() int64 { return fx.now }

func (fx *clockedEffects) Emit(event kernelports.StreamEvent) {
	fx.events = append(fx.events, event)
}

func (fx *clockedEffects) Complete(
	t *Turn,
	req *llmports.ChatCompletionRequest,
) (ModelReply, error) {
	reply, err := fx.localEffects.Complete(t, req)
	fx.now += worldCheckEvery + 1

	return reply, err
}

func TestDrive_TellsTheModelWhenAWatchedRecordChangesElsewhere(t *testing.T) {
	t.Parallel()

	actor := testActor()
	other := pulid.MustNew("usr_")
	feed := &fakeChangeFeed{head: "0.1-0", changes: []realtimeports.WatchedRecordChange{{
		RecordID:    watchedShipment,
		Resource:    "audited:shipment",
		Action:      "updated",
		Fields:      []string{"status"},
		ActorType:   string(permissionports.PrincipalTypeUser),
		ActorUserID: other.String(),
	}}}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		toolTurn("customer_get", map[string]any{"customerId": "cus_1"}),
		textTurn("The shipment moved; I read it again."),
	}}
	rt := newRuntime(completion,
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
			queryTool("customer_get", map[string]any{"name": "Acme"}, nil),
		}},
		&stubActionRegistry{}, nil)
	rt.changes = feed

	turn := rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: testDefinition("customer_get"),
		Actor:      actor,
		Input:      "Is S-1001 ready to bill?",
		Records: []agentpagecontext.EntityRef{
			{Type: "shipment", ID: watchedShipment, Label: "S-1001"},
		},
	})
	require.NotNil(t, turn.world, "a turn with records and a feed watches them")
	fx := &clockedEffects{
		localEffects: &localEffects{
			s:    rt,
			ctx:  t.Context(),
			emit: func(kernelports.StreamEvent) {},
		},
		now: turn.world.CheckedAt,
	}

	result, err := rt.Drive(turn, fx)
	require.NoError(t, err)

	require.Len(t, feed.asked, 1, "the records are checked between steps, once one is due")
	assert.Equal(t, "0.1-0", feed.asked[0].Cursor)
	assert.Contains(t, feed.asked[0].Records, watchedShipment)

	notice := lastUserMessage(completion.Requests[1])
	assert.Contains(t, notice, worldChangePreamble)
	assert.Contains(t, notice, "S-1001")
	assert.Contains(t, notice, "by someone else")
	assert.Contains(t, notice, "status")

	var saved *conversation.Message
	for idx := range result.Messages {
		if result.Messages[idx].Kind == conversation.MessageKindWorldChange {
			saved = &result.Messages[idx]
		}
	}
	require.NotNil(t, saved, "the notice is kept in the conversation")
	require.Len(t, saved.WorldChanges, 1)
	assert.Equal(t, "shipment S-1001", saved.WorldChanges[0].Label)
	assert.Equal(t, "0.2-0", turn.world.Cursor, "the next check starts where this one stopped")

	var announced bool
	for _, event := range fx.events {
		announced = announced || event.Event == assistantports.AssistantEventWorldChanged
	}
	assert.True(t, announced)
}

func TestDrive_DoesNotCheckBeforeACheckIsDue(t *testing.T) {
	t.Parallel()

	feed := &fakeChangeFeed{head: "0.1-0"}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		textTurn("S-1001 is delivered."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	rt.changes = feed
	turn := rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Where is S-1001?",
		Records:    []agentpagecontext.EntityRef{{Type: "shipment", ID: watchedShipment}},
	})
	fx := &clockedEffects{
		localEffects: &localEffects{
			s:    rt,
			ctx:  t.Context(),
			emit: func(kernelports.StreamEvent) {},
		},
		now: turn.world.CheckedAt,
	}

	_, err := rt.Drive(turn, fx)
	require.NoError(t, err)
	assert.Empty(t, feed.asked, "a turn that answers at once never reads the feed")
}

func TestOpenTurn_DelegatedTurnWatchesNothing(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	rt.changes = &fakeChangeFeed{head: "0.1-0"}

	turn := rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Check the carrier.",
		Records:    []agentpagecontext.EntityRef{{Type: "shipment", ID: watchedShipment}},
		Delegation: &agentruntimeports.Delegation{},
	})

	assert.Nil(t, turn.world)
}

func TestWorldState_ReadsWatchTheRecordsAGetCallNamed(t *testing.T) {
	t.Parallel()

	world := openWorld("0.1-0", nil, 100)
	world.read(&llmports.ToolCall{
		Name: "shipment_get",
		Arguments: map[string]any{
			"shipmentId":   watchedShipment,
			"customerID":   otherShipment,
			"includeMoves": true,
			"note":         "shp_01J9NOTANIDNOTANIDNOTANI0",
			"tractorId":    "short",
		},
	}, "S-1001")
	world.read(&llmports.ToolCall{
		Name:      "shipment_list",
		Arguments: map[string]any{"customerId": "cus_01J9LISTLISTLISTLISTLIST0"},
	}, "3 shipments")

	ids := make([]string, 0, len(world.Records))
	for _, record := range world.Records {
		ids = append(ids, record.ID)
	}
	assert.Equal(t, []string{otherShipment, watchedShipment}, ids,
		"only id arguments of a get_ call are watched, in a fixed order")
	assert.Equal(t, "shipment", world.Records[1].Type)
	assert.Equal(t, "S-1001", world.Records[1].Label)
}

func TestWorldState_ItsOwnWriteIsNotNews(t *testing.T) {
	t.Parallel()

	world := openWorld(
		"0.1-0",
		[]agentpagecontext.EntityRef{{Type: "shipment", ID: watchedShipment}},
		100,
	)
	world.wrote(&agentruntimeports.PendingAction{
		Executed: true,
		Target: &agentruntimeports.ProposalTarget{
			Resource: "shipment",
			ID:       pulid.ID(watchedShipment),
		},
	}, 200)
	world.wrote(&agentruntimeports.PendingAction{
		Executed: false,
		Target: &agentruntimeports.ProposalTarget{
			Resource: "shipment",
			ID:       pulid.ID(otherShipment),
		},
	}, 200)

	news := world.news(
		&realtimeports.RecordChanges{Cursor: "0.3-0", Changes: []realtimeports.WatchedRecordChange{
			{RecordID: watchedShipment, Action: "updated", At: 205},
		}},
		210,
	)
	assert.Empty(t, news, "the turn's own write is not a change made elsewhere")
	assert.Len(t, world.Records, 1, "a write only proposed is not watched")

	news = world.news(&realtimeports.RecordChanges{Changes: []realtimeports.WatchedRecordChange{
		{RecordID: watchedShipment, Action: "updated", At: 200 + ownWriteGrace + 1},
	}}, 300)
	assert.Len(t, news, 1, "a later change to the same record is news again")
	assert.Equal(t, "0.3-0", world.Cursor, "an answer without a cursor keeps the last one")
}

func TestWorldState_DueOnlyAfterTheInterval(t *testing.T) {
	t.Parallel()

	world := openWorld("0.1-0", []agentpagecontext.EntityRef{{ID: watchedShipment}}, 100)
	assert.Nil(t, world.due(100+worldCheckEvery-1))
	require.NotNil(t, world.due(100+worldCheckEvery))

	var none *WorldState
	assert.Nil(t, none.due(1000), "a turn that watches nothing is never due")
	assert.Nil(t, openWorld("", nil, 0), "no cursor, no watch")
}

func TestWorldChangeNotice_SaysWhoChangedIt(t *testing.T) {
	t.Parallel()

	actor := testActor()
	notice := WorldChangeNotice([]realtimeports.WatchedRecordChange{
		{RecordID: watchedShipment, Label: "shipment S-1001", Action: "updated",
			ActorType: "session_user", ActorUserID: actor.UserID.String(), Fields: []string{"status"}},
		{RecordID: otherShipment, Label: otherShipment, Action: "deleted", ActorType: "agent"},
	}, actor)

	assert.Contains(t, notice, "shipment S-1001 ("+watchedShipment+") was changed by the person "+
		"you are working for, outside this conversation: status.")
	assert.Contains(t, notice, otherShipment+" was deleted by an agent.")
	assert.NotContains(t, notice, otherShipment+" ("+otherShipment+")")
}

func TestReplayHistory_FramesASteerAndKeepsItInsideItsTurn(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Find a carrier for S-1001."},
		{Role: conversation.RoleAssistant, Content: "Looking."},
		{Role: conversation.RoleUser, Kind: conversation.MessageKindSteer, Content: "Only Werner.",
			Mentions: []agentpagecontext.EntityRef{
				{Type: "carrier", ID: "car_01J9WERNERWERNERWERNER000", Label: "Werner"},
			}},
		{Role: conversation.RoleUser, Kind: conversation.MessageKindWorldChange, Content: "notice"},
		{Role: conversation.RoleAssistant, Content: "Tendered to Werner."},
	}

	messages, _ := replayHistory(history, nil)
	require.Len(t, messages, 6)
	assert.Equal(t, SteerPrompt("Only Werner.", history[2].Mentions), messages[2].Content)
	assert.Equal(t, llmports.RoleUser, messages[2].Role, "the person's words stay theirs")
	assert.Equal(t, llmports.NoticeMessage(steerNotice), messages[3],
		"what to make of them is the system's")
	assert.Equal(t, llmports.NoticeMessage("notice"), messages[4])
	assert.Equal(t, 0, recentTurnStart(history, 1),
		"a steer and a notice arrive inside a turn and open none of their own")
}
