package fakecp

import (
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const (
	maxTelemetrySamples = 30
	maxTelemetryParts   = 50
	telemetryPast       = 2 * time.Hour
	telemetryAhead      = 5 * time.Minute
)

type telemetryState struct {
	reports []instanceapi.TelemetryReport
	byMin   map[time.Time]instanceapi.TelemetrySample
	fail    int
	refuse  int
	calls   int
}

func (s *Server) TelemetryReports() []instanceapi.TelemetryReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.telemetry.reports)
}

func (s *Server) TelemetrySamples() map[time.Time]instanceapi.TelemetrySample {
	s.mu.Lock()
	defer s.mu.Unlock()
	return maps.Clone(s.telemetry.byMin)
}

func (s *Server) TelemetryCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.telemetry.calls
}

func (s *Server) FailTelemetryNext(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.telemetry.fail = n
}

func (s *Server) RefuseTelemetryNext(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.telemetry.refuse = n
}

func (s *Server) reportTelemetry(w http.ResponseWriter, r *http.Request) {
	var report instanceapi.TelemetryReport
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&report); err != nil ||
		!validTelemetry(report, time.Now()) {
		s.countTelemetry()
		problem(w, http.StatusBadRequest, "validation_failed")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.telemetry.calls++
	switch {
	case s.telemetry.fail > 0:
		s.telemetry.fail--
		problem(w, http.StatusInternalServerError, "internal")
		return
	case s.telemetry.refuse > 0:
		s.telemetry.refuse--
		problem(w, http.StatusBadRequest, "validation_failed")
		return
	}
	if s.telemetry.byMin == nil {
		s.telemetry.byMin = map[time.Time]instanceapi.TelemetrySample{}
	}
	s.telemetry.reports = append(s.telemetry.reports, report)
	for _, sample := range report.Samples {
		s.telemetry.byMin[sample.At.UTC()] = sample
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) countTelemetry() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.telemetry.calls++
}

func validTelemetry(report instanceapi.TelemetryReport, now time.Time) bool {
	if len(report.Samples) == 0 || len(report.Samples) > maxTelemetrySamples {
		return false
	}
	var last time.Time
	for i, sample := range report.Samples {
		at := sample.At.UTC()
		if !at.Equal(at.Truncate(time.Minute)) || at.Before(now.Add(-telemetryPast)) ||
			at.After(now.Add(telemetryAhead)) || (i > 0 && !at.After(last)) ||
			len(sample.Parts) > maxTelemetryParts {
			return false
		}
		last = at
		names := make(map[string]bool, len(sample.Parts))
		for _, p := range sample.Parts {
			if p.Name == "" || names[p.Name] {
				return false
			}
			names[p.Name] = true
		}
	}
	return true
}
