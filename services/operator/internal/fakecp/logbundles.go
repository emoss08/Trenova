package fakecp

import (
	"net/http"
	"slices"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const maxBundlePeriod = 7 * 24 * time.Hour

type LogBundle struct {
	ID           uuid.UUID
	Registration *instanceapi.LogBundleRegistration
	Updates      []instanceapi.LogBundleUpdate
}

type bundleState struct {
	byID          map[uuid.UUID]*LogBundle
	byRequest     map[string]uuid.UUID
	registrations int
	updateFail    int
	refuse        int
	refusal       string
}

func newBundleState() bundleState {
	return bundleState{byID: map[uuid.UUID]*LogBundle{}, byRequest: map[string]uuid.UUID{}}
}

func (s *Server) AddLogBundle(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bundles.byID[id] = &LogBundle{ID: id}
}

func (s *Server) LogBundle(id uuid.UUID) (LogBundle, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.bundles.byID[id]
	if !ok {
		return LogBundle{}, false
	}
	return LogBundle{ID: b.ID, Registration: b.Registration, Updates: slices.Clone(b.Updates)}, true
}

func (s *Server) LogBundleRegistrations() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bundles.registrations
}

func (s *Server) FailLogBundleUpdatesNext(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bundles.updateFail = n
}

func (s *Server) RefuseLogBundlesNext(n int, detail string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bundles.refuse, s.bundles.refusal = n, detail
}

func conflict(w http.ResponseWriter, detail string) {
	body := map[string]any{
		"type":   "about:blank",
		"title":  "Conflict",
		"status": http.StatusConflict,
		"code":   "conflict",
	}
	if detail != "" {
		body["detail"] = detail
	}
	out, _ := sonic.Marshal(body)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusConflict)
	_, _ = w.Write(out)
}

func (s *Server) registerLogBundle(w http.ResponseWriter, r *http.Request) {
	var reg instanceapi.LogBundleRegistration
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&reg); err != nil ||
		reg.RequestId == "" || string(reg.RequestedBy.Email) == "" ||
		!reg.Until.After(reg.Since) || reg.Until.Sub(reg.Since) > maxBundlePeriod {
		problem(w, http.StatusBadRequest, "validation_failed")
		return
	}
	s.mu.Lock()
	s.bundles.registrations++
	id, seen := s.bundles.byRequest[reg.RequestId]
	if !seen && s.bundles.refuse > 0 {
		s.bundles.refuse--
		detail := s.bundles.refusal
		s.mu.Unlock()
		conflict(w, detail)
		return
	}
	if !seen {
		id = uuid.New()
		s.bundles.byRequest[reg.RequestId] = id
		s.bundles.byID[id] = &LogBundle{ID: id, Registration: &reg}
	}
	s.mu.Unlock()
	out, err := sonic.Marshal(instanceapi.LogBundleRegistered{BundleId: id})
	if err != nil {
		problem(w, http.StatusInternalServerError, "internal")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(out)
}

func (s *Server) updateLogBundle(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("bundleId"))
	if err != nil {
		problem(w, http.StatusNotFound, "not_found")
		return
	}
	var update instanceapi.LogBundleUpdate
	if err = sonic.ConfigDefault.NewDecoder(r.Body).
		Decode(&update); err != nil ||
		!validUpdate(id, &update) {
		problem(w, http.StatusBadRequest, "validation_failed")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bundles.updateFail > 0 {
		s.bundles.updateFail--
		problem(w, http.StatusInternalServerError, "internal")
		return
	}
	b, ok := s.bundles.byID[id]
	if !ok {
		problem(w, http.StatusNotFound, "not_found")
		return
	}
	b.Updates = append(b.Updates, update)
	w.WriteHeader(http.StatusNoContent)
}

func validUpdate(id uuid.UUID, u *instanceapi.LogBundleUpdate) bool {
	message := u.Message != nil && *u.Message != ""
	switch u.Status {
	case instanceapi.LogBundleCollecting:
		return true
	case instanceapi.LogBundleStored:
		return u.ObjectKey != nil && *u.ObjectKey == "support/logs/"+id.String()+".zip" &&
			u.SizeBytes != nil && u.Components != nil
	case instanceapi.LogBundleFailed, instanceapi.LogBundleRefused:
		return message
	default:
		return false
	}
}
