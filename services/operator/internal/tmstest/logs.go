package tmstest

import (
	"net/http"
	"slices"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/operator/internal/tms"
)

type LogPull struct {
	Email string
	Kind  tms.LogPullKind
}

type LogUpdate struct {
	ID     string
	Update tms.LogRequestUpdate
}

type logState struct {
	requests []tms.LogRequest
	status   map[string]tms.LogRequestStatus
	updates  []LogUpdate
	grants   map[string]bool
	pulls    []LogPull
	pullFail int
}

func newLogState() logState {
	return logState{status: map[string]tms.LogRequestStatus{}, grants: map[string]bool{}}
}

func (s *Server) AddLogRequest(req tms.LogRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs.requests = append(s.logs.requests, req)
}

func (s *Server) Grant(email string, live bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs.grants[email] = live
}

func (s *Server) FailLogPullsNext(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs.pullFail = n
}

func (s *Server) LogPulls() []LogPull {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.logs.pulls)
}

func (s *Server) LogUpdates() []LogUpdate {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.logs.updates)
}

func (s *Server) LoseLogRequestStatus(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs.status[id] = tms.LogRequestCollecting
}

func (s *Server) LogRequestStatus(id string) tms.LogRequestStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.logs.status[id]
}

func (s *Server) getLogRequests(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	pending := []tms.LogRequest{}
	for _, req := range s.logs.requests {
		switch status := s.logs.status[req.ID]; status {
		case tms.LogRequestSent, tms.LogRequestFailed:
		case "":
			req.Status = tms.LogRequestRequested
			pending = append(pending, req)
		default:
			req.Status = status
			pending = append(pending, req)
		}
	}
	s.mu.Unlock()
	if len(pending) > 10 {
		pending = pending[:10]
	}
	out, err := sonic.Marshal(map[string]any{"requests": pending})
	if err != nil {
		problem(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

func (s *Server) putLogRequest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var update tms.LogRequestUpdate
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&update); err != nil {
		problem(w, http.StatusBadRequest, "Invalid update")
		return
	}
	switch update.Status {
	case tms.LogRequestCollecting, tms.LogRequestSent:
	case tms.LogRequestFailed:
		if update.Message == "" {
			problem(w, http.StatusBadRequest, "A failure needs a message")
			return
		}
	default:
		problem(w, http.StatusBadRequest, "Invalid status")
		return
	}
	if update.Status == tms.LogRequestSent && update.BundleID == "" {
		problem(w, http.StatusBadRequest, "A sent request needs its bundle")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	known := slices.ContainsFunc(s.logs.requests, func(req tms.LogRequest) bool { return req.ID == id })
	if !known {
		problem(w, http.StatusNotFound, "Not found")
		return
	}
	current := s.logs.status[id]
	switch {
	case current == update.Status:
		w.WriteHeader(http.StatusNoContent)
		return
	case rank(update.Status) <= rank(current):
		problem(w, http.StatusConflict, "The request has moved past that status")
		return
	}
	s.logs.status[id] = update.Status
	s.logs.updates = append(s.logs.updates, LogUpdate{ID: id, Update: update})
	w.WriteHeader(http.StatusNoContent)
}

func rank(status tms.LogRequestStatus) int {
	switch status {
	case tms.LogRequestCollecting:
		return 1
	case tms.LogRequestSent, tms.LogRequestFailed:
		return 2
	default:
		return 0
	}
}

func (s *Server) postLogPull(w http.ResponseWriter, r *http.Request) {
	var pull struct {
		Email string          `json:"email"`
		Kind  tms.LogPullKind `json:"kind"`
	}
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&pull); err != nil || pull.Email == "" ||
		(pull.Kind != tms.LogPullLines && pull.Kind != tms.LogPullBundle) {
		problem(w, http.StatusBadRequest, "Invalid log pull")
		return
	}
	s.mu.Lock()
	fail := s.logs.pullFail > 0
	if fail {
		s.logs.pullFail--
	}
	live := s.logs.grants[pull.Email]
	if !fail {
		s.logs.pulls = append(s.logs.pulls, LogPull{Email: pull.Email, Kind: pull.Kind})
	}
	s.mu.Unlock()
	switch {
	case fail:
		problem(w, http.StatusServiceUnavailable, "Unavailable")
	case !live:
		w.WriteHeader(http.StatusForbidden)
	default:
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}
}
