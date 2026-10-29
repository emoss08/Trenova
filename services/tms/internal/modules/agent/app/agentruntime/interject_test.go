package agentruntime

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/worldwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const watchedRecordID = "shp_01J9WATCHWATCHWATCHWATCH0"

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
) (modelstream.Reply, error) {
	reply, err := fx.localEffects.Complete(t, req)
	fx.now += worldwatch.WorldCheckEvery + 1

	return reply, err
}

func TestDrive_TellsTheModelWhenAWatchedRecordChangesElsewhere(t *testing.T) {
	t.Parallel()

	actor := runtimetest.Actor()
	other := pulid.MustNew("usr_")
	feed := &fakeChangeFeed{head: "0.1-0", changes: []realtimeports.WatchedRecordChange{{
		RecordID:    watchedRecordID,
		Resource:    "audited:shipment",
		Action:      "updated",
		Fields:      []string{"status"},
		ActorType:   string(permissionports.PrincipalTypeUser),
		ActorUserID: other.String(),
	}}}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("customer_get", map[string]any{"customerId": "cus_1"}),
		runtimetest.TextTurn("The shipment moved; I read it again."),
	}}
	rt := newRuntime(completion,
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
			queryTool("customer_get", map[string]any{"name": "Acme"}, nil),
		}},
		&stubActionRegistry{}, nil)
	rt.changes = feed

	turn := rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("customer_get"),
		Actor:      actor,
		Input:      "Is S-1001 ready to bill?",
		Records: []agentpagecontext.EntityRef{
			{Type: "shipment", ID: watchedRecordID, Label: "S-1001"},
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
	assert.Contains(t, feed.asked[0].Records, watchedRecordID)

	notice := lastUserMessage(completion.Requests[1])
	assert.Contains(t, notice, worldwatch.ChangePreamble)
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
		announced = announced || event.Event == assistantstreamports.AssistantEventWorldChanged
	}
	assert.True(t, announced)
}

func TestDrive_DoesNotCheckBeforeACheckIsDue(t *testing.T) {
	t.Parallel()

	feed := &fakeChangeFeed{head: "0.1-0"}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.TextTurn("S-1001 is delivered."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	rt.changes = feed
	turn := rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		Input:      "Where is S-1001?",
		Records:    []agentpagecontext.EntityRef{{Type: "shipment", ID: watchedRecordID}},
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
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		Input:      "Check the carrier.",
		Records:    []agentpagecontext.EntityRef{{Type: "shipment", ID: watchedRecordID}},
		Delegation: &agentruntimeports.Delegation{},
	})

	assert.Nil(t, turn.world)
}
