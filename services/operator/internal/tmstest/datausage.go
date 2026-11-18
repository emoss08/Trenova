package tmstest

import (
	"net/http"
	"time"

	"github.com/emoss08/trenova/operator/internal/tms"
)

type dataUsageState struct {
	held   *tms.DataUsage
	status int
	calls  int
}

func (s *Server) SetDataUsage(usage *tms.DataUsage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if usage == nil {
		s.dataUsage.held = nil
		return
	}
	held := *usage
	s.dataUsage.held = &held
}

func (s *Server) FailDataUsage(status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dataUsage.status = status
}

func (s *Server) DataUsageCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dataUsage.calls
}

func (s *Server) getDataUsage(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	s.dataUsage.calls++
	status, held := s.dataUsage.status, s.dataUsage.held
	s.mu.Unlock()
	if status != 0 {
		problem(w, status, "")
		return
	}
	out := map[string]any{"measuredAt": time.Now().UTC().Format(time.RFC3339)}
	if held != nil {
		if !held.MeasuredAt.IsZero() {
			out["measuredAt"] = held.MeasuredAt.UTC().Format(time.RFC3339)
		}
		for key, v := range map[string]*int64{
			"databaseBytes":  held.DatabaseBytes,
			"documentsBytes": held.DocumentsBytes,
			"searchBytes":    held.SearchBytes,
		} {
			if v != nil {
				out[key] = *v
			}
		}
	}
	writeJSON(w, out)
}
