package tmstest

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
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
	lists    []tms.FeatureList
	refused  int
	fail     int
	invites  []tms.Invite
	spoiled  map[string]bool
	setUp    bool
	email    *tms.SystemEmail
	logs     logState
	support  supportState
}

func New(t testing.TB, token string) *Server {
	t.Helper()
	s := &Server{token: token, spoiled: map[string]bool{}, logs: newLogState()}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /managed/v1/readings", s.getReadings)
	mux.HandleFunc("PUT /managed/v1/state", s.putState)
	mux.HandleFunc("PUT /managed/v1/features", s.putFeatures)
	mux.HandleFunc("GET /managed/v1/features", s.getFeatures)
	mux.HandleFunc("PUT /managed/v1/first-administrator/invite", s.putInvite)
	mux.HandleFunc("GET /managed/v1/system-email", s.getSystemEmail)
	mux.HandleFunc("PUT /managed/v1/system-email", s.putSystemEmail)
	mux.HandleFunc("DELETE /managed/v1/system-email", s.deleteSystemEmail)
	mux.HandleFunc("GET /managed/v1/log-requests", s.getLogRequests)
	mux.HandleFunc("PUT /managed/v1/log-requests/{id}", s.putLogRequest)
	mux.HandleFunc("POST /managed/v1/support/log-pulls", s.postLogPull)
	s.routeSupport(mux)
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

func (s *Server) Features() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]string, 0, len(s.lists))
	for _, list := range s.lists {
		out = append(out, append([]string{}, list.Features...))
	}
	return out
}

func (s *Server) LastFeatures() ([]string, bool) {
	list, ok := s.LastFeatureList()
	return list.Features, ok
}

func (s *Server) LastFeatureList() (tms.FeatureList, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.lists) == 0 {
		return tms.FeatureList{}, false
	}
	last := s.lists[len(s.lists)-1]
	return tms.FeatureList{
		Features:        append([]string{}, last.Features...),
		PlanURL:         last.PlanURL,
		AddonEndings:    slices.Clone(last.AddonEndings),
		PlanEndsAt:      last.PlanEndsAt,
		FinalBackupDays: last.FinalBackupDays,
	}, true
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

func (s *Server) putFeatures(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Features        []string           `json:"features"`
		PlanURL         *string            `json:"planUrl"`
		AddonEndings    *[]tms.AddonEnding `json:"addonEndings"`
		PlanEndsAt      *time.Time         `json:"planEndsAt"`
		FinalBackupDays int                `json:"finalBackupDays"`
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		problem(w, http.StatusBadRequest, "Invalid feature list")
		return
	}
	var keys map[string]any
	if err = sonic.Unmarshal(raw, &keys); err != nil {
		problem(w, http.StatusBadRequest, "Invalid feature list")
		return
	}
	if _, stated := keys["planEndsAt"]; !stated {
		problem(w, http.StatusBadRequest, "The feature list must state planEndsAt, null when the plan is not ending")
		return
	}
	if err = sonic.Unmarshal(raw, &body); err != nil ||
		body.Features == nil || (body.PlanURL != nil && *body.PlanURL == "") ||
		(body.AddonEndings != nil && len(*body.AddonEndings) > 50) {
		problem(w, http.StatusBadRequest, "Invalid feature list")
		return
	}
	if body.FinalBackupDays < 0 || body.FinalBackupDays > 3650 {
		problem(w, http.StatusBadRequest, "Invalid finalBackupDays")
		return
	}
	list := tms.FeatureList{
		Features:        body.Features,
		PlanEndsAt:      body.PlanEndsAt,
		FinalBackupDays: body.FinalBackupDays,
	}
	if body.PlanURL != nil {
		list.PlanURL = *body.PlanURL
	}
	if body.AddonEndings != nil {
		list.AddonEndings = *body.AddonEndings
	}
	s.mu.Lock()
	s.lists = append(s.lists, list)
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getFeatures(w http.ResponseWriter, _ *http.Request) {
	held, _ := s.LastFeatures()
	if held == nil {
		held = []string{}
	}
	out, err := sonic.Marshal(map[string]any{"features": held})
	if err != nil {
		problem(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
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

func (s *Server) SystemEmail() *tms.SystemEmail {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.email == nil {
		return nil
	}
	held := *s.email
	return &held
}

func (s *Server) getSystemEmail(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	held := tms.HeldSystemEmail{}
	if s.email != nil {
		held = tms.HeldSystemEmail{
			Configured:  true,
			Provider:    s.email.Provider,
			FromAddress: s.email.FromAddress,
			Fingerprint: fingerprint(s.email),
		}
	}
	s.mu.Unlock()
	out, err := sonic.Marshal(held)
	if err != nil {
		problem(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

func (s *Server) putSystemEmail(w http.ResponseWriter, r *http.Request) {
	var sender tms.SystemEmail
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&sender); err != nil ||
		sender.Provider == "" || len(sender.APIKey) < 20 || sender.FromAddress == "" ||
		sender.FromName == "" {
		problem(w, http.StatusBadRequest, "Invalid system email sender")
		return
	}
	s.mu.Lock()
	s.email = &sender
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteSystemEmail(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	s.email = nil
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func fingerprint(sender *tms.SystemEmail) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		"trenova.system-email.v1",
		sender.Provider,
		sender.FromAddress,
		sender.FromName,
		sender.APIKey,
		sender.WebhookSecret,
	}, "\n")))
	return hex.EncodeToString(sum[:])
}

func problem(w http.ResponseWriter, status int, title string) {
	out, _ := sonic.Marshal(map[string]any{"type": "about:blank", "title": title, "status": status})
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_, _ = w.Write(out)
}
