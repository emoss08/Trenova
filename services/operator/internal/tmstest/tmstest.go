package tmstest

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/operator/internal/tms"
)

type Server struct {
	URL string

	mu       sync.Mutex
	token    string
	readings []tms.Reading
	since    []time.Time
	states   []tms.State
	refused  int
	fail     int
	invites  []tms.Invite
	spoiled  map[string]bool
	setUp    bool
}

func New(t testing.TB, token string) *Server {
	t.Helper()
	s := &Server{token: token, spoiled: map[string]bool{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /managed/v1/readings", s.getReadings)
	mux.HandleFunc("PUT /managed/v1/state", s.putState)
	mux.HandleFunc("PUT /managed/v1/first-administrator/invite", s.putInvite)
	srv := httptest.NewServer(s.authenticate(mux))
	t.Cleanup(srv.Close)
	s.URL = srv.URL
	return s
}

func (s *Server) SetToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = token
}

func (s *Server) SetReadings(readings []tms.Reading) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readings = readings
}

func (s *Server) FailNext(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fail = n
}

func (s *Server) Since() []time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]time.Time(nil), s.since...)
}

func (s *Server) States() []tms.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]tms.State(nil), s.states...)
}

func (s *Server) SetAdministratorSetUp(setUp bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setUp = setUp
}

func (s *Server) SpoilInvite(tokenSHA256 string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.spoiled[tokenSHA256] = true
}

func (s *Server) Invites() []tms.Invite {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]tms.Invite(nil), s.invites...)
}

func (s *Server) Refused() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refused
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		ok := s.token != "" && r.Header.Get("Authorization") == "Bearer "+s.token
		if !ok {
			s.refused++
		}
		fail := s.fail > 0
		if fail {
			s.fail--
		}
		s.mu.Unlock()
		switch {
		case !ok:
			problem(w, http.StatusUnauthorized, "Unauthorized")
		case fail:
			problem(w, http.StatusServiceUnavailable, "Unavailable")
		default:
			next.ServeHTTP(w, r)
		}
	})
}

func (s *Server) getReadings(w http.ResponseWriter, r *http.Request) {
	since, err := time.Parse(time.RFC3339, r.URL.Query().Get("since"))
	if err != nil || !since.Equal(since.Truncate(time.Hour)) {
		problem(w, http.StatusBadRequest, "Invalid since")
		return
	}
	s.mu.Lock()
	s.since = append(s.since, since)
	readings := append([]tms.Reading{}, s.readings...)
	s.mu.Unlock()
	out, err := sonic.Marshal(map[string]any{"readings": readings})
	if err != nil {
		problem(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

func (s *Server) putState(w http.ResponseWriter, r *http.Request) {
	var state tms.State
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&state); err != nil ||
		state.Meters == nil {
		problem(w, http.StatusBadRequest, "Invalid state")
		return
	}
	s.mu.Lock()
	s.states = append(s.states, state)
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) putInvite(w http.ResponseWriter, r *http.Request) {
	var invite tms.Invite
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&invite); err != nil ||
		len(invite.TokenSHA256) != 64 || invite.ExpiresAt.IsZero() {
		problem(w, http.StatusBadRequest, "Invalid invite")
		return
	}
	s.mu.Lock()
	s.invites = append(s.invites, invite)
	answer := tms.Administrator{SetUp: s.setUp}
	if !s.setUp && !s.spoiled[invite.TokenSHA256] {
		answer.InviteTokenSHA256 = invite.TokenSHA256
	}
	s.mu.Unlock()
	out, err := sonic.Marshal(answer)
	if err != nil {
		problem(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

func problem(w http.ResponseWriter, status int, title string) {
	out, _ := sonic.Marshal(map[string]any{"type": "about:blank", "title": title, "status": status})
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_, _ = w.Write(out)
}
