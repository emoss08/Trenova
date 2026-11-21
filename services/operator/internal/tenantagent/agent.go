package tenantagent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cenkalti/backoff/v5"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/emailevents"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/usage"
)

const (
	DefaultPoll         = 15 * time.Second
	DefaultUsageEvery   = 15 * time.Minute
	DefaultEventsEvery  = time.Minute
	DefaultStatusEvery  = time.Minute
	DefaultConcurrency  = 4
	maxBackoff          = 5 * time.Minute
	healthyFor          = 3
	tmsComponent        = "tms"
	maxMessage          = 500
	maxComponentMessage = 200
	maxFailures         = 20
)

var ErrNoDesired = errors.New("the control plane has not answered with the tenants yet")

type ControlPlane interface {
	SharedTenants(ctx context.Context, etag string) (controlplane.SharedDesired, bool, error)
	ReportSharedStatus(
		ctx context.Context,
		report *instanceapi.SharedStatusReport,
	) (instanceapi.SharedReportAck, error)
	SharedStorage(ctx context.Context) (instanceapi.SharedStorage, error)
}

type TMS interface {
	PutTenant(ctx context.Context, instanceID string, entry []byte) (tms.TenantStatus, error)
	Tenants(ctx context.Context) ([]tms.TenantStatus, error)
	StartArchive(
		ctx context.Context,
		instanceID string,
		start *tms.ArchiveStart,
	) (tms.TenantCommand, error)
	StartRestore(
		ctx context.Context,
		instanceID string,
		start *tms.RestoreStart,
	) (tms.TenantCommand, error)
	TenantCommand(
		ctx context.Context,
		instanceID, kind, commandID string,
	) (tms.TenantCommand, error)
	PutNightlyGrant(ctx context.Context, grant *tms.NightlyGrant) error
	DeleteNightlyGrant(ctx context.Context) error
}

type SupportKeys interface {
	SyncShared(ctx context.Context, desired *instanceapi.SharedSupport) error
}

type Credentials interface {
	RenewIfDue(ctx context.Context) error
	Recover(ctx context.Context) error
}

type Escrow interface {
	Ensure(ctx context.Context) error
}

type SystemEmail interface {
	Sync(ctx context.Context, desired *instanceapi.SpecSystemEmail) error
}

type Usage interface {
	Run(ctx context.Context, tenants []instanceapi.SharedTenant) (usage.SharedResult, error)
}

type EmailEvents interface {
	Run(ctx context.Context) (emailevents.Result, error)
}

type Params struct {
	ControlPlane ControlPlane
	TMS          TMS
	Credentials  Credentials
	Escrow       Escrow
	SystemEmail  SystemEmail
	Usage        Usage
	EmailEvents  EmailEvents
	SupportKeys  SupportKeys
	Version      string
	Logger       *slog.Logger
	Now          func() time.Time
	Poll         time.Duration
	UsageEvery   time.Duration
	EventsEvery  time.Duration
	StatusEvery  time.Duration
	StorageEvery time.Duration
	Concurrency  int
}

type Agent struct {
	p   Params
	now func() time.Time

	mu       sync.Mutex
	etag     string
	desired  *instanceapi.SharedTenantList
	sent     map[uuid.UUID]string
	silenced map[uuid.UUID]struct{}
	refusals map[string]refusal
	storage  storageState

	lastReport   []byte
	lastReportAt time.Time
	lastUsage    time.Time
	lastEvents   time.Time
	lastOK       time.Time
	lastErr      error
}

func New(params *Params) (*Agent, error) {
	p := *params
	switch {
	case p.ControlPlane == nil, p.TMS == nil, p.Credentials == nil, p.Escrow == nil,
		p.SystemEmail == nil, p.Usage == nil, p.EmailEvents == nil, p.SupportKeys == nil,
		p.Logger == nil:
		return nil, errors.New("tenantagent: every dependency is required")
	}
	if p.Poll <= 0 {
		p.Poll = DefaultPoll
	}
	if p.UsageEvery <= 0 {
		p.UsageEvery = DefaultUsageEvery
	}
	if p.EventsEvery <= 0 {
		p.EventsEvery = DefaultEventsEvery
	}
	if p.StatusEvery <= 0 {
		p.StatusEvery = DefaultStatusEvery
	}
	if p.StorageEvery <= 0 {
		p.StorageEvery = DefaultStorageEvery
	}
	if p.Concurrency <= 0 {
		p.Concurrency = DefaultConcurrency
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Agent{
		p:        p,
		now:      now,
		sent:     make(map[uuid.UUID]string),
		silenced: make(map[uuid.UUID]struct{}),
		refusals: make(map[string]refusal),
		storage:  newStorageState(p.Poll),
	}, nil
}

func (a *Agent) Run(ctx context.Context) error {
	retry := backoff.NewExponentialBackOff()
	retry.InitialInterval = a.p.Poll
	retry.MaxInterval = maxBackoff
	for {
		wait := jitter(a.p.Poll)
		if err := a.Iterate(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			wait = max(wait, retry.NextBackOff())
			a.p.Logger.Warn("an iteration did not finish; the next one tries again",
				"error", err, "retryIn", wait.Round(time.Second))
		} else {
			retry.Reset()
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func jitter(d time.Duration) time.Duration {
	spread := int64(d / 5)
	if spread <= 0 {
		return d
	}
	offset := rand.Int64N(spread) //nolint:gosec // G404: scheduling jitter, not a secret
	return d - time.Duration(spread/2) + time.Duration(offset)
}

func (a *Agent) Healthy() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lastOK.IsZero() {
		if a.lastErr != nil {
			return fmt.Errorf("no iteration has finished: %w", a.lastErr)
		}
		return errors.New("no iteration has finished yet")
	}
	if since := a.now().Sub(a.lastOK); since > healthyFor*max(a.p.Poll, a.p.StatusEvery) {
		return fmt.Errorf(
			"the last iteration that finished was %s ago: %w",
			since.Round(time.Second),
			a.lastErr,
		)
	}
	return nil
}

func (a *Agent) Iterate(ctx context.Context) error {
	err := a.iterate(ctx)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastErr = err
	if err == nil {
		a.lastOK = a.now()
	}
	return err
}

func (a *Agent) iterate(ctx context.Context) error {
	if err := a.p.Credentials.RenewIfDue(ctx); err != nil {
		a.p.Logger.Warn("cannot renew the certificate; the current one is kept", "error", err)
	}
	if err := a.p.Escrow.Ensure(ctx); err != nil {
		return fmt.Errorf("the escrowed secrets: %w", err)
	}
	desired, err := a.fetch(ctx)
	if err != nil {
		return err
	}

	var errs []error
	if serr := a.p.SystemEmail.Sync(ctx, desired.SystemEmail); serr != nil {
		errs = append(errs, fmt.Errorf("the system email sender: %w", serr))
	}
	if kerr := a.p.SupportKeys.SyncShared(ctx, desired.Support); kerr != nil {
		errs = append(errs, fmt.Errorf("the support access keys: %w", kerr))
	}

	statuses, terr := a.p.TMS.Tenants(ctx)
	if terr == nil {
		statuses, terr = a.converge(ctx, desired, statuses)
	}
	if rerr := a.report(ctx, desired, statuses, terr); rerr != nil {
		errs = append(errs, rerr)
	}
	if terr != nil {
		return errors.Join(append(errs, fmt.Errorf("the TMS: %w", terr))...)
	}

	now := a.now()
	if now.Sub(a.lastUsage) >= a.p.UsageEvery {
		result, uerr := a.p.Usage.Run(ctx, desired.Tenants)
		if uerr != nil {
			errs = append(errs, fmt.Errorf("the usage report: %w", uerr))
		} else {
			a.lastUsage = now
			a.p.Logger.Info("usage reported",
				"readings", result.Readings,
				"dropped", result.Dropped,
				"withheld", result.Withheld,
				"answered", result.Answered,
				"refusedByControlPlane", len(result.RefusedByCP),
				"refusedByTMS", len(result.RefusedByTMS))
		}
	}
	if now.Sub(a.lastEvents) >= a.p.EventsEvery {
		if _, eerr := a.p.EmailEvents.Run(ctx); eerr != nil {
			errs = append(errs, fmt.Errorf("the system email events: %w", eerr))
		} else {
			a.lastEvents = now
		}
	}
	return errors.Join(errs...)
}

func (a *Agent) fetch(ctx context.Context) (*instanceapi.SharedTenantList, error) {
	got, changed, err := a.p.ControlPlane.SharedTenants(ctx, a.etag)
	if errors.Is(err, controlplane.ErrUnauthenticated) {
		if rerr := a.p.Credentials.Recover(ctx); rerr != nil {
			return nil, fmt.Errorf("the control plane refuses the certificate: %w", rerr)
		}
		got, changed, err = a.p.ControlPlane.SharedTenants(ctx, a.etag)
	}
	switch {
	case err != nil && a.desired == nil:
		return nil, fmt.Errorf("the tenants: %w", err)
	case err != nil:
		a.p.Logger.Warn("cannot read the tenants; converging to the last ones read", "error", err)
		return a.desired, nil
	case changed:
		list := got.List
		a.desired, a.etag = &list, got.ETag
		for i := range list.Tenants {
			delete(a.silenced, list.Tenants[i].InstanceId)
		}
		a.p.Logger.Info("the tenants changed", "tenants", len(list.Tenants))
	}
	if a.desired == nil {
		return nil, ErrNoDesired
	}
	return a.desired, nil
}

type plan struct {
	entry  *instanceapi.SharedTenant
	body   []byte
	digest string
	target *command
}

type entryBody struct {
	*instanceapi.SharedTenant
	ImportTarget *tms.ArchiveTarget `json:"importTarget,omitempty"`
}

func (a *Agent) converge(
	ctx context.Context,
	desired *instanceapi.SharedTenantList,
	held []tms.TenantStatus,
) ([]tms.TenantStatus, error) {
	a.forgetUnlisted(desired)
	byID := make(map[string]tms.TenantStatus, len(held))
	for _, s := range held {
		byID[s.InstanceID] = s
	}
	a.noteSignals(held)
	var (
		plans  []plan
		starts []start
		wanted []targetKey
		failed []error
	)
	for i := range desired.Tenants {
		entry := &desired.Tenants[i]
		id := entry.InstanceId.String()
		status := byID[id]
		body, err := sonic.Marshal(entry)
		if err != nil {
			return held, fmt.Errorf("encode the tenant %s: %w", entry.InstanceId, err)
		}
		sum := sha256.Sum256(body)
		digest := hex.EncodeToString(sum[:])
		if needsApply(entry, &status, a.sent[entry.InstanceId] == digest) {
			pending, blocked, perr := a.pendingImport(ctx, entry, &status)
			if perr != nil {
				failed = append(failed, fmt.Errorf("tenant %s: %w", id, perr))
				continue
			}
			if blocked {
				continue
			}
			if pending != nil {
				wanted = append(wanted, targetKey{instance: id, command: pending.id})
			}
			plans = append(plans, plan{entry: entry, body: body, digest: digest, target: pending})
		}
		next, found, nerr := a.nextCommand(ctx, entry, &status)
		if nerr != nil {
			failed = append(failed, fmt.Errorf("tenant %s: %w", id, nerr))
			continue
		}
		if found {
			wanted = append(wanted, targetKey{instance: id, command: next.id})
			starts = append(starts, start{instance: entry.InstanceId, cmd: *next})
		}
	}
	targets := a.targets(ctx, desired.StorageRevision, wanted)
	a.syncNightly(ctx)
	applied, perr := a.apply(ctx, plans, targets, byID)
	if perr != nil {
		failed = append(failed, perr)
	}
	if serr := a.startAll(ctx, starts, targets); serr != nil {
		failed = append(failed, serr)
	}
	out := make([]tms.TenantStatus, 0, len(byID))
	for _, s := range byID {
		out = append(out, s)
	}
	if applied > 0 || len(failed) > 0 {
		a.p.Logger.Info("tenants applied", "asked", applied, "failed", len(failed))
	}
	return out, errors.Join(failed...)
}

func (a *Agent) apply(
	ctx context.Context,
	plans []plan,
	targets map[targetKey]instanceapi.SharedArchiveTarget,
	byID map[string]tms.TenantStatus,
) (int, error) {
	ready := make([]plan, 0, len(plans))
	for _, pl := range plans {
		if pl.target != nil {
			target, ok := a.targetFor(targets, pl.entry.InstanceId, pl.target)
			if !ok {
				a.p.Logger.Info("a tenant made from an archive waits for its archive's location",
					"instanceId", pl.entry.InstanceId, "command", pl.target.id)
				continue
			}
			body, err := sonic.Marshal(entryBody{SharedTenant: pl.entry, ImportTarget: target})
			if err != nil {
				return 0, fmt.Errorf("encode the tenant %s: %w", pl.entry.InstanceId, err)
			}
			pl.body = body
		}
		ready = append(ready, pl)
	}
	if len(ready) == 0 {
		return 0, nil
	}
	var mu sync.Mutex
	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(a.p.Concurrency)
	failed := make([]error, 0)
	for _, pl := range ready {
		group.Go(func() error {
			id := pl.entry.InstanceId
			status, err := a.p.TMS.PutTenant(gctx, id.String(), pl.body)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if reason, refused := tms.Refused(err); refused && pl.target != nil {
					a.refusals[pl.target.id] = refusal{
						kind: tms.CommandImport, reason: reason, at: a.now().UTC(),
					}
					a.p.Logger.Warn(
						"the TMS refused the import of a tenant; it is reported as failed",
						"instanceId",
						id,
						"command",
						pl.target.id,
						"reason",
						reason,
					)
					return nil
				}
				failed = append(failed, fmt.Errorf("tenant %s: %w", id, err))
				return nil
			}
			a.sent[id] = pl.digest
			byID[status.InstanceID] = status
			if pl.target != nil {
				a.p.Logger.Info("a tenant is being made from an archive",
					"instanceId", id, "command", pl.target.id)
			}
			return nil
		})
	}
	_ = group.Wait()
	return len(ready), errors.Join(failed...)
}

func needsApply(entry *instanceapi.SharedTenant, held *tms.TenantStatus, sent bool) bool {
	if held.InstanceID == "" || !sent {
		return true
	}
	if held.State == string(instanceapi.TenantRemoved) {
		return false
	}
	applied := int64(0)
	if held.AppliedRevision != nil {
		applied = *held.AppliedRevision
	}
	refused := held.Refused != nil && held.Refused.Revision >= entry.Revision
	return applied < entry.Revision && !refused && held.State != string(instanceapi.TenantApplying)
}

func (a *Agent) report(
	ctx context.Context,
	desired *instanceapi.SharedTenantList,
	held []tms.TenantStatus,
	tmsErr error,
) error {
	report := &instanceapi.SharedStatusReport{
		Ready:      tmsErr == nil,
		Components: []instanceapi.ComponentStatus{{Name: tmsComponent, Ready: tmsErr == nil}},
		Tenants:    a.tenantStatuses(desired, held),
	}
	if a.p.Version != "" {
		version := a.p.Version
		report.Version = &version
	}
	if tmsErr != nil {
		message := clip("the TMS's managed listener cannot be read: " + tmsErr.Error())
		component := clipTo(message, maxComponentMessage)
		report.Message = &message
		report.Components[0].Message = &component
	}
	if storage, known := a.storageComponent(); known {
		report.Components = append(report.Components, storage)
		if !storage.Ready && report.Ready {
			report.Ready = false
			report.Message = storage.Message
		}
	}
	body, err := sonic.Marshal(report)
	if err != nil {
		return fmt.Errorf("encode the status report: %w", err)
	}
	now := a.now()
	if bytes.Equal(body, a.lastReport) && now.Sub(a.lastReportAt) < a.p.StatusEvery {
		return nil
	}
	ack, err := a.p.ControlPlane.ReportSharedStatus(ctx, report)
	if err != nil {
		return fmt.Errorf("the status report: %w", err)
	}
	a.lastReport, a.lastReportAt = body, now
	for _, refused := range ack.Refused {
		a.silenced[refused.InstanceId] = struct{}{}
		a.p.Logger.Warn(
			"the control plane says a tenant is not placed here; it is no longer reported",
			"instanceId",
			refused.InstanceId,
		)
	}
	return nil
}

func (a *Agent) tenantStatuses(
	desired *instanceapi.SharedTenantList,
	held []tms.TenantStatus,
) []instanceapi.SharedTenantStatus {
	listed := make(map[string]*instanceapi.SharedTenant, len(desired.Tenants))
	for i := range desired.Tenants {
		listed[desired.Tenants[i].InstanceId.String()] = &desired.Tenants[i]
	}
	out := make([]instanceapi.SharedTenantStatus, 0, len(held))
	for _, s := range held {
		id, err := uuid.Parse(s.InstanceID)
		if err != nil {
			continue
		}
		entry, ok := listed[s.InstanceID]
		if !ok {
			continue
		}
		if _, quiet := a.silenced[id]; quiet {
			continue
		}
		status := a.statusOf(id, &s)
		a.reportCommands(entry, &s, &status)
		out = append(out, status)
	}
	return out
}

func (a *Agent) statusOf(id uuid.UUID, s *tms.TenantStatus) instanceapi.SharedTenantStatus {
	out := instanceapi.SharedTenantStatus{
		InstanceId:      id,
		AppliedRevision: s.AppliedRevision,
		State:           instanceapi.SharedTenantStatusState(s.State),
	}
	if s.Administrator != nil {
		admin := &instanceapi.AdministratorStatus{SetUp: s.Administrator.SetUp}
		if s.Administrator.InviteTokenSHA256 != "" {
			token := s.Administrator.InviteTokenSHA256
			admin.InviteTokenSha256 = &token
		}
		out.Administrator = admin
	}
	if s.LastArchive != nil {
		if commandID, err := uuid.Parse(s.LastArchive.CommandID); err == nil {
			out.LastArchive = &instanceapi.SharedArchiveReport{
				CommandId: commandID,
				Location:  s.LastArchive.Location,
				SizeBytes: s.LastArchive.SizeBytes,
				Sha256:    s.LastArchive.SHA256,
				TakenAt:   s.LastArchive.TakenAt,
			}
		}
	}
	out.LastRestore = restoreReport(s.LastRestore)
	if s.SupportGrants != nil {
		out.SupportGrants = a.grantsOf(id, *s.SupportGrants)
	}
	if s.Refused != nil {
		out.Refused = &instanceapi.SharedTenantRefusal{
			Revision: s.Refused.Revision,
			Code:     instanceapi.SharedTenantRefusalCode(s.Refused.Code),
			Reason:   clip(s.Refused.Reason),
		}
	}
	return out
}

func clip(s string) string {
	return clipTo(s, maxMessage)
}

func clipTo(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit-1]) + "…"
}
