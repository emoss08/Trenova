package assistantturnservice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// recordingRealtime remembers every announcement, and fails them all when told
// to.
type recordingRealtime struct {
	mu        sync.Mutex
	published []*serviceports.PublishResourceInvalidationRequest
	err       error
}

func (r *recordingRealtime) PublishResourceInvalidation(
	_ context.Context,
	req *serviceports.PublishResourceInvalidationRequest,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.published = append(r.published, req)

	return r.err
}

func (r *recordingRealtime) actions() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]string, 0, len(r.published))
	for _, req := range r.published {
		out = append(out, req.Action)
	}

	return out
}

func announcing(turns repositories.AssistantTurnRepository) (*Service, *recordingRealtime) {
	realtime := &recordingRealtime{}

	return &Service{l: zap.NewNop(), turns: turns, realtime: realtime}, realtime
}

func announced(t *testing.T, req *serviceports.PublishResourceInvalidationRequest) TurnAnnouncement {
	t.Helper()

	entity, ok := req.Entity.(TurnAnnouncement)
	require.True(t, ok, "the announcement carries a TurnAnnouncement")

	return entity
}

// A tab that did not start a reply learns one began without polling, once the
// execution carrying it is recorded.
func TestStartTurn_AnnouncesTheTurnOnceItsWorkflowIsRecorded(t *testing.T) {
	t.Parallel()

	turns := &recordingTurns{}
	svc, realtime := announcing(turns)
	req := startRequest()

	turn, err := svc.StartTurn(t.Context(), req,
		func(*conversation.AssistantTurn) (string, error) {
			return "assistant-turn:atrn_1", nil
		},
	)
	require.NoError(t, err)

	require.Len(t, realtime.published, 1)
	published := realtime.published[0]
	assert.Equal(t, TurnsResource, published.Resource)
	assert.Equal(t, "started", published.Action)
	assert.Equal(t, req.TenantInfo.OrgID, published.OrganizationID)
	assert.Equal(t, req.TenantInfo.BuID, published.BusinessUnitID)
	assert.Equal(t, turn.ID, published.RecordID)
	assert.Equal(t, req.UserID, published.AudienceUserID)

	entity := announced(t, published)
	assert.Equal(t, turn.ID, entity.TurnID)
	assert.Equal(t, req.ThreadID, entity.ThreadID)
	assert.Equal(t, req.UserID, entity.UserID)
	assert.Equal(t, conversation.AssistantTurnStatusRunning, entity.Status)
}

// A turn that never reached a worker is closed, and its close is announced
// like any other, so no tab is left showing a reply being written.
func TestStartTurn_AnnouncesATurnThatCouldNotStartAsFinished(t *testing.T) {
	t.Parallel()

	svc, realtime := announcing(&recordingTurns{})

	_, err := svc.StartTurn(t.Context(), startRequest(),
		func(*conversation.AssistantTurn) (string, error) {
			return "", errors.New("temporal is unreachable")
		},
	)
	require.Error(t, err)

	assert.Equal(t, []string{"finished"}, realtime.actions())
	assert.Equal(t, conversation.AssistantTurnStatusFailed,
		announced(t, realtime.published[0]).Status)
}

// The announcement is a hint. A realtime outage costs a tab a refresh, never
// the reply.
func TestStartTurn_StartsWhenTheAnnouncementFails(t *testing.T) {
	t.Parallel()

	turns := &recordingTurns{}
	svc, realtime := announcing(turns)
	realtime.err = errors.New("realtime is down")

	turn, err := svc.StartTurn(t.Context(), startRequest(),
		func(*conversation.AssistantTurn) (string, error) {
			return "assistant-turn:atrn_1", nil
		},
	)

	require.NoError(t, err)
	assert.NotNil(t, turn)
	assert.Empty(t, turns.completed)
}

func TestComplete_AnnouncesHowTheTurnEnded(t *testing.T) {
	t.Parallel()

	svc, realtime := announcing(&recordingTurns{})
	turn := runningTurn()
	turn.UserID = pulid.MustNew("usr_")

	svc.Complete(t.Context(), turn, conversation.AssistantTurnStatusCompleted, nil)

	require.Len(t, realtime.published, 1)
	assert.Equal(t, "finished", realtime.published[0].Action)
	entity := announced(t, realtime.published[0])
	assert.Equal(t, turn.ID, entity.TurnID)
	assert.Equal(t, turn.ThreadID, entity.ThreadID)
	assert.Equal(t, turn.UserID, entity.UserID)
	assert.Equal(t, conversation.AssistantTurnStatusCompleted, entity.Status)
}

// failingTurns cannot close anything.
type failingTurns struct {
	recordingTurns
}

func (f *failingTurns) Complete(context.Context, repositories.CompleteAssistantTurnRequest) error {
	return errors.New("database is down")
}

// A record that did not close did not finish, and saying it had would have a
// tab stop showing a reply that is still holding the conversation.
func TestClose_DoesNotAnnounceARecordItCouldNotClose(t *testing.T) {
	t.Parallel()

	svc, realtime := announcing(&failingTurns{})

	err := svc.Close(t.Context(), runningTurn(), conversation.AssistantTurnStatusFailed, "db")

	require.Error(t, err)
	assert.Empty(t, realtime.published)
}

// A stop that finds no execution closes the record itself, so it is the one
// that announces the end.
func TestStop_AnnouncesATurnItClosedItself(t *testing.T) {
	t.Parallel()

	svc, realtime := announcing(&recordingTurns{})
	svc.canceller = serviceports.AssistantTurnCancellerFunc(
		func(context.Context, string) error { return ErrNoExecution },
	)

	require.NoError(t, svc.Stop(t.Context(), runningTurn()))

	assert.Equal(t, []string{"finished"}, realtime.actions())
	assert.Equal(t, conversation.AssistantTurnStatusStopped,
		announced(t, realtime.published[0]).Status)
}

// A stop that reached the execution leaves closing, and announcing, to it.
func TestStop_LeavesTheAnnouncementToAnExecutionItCancelled(t *testing.T) {
	t.Parallel()

	svc, realtime := announcing(&recordingTurns{})
	svc.canceller = serviceports.AssistantTurnCancellerFunc(
		func(context.Context, string) error { return nil },
	)

	require.NoError(t, svc.Stop(t.Context(), runningTurn()))

	assert.Empty(t, realtime.published)
}

// liveTurns is a person with several replies in progress.
type liveTurns struct {
	recordingTurns

	live  []*repositories.LiveAssistantTurn
	asked repositories.ListLiveAssistantTurnsRequest
}

func (l *liveTurns) ListLive(
	_ context.Context,
	req repositories.ListLiveAssistantTurnsRequest,
) ([]*repositories.LiveAssistantTurn, error) {
	l.asked = req

	return l.live, nil
}

func liveTurn() *repositories.LiveAssistantTurn {
	turn := runningTurn()
	turn.WorkflowID = conversation.AssistantTurnWorkflowID(turn.ID)

	return &repositories.LiveAssistantTurn{AssistantTurn: *turn}
}

/*
Signing out stops every reply the person has in progress. One that cannot be
stopped does not keep the rest running, one whose execution is gone is closed
by hand, and the failure is still reported.
*/
func TestStopAllForUser_StopsEveryReplyAndReportsTheOnesItCouldNot(t *testing.T) {
	t.Parallel()

	cancelled, gone, stuck := liveTurn(), liveTurn(), liveTurn()
	turns := &liveTurns{live: []*repositories.LiveAssistantTurn{cancelled, gone, stuck}}
	svc := &Service{l: zap.NewNop(), turns: turns}

	var asked []string
	svc.canceller = serviceports.AssistantTurnCancellerFunc(
		func(_ context.Context, workflowID string) error {
			asked = append(asked, workflowID)
			switch workflowID {
			case gone.ExecutionID():
				return fmt.Errorf("cancel: %w", ErrNoExecution)
			case stuck.ExecutionID():
				return errors.New("temporal is down")
			default:
				return nil
			}
		},
	)

	req := serviceports.StopUserTurnsRequest{
		UserID: pulid.MustNew("usr_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}
	err := svc.StopAllForUser(t.Context(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), stuck.ID.String())
	assert.Equal(t, req.UserID, turns.asked.UserID, "only the person's own replies are read")
	assert.Equal(t, req.TenantInfo, turns.asked.TenantInfo)
	assert.Equal(t,
		[]string{cancelled.ExecutionID(), gone.ExecutionID(), stuck.ExecutionID()},
		asked,
		"every reply is asked to stop, whatever happened to the one before it",
	)
	require.Len(t, turns.completed, 1)
	assert.Equal(t, gone.ID, turns.completed[0].ID)
	assert.Equal(t, conversation.AssistantTurnStatusStopped, turns.completed[0].Status)
}

func TestStopAllForUser_HasNothingToDoForSomeoneWithNoReplies(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop(), turns: &liveTurns{}}
	svc.canceller = serviceports.AssistantTurnCancellerFunc(
		func(context.Context, string) error {
			t.Fatal("nothing is in progress, so nothing is cancelled")

			return nil
		},
	)

	require.NoError(t, svc.StopAllForUser(t.Context(), serviceports.StopUserTurnsRequest{
		UserID: pulid.MustNew("usr_"),
	}))
}
