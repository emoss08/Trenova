package fakecp

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

type sharedState struct {
	tenants     []instanceapi.SharedTenant
	systemEmail *instanceapi.SpecSystemEmail
	listings    int
	unchanged   int
	reports     []instanceapi.SharedStatusReport
	usage       []instanceapi.SharedUsageReport
	answers     map[uuid.UUID]instanceapi.SharedTenantUsage
	usageFail   int
}

func newSharedState() sharedState {
	return sharedState{answers: map[uuid.UUID]instanceapi.SharedTenantUsage{}}
}

func (s *Server) routeShared(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/shared/tenants", s.sharedOnly(s.listSharedTenants))
	mux.HandleFunc("PUT /v1/shared/status", s.sharedOnly(s.reportSharedStatus))
	mux.HandleFunc("POST /v1/shared/usage", s.sharedOnly(s.reportSharedUsage))
}

func (s *Server) sharedOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.Kind != credential.KindSharedServer {
			problem(w, http.StatusForbidden, "forbidden")
			return
		}
		next(w, r)
	}
}

func (s *Server) SetSharedTenants(tenants []instanceapi.SharedTenant) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shared.tenants = slices.Clone(tenants)
	slices.SortFunc(s.shared.tenants, func(a, b instanceapi.SharedTenant) int {
		return compareUUID(a.InstanceId, b.InstanceId)
	})
}

func (s *Server) SetSharedSystemEmail(e *instanceapi.SpecSystemEmail) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shared.systemEmail = e
}

func (s *Server) SetSharedUsageAnswer(answer instanceapi.SharedTenantUsage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shared.answers[answer.InstanceId] = answer
}

func (s *Server) FailSharedUsage(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shared.usageFail = n
}

func (s *Server) SharedListings() (listed, unchanged int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shared.listings, s.shared.unchanged
}

func (s *Server) SharedReports() []instanceapi.SharedStatusReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.shared.reports)
}

func (s *Server) LastSharedReport() (instanceapi.SharedStatusReport, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.shared.reports) == 0 {
		return instanceapi.SharedStatusReport{}, false
	}
	return s.shared.reports[len(s.shared.reports)-1], true
}

func (s *Server) SharedUsageReports() []instanceapi.SharedUsageReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.shared.usage)
}

func (s *Server) listSharedTenants(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	list := instanceapi.SharedTenantList{
		Tenants:     slices.Clone(s.shared.tenants),
		SystemEmail: s.shared.systemEmail,
	}
	s.shared.listings++
	s.mu.Unlock()
	if list.Tenants == nil {
		list.Tenants = []instanceapi.SharedTenant{}
	}
	body, err := sonic.Marshal(list)
	if err != nil {
		problem(w, http.StatusInternalServerError, "internal")
		return
	}
	sum := sha256.Sum256(body)
	etag := `"` + hex.EncodeToString(sum[:]) + `"`
	if r.Header.Get("If-None-Match") == etag {
		s.mu.Lock()
		s.shared.unchanged++
		s.mu.Unlock()
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (s *Server) placed(id uuid.UUID) bool {
	for _, t := range s.shared.tenants {
		if t.InstanceId == id {
			return true
		}
	}
	return false
}

func (s *Server) reportSharedStatus(w http.ResponseWriter, r *http.Request) {
	var report instanceapi.SharedStatusReport
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&report); err != nil {
		problem(w, http.StatusBadRequest, "invalid_request")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shared.reports = append(s.shared.reports, report)
	ack := instanceapi.SharedReportAck{Refused: []instanceapi.SharedEntryRefused{}}
	for _, t := range report.Tenants {
		if !s.placed(t.InstanceId) {
			ack.Refused = append(ack.Refused, instanceapi.SharedEntryRefused{
				InstanceId: t.InstanceId,
				Code:       instanceapi.SharedEntryNotPlaced,
			})
		}
	}
	write(w, ack)
}

func (s *Server) reportSharedUsage(w http.ResponseWriter, r *http.Request) {
	var report instanceapi.SharedUsageReport
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&report); err != nil {
		problem(w, http.StatusBadRequest, "invalid_request")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shared.usageFail > 0 {
		s.shared.usageFail--
		problem(w, http.StatusServiceUnavailable, "unavailable")
		return
	}
	if len(report.Readings) > 5000 {
		problem(w, http.StatusBadRequest, "validation_failed")
		return
	}
	s.shared.usage = append(s.shared.usage, report)
	ack := instanceapi.SharedUsageAck{
		Tenants: []instanceapi.SharedTenantUsage{},
		Refused: []instanceapi.SharedEntryRefused{},
	}
	refused := map[uuid.UUID]bool{}
	for _, rd := range report.Readings {
		if !s.placed(rd.InstanceId) && !refused[rd.InstanceId] {
			refused[rd.InstanceId] = true
			ack.Refused = append(ack.Refused, instanceapi.SharedEntryRefused{
				InstanceId: rd.InstanceId,
				Code:       instanceapi.SharedEntryNotPlaced,
			})
		}
	}
	for _, t := range s.shared.tenants {
		answer, ok := s.shared.answers[t.InstanceId]
		if !ok {
			answer = instanceapi.SharedTenantUsage{
				InstanceId: t.InstanceId,
				Meters:     []instanceapi.MeterUsage{},
			}
		}
		ack.Tenants = append(ack.Tenants, answer)
	}
	write(w, ack)
}

func compareUUID(a, b uuid.UUID) int {
	return slices.Compare(a[:], b[:])
}
