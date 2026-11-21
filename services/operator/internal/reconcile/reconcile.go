package reconcile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"helm.sh/helm/v3/pkg/chart"

	"github.com/emoss08/trenova/operator/internal/backup"
	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/databases"
	"github.com/emoss08/trenova/operator/internal/emailevents"
	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/release"
	"github.com/emoss08/trenova/operator/internal/selfupgrade"
	"github.com/emoss08/trenova/operator/internal/supportkeys"
	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/usage"
	"github.com/emoss08/trenova/operator/internal/values"
)

const (
	maxComponents   = 50
	maxMessage      = 500
	maxVersion      = 64
	maxFeatures     = 100
	usageRetry      = time.Minute
	usageTimeout    = time.Minute
	defaultUsage    = 15 * time.Minute
	inviteRecheck   = time.Minute
	inviteTimeout   = 30 * time.Second
	featureTimeout  = 30 * time.Second
	emailTimeout    = 30 * time.Second
	supportTimeout  = 30 * time.Second
	commandsTimeout = 2 * time.Minute

	logRequestsTimeout = 30 * time.Second
	emailEventsTimeout = 30 * time.Second
)

var (
	systemEmailHeldShape = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type ControlPlane interface {
	Spec(ctx context.Context, etag string) (controlplane.Desired, bool, error)
	ReportStatus(ctx context.Context, report *instanceapi.StatusReport) error
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

type Escrow interface {
	Ensure(ctx context.Context) error
}

type Usage interface {
	Run(ctx context.Context, allowances []instanceapi.SpecAllowance) (usage.Result, error)
}

type Commands interface {
	Run(ctx context.Context) error
	EmailEventsWaiting() bool
}

type EmailEvents interface {
	Run(ctx context.Context) (emailevents.Result, error)
}

type LogRequests interface {
	Run(ctx context.Context) error
	Relay(ctx context.Context) error
}

type Telemetry interface {
	Run(ctx context.Context) error
	Observe(p health.CheckParams)
}

type Administrators interface {
	InstallInvite(ctx context.Context, invite tms.Invite) (tms.Administrator, error)
}

type Features interface {
	PutFeatures(ctx context.Context, list *tms.FeatureList) error
	HeldFeatures(ctx context.Context) ([]string, error)
}

type SystemEmail interface {
	Sync(ctx context.Context, desired *instanceapi.SpecSystemEmail) error
	Held(ctx context.Context) (string, error)
}

type SupportKeys interface {
	Sync(ctx context.Context, desired *instanceapi.SpecSupport) error
}

type SupportGrants interface {
	SupportGrants(ctx context.Context) ([]tms.SupportGrant, error)
}

type Storage interface {
	Ensure(ctx context.Context, hostnames []string, revision int64) (values.Storage, error)
}

type Backups interface {
	Run(ctx context.Context, in *backup.Input) backup.Result
}

type Imports interface {
	Prepare(
		ctx context.Context,
		spec *instanceapi.InstanceSpec,
		storage *values.Storage,
	) (values.Import, error)
	Observe(
		ctx context.Context,
		spec *instanceapi.InstanceSpec,
		releaseJob string,
	) (status *instanceapi.ImportStatus, reported bool, err error)
}

type Operator interface {
	Converge(ctx context.Context, want string) selfupgrade.Status
}

type Databases interface {
	Gate(ctx context.Context, in databases.GateInput) databases.Gate
	Settle(ctx context.Context, in databases.SettleInput) databases.Result
}

type inviteAsked struct {
	tokenSHA256 string
	at          time.Time
	answer      *instanceapi.AdministratorStatus
}

type Loop struct {
	cp       ControlPlane
	creds    Credentials
	charts   Charts
	releases Releases
	health   Health
	escrow   Escrow
	usage    Usage
	admins   Administrators
	features Features
	email    SystemEmail
	keys     SupportKeys
	grants   SupportGrants
	storage  Storage
	backups  Backups
	dbs      Databases
	commands Commands
	events   EmailEvents
	logs     LogRequests
	telem    Telemetry
	operator Operator
	imports  Imports
	imported bool
	relJob   string
	located  values.Storage
	connects databases.Connected
	capacity values.Capacity
	base     map[string]any
	ownNS    string
	logger   *slog.Logger
	now      func() time.Time
	poll     time.Duration
	report   time.Duration
	every    time.Duration

	usageNext    time.Time
	invite       inviteAsked
	etag         string
	desired      *controlplane.Desired
	applied      int64
	last         instanceapi.StatusReport
	accepted     acceptedLists
	lastSent     time.Time
	reported     bool
	heartbeat    atomic.Int64
	enrolled     atomic.Bool
	retryAfter   time.Time
	commandsWait string
}

type Params struct {
	ControlPlane   ControlPlane
	Credentials    Credentials
	Charts         Charts
	Releases       Releases
	Health         Health
	Escrow         Escrow
	Usage          Usage
	Administrators Administrators
	Features       Features
	SystemEmail    SystemEmail
	SupportKeys    SupportKeys
	SupportGrants  SupportGrants
	Storage        Storage
	Backups        Backups
	Databases      Databases
	Commands       Commands
	EmailEvents    EmailEvents
	Logs           LogRequests
	Telemetry      Telemetry
	Operator       Operator
	Imports        Imports
	Capacity       values.Capacity
	Base           map[string]any
	Namespace      string
	Logger         *slog.Logger
	Now            func() time.Time
	Poll           time.Duration
	Report         time.Duration
	UsageEvery     time.Duration
}

func New(p *Params) (*Loop, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
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
		escrow:   p.Escrow,
		usage:    p.Usage,
		admins:   p.Administrators,
		features: p.Features,
		email:    p.SystemEmail,
		keys:     p.SupportKeys,
		grants:   p.SupportGrants,
		storage:  p.Storage,
		backups:  p.Backups,
		dbs:      p.Databases,
		commands: p.Commands,
		events:   p.EmailEvents,
		logs:     p.Logs,
		telem:    p.Telemetry,
		operator: p.Operator,
		imports:  p.Imports,
		capacity: p.Capacity,
		base:     p.Base,
		ownNS:    p.Namespace,
		logger:   p.Logger,
		now:      now,
		poll:     p.Poll,
		report:   p.Report,
		every:    every,
	}, nil
}

func (p *Params) validate() error {
	needs := []struct {
		name    string
		missing bool
	}{
		{"ControlPlane", p.ControlPlane == nil},
		{"Credentials", p.Credentials == nil},
		{"Charts", p.Charts == nil},
		{"Releases", p.Releases == nil},
		{"Health", p.Health == nil},
		{"Escrow", p.Escrow == nil},
		{"Usage", p.Usage == nil},
		{"Administrators", p.Administrators == nil},
		{"Features", p.Features == nil},
		{"SystemEmail", p.SystemEmail == nil},
		{"SupportKeys", p.SupportKeys == nil},
		{"SupportGrants", p.SupportGrants == nil},
		{"Storage", p.Storage == nil},
		{"Backups", p.Backups == nil},
		{"Databases", p.Databases == nil},
		{"Commands", p.Commands == nil},
		{"EmailEvents", p.EmailEvents == nil},
		{"Logs", p.Logs == nil},
		{"Telemetry", p.Telemetry == nil},
		{"Operator", p.Operator == nil},
		{"Imports", p.Imports == nil},
		{"Base", p.Base == nil},
		{"Namespace", p.Namespace == ""},
		{"Logger", p.Logger == nil},
		{"Poll", p.Poll <= 0},
		{"Report", p.Report <= 0},
	}
	var missing []string
	for _, n := range needs {
		if n.missing {
			missing = append(missing, n.name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("reconcile: the loop needs %s", strings.Join(missing, ", "))
	}
	return nil
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
	go func() {
		if err := l.logs.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			l.logger.Error("the log bundle worker stopped", "error", err)
		}
	}()
	var beside sync.WaitGroup
	defer beside.Wait()
	beside.Go(func() {
		if err := l.telem.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			l.logger.Error("the telemetry relay stopped", "error", err)
		}
	})
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
	ready := false
	defer func() { l.runCommands(ctx, ready) }()
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
	report, ok := l.converge(ctx)
	if !ok {
		return
	}
	ready = report.Ready
	l.heartbeat.Store(l.now().UnixNano())
	report.Import = l.importStatus(ctx)
	report.Administrator = l.administrator(ctx, report.Ready)
	l.backupReport(ctx, &report)
	l.databaseReport(ctx, &report)
	l.sendFeatures(ctx, report.Ready)
	report.Features = l.heldFeatures(ctx, report.Ready)
	l.sendSystemEmail(ctx, report.Ready)
	report.SystemEmailFingerprint = l.heldSystemEmail(ctx, report.Ready)
	l.sendSupportKeys(ctx, &report)
	report.SupportGrants = l.supportGrants(ctx, report.Ready)
	capabilities := values.Capabilities()
	report.Capabilities = &capabilities
	if l.shouldReport(&report) {
		if err = l.cp.ReportStatus(ctx, &report); err != nil {
			if !l.handle(ctx, err) {
				l.logger.Warn("cannot report status", "error", err)
			}
			return
		}
		l.last, l.lastSent, l.reported = report, l.now(), true
		l.accepted.record(&report)
	}
	l.relayUsage(ctx, report.Ready)
	l.relayLogRequests(ctx, report.Ready)
}

func (l *Loop) relayLogRequests(ctx context.Context, ready bool) {
	if !ready || l.desired.Spec.Lifecycle != instanceapi.Active {
		return
	}
	run, cancel := context.WithTimeout(ctx, logRequestsTimeout)
	defer cancel()
	if err := l.logs.Relay(run); err != nil && !l.handle(ctx, err) {
		l.logger.Warn("cannot carry out the log requests made in Trenova; retrying", "error", err)
	}
}

func (l *Loop) runCommands(ctx context.Context, ready bool) {
	if l.now().Before(l.retryAfter) {
		l.waitCommands("the control plane asked the operator to wait")
		return
	}
	if l.commandsWait != "" {
		l.logger.Info("carrying out the control plane's commands again")
		l.commandsWait = ""
	}
	run, cancel := context.WithTimeout(ctx, commandsTimeout)
	defer cancel()
	if err := l.commands.Run(run); err != nil && !l.handle(ctx, err) {
		l.logger.Warn("cannot carry out the control plane's commands; retrying", "error", err)
	}
	l.relayEmailEvents(ctx, ready)
}

func (l *Loop) relayEmailEvents(ctx context.Context, ready bool) {
	if !ready || l.desired == nil || l.desired.Spec.Lifecycle != instanceapi.Active ||
		l.now().Before(l.retryAfter) || !l.commands.EmailEventsWaiting() {
		return
	}
	run, cancel := context.WithTimeout(ctx, emailEventsTimeout)
	defer cancel()
	res, err := l.events.Run(run)
	if err != nil {
		if !l.handle(ctx, err) {
			l.logger.Warn("cannot relay the system email events; retrying",
				"listed", res.Listed, "error", err)
		}
		return
	}
	if res.Listed > 0 {
		l.logger.Info("system email events relayed", "listed", res.Listed, "handled", res.Handled)
	}
}

func (l *Loop) waitCommands(reason string) {
	if l.commandsWait == reason {
		return
	}
	l.commandsWait = reason
	l.logger.Info("the control plane's commands wait", "reason", reason)
}

func (l *Loop) importStatus(ctx context.Context) *instanceapi.ImportStatus {
	if l.desired.Spec.Import == nil {
		return nil
	}
	status, reported, err := l.imports.Observe(ctx, &l.desired.Spec, l.relJob)
	if err != nil {
		l.logger.Warn("cannot read where the import stands; the report leaves it out", "error", err)
		return nil
	}
	if !reported {
		return nil
	}
	return status
}

func (l *Loop) administrator(ctx context.Context, ready bool) *instanceapi.AdministratorStatus {
	invite := l.desired.Spec.FirstAdministrator.Invite
	if invite == nil || !ready || l.imported || l.desired.Spec.Import != nil ||
		l.desired.Spec.Lifecycle != instanceapi.Active {
		l.invite = inviteAsked{}
		return nil
	}
	if l.invite.answer != nil && l.invite.tokenSHA256 == invite.TokenSha256 &&
		l.now().Sub(l.invite.at) < inviteRecheck {
		return l.invite.answer
	}
	run, cancel := context.WithTimeout(ctx, inviteTimeout)
	defer cancel()
	got, err := l.admins.InstallInvite(run, tms.Invite{
		TokenSHA256: invite.TokenSha256,
		ExpiresAt:   invite.ExpiresAt,
	})
	if err != nil {
		l.invite = inviteAsked{}
		l.logger.Warn("cannot install the first administrator's invite; retrying", "error", err)
		return nil
	}
	answer := &instanceapi.AdministratorStatus{SetUp: got.SetUp}
	if got.InviteTokenSHA256 != "" {
		hash := got.InviteTokenSHA256
		answer.InviteTokenSha256 = &hash
	}
	if !sameAdministrator(l.invite.answer, answer) {
		l.logger.Info(
			"first administrator",
			"setUp", answer.SetUp,
			"inviteLive", answer.InviteTokenSha256 != nil,
		)
	}
	l.invite = inviteAsked{tokenSHA256: invite.TokenSha256, at: l.now(), answer: answer}
	return answer
}

func (l *Loop) sendFeatures(ctx context.Context, ready bool) {
	if !ready || l.desired.Spec.Lifecycle != instanceapi.Active {
		return
	}
	run, cancel := context.WithTimeout(ctx, featureTimeout)
	defer cancel()
	list := featureList(&l.desired.Spec)
	if err := l.features.PutFeatures(run, &list); err != nil {
		l.logger.Warn("cannot send the feature list to the TMS; retrying", "error", err)
		return
	}
	l.logger.Debug(
		"feature list sent",
		"features", len(list.Features),
		"addonEndings", len(list.AddonEndings),
		"planUrl", list.PlanURL != "",
		"planEnding", list.PlanEndsAt != nil,
	)
}

func featureList(spec *instanceapi.InstanceSpec) tms.FeatureList {
	list := tms.FeatureList{Features: spec.Features, AddonEndings: []tms.AddonEnding{}}
	if spec.Account != nil {
		list.PlanURL = spec.Account.PlanUrl
		list.FinalBackupDays = int(spec.Account.FinalBackupDays)
		if spec.Account.EndsAt != nil {
			ends := *spec.Account.EndsAt
			list.PlanEndsAt = &ends
		}
	}
	if spec.AddonEndings != nil {
		for _, e := range *spec.AddonEndings {
			list.AddonEndings = append(
				list.AddonEndings,
				tms.AddonEnding{Addon: e.Addon, EndsAt: e.EndsAt},
			)
		}
	}
	return list
}

func (l *Loop) heldFeatures(ctx context.Context, ready bool) *[]string {
	if !ready || l.desired.Spec.Lifecycle != instanceapi.Active {
		return nil
	}
	run, cancel := context.WithTimeout(ctx, featureTimeout)
	defer cancel()
	held, err := l.features.HeldFeatures(run)
	if err != nil {
		l.logger.Warn(
			"cannot read the features the TMS holds; the report leaves them out",
			"error",
			err,
		)
		return nil
	}
	if held == nil {
		held = []string{}
	}
	if len(held) > maxFeatures {
		held = held[:maxFeatures]
	}
	return &held
}

func (l *Loop) sendSystemEmail(ctx context.Context, ready bool) {
	if !ready || l.desired.Spec.Lifecycle != instanceapi.Active {
		return
	}
	run, cancel := context.WithTimeout(ctx, emailTimeout)
	defer cancel()
	err := l.email.Sync(run, l.desired.Spec.SystemEmail)
	if err == nil {
		return
	}
	if errors.Is(err, controlplane.ErrUnauthenticated) {
		l.handle(ctx, err)
		return
	}
	l.logger.Warn("cannot bring the TMS's system email sender up to date; retrying", "error", err)
}

func (l *Loop) heldSystemEmail(ctx context.Context, ready bool) *string {
	if !ready || l.desired.Spec.Lifecycle != instanceapi.Active {
		return nil
	}
	run, cancel := context.WithTimeout(ctx, emailTimeout)
	defer cancel()
	held, err := l.email.Held(run)
	if err != nil {
		l.logger.Warn(
			"cannot read the system email sender the TMS holds; the report leaves it out",
			"error",
			err,
		)
		return nil
	}
	if !systemEmailHeldShape.MatchString(held) {
		return nil
	}
	return &held
}

func (l *Loop) sendSupportKeys(ctx context.Context, report *instanceapi.StatusReport) {
	desired := l.desired.Spec.Support
	if !report.Ready || l.desired.Spec.Lifecycle != instanceapi.Active || desired == nil {
		return
	}
	run, cancel := context.WithTimeout(ctx, supportTimeout)
	defer cancel()
	err := l.keys.Sync(run, desired)
	if err == nil {
		return
	}
	l.logger.Warn("cannot bring the TMS's support access keys up to date; retrying", "error", err)
	if report.Message == nil {
		msg := trim("support access keys not delivered: " + err.Error())
		report.Message = &msg
	}
}

func (l *Loop) supportGrants(ctx context.Context, ready bool) *[]instanceapi.SupportGrant {
	if !ready || l.desired.Spec.Lifecycle != instanceapi.Active {
		return nil
	}
	run, cancel := context.WithTimeout(ctx, supportTimeout)
	defer cancel()
	held, err := l.grants.SupportGrants(run)
	if err != nil {
		l.logger.Warn(
			"cannot read the TMS's live support grants; the report leaves them out",
			"error",
			err,
		)
		return nil
	}
	out, dropped := supportkeys.Grants(held)
	for _, id := range dropped {
		l.logger.Warn(
			"the TMS listed a support grant the contract cannot carry; it is left out",
			"grant",
			id,
		)
	}
	return &out
}

func (l *Loop) relayUsage(ctx context.Context, ready bool) {
	if !ready || l.desired.Spec.Lifecycle != instanceapi.Active ||
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

func (l *Loop) converge(ctx context.Context) (instanceapi.StatusReport, bool) {
	report := instanceapi.StatusReport{Components: []instanceapi.ComponentStatus{}}
	l.connects = databases.Connected{}
	if own := l.creds.InstanceID(); l.desired.Spec.InstanceId != own {
		return l.refuse(ctx, &report, instanceapi.RefusedRevisionCodeInvalid, fmt.Sprintf(
			"the spec is for instance %s and this operator is instance %s",
			l.desired.Spec.InstanceId,
			own,
		)), true
	}
	op := l.operator.Converge(ctx, l.desired.Spec.Release.Version)
	report.Operator = operatorReport(op)
	if op.Holds() {
		return l.hold(ctx, &report, op), true
	}
	located, err := l.objectStorage(ctx)
	if err != nil {
		if l.handle(ctx, err) {
			return report, false
		}
		l.logger.Error("cannot keep the instance's storage; nothing is applied", "error", err)
		return l.notReady(
			&report,
			"nothing is applied without the instance's storage: "+err.Error(),
		), true
	}
	l.located = located
	spec := l.desired.Spec
	spec.Plan = l.capacity.Fit(&spec.Plan)
	rendered, err := values.Render(&spec, l.base, l.ownNS, &located)
	if errors.Is(err, values.ErrInvalidSpec) || errors.Is(err, values.ErrSchemaVersion) {
		return l.refuse(ctx, &report, instanceapi.RefusedRevisionCodeInvalid, err.Error()), true
	}
	if err != nil {
		return l.notReady(&report, err.Error()), true
	}
	imp, err := l.imports.Prepare(ctx, &l.desired.Spec, &located)
	if err == nil {
		err = values.ApplyImport(rendered.Values, &located, &imp)
	}
	if errors.Is(err, values.ErrInvalidSpec) {
		return l.refuse(ctx, &report, instanceapi.RefusedRevisionCodeInvalid, err.Error()), true
	}
	if err != nil {
		if l.handle(ctx, err) {
			return report, false
		}
		return l.notReady(
			&report,
			"nothing is applied without the import's state: "+err.Error(),
		), true
	}
	l.imported = imp.ImportedBefore
	if err = l.escrow.Ensure(ctx); err != nil {
		if l.handle(ctx, err) {
			return report, false
		}
		l.logger.Error("cannot keep the escrowed secrets; nothing is applied", "error", err)
		return l.notReady(
			&report,
			"nothing is applied without the escrowed secrets: "+err.Error(),
		), true
	}
	return l.apply(ctx, &report, rendered), true
}

func (l *Loop) apply(
	ctx context.Context,
	base *instanceapi.StatusReport,
	rendered values.Rendered,
) instanceapi.StatusReport {
	report := *base
	ch, err := l.charts.Load(rendered.Version)
	if err != nil {
		return l.notReady(&report, err.Error())
	}
	deployed, err := l.releases.Deployed()
	if err != nil {
		return l.notReady(&report, err.Error())
	}
	l.connects = connectedTo(&deployed)
	gate := l.dbs.Gate(ctx, databases.GateInput{Spec: &l.desired.Spec, Connected: l.connects})
	if gate.Refused != "" {
		l.connects = databases.Connected{}
		report = l.stay(ctx, &report, &deployed, gate.Message)
		report.Refused = l.refusal(gate.Refused, gate.Message)
		return report
	}
	if gate.Hold {
		return l.stay(ctx, &report, &deployed, gate.Message)
	}
	need, err := release.NeedsApply(&deployed, ch, rendered.Values)
	if err != nil {
		return l.notReady(&report, err.Error())
	}
	if need {
		if err = l.releases.Apply(ctx, ch, rendered.Values); err != nil {
			l.connects = databases.Connected{}
			l.logger.Error("cannot apply the spec", "revision", l.desired.Revision, "error", err)
			return l.notReady(&report, err.Error())
		}
		if deployed, err = l.releases.Deployed(); err != nil {
			l.connects = databases.Connected{}
			return l.notReady(&report, err.Error())
		}
		l.connects = connectedTo(&deployed)
	}
	l.applied = l.desired.Revision
	return l.observe(ctx, &report, &deployed, rendered.Values)
}

// stay keeps the deployed release, reports it as it runs and says why the
// desired revision waits.
func (l *Loop) stay(
	ctx context.Context,
	base *instanceapi.StatusReport,
	deployed *release.Deployed,
	why string,
) instanceapi.StatusReport {
	report := *base
	if !deployed.Exists {
		return l.notReady(&report, why)
	}
	report = l.observe(ctx, &report, deployed, deployed.Config)
	msg := trim(why)
	report.Message = &msg
	return report
}

// refuse declines the desired revision as written: nothing of it is applied
// (its retirements included, so the databases are left as they are), the
// release that runs is reported as it is, the message says why for people and
// refused says it for the control plane. It is worked out again on every
// tick, so it stands while the cause does and is gone once a later revision
// is applied or this one becomes applicable.
func (l *Loop) refuse(
	ctx context.Context,
	base *instanceapi.StatusReport,
	code instanceapi.RefusedRevisionCode,
	why string,
) instanceapi.StatusReport {
	report := *base
	l.logger.Warn(
		"the revision is refused",
		"revision",
		l.desired.Revision,
		"code",
		code,
		"reason",
		why,
	)
	deployed, err := l.releases.Deployed()
	if err != nil {
		report = l.notReady(&report, why)
	} else {
		report = l.stay(ctx, &report, &deployed, why)
	}
	report.Refused = l.refusal(code, why)
	return report
}

func (l *Loop) refusal(
	code instanceapi.RefusedRevisionCode,
	why string,
) *instanceapi.RefusedRevision {
	return &instanceapi.RefusedRevision{Revision: l.desired.Revision, Code: code, Reason: trim(why)}
}

func connectedTo(d *release.Deployed) databases.Connected {
	out := databases.Connected{Known: true, Exists: d.Exists}
	if pg, ok := d.Config["postgres"].(map[string]any); ok && d.Exists {
		out.Generation, _ = pg["generation"].(string)
	}
	return out
}

func (l *Loop) hold(
	ctx context.Context,
	base *instanceapi.StatusReport,
	op selfupgrade.Status,
) instanceapi.StatusReport {
	report := *base
	deployed, err := l.releases.Deployed()
	if err != nil {
		return l.notReady(&report, err.Error())
	}
	l.connects = connectedTo(&deployed)
	if !deployed.Exists {
		return l.notReady(&report, fmt.Sprintf(
			"Trenova is installed once the operator runs %s: %s", op.Target, op.Message,
		))
	}
	report = l.observe(ctx, &report, &deployed, deployed.Config)
	held := trim(fmt.Sprintf(
		"Trenova stays on %s until the operator runs %s: %s",
		deployed.ChartVersion, op.Target, op.Message,
	))
	report.Message = &held
	return report
}

func (l *Loop) observe(
	ctx context.Context,
	base *instanceapi.StatusReport,
	deployed *release.Deployed,
	vals map[string]any,
) instanceapi.StatusReport {
	report := *base
	if l.applied > 0 {
		applied := l.applied
		report.AppliedRevision = &applied
	}
	if deployed.Exists {
		v := deployed.AppVersion
		report.Version = &v
		l.relJob = deployed.ReleaseJob
	}
	params := healthParams(deployed.ReleaseJob, vals)
	l.telem.Observe(params)
	comps, err := l.health.Check(ctx, params)
	if err != nil {
		return l.notReady(&report, err.Error())
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

func healthParams(releaseJob string, vals map[string]any) health.CheckParams {
	stopped, _ := vals["stopped"].(bool)
	hibernated := false
	generation := ""
	if pg, ok := vals["postgres"].(map[string]any); ok {
		hibernated, _ = pg["hibernate"].(bool)
		generation, _ = pg["generation"].(string)
	}
	return health.CheckParams{
		ReleaseJob: releaseJob,
		Stopped:    stopped,
		Hibernated: hibernated,
		Generation: generation,
	}
}

func (l *Loop) objectStorage(ctx context.Context) (values.Storage, error) {
	hostnames := []string{l.desired.Spec.Hostname}
	if custom, err := values.CustomHostname(&l.desired.Spec); err == nil && custom != "" {
		hostnames = append(hostnames, custom)
	}
	return l.storage.Ensure(ctx, hostnames, l.desired.Revision)
}

func (l *Loop) backupReport(ctx context.Context, report *instanceapi.StatusReport) {
	running, ok := l.runningOn()
	if !ok {
		return
	}
	applied := report.AppliedRevision != nil && *report.AppliedRevision == l.desired.Revision
	res := l.backups.Run(ctx, &backup.Input{
		Spec:       &l.desired.Spec,
		Generation: running,
		Revision:   l.desired.Revision,
		Applied:    applied,
		Ready:      report.Ready,
		Storage:    l.located,
	})
	report.Backup = res.Health
	report.Restore = res.Restore
	if res.Final != nil && applied {
		report.FinalBackup = res.Final
	}
	if res.Health != nil && res.Backups != nil &&
		backupsDigest(*res.Backups) != l.accepted.backups {
		h := *res.Health
		list := slices.Clone(*res.Backups)
		h.Backups = &list
		report.Backup = &h
	}
}

// runningOn is the database the application runs on: the one the deployed
// release points it at, read again when this tick's converge stopped before
// reading the release. Without a release nothing runs yet, and the spec's
// database is the one the chart makes. When the release cannot be read the
// backup report is left out, since any other database's health would be read
// as the instance's; the control plane keeps the last one it has.
func (l *Loop) runningOn() (string, bool) {
	c := l.connects
	if !c.Known {
		deployed, err := l.releases.Deployed()
		if err != nil {
			l.logger.Warn("cannot read the release; the report leaves backups out", "error", err)
			return "", false
		}
		c = connectedTo(&deployed)
	}
	if !c.Exists {
		return databases.Live(&l.desired.Spec), true
	}
	return c.Generation, true
}

func (l *Loop) databaseReport(ctx context.Context, report *instanceapi.StatusReport) {
	applied := report.AppliedRevision != nil && *report.AppliedRevision == l.desired.Revision
	res := l.dbs.Settle(ctx, databases.SettleInput{
		Spec:      &l.desired.Spec,
		Connected: l.connects,
		Applied:   applied,
		Ready:     report.Ready,
	})
	if res.Message != "" {
		msg := res.Message
		if report.Message != nil {
			msg = *report.Message + "; " + msg
		}
		msg = trim(msg)
		report.Message = &msg
	}
	if res.Databases != nil && databasesDigest(*res.Databases) != l.accepted.databases {
		list := slices.Clone(*res.Databases)
		report.Databases = &list
	}
}

// acceptedLists holds the digest of each list the control plane last
// answered 2xx to. A list goes on a report when its digest differs, which
// includes the first report after a start; a refused or failed report records
// nothing, so the list goes again on the next tick.
type acceptedLists struct {
	backups   string
	databases string
}

func (a *acceptedLists) record(report *instanceapi.StatusReport) {
	if b := report.Backup; b != nil && b.Backups != nil {
		a.backups = backupsDigest(*b.Backups)
	}
	if report.Databases != nil {
		a.databases = databasesDigest(*report.Databases)
	}
}

func (a *acceptedLists) carried(report *instanceapi.StatusReport) bool {
	return (report.Backup != nil && report.Backup.Backups != nil) || report.Databases != nil
}

func listDigest(rows [][]string) string {
	sum := sha256.New()
	for _, fields := range rows {
		sum.Write([]byte(strings.Join(fields, "\x00") + "\n"))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func backupsDigest(list []instanceapi.BackupEntry) string {
	rows := make([][]string, 0, len(list))
	for _, e := range list {
		origin := ""
		if e.Origin != nil {
			origin = string(*e.Origin)
		}
		rows = append(rows, []string{
			e.Id,
			string(e.Phase),
			e.StartedAt.UTC().Format(time.RFC3339Nano),
			stampNano(e.CompletedAt),
			origin,
			str(e.RequestId),
			sizeText(e.SizeBytes),
			str(e.Message),
		})
	}
	return listDigest(rows)
}

func databasesDigest(list []instanceapi.DatabaseState) string {
	rows := make([][]string, 0, len(list))
	for _, d := range list {
		rows = append(rows, []string{
			d.Generation,
			string(d.State),
			strconv.FormatBool(d.Live),
			sizeText(d.SizeBytes),
			stampNano(d.CreatedAt),
		})
	}
	return listDigest(rows)
}

func sizeText(n *int64) string {
	if n == nil {
		return ""
	}
	return strconv.FormatInt(*n, 10)
}

func stampNano(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func backupSummary(r *instanceapi.StatusReport) string {
	out := ""
	if b := r.Backup; b != nil {
		out += string(b.WalArchiving) + "|" + str(b.Message) + "|" + stamp(b.LastBaseBackupAt)
		if q := b.Request; q != nil {
			out += "|" + q.Id + "/" + string(q.Phase)
		}
	}
	if p := r.Restore; p != nil {
		out += "|" + p.Id.String() + "/" + string(p.Phase)
	}
	if f := r.FinalBackup; f != nil {
		out += "|final/" + f.Id
	}
	return out
}

func stamp(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func (l *Loop) notReady(base *instanceapi.StatusReport, msg string) instanceapi.StatusReport {
	report := *base
	if l.applied > 0 {
		applied := l.applied
		report.AppliedRevision = &applied
	}
	m := trim(msg)
	report.Message = &m
	report.Ready = false
	return report
}

func (l *Loop) shouldReport(r *instanceapi.StatusReport) bool {
	if !l.reported || l.now().Sub(l.lastSent) >= l.report || l.accepted.carried(r) {
		return true
	}
	return r.Ready != l.last.Ready ||
		value(r.AppliedRevision) != value(l.last.AppliedRevision) ||
		str(r.Version) != str(l.last.Version) ||
		str(r.Message) != str(l.last.Message) ||
		!sameAdministrator(r.Administrator, l.last.Administrator) ||
		!sameFeatures(r.Features, l.last.Features) ||
		!supportkeys.SameGrants(r.SupportGrants, l.last.SupportGrants) ||
		str(r.SystemEmailFingerprint) != str(l.last.SystemEmailFingerprint) ||
		backupSummary(r) != backupSummary(&l.last) ||
		refusedSummary(r.Refused) != refusedSummary(l.last.Refused) ||
		importSummary(r.Import) != importSummary(l.last.Import) ||
		operatorSummary(r.Operator) != operatorSummary(l.last.Operator)
}

func importSummary(i *instanceapi.ImportStatus) string {
	if i == nil {
		return ""
	}
	return i.Id.String() + "|" + string(i.State) + "|" + str(i.Reason)
}

func refusedSummary(r *instanceapi.RefusedRevision) string {
	if r == nil {
		return ""
	}
	return strconv.FormatInt(r.Revision, 10) + "|" + string(r.Code) + "|" + r.Reason
}

func operatorReport(st selfupgrade.Status) *instanceapi.OperatorStatus {
	out := &instanceapi.OperatorStatus{Version: health.Clean(st.Version, maxVersion)}
	if st.Holds() {
		out.Upgrade = &instanceapi.OperatorUpgrade{
			Version: health.Clean(st.Target, maxVersion),
			Phase:   instanceapi.OperatorUpgradePhase(st.Phase),
		}
	}
	if st.Message != "" {
		msg := trim(st.Message)
		out.Message = &msg
	}
	return out
}

func operatorSummary(o *instanceapi.OperatorStatus) string {
	if o == nil {
		return ""
	}
	out := o.Version + "|" + str(o.Message)
	if u := o.Upgrade; u != nil {
		out += "|" + u.Version + "/" + string(u.Phase)
	}
	return out
}

func sameFeatures(held, last *[]string) bool {
	if held == nil {
		return true
	}
	return last != nil && slices.Equal(*held, *last)
}

func sameAdministrator(a, b *instanceapi.AdministratorStatus) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.SetUp == b.SetUp && str(a.InviteTokenSha256) == str(b.InviteTokenSha256)
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

func trim(s string) string { return health.Clean(s, maxMessage) }

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
