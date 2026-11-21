package tmstest

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/operator/internal/tms"
)

const maxSupportKeys = 4

var strictJSON = sonic.Config{DisallowUnknownFields: true}.Froze()

type supportState struct {
	keys     *tms.SupportKeys
	received time.Time
	puts     []tms.SupportKeys
	grants   []tms.SupportGrant
	grantErr bool
	deletes  int
}

func (s *Server) routeSupport(mux *http.ServeMux) {
	mux.HandleFunc("GET /managed/v1/support-access", s.getSupportKeys)
	mux.HandleFunc("PUT /managed/v1/support-access", s.putSupportKeys)
	mux.HandleFunc("DELETE /managed/v1/support-access", s.deleteSupportKeys)
	mux.HandleFunc("GET /managed/v1/support-access/grants", s.getSupportGrants)
}

func (s *Server) SupportKeys() *tms.SupportKeys {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.support.keys == nil {
		return nil
	}
	held := *s.support.keys
	held.Keys = append([]tms.SupportKey(nil), held.Keys...)
	return &held
}

func (s *Server) SupportKeyPuts() []tms.SupportKeys {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]tms.SupportKeys(nil), s.support.puts...)
}

func (s *Server) HoldSupportKeys(keys tms.SupportKeys) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.support.keys = &keys
	s.support.received = time.Now().UTC()
}

func (s *Server) SetSupportGrants(grants []tms.SupportGrant) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.support.grants = append([]tms.SupportGrant(nil), grants...)
}

func (s *Server) FailSupportGrants(fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.support.grantErr = fail
}

func (s *Server) getSupportKeys(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	held := tms.HeldSupportKeys{KeyIDs: []string{}}
	if k := s.support.keys; k != nil {
		held.InstanceID = k.InstanceID
		held.StaffDomain = k.StaffDomain
		for _, key := range k.Keys {
			held.KeyIDs = append(held.KeyIDs, key.Kid)
		}
		held.ReceivedAt = s.support.received.Format(time.RFC3339)
	}
	s.mu.Unlock()
	writeJSON(w, held)
}

func (s *Server) putSupportKeys(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 16<<10))
	if err != nil {
		problem(w, http.StatusBadRequest, "Invalid support access keys")
		return
	}
	var in tms.SupportKeys
	if err = strictJSON.Unmarshal(raw, &in); err != nil {
		problem(w, http.StatusBadRequest,
			"the body must be {instanceId, staffDomain, keys} with keys a list of Ed25519 JWKs")
		return
	}
	if !validSupportKeys(&in) {
		problem(w, http.StatusBadRequest, "Invalid support access keys")
		return
	}
	in.StaffDomain = strings.ToLower(in.StaffDomain)
	s.mu.Lock()
	s.support.keys = &in
	s.support.received = time.Now().UTC()
	s.support.puts = append(s.support.puts, in)
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func validSupportKeys(in *tms.SupportKeys) bool {
	if !strings.Contains(in.StaffDomain, ".") ||
		len(in.Keys) == 0 || len(in.Keys) > maxSupportKeys {
		return false
	}
	seen := map[string]bool{}
	for _, k := range in.Keys {
		if k.Kid == "" || seen[k.Kid] || k.Kty != "OKP" || k.Crv != "Ed25519" ||
			k.Alg != "EdDSA" || k.Use != "sig" || len(k.X) != 43 {
			return false
		}
		seen[k.Kid] = true
	}
	return true
}

func (s *Server) SupportKeyDeletes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.support.deletes
}

func (s *Server) deleteSupportKeys(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	s.support.keys = nil
	s.support.deletes++
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getSupportGrants(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	fail := s.support.grantErr
	grants := append([]tms.SupportGrant{}, s.support.grants...)
	s.mu.Unlock()
	if fail {
		problem(w, http.StatusInternalServerError, "")
		return
	}
	out := make([]map[string]string, 0, len(grants))
	for _, g := range grants {
		out = append(out, map[string]string{
			"id":               g.ID,
			"organizationId":   g.OrganizationID,
			"organizationName": g.OrganizationName,
			"staffEmail":       g.StaffEmail,
			"scope":            g.Scope,
			"expiresAt":        g.ExpiresAt.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, map[string]any{"grants": out})
}

func writeJSON(w http.ResponseWriter, v any) {
	out, err := sonic.Marshal(v)
	if err != nil {
		problem(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}
