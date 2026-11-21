package tmstest

import (
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/operator/internal/tms"
)

type heldCommand struct {
	kind     string
	instance string
	id       string
	purpose  string
	state    string
	expect   string
	target   tms.ArchiveTarget
	starts   int
}

func (s *Server) routeArchives(mux *http.ServeMux) {
	mux.HandleFunc("POST /managed/v1/tenants/{instanceId}/archives", s.postArchive)
	mux.HandleFunc("POST /managed/v1/tenants/{instanceId}/restores", s.postRestore)
	mux.HandleFunc(
		"GET /managed/v1/tenants/{instanceId}/archives/{commandId}",
		s.getCommand(tms.CommandArchive),
	)
	mux.HandleFunc(
		"GET /managed/v1/tenants/{instanceId}/restores/{commandId}",
		s.getCommand(tms.CommandRestore),
	)
	mux.HandleFunc(
		"GET /managed/v1/tenants/{instanceId}/imports/{commandId}",
		s.getCommand(tms.CommandImport),
	)
	mux.HandleFunc("PUT /managed/v1/tenants/archive-storage", s.putNightly)
	mux.HandleFunc("DELETE /managed/v1/tenants/archive-storage", s.deleteNightly)
}

func (s *Server) RefuseCommand(commandID, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants.refuseCmd[commandID] = reason
}

func (s *Server) SetBusy(instanceID string, busy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants.busy[instanceID] = busy
}

func (s *Server) CommandStarts(commandID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.tenants.commands[commandID]; ok {
		return c.starts
	}
	return 0
}

func (s *Server) CommandPosts(commandID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tenants.posted[commandID]
}

func (s *Server) CommandState(commandID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.tenants.commands[commandID]; ok {
		return c.state
	}
	return ""
}

func (s *Server) CommandTarget(commandID string) (tms.ArchiveTarget, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.tenants.commands[commandID]
	if !ok {
		return tms.ArchiveTarget{}, false
	}
	return c.target, true
}

func (s *Server) NightlyGrants() []tms.NightlyGrant {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.tenants.nightly)
}

func (s *Server) NightlyGrantDeletes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tenants.nightlyDeletes
}

func (s *Server) HeldNightlyGrant() (tms.NightlyGrant, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tenants.heldNightly == nil {
		return tms.NightlyGrant{}, false
	}
	return *s.tenants.heldNightly, true
}

func (s *Server) RefuseNightlyGrant(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants.refuseNightly = reason
}

func (s *Server) SetTenantGrants(instanceID string, grants []tms.SupportGrant) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.tenants.statuses[instanceID]
	held := slices.Clone(grants)
	status.SupportGrants = &held
	s.tenants.statuses[instanceID] = status
}

func (s *Server) FinishArchive(commandID string, size int64, sha string, takenAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.tenants.commands[commandID]
	c.state = "done"
	status := s.tenants.statuses[c.instance]
	status.LastArchive = &tms.TenantArchive{
		CommandID: c.id,
		Location:  "s3://" + c.target.Bucket + "/" + c.target.Prefix,
		SizeBytes: size,
		SHA256:    sha,
		TakenAt:   takenAt,
	}
	s.tenants.statuses[c.instance] = status
}

func (s *Server) FailCommand(commandID, reason string) {
	s.FailCommandWithCode(commandID, "failed", reason)
}

func (s *Server) FailCommandWithCode(commandID, code, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.tenants.commands[commandID]
	c.state = "failed"
	status := s.tenants.statuses[c.instance]
	status.CommandFailures = append([]tms.TenantCommandFailure{{
		CommandID: c.id, Kind: c.kind, Code: code, Reason: reason,
	}}, status.CommandFailures...)
	s.tenants.statuses[c.instance] = status
}

func (s *Server) FailNightly(instanceID, commandID, code, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.tenants.statuses[instanceID]
	status.CommandFailures = append([]tms.TenantCommandFailure{{
		CommandID: commandID, Kind: tms.CommandArchive, Code: code, Reason: reason,
	}}, status.CommandFailures...)
	s.tenants.statuses[instanceID] = status
}

func (s *Server) FinishRestore(commandID string, at time.Time, reason string) {
	s.FinishRestoreNoted(commandID, at, reason, "")
}

func (s *Server) FinishRestoreNoted(commandID string, at time.Time, reason, note string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.tenants.commands[commandID]
	status := s.tenants.statuses[c.instance]
	restore := &tms.TenantRestore{CommandID: c.id, State: "done", At: at, Note: note}
	c.state = "done"
	status.State = "suspended"
	if reason != "" {
		c.state = "failed"
		restore.State, restore.Reason = "failed", reason
	}
	status.LastRestore = restore
	s.tenants.statuses[c.instance] = status
}

func (s *Server) FinishImport(commandID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.tenants.commands[commandID]
	c.state = "done"
	entry := s.tenants.pending[c.instance]
	status := s.tenants.statuses[c.instance]
	revision := entry.Revision
	status.AppliedRevision = &revision
	status.State = entry.Lifecycle
	status.Administrator = &tms.TenantAdministrator{SetUp: true}
	s.tenants.statuses[c.instance] = status
}

func (s *Server) importing(
	w http.ResponseWriter,
	entry *tenantEntry,
	status *tms.TenantStatus,
	known bool,
) bool {
	if known && status.AppliedRevision != nil {
		return false
	}
	var importID, location string
	for _, c := range entry.Commands {
		if c.Kind == tms.CommandImport {
			importID, location = c.ID, c.Location
		}
	}
	if importID == "" {
		return false
	}
	if c, held := s.tenants.commands[importID]; held {
		c.starts += boolInt(entry.ImportTarget != nil)
		writeJSON(w, status)
		return true
	}
	t := entry.ImportTarget
	switch {
	case t == nil:
		status.State = "applying"
		status.Refused = &tms.TenantRefusal{
			Revision: entry.Revision, Code: "invalid",
			Reason: "an entry that imports the tenant must carry importSource",
		}
	case t.InstanceID != entry.InstanceID || t.CommandID != importID ||
		t.Kind != tms.CommandImport || t.Access != "read" ||
		strings.TrimSuffix("s3://"+t.Bucket+"/"+t.Prefix, "/") != strings.TrimSuffix(location, "/"):
		problem(w, http.StatusBadRequest, "the target must name this tenant and this command")
		return true
	default:
		s.tenants.commands[importID] = &heldCommand{
			kind: tms.CommandImport, instance: entry.InstanceID, id: importID,
			state: "running", target: *t, starts: 1,
		}
		s.tenants.pending[entry.InstanceID] = *entry
		status.State = "applying"
		status.Refused = nil
	}
	s.tenants.statuses[entry.InstanceID] = *status
	writeJSON(w, status)
	return true
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type startBody struct {
	CommandID    string            `json:"commandId"`
	Purpose      string            `json:"purpose"`
	ExpectSHA256 string            `json:"expectSha256"`
	Target       tms.ArchiveTarget `json:"target"`
}

func (s *Server) postArchive(w http.ResponseWriter, r *http.Request) {
	s.startCommand(w, r, tms.CommandArchive)
}

func (s *Server) postRestore(w http.ResponseWriter, r *http.Request) {
	s.startCommand(w, r, tms.CommandRestore)
}

func (s *Server) startCommand(w http.ResponseWriter, r *http.Request, kind string) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 16<<10))
	var body startBody
	if err != nil || sonic.Unmarshal(raw, &body) != nil {
		problem(w, http.StatusBadRequest, "the body must be {commandId, target}")
		return
	}
	instance := r.PathValue("instanceId")
	want := "read"
	if kind == tms.CommandArchive {
		want = "write"
	}
	t := body.Target
	if t.InstanceID != instance || t.CommandID != body.CommandID || t.Kind != kind ||
		t.Access != want {
		problem(
			w,
			http.StatusBadRequest,
			"the target must name this tenant, this command and its kind",
		)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants.posted[body.CommandID]++
	if held, ok := s.tenants.commands[body.CommandID]; ok {
		held.starts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(commandBody(held))
		return
	}
	if reason, refused := s.tenants.refuseCmd[body.CommandID]; refused {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"title":"Conflict","status":409,"detail":"` + reason + `"}`))
		return
	}
	if s.tenants.busy[instance] {
		problem(w, http.StatusLocked, "another archive, restore or import of the tenant is running")
		return
	}
	c := &heldCommand{
		kind: kind, instance: instance, id: body.CommandID, purpose: body.Purpose,
		state: "running", expect: body.ExpectSHA256, target: t, starts: 1,
	}
	s.tenants.commands[body.CommandID] = c
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write(commandBody(c))
}

func commandBody(c *heldCommand) []byte {
	out, _ := sonic.Marshal(tms.TenantCommand{
		CommandID: c.id, InstanceID: c.instance, Kind: c.kind, Purpose: c.purpose,
		State: c.state, Location: "s3://" + c.target.Bucket + "/" + c.target.Prefix,
	})
	return out
}

func (s *Server) getCommand(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		c, ok := s.tenants.commands[r.PathValue("commandId")]
		if !ok || c.kind != kind || c.instance != r.PathValue("instanceId") {
			problem(w, http.StatusNotFound, "no command by that id is on this server")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(commandBody(c))
	}
}

func (s *Server) putNightly(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 16<<10))
	var grant tms.NightlyGrant
	if err != nil || sonic.Unmarshal(raw, &grant) != nil || grant.Bucket == "" ||
		grant.SecretAccessKey == "" || len(grant.ArchiveKeys) == 0 {
		problem(w, http.StatusBadRequest, "the body must be a nightly grant")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tenants.shared {
		problem(w, http.StatusConflict, "tenants are managed only on a shared server")
		return
	}
	switch {
	case grant.Prefix != "tenants/":
		problem(w, http.StatusBadRequest, "the nightly archive storage's prefix must be tenants/")
		return
	case s.tenants.refuseNightly != "":
		problem(w, http.StatusBadRequest, s.tenants.refuseNightly)
		return
	}
	s.tenants.nightly = append(s.tenants.nightly, grant)
	s.tenants.heldNightly = &grant
	writeJSON(w, map[string]any{"expiresAt": grant.ExpiresAt})
}

func (s *Server) deleteNightly(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tenants.shared {
		problem(w, http.StatusConflict, "tenants are managed only on a shared server")
		return
	}
	s.tenants.nightlyDeletes++
	s.tenants.heldNightly = nil
	w.WriteHeader(http.StatusNoContent)
}
