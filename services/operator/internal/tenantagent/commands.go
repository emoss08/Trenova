package tenantagent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const (
	DefaultStorageEvery = time.Minute
	restoreUntouched    = "Nothing was changed: "
	restoreIncomplete   = "The tenant is incomplete: "
	waitingNoBucket     = string(instanceapi.SharedArchiveNoBucket)
)

type command struct {
	kind     string
	id       string
	purpose  string
	location string
	expect   string
}

func commandsOf(entry *instanceapi.SharedTenant) []command {
	out := make([]command, 0, len(entry.Commands))
	for _, raw := range entry.Commands {
		kind, err := raw.Discriminator()
		if err != nil {
			continue
		}
		switch kind {
		case tms.CommandArchive:
			c, cerr := raw.AsSharedArchiveCommand()
			if cerr == nil {
				out = append(out, command{
					kind: kind, id: c.Id.String(), purpose: string(c.Purpose),
				})
			}
		case tms.CommandRestore:
			c, cerr := raw.AsSharedRestoreCommand()
			if cerr == nil {
				out = append(out, command{
					kind: kind, id: c.Id.String(), location: c.Location, expect: c.ExpectSha256,
				})
			}
		case tms.CommandImport:
			c, cerr := raw.AsSharedImportCommand()
			if cerr == nil {
				out = append(out, command{
					kind: kind, id: c.Id.String(), location: c.Location, expect: c.ExpectSha256,
				})
			}
		}
	}
	return out
}

type refusal struct {
	kind   string
	reason string
	at     time.Time
}

type targetKey struct {
	instance string
	command  string
}

type start struct {
	instance uuid.UUID
	cmd      command
}

func (a *Agent) forgetUnlisted(desired *instanceapi.SharedTenantList) {
	listed := make(map[string]struct{})
	for i := range desired.Tenants {
		for _, c := range commandsOf(&desired.Tenants[i]) {
			listed[c.id] = struct{}{}
		}
	}
	for id := range a.refusals {
		if _, ok := listed[id]; !ok {
			delete(a.refusals, id)
		}
	}
}

func settled(c *command, held *tms.TenantStatus, refused map[string]refusal) bool {
	if _, ok := refused[c.id]; ok {
		return true
	}
	for _, f := range held.CommandFailures {
		if f.CommandID == c.id {
			return true
		}
	}
	switch c.kind {
	case tms.CommandArchive:
		return held.LastArchive != nil && held.LastArchive.CommandID == c.id
	case tms.CommandRestore:
		return held.LastRestore != nil && held.LastRestore.CommandID == c.id
	case tms.CommandImport:
		return held.AppliedRevision != nil
	default:
		return true
	}
}

func holdsLifecycle(entry *instanceapi.SharedTenant, held *tms.TenantStatus) bool {
	if held.AppliedRevision == nil || *held.AppliedRevision < entry.Revision {
		return false
	}
	switch entry.Lifecycle {
	case instanceapi.SharedTenantActive, instanceapi.SharedTenantSuspended:
		return held.State == string(entry.Lifecycle)
	case instanceapi.SharedTenantRemoved:
		return false
	default:
		return false
	}
}

func (a *Agent) pendingImport(
	ctx context.Context,
	entry *instanceapi.SharedTenant,
	held *tms.TenantStatus,
) (pending *command, blocked bool, err error) {
	for _, c := range commandsOf(entry) {
		if c.kind != tms.CommandImport {
			continue
		}
		if _, refused := a.refusals[c.id]; refused && held.AppliedRevision == nil {
			return nil, true, nil
		}
		if settled(&c, held, a.refusals) {
			continue
		}
		_, gerr := a.p.TMS.TenantCommand(ctx, entry.InstanceId.String(), c.kind, c.id)
		switch {
		case errors.Is(gerr, tms.ErrCommandNotFound):
			return &c, false, nil
		case gerr != nil:
			return nil, false, gerr
		default:
			return nil, false, nil
		}
	}
	return nil, false, nil
}

func (a *Agent) nextCommand(
	ctx context.Context,
	entry *instanceapi.SharedTenant,
	held *tms.TenantStatus,
) (next *command, found bool, err error) {
	if !holdsLifecycle(entry, held) {
		return nil, false, nil
	}
	id := entry.InstanceId.String()
	for _, c := range commandsOf(entry) {
		if c.kind == tms.CommandImport || settled(&c, held, a.refusals) {
			continue
		}
		got, gerr := a.p.TMS.TenantCommand(ctx, id, c.kind, c.id)
		switch {
		case errors.Is(gerr, tms.ErrCommandNotFound):
			return &c, true, nil
		case gerr != nil:
			return nil, false, gerr
		case got.Settled():
			continue
		default:
			a.p.Logger.Debug("a tenant command is running",
				"instanceId", id, "command", c.id, "kind", c.kind,
				"phase", got.Progress.Phase, "bytes", got.Progress.Bytes)
			return nil, false, nil
		}
	}
	return nil, false, nil
}

func (a *Agent) targetFor(
	targets map[targetKey]instanceapi.SharedArchiveTarget,
	instanceID uuid.UUID,
	c *command,
) (*tms.ArchiveTarget, bool) {
	t, ok := targets[targetKey{instance: instanceID.String(), command: c.id}]
	if !ok {
		return nil, false
	}
	nightly := a.heldNightly()
	if nightly == nil {
		a.p.Logger.Info(
			"a tenant command waits: an archive, restore or import goes only to the server's own bucket, and the control plane names none",
			"instanceId",
			instanceID,
			"command",
			c.id,
			"kind",
			c.kind,
			"reason",
			waitingNoBucket,
		)
		return nil, false
	}
	if why := mismatch(&t, instanceID, c, nightly); why != "" {
		a.p.Logger.Warn(
			"the control plane's archive target does not fit its command; it is not used",
			"instanceId",
			instanceID,
			"command",
			c.id,
			"kind",
			c.kind,
			"why",
			why,
		)
		return nil, false
	}
	return convertTarget(&t), true
}

func mismatch(
	t *instanceapi.SharedArchiveTarget,
	instanceID uuid.UUID,
	c *command,
	nightly *instanceapi.SharedNightlyTarget,
) string {
	switch {
	case t.InstanceId != instanceID || t.CommandId.String() != c.id:
		return "it names another tenant or command"
	case !httpsEndpoint(t.Endpoint):
		return "its endpoint is not an https URL"
	case t.Bucket != nightly.Bucket || !sameEndpoint(t.Endpoint, nightly.Endpoint):
		return "it names another bucket than the server's own"
	case string(t.Kind) != c.kind:
		return "it is for another kind of command"
	case c.kind == tms.CommandArchive && t.Access != instanceapi.TargetWrite,
		c.kind != tms.CommandArchive && t.Access != instanceapi.TargetRead:
		return "its access does not fit the command"
	case c.kind == tms.CommandArchive &&
		t.Prefix != "tenants/"+instanceID.String()+"/archives/"+c.id+"/":
		return "its prefix is not the command's archive"
	case c.kind != tms.CommandArchive && !samePlace(t.Bucket, t.Prefix, c.location):
		return "its prefix is not the archive the command names"
	default:
		return keysMismatch(t, c)
	}
}

func keysMismatch(t *instanceapi.SharedArchiveTarget, c *command) string {
	keys := t.ArchiveKeys != nil && len(*t.ArchiveKeys) > 0
	switch {
	case c.exports() && keys:
		return "an export is written without archive keys"
	case !c.exports() && !keys:
		return "it carries no archive keys"
	case keys && !oneCurrent(*t.ArchiveKeys):
		return "its archive keys do not have exactly one current key"
	default:
		return ""
	}
}

func (c *command) exports() bool {
	return c.kind == tms.CommandArchive && c.purpose == string(instanceapi.SharedArchiveExport)
}

func samePlace(bucket, prefix, location string) bool {
	slash := func(s string) string { return strings.TrimSuffix(s, "/") + "/" }
	return bucket != "" && prefix != "" && slash("s3://"+bucket+"/"+prefix) == slash(location)
}

func convertTarget(t *instanceapi.SharedArchiveTarget) *tms.ArchiveTarget {
	out := &tms.ArchiveTarget{
		InstanceID:      t.InstanceId.String(),
		CommandID:       t.CommandId.String(),
		Kind:            string(t.Kind),
		Access:          string(t.Access),
		Endpoint:        t.Endpoint,
		Region:          t.Region,
		Bucket:          t.Bucket,
		Prefix:          t.Prefix,
		AccessKeyID:     t.AccessKeyId,
		SecretAccessKey: t.SecretAccessKey,
		ExpiresAt:       t.ExpiresAt,
		ArchiveKeys:     archiveKeys(t.ArchiveKeys),
	}
	if t.SessionToken != nil {
		out.SessionToken = *t.SessionToken
	}
	return out
}

func archiveKeys(keys *[]instanceapi.ArchiveKey) []tms.ArchiveKey {
	if keys == nil {
		return nil
	}
	out := make([]tms.ArchiveKey, 0, len(*keys))
	for _, k := range *keys {
		out = append(out, tms.ArchiveKey{ID: k.Id, Key: k.Key, Current: k.Current})
	}
	return out
}

func (a *Agent) startAll(
	ctx context.Context,
	starts []start,
	targets map[targetKey]instanceapi.SharedArchiveTarget,
) error {
	var errs []error
	for i := range starts {
		s := &starts[i]
		target, ok := a.targetFor(targets, s.instance, &s.cmd)
		if !ok {
			continue
		}
		if err := a.startOne(ctx, s, target); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (a *Agent) startOne(ctx context.Context, s *start, target *tms.ArchiveTarget) error {
	id := s.instance.String()
	var err error
	switch s.cmd.kind {
	case tms.CommandArchive:
		_, err = a.p.TMS.StartArchive(ctx, id, &tms.ArchiveStart{
			CommandID: s.cmd.id, Purpose: s.cmd.purpose, Target: *target,
		})
	case tms.CommandRestore:
		_, err = a.p.TMS.StartRestore(ctx, id, &tms.RestoreStart{
			CommandID: s.cmd.id, ExpectSHA256: s.cmd.expect, Target: *target,
		})
	default:
		return nil
	}
	return a.started(id, &s.cmd, err)
}

func (a *Agent) started(instanceID string, c *command, err error) error {
	if err == nil {
		a.p.Logger.Info("a tenant command was started",
			"instanceId", instanceID, "command", c.id, "kind", c.kind, "purpose", c.purpose)
		return nil
	}
	if errors.Is(err, tms.ErrTenantBusy) {
		a.p.Logger.Info("a tenant command waits for the tenant's running one",
			"instanceId", instanceID, "command", c.id, "kind", c.kind)
		return nil
	}
	if reason, ok := tms.Refused(err); ok {
		a.refusals[c.id] = refusal{kind: c.kind, reason: reason, at: a.now().UTC()}
		a.p.Logger.Warn("the TMS refused a tenant command; it is reported as failed",
			"instanceId", instanceID, "command", c.id, "kind", c.kind, "reason", reason)
		return nil
	}
	return fmt.Errorf("tenant %s command %s: %w", instanceID, c.id, err)
}

func (a *Agent) reportCommands(
	entry *instanceapi.SharedTenant,
	held *tms.TenantStatus,
	out *instanceapi.SharedTenantStatus,
) {
	failures := make([]instanceapi.SharedCommandFailure, 0, len(held.CommandFailures))
	seen := map[string]bool{}
	add := func(id, kind, reason string) {
		parsed, err := uuid.Parse(id)
		if err != nil || seen[id] || len(failures) == maxFailures ||
			(kind != tms.CommandArchive && kind != tms.CommandImport) {
			return
		}
		seen[id] = true
		if strings.TrimSpace(reason) == "" {
			reason = "the command could not be completed"
		}
		failures = append(failures, instanceapi.SharedCommandFailure{
			CommandId: parsed,
			Kind:      instanceapi.SharedCommandFailureKind(kind),
			Reason:    clip(reason),
		})
	}
	for _, c := range commandsOf(entry) {
		r, ok := a.refusals[c.id]
		if !ok {
			continue
		}
		if c.kind == tms.CommandRestore {
			if parsed, err := uuid.Parse(c.id); err == nil {
				reason := clip(
					restoreUntouched + "the server refused to start the restore: " + r.reason,
				)
				out.LastRestore = &instanceapi.SharedRestoreReport{
					CommandId: parsed,
					State:     instanceapi.RestoreFailed,
					At:        r.at,
					Reason:    &reason,
				}
			}
			continue
		}
		add(c.id, r.kind, r.reason)
	}
	for _, f := range held.CommandFailures {
		add(f.CommandID, f.Kind, f.Reason)
	}
	if len(failures) > 0 {
		out.CommandFailures = &failures
	}
}

func restoreReport(r *tms.TenantRestore) *instanceapi.SharedRestoreReport {
	if r == nil || (r.State != string(instanceapi.RestoreDone) &&
		r.State != string(instanceapi.RestoreFailed)) {
		return nil
	}
	parsed, err := uuid.Parse(r.CommandID)
	if err != nil {
		return nil
	}
	out := &instanceapi.SharedRestoreReport{
		CommandId: parsed,
		State:     instanceapi.SharedRestoreReportState(r.State),
		At:        r.At.UTC(),
	}
	switch instanceapi.SharedRestoreReportState(r.State) {
	case instanceapi.RestoreFailed:
		reason := r.Reason
		if !strings.HasPrefix(reason, restoreUntouched) &&
			!strings.HasPrefix(reason, restoreIncomplete) {
			reason = restoreIncomplete + reason
		}
		reason = clip(reason)
		out.Reason = &reason
	case instanceapi.RestoreDone:
		if note := printableLine(r.Note); note != "" {
			note = clip(note)
			out.Note = &note
		}
	}
	return out
}

func printableLine(s string) string {
	return strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return ' '
	}, s)), " ")
}
