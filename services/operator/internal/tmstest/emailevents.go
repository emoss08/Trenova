package tmstest

import (
	"net/http"
	"slices"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/operator/internal/tms"
)

const maxEmailEvents = 50

type emailEventState struct {
	received  []tms.EmailEvent
	unhandled map[string]bool
	fail      int
}

func newEmailEventState() emailEventState {
	return emailEventState{unhandled: map[string]bool{}}
}

func (s *Server) EmailEvents() []tms.EmailEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.emailEvents.received)
}

func (s *Server) LeaveEmailEventsUnhandled(ids ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		s.emailEvents.unhandled[id] = true
	}
}

func (s *Server) HandleEmailEvents(ids ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		delete(s.emailEvents.unhandled, id)
	}
}

func (s *Server) FailEmailEventsNext(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emailEvents.fail = n
}

func (s *Server) postEmailEvents(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Events []tms.EmailEvent `json:"events"`
	}
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&body); err != nil ||
		len(body.Events) == 0 || len(body.Events) > maxEmailEvents {
		problem(w, http.StatusBadRequest, "Invalid system email events")
		return
	}
	for _, e := range body.Events {
		if e.ID == "" || e.Payload == "" || (e.Provider != "Resend" && e.Provider != "Postmark") {
			problem(w, http.StatusBadRequest, "Invalid system email event")
			return
		}
	}
	s.mu.Lock()
	if s.emailEvents.fail > 0 {
		s.emailEvents.fail--
		s.mu.Unlock()
		problem(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	handled := make([]string, 0, len(body.Events))
	for _, e := range body.Events {
		s.emailEvents.received = append(s.emailEvents.received, e)
		if !s.emailEvents.unhandled[e.ID] {
			handled = append(handled, e.ID)
		}
	}
	s.mu.Unlock()
	out, err := sonic.Marshal(map[string]any{"handled": handled})
	if err != nil {
		problem(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}
