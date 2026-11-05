package reconcile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"helm.sh/helm/v3/pkg/chart"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/release"
	"github.com/emoss08/trenova/operator/internal/usage"
	"github.com/emoss08/trenova/operator/internal/values"
)

const (
	maxComponents = 50
	maxMessage    = 500
	usageRetry    = time.Minute
	usageTimeout  = time.Minute
	defaultUsage  = 15 * time.Minute
)

type ControlPlane interface {
	Spec(ctx context.Context, etag string) (controlplane.Desired, bool, error)
	ReportStatus(ctx context.Context, report instanceapi.StatusReport) error
}

type Credentials interface {
	Start(ctx context.Context) error
	RenewIfDue(ctx context.Context) error
	Recover(ctx context.Context) error
	InstanceID() uuid.UUID
}

type Charts interface {
	Load(version string) (*chart.Chart, error)
}

type Releases interface {
	Deployed() (release.Deployed, error)
	Apply(ctx context.Context, ch *chart.Chart, vals map[string]any) error
}

type Health interface {
	Check(ctx context.Context, p health.CheckParams) ([]health.Component, error)
}

type Usage interface {
	Run(ctx context.Context, allowances []instanceapi.SpecAllowance) (usage.Result, error)
}

type Loop struct {
	cp       ControlPlane
	creds    Credentials
	charts   Charts
	releases Releases
	health   Health
	usage    Usage
	base     map[string]any
	ownNS    string
	logger   *slog.Logger
	now      func() time.Time
	poll     time.Duration
	report   time.Duration
	every    time.Duration

	usageNext  time.Time
	etag       string
	desired    *controlplane.Desired
	applied    int64
	last       instanceapi.StatusReport
	lastSent   time.Time
	reported   bool
	heartbeat  atomic.Int64
	enrolled   atomic.Bool
	retryAfter time.Time
}

type Params struct {
	ControlPlane ControlPlane
	Credentials  Credentials
	Charts       Charts
	Releases     Releases
	Health       Health
	Usage        Usage
	Base         map[string]any
	Namespace    string
	Logger       *slog.Logger
	Now          func() time.Time
	Poll         time.Duration
	Report       time.Duration
	UsageEvery   time.Duration
}

func New(p *Params) *Loop {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	every := p.UsageEvery
	if every <= 0 {
		every = defaultUsage
	}
	return &Loop{
		cp:       p.ControlPlane,
		creds:    p.Credentials,
		charts:   p.Charts,
		releases: p.Releases,
		health:   p.Health,
		usage:    p.Usage,
		base:     p.Base,
		ownNS:    p.Namespace,
		logger:   p.Logger,
		now:      now,
		poll:     p.Poll,
		report:   p.Report,
		every:    every,
	}
}

func (l *Loop) Healthy() error {
	if !l.enrolled.Load() {
		return errors.New("not enrolled")
	}
	last := time.Unix(0, l.heartbeat.Load())
	if l.now().Sub(last) > 4*l.poll+time.Minute {
		return fmt.Errorf("no reconcile since %s", last.UTC().Format(time.RFC3339))
	}
	return nil
}

func (l *Loop) Run(ctx context.Context) error {
	for {
		err := l.creds.Start(ctx)
		if err == nil {
			break
		}
		l.logger.Error("cannot start: the operator has no usable credentials", "error", err)
		if !sleep(ctx, 30*time.Second) {
			return ctx.Err()
		}
	}
	l.enrolled.Store(true)
	ticker := time.NewTicker(l.poll)
	defer ticker.Stop()
	for {
		l.Tick(ctx)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (l *Loop) Tick(ctx context.Context) {
	if l.now().Before(l.retryAfter) {
		return
	}
	if err := l.creds.RenewIfDue(ctx); err != nil && !l.handle(ctx, err) {
		l.logger.Warn(
			"certificate renewal failed; the current certificate is still used",
			"error",
			err,
		)
	}
	desired, changed, err := l.cp.Spec(ctx, l.etag)
	if err != nil {
		if !l.handle(ctx, err) {
			l.logger.Warn("cannot read the spec", "error", err)
		}
		return
	}
	l.enrolled.Store(true)
	if changed {
		l.desired, l.etag = &desired, desired.ETag
		l.logger.Info(
			"desired spec",
			"revision",
			desired.Revision,
			"lifecycle",
			desired.Spec.Lifecycle,
			"version",
			desired.Spec.Release.Version,
		)
	}
	if l.desired == nil {
		return
	}
	report := l.converge(ctx)
	l.heartbeat.Store(l.now().UnixNano())
	if l.shouldReport(report) {
		if err = l.cp.ReportStatus(ctx, report); err != nil {
			if !l.handle(ctx, err) {
				l.logger.Warn("cannot report status", "error", err)
			}
			return
		}
		l.last, l.lastSent, l.reported = report, l.now(), true
	}
	l.relayUsage(ctx, report.Ready)
}

func (l *Loop) relayUsage(ctx context.Context, ready bool) {
	if l.usage == nil || !ready || l.desired.Spec.Lifecycle != instanceapi.Active ||
		l.now().Before(l.usageNext) {
		return
	}
	run, cancel := context.WithTimeout(ctx, usageTimeout)
	defer cancel()
	res, err := l.usage.Run(run, l.desired.Spec.Allowances)
	if err == nil {
		l.usageNext = l.now().Add(l.every)
		l.logger.Debug(
			"usage relayed",
			"readings", res.Readings,
			"dropped", res.Dropped,
			"meters", res.Meters,
		)
		return
	}
	l.usageNext = l.now().Add(usageRetry)
	var later *controlplane.RetryLaterError
	switch {
	case errors.As(err, &later):
		l.usageNext = l.now().Add(max(later.RetryAfter, usageRetry))
		l.logger.Warn(
			"the control plane asks to wait before the next usage report",
			"reason", later.Reason,
			"retryAfter", later.RetryAfter,
		)
	case errors.Is(err, controlplane.ErrUnauthenticated):
		l.handle(ctx, err)
	default:
		l.logger.Warn("cannot relay usage; retrying in a minute", "error", err)
	}
}

func (l *Loop) converge(ctx context.Context) instanceapi.StatusReport {
	report := instanceapi.StatusReport{Components: []instanceapi.ComponentStatus{}}
	if own := l.creds.InstanceID(); l.desired.Spec.InstanceId != own {
		return l.notReady(report, fmt.Sprintf(
			"the spec is for instance %s and this operator is instance %s",
			l.desired.Spec.InstanceId,
			own,
		))
	}
	rendered, err := values.Render(&l.desired.Spec, l.base, l.ownNS)
	if err != nil {
		return l.notReady(report, err.Error())
	}
	ch, err := l.charts.Load(rendered.Version)
	if err != nil {
		return l.notReady(report, err.Error())
	}
	deployed, err := l.releases.Deployed()
	if err != nil {
		return l.notReady(report, err.Error())
	}
	need, err := release.NeedsApply(deployed, ch, rendered.Values)
	if err != nil {
		return l.notReady(report, err.Error())
	}
	if need {
		if err = l.releases.Apply(ctx, ch, rendered.Values); err != nil {
			l.logger.Error("cannot apply the spec", "revision", l.desired.Revision, "error", err)
			return l.notReady(report, err.Error())
		}
		if deployed, err = l.releases.Deployed(); err != nil {
			return l.notReady(report, err.Error())
		}
	}
	l.applied = l.desired.Revision
	applied := l.applied
	report.AppliedRevision = &applied
	if deployed.Exists {
		v := deployed.AppVersion
		report.Version = &v
	}
	stopped, _ := rendered.Values["stopped"].(bool)
	hibernated := false
	if pg, ok := rendered.Values["postgres"].(map[string]any); ok {
		hibernated, _ = pg["hibernate"].(bool)
	}
	comps, err := l.health.Check(ctx, health.CheckParams{
		HelmRevision: deployed.Revision,
		Stopped:      stopped,
		Hibernated:   hibernated,
	})
	if err != nil {
		return l.notReady(report, err.Error())
	}
	report.Ready = true
	for i, c := range comps {
		if i < maxComponents {
			cs := instanceapi.ComponentStatus{Name: c.Name, Ready: c.Ready}
			if c.Message != "" {
				msg := c.Message
				cs.Message = &msg
			}
			report.Components = append(report.Components, cs)
		}
		if !c.Ready {
			if report.Ready {
				msg := trim(c.Name + ": " + c.Message)
				report.Message = &msg
			}
			report.Ready = false
		}
	}
	if report.Ready && l.desired.Spec.Lifecycle == instanceapi.Decommissioned {
		msg := "workloads stopped; PostgreSQL stays up for the final backup"
		report.Message = &msg
	}
	return report
}

func (l *Loop) notReady(report instanceapi.StatusReport, msg string) instanceapi.StatusReport {
	if l.applied > 0 {
		applied := l.applied
		report.AppliedRevision = &applied
	}
	m := trim(msg)
	report.Message = &m
	report.Ready = false
	return report
}

func (l *Loop) shouldReport(r instanceapi.StatusReport) bool {
	if !l.reported || l.now().Sub(l.lastSent) >= l.report {
		return true
	}
	return r.Ready != l.last.Ready ||
		value(r.AppliedRevision) != value(l.last.AppliedRevision) ||
		str(r.Version) != str(l.last.Version) ||
		str(r.Message) != str(l.last.Message)
}

func (l *Loop) handle(ctx context.Context, err error) bool {
	var later *controlplane.RetryLaterError
	switch {
	case errors.As(err, &later):
		l.retryAfter = l.now().Add(later.RetryAfter)
		l.logger.Warn(
			"waiting for the control plane",
			"reason",
			later.Reason,
			"retryAfter",
			later.RetryAfter,
		)
		return true
	case errors.Is(err, controlplane.ErrUnauthenticated):
		if rerr := l.creds.Recover(ctx); rerr != nil {
			l.logger.Error("the control plane refuses this operator", "error", rerr)
			l.retryAfter = l.now().Add(5 * time.Minute)
			if errors.Is(rerr, credential.ErrReenrolmentNeeded) {
				l.enrolled.Store(false)
			}
			return true
		}
		l.enrolled.Store(true)
		l.etag = ""
		return true
	}
	return false
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func trim(s string) string {
	if len(s) <= maxMessage {
		return s
	}
	return s[:maxMessage-1] + "…"
}

func value(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
