package agentruneventservice

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRepo struct {
	mu sync.Mutex

	appended  []*agent.AgentRunEvent
	next      int
	appendErr error
	seqErr    error
}

func (f *fakeRepo) Append(_ context.Context, req repositories.AppendAgentRunEventsRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.appendErr != nil {
		return f.appendErr
	}
	f.appended = append(f.appended, req.Events...)

	return nil
}

func (f *fakeRepo) NextSequence(
	_ context.Context,
	_ pagination.TenantInfo,
	_ string,
	_ pulid.ID,
) (int, error) {
	if f.seqErr != nil {
		return 0, f.seqErr
	}
	if f.next == 0 {
		return 1, nil
	}

	return f.next, nil
}

func (f *fakeRepo) List(
	_ context.Context,
	_ repositories.ListAgentRunEventsRequest,
) ([]*agent.AgentRunEvent, error) {
	return nil, nil
}

func (f *fakeRepo) ListConnection(
	_ context.Context,
	_ *repositories.ListAgentRunEventConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentRunEvent], error) {
	return nil, nil
}

func (f *fakeRepo) Prune(
	_ context.Context,
	_ repositories.PruneAgentRunEventsRequest,
) (int, error) {
	return 0, nil
}

func (f *fakeRepo) rows() []*agent.AgentRunEvent {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]*agent.AgentRunEvent(nil), f.appended...)
}

func newRecorder(t *testing.T, repo *fakeRepo) serviceports.AgentRunEventRecorder {
	t.Helper()

	return New(Params{Logger: zap.NewNop(), Repo: repo})
}

func owner() serviceports.RunStepOwner {
	return serviceports.RunStepOwner{
		Kind: serviceports.RunStepOwnerAgentRun,
		ID:   pulid.MustNew("ar_"),
	}
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

// The order is the whole value of the record. An account read back in the wrong
// order reports the agent doing things it never did.
func TestRecorder_NumbersEventsInTheOrderTheyHappened(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	writer := newRecorder(t, repo).Recorder(t.Context(), tenant(), owner())

	for _, kind := range []string{"accepted", "tool_started", "tool_finished", "done"} {
		writer.Record(t.Context(), serviceports.StreamEvent{Event: kind})
	}
	writer.Flush(t.Context())

	rows := repo.rows()
	require.Len(t, rows, 4)
	for i, row := range rows {
		assert.Equal(t, i+1, row.Sequence)
	}
	assert.Equal(t, "accepted", rows[0].Kind)
	assert.Equal(t, "done", rows[3].Kind)
}

// A retried run opens a fresh writer with no memory of the numbers the last
// attempt used. Starting again at one collides with them, and the unique index
// would then refuse every event of the retry.
func TestRecorder_ResumesNumberingAfterARetry(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{next: 12}
	writer := newRecorder(t, repo).Recorder(t.Context(), tenant(), owner())

	writer.Record(t.Context(), serviceports.StreamEvent{Event: "accepted"})
	writer.Flush(t.Context())

	rows := repo.rows()
	require.Len(t, rows, 1)
	assert.Equal(t, 12, rows[0].Sequence)
}

// The rule the whole design rests on: an agent that finishes its work and then
// dies filing the paperwork is worse than one that files none.
func TestRecorder_AFailedWriteDoesNotReachTheCaller(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{appendErr: errors.New("postgres is gone")}
	writer := newRecorder(t, repo).Recorder(t.Context(), tenant(), owner())

	assert.NotPanics(t, func() {
		writer.Record(t.Context(), serviceports.StreamEvent{Event: "tool_started"})
		writer.Flush(t.Context())
	})
	assert.Empty(t, repo.rows())
}

// Nor does a recorder that could not find its place refuse to be used.
func TestRecorder_SurvivesNotKnowingWhereItLeftOff(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{seqErr: errors.New("postgres is gone")}
	writer := newRecorder(t, repo).Recorder(t.Context(), tenant(), owner())

	require.NotNil(t, writer, "a recorder that cannot resume still has to be callable")
	assert.NotPanics(t, func() {
		writer.Record(t.Context(), serviceports.StreamEvent{Event: "done"})
		writer.Flush(t.Context())
	})
	assert.Empty(t, repo.rows())
}

// A reader that cannot tell a short payload from a shortened one will
// eventually mistake one for the other.
func TestRecorder_SaysWhenAPayloadWasTooLargeToKeep(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	writer := newRecorder(t, repo).Recorder(t.Context(), tenant(), owner())

	writer.Record(t.Context(), serviceports.StreamEvent{
		Event: "tool_finished",
		Data:  map[string]any{"content": strings.Repeat("x", maxPayloadChars+1)},
	})
	writer.Flush(t.Context())

	rows := repo.rows()
	require.Len(t, rows, 1)
	assert.True(t, rows[0].Truncated, "an oversized payload must be marked, not silently shortened")
	assert.NotContains(t, rows[0].Payload, "content")
}

// The tool's call id is lifted out so a start and its finish can be paired
// without opening the payload.
func TestRecorder_LiftsTheCallIDOutOfAToolEvent(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	writer := newRecorder(t, repo).Recorder(t.Context(), tenant(), owner())

	writer.Record(t.Context(), serviceports.StreamEvent{
		Event: "tool_started",
		Data:  serviceports.AssistantToolStartedEvent{CallID: "call_42", Name: "assign_worker"},
	})
	writer.Flush(t.Context())

	rows := repo.rows()
	require.Len(t, rows, 1)
	assert.Equal(t, "call_42", rows[0].CallID)
}

// Both owner kinds share the table, exactly as they share the step ledger.
func TestRecorder_RecordsAConversationTurnToo(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	turn := serviceports.RunStepOwner{
		Kind: serviceports.RunStepOwnerAssistantTurn,
		ID:   pulid.MustNew("at_"),
	}
	writer := newRecorder(t, repo).Recorder(t.Context(), tenant(), turn)

	writer.Record(t.Context(), serviceports.StreamEvent{Event: "delta", Data: "hello"})
	writer.Flush(t.Context())

	rows := repo.rows()
	require.Len(t, rows, 1)
	assert.Equal(t, "AssistantTurn", rows[0].OwnerKind)
}

// A full buffer writes without waiting to be asked, so a long run is not one
// unbounded slice held until it ends.
func TestRecorder_WritesOnceItHasEnoughToBeWorthWriting(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	writer := newRecorder(t, repo).Recorder(t.Context(), tenant(), owner())

	for range flushAt {
		writer.Record(t.Context(), serviceports.StreamEvent{Event: "delta"})
	}

	assert.Len(t, repo.rows(), flushAt, "the buffer writes itself out once full")
}

// Nil is a legitimate recorder: an installation without one still runs agents.
func TestRecordTrajectory_IsSafeWithoutARecorder(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		serviceports.RecordTrajectory(nil, t.Context(), serviceports.StreamEvent{Event: "done"})
		serviceports.FlushTrajectory(nil, t.Context())
	})
}

func TestRecord_KeepsWhenTheEventHappened(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	writer := newRecorder(t, repo).Recorder(t.Context(), pagination.TenantInfo{},
		serviceports.RunStepOwner{Kind: serviceports.RunStepOwnerAgentRun, ID: pulid.MustNew("ar_")})

	started := int64(1_790_000_000)
	writer.Record(t.Context(), serviceports.StreamEvent{
		Event: serviceports.AssistantEventToolStarted,
		Data:  serviceports.AssistantToolStartedEvent{CallID: "call_1"},
		At:    started,
	})
	writer.Record(t.Context(), serviceports.StreamEvent{
		Event: serviceports.AssistantEventToolFinished,
		Data:  serviceports.AssistantToolFinishedEvent{CallID: "call_1"},
		At:    started + 95,
	})
	writer.Record(t.Context(), serviceports.StreamEvent{Event: serviceports.AssistantEventDone})
	writer.Flush(t.Context())

	rows := repo.rows()
	require.Len(t, rows, 3)
	assert.Equal(t, started, rows[0].OccurredAt, "a durable run's events keep their own times")
	assert.Equal(t, started+95, rows[1].OccurredAt)
	assert.Greater(t, rows[2].OccurredAt, started+95,
		"an event from before times were kept is stamped when it is written")
}
