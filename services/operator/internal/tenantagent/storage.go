package tenantagent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cenkalti/backoff/v5"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const (
	storageComponent = "archive-storage"
	nightlyPrefix    = "tenants/"
	nightlyAbsent    = "absent"
	nightlyOff       = "the control plane names no nightly archive target: nightly archives are off"
)

type storageState struct {
	held     *instanceapi.SharedStorage
	revision string
	readAt   time.Time
	stale    bool
	retry    *backoff.ExponentialBackOff
	retryAt  time.Time

	signalled map[string]struct{}
	primed    bool

	nightlySent    string
	nightlyRefused string
	nightlyProblem string
}

func newStorageState(poll time.Duration) storageState {
	retry := backoff.NewExponentialBackOff()
	retry.InitialInterval = poll
	retry.MaxInterval = maxBackoff
	return storageState{retry: retry, signalled: make(map[string]struct{})}
}

func (a *Agent) heldNightly() *instanceapi.SharedNightlyTarget {
	if a.storage.held == nil {
		return nil
	}
	return a.storage.held.Nightly
}

func (a *Agent) noteSignals(held []tms.TenantStatus) {
	current := make(map[string]struct{})
	for i := range held {
		for _, f := range held[i].CommandFailures {
			if f.Code != tms.FailureStorageRefused && f.Code != tms.FailureKeyUnknown {
				continue
			}
			current[f.CommandID] = struct{}{}
			if _, seen := a.storage.signalled[f.CommandID]; seen || !a.storage.primed {
				continue
			}
			a.storage.stale = true
			a.p.Logger.Info(
				"the TMS reports a bucket that refused the credential or an archive key it was not given; the archive storage is read again",
				"instanceId",
				held[i].InstanceID,
				"command",
				f.CommandID,
				"code",
				f.Code,
			)
		}
	}
	a.storage.signalled = current
	a.storage.primed = true
}

func (a *Agent) storageDue(revision string, wanted []targetKey) bool {
	s := &a.storage
	now := a.now()
	if now.Before(s.retryAt) {
		return false
	}
	switch {
	case s.held == nil, s.revision != revision, s.stale:
		return true
	case now.Sub(s.readAt) < a.p.StorageEvery:
		return false
	}
	for _, k := range wanted {
		for _, w := range s.held.Waiting {
			if w.InstanceId.String() == k.instance && w.CommandId.String() == k.command {
				return true
			}
		}
	}
	return false
}

func (a *Agent) targets(
	ctx context.Context,
	revision string,
	wanted []targetKey,
) map[targetKey]instanceapi.SharedArchiveTarget {
	if a.storageDue(revision, wanted) {
		a.readStorage(ctx, revision)
	}
	if a.storage.held == nil {
		return nil
	}
	archives := a.storage.held.Archives
	out := make(map[targetKey]instanceapi.SharedArchiveTarget, len(archives))
	for i := range archives {
		t := &archives[i]
		out[targetKey{instance: t.InstanceId.String(), command: t.CommandId.String()}] = *t
	}
	return out
}

func (a *Agent) readStorage(ctx context.Context, revision string) {
	s := &a.storage
	got, err := a.p.ControlPlane.SharedStorage(ctx)
	if err != nil {
		wait := s.retry.NextBackOff()
		s.retryAt = a.now().Add(wait)
		a.p.Logger.Warn(
			"cannot read the archive storage; what was last read is kept and it is read again later",
			"error",
			err,
			"retryIn",
			wait.Round(time.Second),
		)
		return
	}
	s.retry.Reset()
	s.retryAt = time.Time{}
	s.held, s.revision, s.readAt, s.stale = &got, revision, a.now(), false
	for _, w := range got.Waiting {
		a.p.Logger.Info("the control plane has no archive credential for a command yet",
			"instanceId", w.InstanceId, "command", w.CommandId, "reason", w.Reason)
	}
	a.p.Logger.Info("the archive storage was read",
		"storageRevision", revision,
		"targets", len(got.Archives),
		"waiting", len(got.Waiting),
		"nightly", got.Nightly != nil)
}

func (a *Agent) syncNightly(ctx context.Context) {
	s := &a.storage
	if s.held == nil {
		return
	}
	target := s.held.Nightly
	if target == nil {
		a.removeNightly(ctx)
		return
	}
	grant := nightlyGrant(target)
	digest, err := digestOf(grant)
	if err != nil || digest == s.nightlySent || digest == s.nightlyRefused {
		return
	}
	if why := unusableNightly(target); why != "" {
		a.refuseNightly(digest, "the control plane's nightly archive target is not usable: "+why,
			target.Bucket)
		return
	}
	err = a.p.TMS.PutNightlyGrant(ctx, grant)
	if err == nil {
		s.nightlySent, s.nightlyRefused, s.nightlyProblem = digest, "", ""
		ids := make([]string, 0, len(grant.ArchiveKeys))
		for _, k := range grant.ArchiveKeys {
			ids = append(ids, k.ID)
		}
		a.p.Logger.Info("the nightly archive storage was given to the TMS",
			"bucket", grant.Bucket, "archiveKeyIds", ids, "expires", grant.ExpiresAt != nil)
		return
	}
	if reason, refused := tms.Refused(err); refused && !errors.Is(err, tms.ErrNotShared) {
		a.refuseNightly(digest, "the TMS refused the nightly archive storage: "+reason,
			target.Bucket)
		return
	}
	a.p.Logger.Warn("cannot give the TMS the nightly archive storage; it is sent again later",
		"bucket", grant.Bucket, "error", err)
}

func (a *Agent) refuseNightly(digest, problem, bucket string) {
	s := &a.storage
	s.nightlyRefused, s.nightlyProblem = digest, problem
	a.p.Logger.Error(
		"nightly archives cannot use the storage the control plane names; the TMS keeps what it held",
		"bucket",
		bucket,
		"why",
		problem,
	)
}

func (a *Agent) removeNightly(ctx context.Context) {
	s := &a.storage
	if s.nightlySent == nightlyAbsent {
		return
	}
	if err := a.p.TMS.DeleteNightlyGrant(ctx); err != nil {
		a.p.Logger.Warn(
			"cannot remove the TMS's nightly archive storage; it is removed again later",
			"error", err)
		return
	}
	s.nightlySent, s.nightlyRefused, s.nightlyProblem = nightlyAbsent, "", ""
	a.p.Logger.Warn(
		"the control plane names no nightly archive target; the TMS's nightly archive storage was removed and nightly archives stop",
	)
}

func (a *Agent) storageComponent() (instanceapi.ComponentStatus, bool) {
	s := &a.storage
	if s.held == nil {
		return instanceapi.ComponentStatus{}, false
	}
	out := instanceapi.ComponentStatus{Name: storageComponent, Ready: s.nightlyProblem == ""}
	switch {
	case s.nightlyProblem != "":
		message := clipTo(s.nightlyProblem, maxComponentMessage)
		out.Message = &message
	case s.held.Nightly == nil:
		message := nightlyOff
		out.Message = &message
	}
	return out, true
}

func nightlyGrant(t *instanceapi.SharedNightlyTarget) *tms.NightlyGrant {
	grant := &tms.NightlyGrant{
		Endpoint:        t.Endpoint,
		Region:          t.Region,
		Bucket:          t.Bucket,
		Prefix:          t.Prefix,
		AccessKeyID:     t.AccessKeyId,
		SecretAccessKey: t.SecretAccessKey,
		ExpiresAt:       t.ExpiresAt,
		ArchiveKeys:     archiveKeys(&t.ArchiveKeys),
	}
	if t.SessionToken != nil {
		grant.SessionToken = *t.SessionToken
	}
	return grant
}

func unusableNightly(t *instanceapi.SharedNightlyTarget) string {
	switch {
	case t.Access != instanceapi.NightlyWrite:
		return "its access is not write"
	case t.Prefix != nightlyPrefix:
		return "its prefix is not " + nightlyPrefix
	case !httpsEndpoint(t.Endpoint):
		return "its endpoint is not an https URL"
	case strings.TrimSpace(t.Bucket) == "":
		return "it names no bucket"
	case strings.TrimSpace(t.AccessKeyId) == "" || strings.TrimSpace(t.SecretAccessKey) == "":
		return "it carries no access key"
	case !oneCurrent(t.ArchiveKeys):
		return "its archive keys do not have exactly one current key"
	default:
		return ""
	}
}

func digestOf(grant *tms.NightlyGrant) (string, error) {
	body, err := sonic.Marshal(grant)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func oneCurrent(keys []instanceapi.ArchiveKey) bool {
	current := 0
	for _, k := range keys {
		if k.Current {
			current++
		}
	}
	return current == 1
}

func httpsEndpoint(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil &&
		(u.Path == "" || u.Path == "/") && u.RawQuery == "" && u.Fragment == ""
}

func sameEndpoint(a, b string) bool {
	trim := func(s string) string { return strings.TrimSuffix(strings.TrimSpace(s), "/") }
	return strings.EqualFold(trim(a), trim(b))
}
