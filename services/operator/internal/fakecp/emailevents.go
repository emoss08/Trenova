package fakecp

import (
	"net/http"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const maxEmailEvents = 50

type emailEventState struct {
	queued   []instanceapi.EmailEvent
	acks     [][]uuid.UUID
	listings int
}

func (s *Server) QueueEmailEvent(event instanceapi.EmailEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emailEvents.queued = append(s.emailEvents.queued, event)
}

func (s *Server) QueuedEmailEvents() []instanceapi.EmailEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.emailEvents.queued)
}

func (s *Server) EmailEventAcks() [][]uuid.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]uuid.UUID, 0, len(s.emailEvents.acks))
	for _, ack := range s.emailEvents.acks {
		out = append(out, slices.Clone(ack))
	}
	return out
}

func (s *Server) EmailEventListings() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.emailEvents.listings
}

func (s *Server) emailEventsWaiting() bool {
	return len(s.emailEvents.queued) > 0
}

func (s *Server) listEmailEvents(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	s.emailEvents.listings++
	events := slices.Clone(s.emailEvents.queued[:min(len(s.emailEvents.queued), maxEmailEvents)])
	s.mu.Unlock()
	if events == nil {
		events = []instanceapi.EmailEvent{}
	}
	w.Header().Set("Cache-Control", "no-store")
	write(w, instanceapi.EmailEventList{Events: events})
}

func (s *Server) ackEmailEvents(w http.ResponseWriter, r *http.Request) {
	var ack instanceapi.EmailEventAck
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&ack); err != nil ||
		len(ack.Ids) == 0 || len(ack.Ids) > maxEmailEvents || !unique(ack.Ids) {
		problem(w, http.StatusBadRequest, "validation_failed")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emailEvents.acks = append(s.emailEvents.acks, slices.Clone(ack.Ids))
	s.emailEvents.queued = slices.DeleteFunc(
		s.emailEvents.queued,
		func(e instanceapi.EmailEvent) bool {
			return slices.Contains(ack.Ids, e.Id)
		},
	)
	w.WriteHeader(http.StatusNoContent)
}

func unique(ids []uuid.UUID) bool {
	seen := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}
