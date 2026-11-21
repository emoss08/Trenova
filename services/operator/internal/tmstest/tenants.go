package tmstest

import (
	"io"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/operator/internal/tms"
)

type tenantEntry struct {
	InstanceID string `json:"instanceId"`
	Revision   int64  `json:"revision"`
	Lifecycle  string `json:"lifecycle"`
	Commands   []struct {
		Kind     string `json:"kind"`
		ID       string `json:"id"`
		Location string `json:"location"`
	} `json:"commands"`
	ImportTarget *tms.ArchiveTarget `json:"importTarget"`
}

type tenantState struct {
	shared    bool
	statuses  map[string]tms.TenantStatus
	entries   map[string][]byte
	puts      map[string]int
	refuse    map[string]string
	readings  map[string][]tms.TenantReading
	asked     []time.Time
	usage     [][]tms.TenantUsage
	fail      int
	commands  map[string]*heldCommand
	refuseCmd map[string]string
	busy      map[string]bool
	nightly   []tms.NightlyGrant
	pending   map[string]tenantEntry
	posted    map[string]int

	heldNightly    *tms.NightlyGrant
	nightlyDeletes int
	refuseNightly  string
}

func newTenantState() tenantState {
	return tenantState{
		shared:    true,
		statuses:  map[string]tms.TenantStatus{},
		entries:   map[string][]byte{},
		puts:      map[string]int{},
		refuse:    map[string]string{},
		readings:  map[string][]tms.TenantReading{},
		commands:  map[string]*heldCommand{},
		refuseCmd: map[string]string{},
		busy:      map[string]bool{},
		pending:   map[string]tenantEntry{},
		posted:    map[string]int{},
	}
}

func (s *Server) routeTenants(mux *http.ServeMux) {
	mux.HandleFunc("GET /managed/v1/tenants", s.getTenants)
	mux.HandleFunc("GET /managed/v1/tenants/readings", s.getTenantReadings)
	mux.HandleFunc("PUT /managed/v1/tenants/usage", s.putTenantUsage)
	mux.HandleFunc("PUT /managed/v1/tenants/{instanceId}", s.putTenant)
	s.routeArchives(mux)
}

func (s *Server) SetDedicated() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants.shared = false
}

func (s *Server) FailTenants(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants.fail = n
}

func (s *Server) RefuseTenant(instanceID, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants.refuse[instanceID] = reason
}

func (s *Server) SetTenantReadings(instanceID string, readings []tms.TenantReading) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants.readings[instanceID] = slices.Clone(readings)
}

func (s *Server) TenantPuts(instanceID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tenants.puts[instanceID]
}

func (s *Server) TenantStatus(instanceID string) (tms.TenantStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status, ok := s.tenants.statuses[instanceID]
	return status, ok
}

func (s *Server) TenantEntry(instanceID string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.tenants.entries[instanceID])
}

func (s *Server) TenantUsage() [][]tms.TenantUsage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.tenants.usage)
}

func (s *Server) TenantReadingsAsked() []time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.tenants.asked)
}

func (s *Server) failing(w http.ResponseWriter) bool {
	if s.tenants.fail > 0 {
		s.tenants.fail--
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return true
	}
	if !s.tenants.shared {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"title":"Conflict","detail":"not a shared server"}`))
		return true
	}
	return false
}

func (s *Server) putTenant(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	var entry tenantEntry
	if err != nil || sonic.Unmarshal(body, &entry) != nil ||
		entry.InstanceID != r.PathValue("instanceId") {
		http.Error(w, "bad entry", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failing(w) {
		return
	}
	s.tenants.puts[entry.InstanceID]++
	s.tenants.entries[entry.InstanceID] = body
	status, known := s.tenants.statuses[entry.InstanceID]
	status.InstanceID = entry.InstanceID
	if s.importing(w, &entry, &status, known) {
		return
	}
	if reason, refused := s.tenants.refuse[entry.InstanceID]; refused {
		if status.State == "" {
			status.State = "applying"
		}
		status.Refused = &tms.TenantRefusal{
			Revision: entry.Revision,
			Code:     "invalid",
			Reason:   reason,
		}
		s.tenants.statuses[entry.InstanceID] = status
		writeJSON(w, status)
		return
	}
	revision := entry.Revision
	status.AppliedRevision = &revision
	status.Refused = nil
	switch entry.Lifecycle {
	case "suspended":
		status.State = "suspended"
	case "removed":
		status.State = "removed"
		status.Administrator = nil
	default:
		status.State = "active"
	}
	if status.State != "removed" && status.Administrator == nil {
		status.Administrator = &tms.TenantAdministrator{}
	}
	s.tenants.statuses[entry.InstanceID] = status
	writeJSON(w, status)
}

func (s *Server) getTenants(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failing(w) {
		return
	}
	ids := make([]string, 0, len(s.tenants.statuses))
	after := r.URL.Query().Get("after")
	for id := range s.tenants.statuses {
		if id > after {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	page := struct {
		Tenants []tms.TenantStatus `json:"tenants"`
		Next    string             `json:"next,omitempty"`
	}{Tenants: []tms.TenantStatus{}}
	for i, id := range ids {
		if i == limit {
			page.Next = ids[i-1]
			break
		}
		page.Tenants = append(page.Tenants, s.tenants.statuses[id])
	}
	writeJSON(w, page)
}

func (s *Server) getTenantReadings(w http.ResponseWriter, r *http.Request) {
	from, ferr := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, terr := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if ferr != nil || terr != nil || from.After(to) {
		http.Error(w, "bad window", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failing(w) {
		return
	}
	s.tenants.asked = append(s.tenants.asked, from)
	page := struct {
		Tenants []tms.TenantReadings `json:"tenants"`
	}{Tenants: []tms.TenantReadings{}}
	ids := make([]string, 0, len(s.tenants.statuses))
	for id, status := range s.tenants.statuses {
		if status.State == "active" || status.State == "suspended" {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	for _, id := range ids {
		entry := tms.TenantReadings{InstanceID: id, Readings: []tms.TenantReading{}}
		for _, rd := range s.tenants.readings[id] {
			if !rd.Hour.Before(from) && !rd.Hour.After(to) {
				entry.Readings = append(entry.Readings, rd)
			}
		}
		page.Tenants = append(page.Tenants, entry)
	}
	writeJSON(w, page)
}

func (s *Server) putTenantUsage(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Tenants []tms.TenantUsage `json:"tenants"`
	}
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&in); err != nil || in.Tenants == nil {
		http.Error(w, "bad usage", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failing(w) {
		return
	}
	s.tenants.usage = append(s.tenants.usage, in.Tenants)
	answer := tms.TenantUsageAnswer{Refused: []tms.TenantUsageRefused{}}
	for _, t := range in.Tenants {
		if _, ok := s.tenants.statuses[t.InstanceID]; ok {
			answer.Applied++
			continue
		}
		answer.Refused = append(answer.Refused, tms.TenantUsageRefused{
			InstanceID: t.InstanceID,
			Code:       "not_placed",
		})
	}
	writeJSON(w, answer)
}
