// Package agentruneventservice keeps what a run said about itself.
//
// The runtime already narrates its own work — every tool it reaches for, every
// refusal, every give-up — to whoever is listening. Until now the listeners were
// a redis stream that is dropped a quarter of an hour after a reply ends, and,
// for a background run, a function that used the events as a heartbeat tick and
// threw them away. This package is the listener that writes them down.
//
// Two rules shape everything here. Recording must never be why a run fails: a
// write that cannot happen is logged and dropped, because an agent that
// finishes its work and then dies filing the paperwork is worse than one that
// files none. And recording must not slow the run down: events are buffered and
// written in batches, since a run emits them far faster than a round trip to
// postgres.
package agentruneventservice

import (
	"context"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// maxPayloadChars bounds one event's payload.
//
// The same reasoning as the step ledger's outcome bound, and deliberately the
// same size: a query tool's answer can be a whole report, and keeping every one
// of those verbatim would fill this table with material the transcript already
// holds. Past the bound the payload is replaced rather than clipped, and the
// row says it was, because a reader that cannot tell a short payload from a
// shortened one will eventually mistake one for the other.
const maxPayloadChars = 64 * 1024

// flushAt is how many events are held before a write.
//
// Small enough that a crash loses little, large enough that a chatty run is not
// writing a row per token.
const flushAt = 32

type Params struct {
	fx.In

	Logger  *zap.Logger
	Repo    repositories.AgentRunEventRepository
	Metrics *metrics.Registry `optional:"true"`
}

type Service struct {
	l       *zap.Logger
	repo    repositories.AgentRunEventRepository
	metrics *metrics.Assistant
}

var _ serviceports.AgentRunEventRecorder = (*Service)(nil)

func New(p Params) serviceports.AgentRunEventRecorder {
	service := &Service{
		l:    p.Logger.Named("service.agentrunevent"),
		repo: p.Repo,
	}
	if p.Metrics != nil {
		service.metrics = p.Metrics.Assistant
	}

	return service
}

// Recorder opens a writer for one run or turn.
//
// The sequence is read back from what is already stored rather than started at
// one, because a retried run starts a fresh writer with no memory of the
// numbers the last attempt used. Starting again at one would collide with them,
// and the unique index would refuse every event of the retry.
func (s *Service) Recorder(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	owner serviceports.RunStepOwner,
) serviceports.AgentRunEventWriter {
	next, err := s.repo.NextSequence(ctx, tenantInfo, string(owner.Kind), owner.ID)
	if err != nil {
		// A recorder that cannot find its place still accepts events and drops
		// them, rather than handing back nil for every caller to check.
		s.l.Warn("could not resume the event sequence; this run will not be recorded",
			zap.String("owner", owner.ID.String()),
			zap.Error(err),
		)

		return &writer{disabled: true}
	}

	return &writer{
		service:  s,
		tenant:   tenantInfo,
		owner:    owner,
		sequence: next,
	}
}

type writer struct {
	service  *Service
	tenant   pagination.TenantInfo
	owner    serviceports.RunStepOwner
	disabled bool

	mu       sync.Mutex
	sequence int
	buffered []*agent.AgentRunEvent
}

// Record files one event. It never blocks on postgres except when the buffer
// fills, and never returns an error, because no caller has anything useful to
// do with one.
func (w *writer) Record(ctx context.Context, event serviceports.StreamEvent) {
	if w == nil || w.disabled {
		return
	}

	row := w.rowFor(event)
	if row == nil {
		return
	}

	w.mu.Lock()
	w.buffered = append(w.buffered, row)
	full := len(w.buffered) >= flushAt
	w.mu.Unlock()

	if full {
		w.Flush(ctx)
	}
}

// Flush writes what is held. Callers close a run by calling it once more.
func (w *writer) Flush(ctx context.Context) {
	if w == nil || w.disabled {
		return
	}

	w.mu.Lock()
	pending := w.buffered
	w.buffered = nil
	w.mu.Unlock()

	if len(pending) == 0 {
		return
	}

	err := w.service.repo.Append(ctx, repositories.AppendAgentRunEventsRequest{
		TenantInfo: w.tenant,
		Events:     pending,
	})
	if err != nil {
		// Deliberately swallowed. Recording is not what the run is for.
		w.service.l.Error("could not record part of a run's trajectory",
			zap.String("owner", w.owner.ID.String()),
			zap.Int("events", len(pending)),
			zap.Error(err),
		)
		w.service.metrics.RecordTrajectoryDropped(string(w.owner.Kind), len(pending))

		return
	}

	w.service.metrics.RecordTrajectoryWritten(string(w.owner.Kind), len(pending))
}

func (w *writer) rowFor(event serviceports.StreamEvent) *agent.AgentRunEvent {
	payload, truncated := encodePayload(event.Data)

	w.mu.Lock()
	sequence := w.sequence
	w.sequence++
	w.mu.Unlock()

	return &agent.AgentRunEvent{
		OrganizationID: w.tenant.OrgID,
		BusinessUnitID: w.tenant.BuID,
		OwnerKind:      string(w.owner.Kind),
		OwnerID:        w.owner.ID,
		Sequence:       sequence,
		Kind:           event.Event,
		CallID:         callIDOf(event.Data),
		Payload:        payload,
		Truncated:      truncated,
		OccurredAt:     timeutils.NowUnix(),
	}
}

// encodePayload turns a typed event into something a jsonb column can hold.
//
// The payload is round-tripped through sonic rather than type-switched over
// every event shape, so an event kind added later is recorded without this
// package being taught about it.
func encodePayload(data any) (map[string]any, bool) {
	if data == nil {
		return map[string]any{}, false
	}

	encoded, err := sonic.Marshal(data)
	if err != nil {
		return map[string]any{"unencodable": true}, false
	}

	if len(encoded) > maxPayloadChars {
		return map[string]any{"omitted": "payload too large to keep"}, true
	}

	payload := make(map[string]any)
	if err = sonic.Unmarshal(encoded, &payload); err != nil {
		// Not every event's data is an object — a delta is a bare string. Keep
		// it under a known key rather than losing it.
		return map[string]any{"value": string(encoded)}, false
	}

	return payload, false
}

// callIDOf lifts the tool call id out of the events that carry one, so a tool's
// start and finish can be found without opening the payload.
func callIDOf(data any) string {
	switch typed := data.(type) {
	case serviceports.AssistantToolStartedEvent:
		return typed.CallID
	case *serviceports.AssistantToolStartedEvent:
		return typed.CallID
	case serviceports.AssistantToolFinishedEvent:
		return typed.CallID
	case *serviceports.AssistantToolFinishedEvent:
		return typed.CallID
	default:
		return ""
	}
}
