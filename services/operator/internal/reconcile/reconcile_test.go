package reconcile_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/commands"
	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/credential/secretstore"
	"github.com/emoss08/trenova/operator/internal/databases"
	"github.com/emoss08/trenova/operator/internal/escrow"
	"github.com/emoss08/trenova/operator/internal/escrow/escrowsecret"
	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/importer"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/reconcile"
	"github.com/emoss08/trenova/operator/internal/release"
	"github.com/emoss08/trenova/operator/internal/selfupgrade"
	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/tmssecret"
	"github.com/emoss08/trenova/operator/internal/tmstest"
	"github.com/emoss08/trenova/operator/internal/usage"
	"github.com/emoss08/trenova/operator/internal/values"
)

var chartDir = filepath.Join("..", "..", "..", "..", "deploy", "helm", "trenova")

type fakeHealth struct {
	mu     sync.Mutex
	ready  bool
	params []health.CheckParams
}

func (f *fakeHealth) Check(_ context.Context, p health.CheckParams) ([]health.Component, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.params = append(f.params, p)
	if f.ready {
		return []health.Component{{Name: "api", Ready: true}, {Name: "postgres", Ready: true}}, nil
	}
	return []health.Component{
		{Name: "api", Ready: true},
		{Name: "release", Message: "running"},
	}, nil
}

func (f *fakeHealth) set(ready bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ready = ready
}

func (f *fakeHealth) last() health.CheckParams {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.params[len(f.params)-1]
}

type countingReleases struct {
	*release.Releaser
	applies int
	fail    error
	before  func(vals map[string]any)
}

func (c *countingReleases) Apply(ctx context.Context, ch *chart.Chart, vals map[string]any) error {
	c.applies++
	if c.before != nil {
		c.before(vals)
	}
	if c.fail != nil {
		return c.fail
	}
	return c.Releaser.Apply(ctx, ch, vals)
}

type noDatabases struct{}

func (noDatabases) Gate(
	context.Context,
	databases.GateInput,
) databases.Gate {
	return databases.Gate{}
}

func (noDatabases) Settle(context.Context, databases.SettleInput) databases.Result {
	return databases.Result{}
}

type fakeUsage struct {
	mu   sync.Mutex
	now  func() time.Time
	runs []time.Time
	err  error
}

func (f *fakeUsage) Run(_ context.Context, _ []instanceapi.SpecAllowance) (usage.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.runs = append(f.runs, f.now())
	return usage.Result{}, f.err
}

func (f *fakeUsage) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.runs)
}

func (f *fakeUsage) fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

type fakeAdmins struct {
	mu      sync.Mutex
	invites []tms.Invite
	answer  tms.Administrator
	err     error
}

func (f *fakeAdmins) InstallInvite(
	_ context.Context,
	invite tms.Invite,
) (tms.Administrator, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.invites = append(f.invites, invite)
	if f.err != nil {
		return tms.Administrator{}, f.err
	}
	if f.answer.SetUp {
		return tms.Administrator{SetUp: true}, nil
	}
	return tms.Administrator{InviteTokenSHA256: invite.TokenSHA256}, nil
}

func (f *fakeAdmins) calls() []tms.Invite {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]tms.Invite(nil), f.invites...)
}

func (f *fakeAdmins) set(answer tms.Administrator, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.answer, f.err = answer, err
}

type fakeFeatures struct {
	mu      sync.Mutex
	lists   [][]string
	sent    []tms.FeatureList
	err     error
	held    []string
	heldErr error
}

func (f *fakeFeatures) HeldFeatures(context.Context) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.heldErr != nil {
		return nil, f.heldErr
	}
	return append([]string{}, f.held...), nil
}

func (f *fakeFeatures) hold(features []string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.held, f.heldErr = features, err
}

func (f *fakeFeatures) PutFeatures(_ context.Context, list *tms.FeatureList) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lists = append(f.lists, append([]string{}, list.Features...))
	f.sent = append(f.sent, *list)
	return f.err
}

func (f *fakeFeatures) lastSent() (tms.FeatureList, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.sent) == 0 {
		return tms.FeatureList{}, false
	}
	return f.sent[len(f.sent)-1], true
}

func (f *fakeFeatures) calls() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]string(nil), f.lists...)
}

func (f *fakeFeatures) fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

type world struct {
	cp       *controlplanetest.Server
	kube     *fake.Clientset
	loop     *reconcile.Loop
	health   *fakeHealth
	releases *countingReleases
	usage    *fakeUsage
	admins   *fakeAdmins
	features *fakeFeatures
	email    *fakeSystemEmail
	keys     *fakeSupportKeys
	grants   *fakeSupportGrants
	operator *fakeOperator
	logs     *fakeLogKeeping
	events   *fakeEmailEvents
	telem    *fakeTelemetry
	client   *controlplane.Client
	now      time.Time
}

type relayParams struct {
	kube   *fake.Clientset
	client *controlplane.Client
	now    func() time.Time
}

func newWorld(t *testing.T) *world {
	t.Helper()
	return newWorldWith(t, controlplanetest.Spec(uuid.New()), nil)
}

func newWorldWith(
	t *testing.T,
	spec instanceapi.InstanceSpec,
	relay func(relayParams) reconcile.Usage,
	options ...func(*reconcile.Params),
) *world {
	t.Helper()
	cp := controlplanetest.New(t, spec)
	kube := fake.NewClientset()
	body, err := sonic.Marshal(cp.Bundle(7 * 24 * time.Hour))
	require.NoError(t, err)
	_, err = kube.CoreV1().Secrets("trenova-system").Create(t.Context(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "enrolment", Namespace: "trenova-system"},
		Data:       map[string][]byte{credential.BundleKey: body},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(testWriter{t}, nil))
	transport := &credential.Transport{}
	client, err := controlplane.New(
		cp.URL,
		&http.Client{Transport: transport, Timeout: 10 * time.Second},
	)
	require.NoError(t, err)
	w := &world{
		cp:       cp,
		kube:     kube,
		health:   &fakeHealth{},
		admins:   &fakeAdmins{},
		features: &fakeFeatures{},
		email:    &fakeSystemEmail{},
		keys:     &fakeSupportKeys{},
		grants:   &fakeSupportGrants{},
		operator: &fakeOperator{status: selfupgrade.Status{
			Version: "0.11.0", Phase: selfupgrade.PhaseCurrent,
		}},
		logs:   &fakeLogKeeping{},
		events: &fakeEmailEvents{},
		telem:  &fakeTelemetry{},
		client: client,
		now:    time.Now(),
	}
	w.usage = &fakeUsage{now: func() time.Time { return w.now }}
	var relayer reconcile.Usage = w.usage
	if relay != nil {
		relayer = relay(
			relayParams{kube: kube, client: client, now: func() time.Time { return w.now }},
		)
	}
	creds := credential.NewManager(&credential.ManagerParams{
		Store: secretstore.New(secretstore.Params{
			Client: kube, Namespace: "trenova-system", Enrolment: "enrolment", Identity: "identity",
		}),
		Transport: transport,
		Renewer:   client,
		Logger:    logger,
		Now:       func() time.Time { return w.now },
	})
	require.NoError(t, creds.Start(t.Context()))

	caps := chartutil.DefaultCapabilities.Copy()
	caps.KubeVersion = chartutil.KubeVersion{Version: "v1.31.5", Major: "1", Minor: "31"}
	cfg := &action.Configuration{
		Releases:     storage.Init(driver.NewMemory()),
		KubeClient:   &kubefake.PrintingKubeClient{Out: io.Discard},
		Capabilities: caps,
		Log:          func(string, ...any) {},
	}
	w.releases = &countingReleases{Releaser: release.NewReleaser(release.ReleaserParams{
		Config: cfg, Kube: kube, Name: "trenova", Namespace: "trenova", Logger: logger,
	})}
	runner, err := commands.New(&commands.Params{
		ControlPlane: client, Kube: kube, Grants: noGrants{}, Logs: w.logs,
		Namespace: "trenova", Release: "trenova", Chart: "trenova",
		Logger: logger, Now: func() time.Time { return w.now },
	})
	require.NoError(t, err)
	params := &reconcile.Params{
		ControlPlane: client,
		Credentials:  creds,
		Charts:       release.NewSource(release.SourceParams{Path: chartDir}),
		Releases:     w.releases,
		Health:       w.health,
		Escrow: escrowsecret.New(&escrowsecret.Params{
			Client:       kube,
			ControlPlane: client,
			Namespace:    "trenova",
			Name:         values.EscrowSecret,
			Logger:       logger,
		}),
		Usage:          relayer,
		Administrators: w.admins,
		Features:       w.features,
		SystemEmail:    w.email,
		SupportKeys:    w.keys,
		SupportGrants:  w.grants,
		Storage:        noStorage{},
		Backups:        &fakeBackups{},
		Databases:      noDatabases{},
		Commands:       runner,
		EmailEvents:    w.events,
		Logs:           w.logs,
		Telemetry:      w.telem,
		Imports:        mustImporter(t, kube, logger),
		Operator:       w.operator,
		Base:           map[string]any{"clamav": map[string]any{"enabled": true}},
		Namespace:      "trenova-system",
		Logger:         logger,
		Now:            func() time.Time { return w.now },
		Poll:           15 * time.Second,
		Report:         time.Minute,
		UsageEvery:     15 * time.Minute,
	}
	for _, option := range options {
		option(params)
	}
	w.loop, err = reconcile.New(params)
	require.NoError(t, err)
	return w
}

func mustImporter(t *testing.T, kube *fake.Clientset, logger *slog.Logger) *importer.Importer {
	t.Helper()
	imp, err := importer.New(&importer.Params{
		Kube: kube, Namespace: "trenova", Release: "trenova", Logger: logger,
	})
	require.NoError(t, err)
	return imp
}

type noStorage struct{}

func (noStorage) Ensure(context.Context, []string, int64) (values.Storage, error) {
	return values.Storage{}, nil
}

type fakeOperator struct {
	mu     sync.Mutex
	status selfupgrade.Status
	wants  []string
}

func (f *fakeOperator) Converge(_ context.Context, want string) selfupgrade.Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.wants = append(f.wants, want)
	st := f.status
	st.Target = want
	return st
}

func (f *fakeOperator) set(phase selfupgrade.Phase, message string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status.Phase, f.status.Message = phase, message
}

func (w *world) lastReport(t *testing.T) instanceapi.StatusReport {
	t.Helper()
	reports := w.cp.Reports()
	require.NotEmpty(t, reports)
	return reports[len(reports)-1]
}

func TestConvergesThenReportsReady(t *testing.T) {
	t.Parallel()
	w := newWorld(t)

	w.loop.Tick(t.Context())
	require.Equal(t, 1, w.releases.applies)
	r := w.lastReport(t)
	require.False(t, r.Ready)
	require.Equal(t, int64(1), *r.AppliedRevision)
	require.Equal(t, "0.0.0-dev", *r.Version)
	require.Equal(t, "release: running", *r.Message)

	w.health.set(true)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 1, w.releases.applies, "nothing changed, so nothing is applied again")
	r = w.lastReport(t)
	require.True(t, r.Ready)
	require.Nil(t, r.Message)
	require.Len(t, w.cp.Reports(), 2)

	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Len(t, w.cp.Reports(), 2, "an unchanged report waits for the interval")

	w.now = w.now.Add(time.Minute)
	w.loop.Tick(t.Context())
	require.Len(t, w.cp.Reports(), 3)
	require.NoError(t, w.loop.Healthy())
}

func TestSuspendingStopsTheWorkloads(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	w.loop.Tick(t.Context())
	require.Equal(t, "trenova-release-1", w.health.last().ReleaseJob,
		"the release Job is the one the deployed manifest names")

	spec := controlplanetest.Spec(w.cp.InstanceID)
	spec.Lifecycle = instanceapi.Suspended
	w.cp.SetSpec(spec)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())

	require.Equal(t, 2, w.releases.applies)
	p := w.health.last()
	require.True(t, p.Stopped)
	require.True(t, p.Hibernated)
	require.Empty(t, p.ReleaseJob, "a stopped release renders no release Job")
	r := w.lastReport(t)
	require.Equal(t, int64(2), *r.AppliedRevision)
	require.True(t, r.Ready)

	spec.Lifecycle = instanceapi.Decommissioned
	w.cp.SetSpec(spec)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	p = w.health.last()
	require.True(t, p.Stopped)
	require.False(t, p.Hibernated, "PostgreSQL stays up for the final backup")
	r = w.lastReport(t)
	require.Equal(t, int64(3), *r.AppliedRevision)
	require.Contains(t, *r.Message, "final backup")
}

func TestAnUnknownSchemaIsReportedNotApplied(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	w.loop.Tick(t.Context())

	spec := controlplanetest.Spec(w.cp.InstanceID)
	spec.SchemaVersion = 2
	w.cp.SetSpec(spec)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())

	require.Equal(t, 1, w.releases.applies)
	r := w.lastReport(t)
	require.True(t, r.Ready, "the instance's real state: revision 1 runs and is ready")
	require.Equal(t, int64(1), *r.AppliedRevision, "it still runs revision 1")
	require.Contains(t, *r.Message, "schema version")
	require.NotNil(t, r.Refused)
	require.Equal(t, int64(2), r.Refused.Revision)
	require.Equal(t, instanceapi.RefusedRevisionCodeInvalid, r.Refused.Code)
	require.Contains(t, r.Refused.Reason, "schema version")
}

func TestAFailedApplyIsReportedAndRetried(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.releases.fail = errors.New("the API server is unreachable")
	w.loop.Tick(t.Context())
	r := w.lastReport(t)
	require.False(t, r.Ready)
	require.Nil(t, r.AppliedRevision)
	require.Contains(t, *r.Message, "unreachable")

	w.releases.fail = nil
	w.health.set(true)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 2, w.releases.applies)
	require.True(t, w.lastReport(t).Ready)
}

func TestWaitsOutARateLimit(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.cp.RateLimitNext(1)
	w.loop.Tick(t.Context())
	require.Empty(t, w.cp.Reports())
	require.Zero(t, w.releases.applies)

	w.loop.Tick(t.Context())
	require.Empty(t, w.cp.Reports(), "it does not call again before Retry-After")

	w.now = w.now.Add(2 * time.Second)
	w.loop.Tick(t.Context())
	require.Len(t, w.cp.Reports(), 1)
}

func TestWaitsForAnUnreadySpec(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.cp.UnavailableNext(1)
	w.loop.Tick(t.Context())
	require.Empty(t, w.cp.Reports())
	require.Zero(t, w.releases.applies)

	w.now = w.now.Add(2 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 1, w.releases.applies)
	require.Len(t, w.cp.Reports(), 1)
}

func TestRecoversFromAReenrolment(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	w.loop.Tick(t.Context())
	require.Len(t, w.cp.Reports(), 1)

	w.cp.RevokeAll()
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Len(t, w.cp.Reports(), 1)
	require.Error(t, w.loop.Healthy())
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

func TestRefusesASpecForAnotherInstance(t *testing.T) {
	w := newWorld(t)
	spec := w.cp.Spec()
	spec.InstanceId = uuid.New()
	w.cp.SetSpec(spec)

	w.loop.Tick(t.Context())

	assert.Zero(t, w.releases.applies)
	reports := w.cp.Reports()
	require.NotEmpty(t, reports)
	last := reports[len(reports)-1]
	assert.False(t, last.Ready)
	require.NotNil(t, last.Message)
	assert.Contains(t, *last.Message, "is for instance")
	require.NotNil(t, last.Refused)
	assert.Equal(t, instanceapi.RefusedRevisionCodeInvalid, last.Refused.Code)
	assert.Equal(t, w.cp.Revision(), last.Refused.Revision)
	assert.Nil(t, last.AppliedRevision, "nothing was ever applied")
}

func TestRelaysUsageOnlyWhenActiveAndReady(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.loop.Tick(t.Context())
	require.Zero(t, w.usage.count(), "not while the release is not ready")

	w.health.set(true)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 1, w.usage.count(), "as soon as it is ready")

	w.now = w.now.Add(14 * time.Minute)
	w.loop.Tick(t.Context())
	require.Equal(t, 1, w.usage.count(), "at most every interval")

	w.now = w.now.Add(time.Minute)
	w.loop.Tick(t.Context())
	require.Equal(t, 2, w.usage.count())

	w.health.set(false)
	w.now = w.now.Add(15 * time.Minute)
	w.loop.Tick(t.Context())
	require.Equal(t, 2, w.usage.count(), "not while a component is not ready")

	w.health.set(true)
	spec := controlplanetest.Spec(w.cp.InstanceID)
	spec.Lifecycle = instanceapi.Suspended
	w.cp.SetSpec(spec)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.True(t, w.lastReport(t).Ready)
	require.Equal(t, 2, w.usage.count(), "not while suspended")

	spec.Lifecycle = instanceapi.Active
	w.cp.SetSpec(spec)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 3, w.usage.count(), "at once on resuming")
}

func TestRetriesUsageAfterAFailure(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		wait time.Duration
	}{
		{name: "failure", err: errors.New("the TMS is unreachable"), wait: time.Minute},
		{
			name: "rate limited",
			err: &controlplane.RetryLaterError{
				Reason:     "rate limited",
				RetryAfter: 5 * time.Minute,
			},
			wait: 5 * time.Minute,
		},
		{
			name: "a short Retry-After waits the retry minute",
			err:  &controlplane.RetryLaterError{Reason: "rate limited", RetryAfter: time.Second},
			wait: time.Minute,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := newWorld(t)
			w.health.set(true)
			w.usage.fail(tc.err)
			w.loop.Tick(t.Context())
			require.Equal(t, 1, w.usage.count())

			w.now = w.now.Add(tc.wait - time.Second)
			w.loop.Tick(t.Context())
			require.Equal(t, 1, w.usage.count())

			w.usage.fail(nil)
			w.now = w.now.Add(time.Second)
			w.loop.Tick(t.Context())
			require.Equal(t, 2, w.usage.count())

			w.now = w.now.Add(time.Minute)
			w.loop.Tick(t.Context())
			require.Equal(t, 2, w.usage.count(), "back to the interval after a success")
		})
	}
}

func TestAUsageFailureDoesNotStopConverging(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	w.usage.fail(errors.New("boom"))
	w.loop.Tick(t.Context())
	require.True(t, w.lastReport(t).Ready)

	spec := controlplanetest.Spec(w.cp.InstanceID)
	spec.Plan.Vcpu = 16
	spec.Plan.MemoryGiB = 64
	w.cp.SetSpec(spec)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 2, w.releases.applies)
	require.Equal(t, int64(2), *w.lastReport(t).AppliedRevision)
	require.NoError(t, w.loop.Healthy())
}

func TestRelaysUsageBetweenTheTMSAndTheControlPlane(t *testing.T) {
	t.Parallel()
	app := "shipments.total"
	spec := controlplanetest.Spec(uuid.New())
	spec.Allowances = []instanceapi.SpecAllowance{
		{
			Meter:    "shipments",
			AppMeter: &app,
			Unit:     instanceapi.SpecAllowanceUnitCount,
			Quantity: 10,
			HardCap:  true,
		},
	}
	srv := tmstest.New(t, "")
	var kube *fake.Clientset
	w := newWorldWith(t, spec, func(p relayParams) reconcile.Usage {
		kube = p.kube
		client, err := tms.New(tms.Params{
			URL: srv.URL,
			Tokens: tmssecret.NewSecretTokens(tmssecret.SecretTokensParams{
				Client: p.kube, Namespace: "trenova", Name: "trenova-secrets",
			}),
		})
		require.NoError(t, err)
		return usage.New(usage.Params{ControlPlane: p.client, TMS: client, Now: p.now})
	})
	current := w.now.UTC().Truncate(time.Hour)
	srv.SetReadings([]tms.Reading{
		{Meter: app, Hour: current, Quantity: 12},
		{Meter: "organizations.total", Hour: current, Quantity: 2},
	})
	w.health.set(true)

	w.loop.Tick(t.Context())
	require.True(t, w.lastReport(t).Ready)
	require.Empty(t, w.cp.UsageReports(), "the chart has not written the token yet")

	srv.SetToken("managed-token")
	_, err := kube.CoreV1().Secrets("trenova").Create(t.Context(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "trenova-secrets", Namespace: "trenova"},
		Data:       map[string][]byte{tmssecret.TokenKey: []byte("managed-token")},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	w.now = w.now.Add(time.Minute)
	w.loop.Tick(t.Context())

	reports := w.cp.UsageReports()
	require.Len(t, reports, 1)
	require.Len(t, reports[0].Readings, 1, "organizations.total has no allowance here")
	since := srv.Since()
	require.Equal(t, w.now.UTC().Truncate(time.Hour).Add(-2*time.Hour), since[len(since)-1])
	states := srv.States()
	require.Len(t, states, 1)
	require.Equal(t, []tms.Meter{{
		Meter: app, Unit: "count", Quantity: 12, Allowance: 10, HardCap: true, Stopped: true,
	}}, states[0].Meters)
}

func (w *world) escrowSecret(t *testing.T) (*corev1.Secret, error) {
	t.Helper()
	return w.kube.CoreV1().Secrets("trenova").
		Get(t.Context(), values.EscrowSecret, metav1.GetOptions{})
}

func TestFetchesTheEscrowedSecretsOnceBeforeInstalling(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	w.loop.Tick(t.Context())

	require.Equal(t, 1, w.cp.SecretCalls())
	require.Equal(t, 1, w.releases.applies)
	sec, err := w.escrowSecret(t)
	require.NoError(t, err)
	want := controlplanetest.Secrets()
	require.Equal(t, want.EncryptionKey, string(sec.Data[escrow.KeyEncryptionKey]))
	require.Equal(t, want.RlsScopeKeyId, string(sec.Data[escrow.KeyRLSScopeKeyID]))
	require.Equal(t, want.RlsScopeKey, string(sec.Data[escrow.KeyRLSScopeKey]))
	require.Equal(t, want.MfaRecoveryCodePepper, string(sec.Data[escrow.KeyMFARecoveryCodePepper]))
	require.True(t, w.lastReport(t).Ready)

	for range 3 {
		w.now = w.now.Add(15 * time.Second)
		w.loop.Tick(t.Context())
	}
	require.Equal(t, 1, w.cp.SecretCalls(), "a complete Secret is not fetched again")

	require.NoError(t, w.kube.CoreV1().Secrets("trenova").
		Delete(t.Context(), values.EscrowSecret, metav1.DeleteOptions{}))
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 2, w.cp.SecretCalls(), "a deleted Secret is restored")
	_, err = w.escrowSecret(t)
	require.NoError(t, err)
}

func TestAPresentEscrowSecretIsKept(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	present := map[string][]byte{
		escrow.KeyEncryptionKey:         []byte("present-encryption-key-0123456789abcdef"),
		escrow.KeyRLSScopeKeyID:         []byte("kpresent"),
		escrow.KeyRLSScopeKey:           []byte("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="),
		escrow.KeyMFARecoveryCodePepper: []byte("present-pepper-0123456789abcdefghijkl"),
	}
	_, err := w.kube.CoreV1().Secrets("trenova").Create(t.Context(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: values.EscrowSecret, Namespace: "trenova"},
		Data:       present,
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	w.loop.Tick(t.Context())
	require.Zero(t, w.cp.SecretCalls())
	require.Equal(t, 1, w.releases.applies)
	sec, err := w.escrowSecret(t)
	require.NoError(t, err)
	require.Equal(t, present, sec.Data)
}

func TestInstallWaitsForTheEscrowedSecrets(t *testing.T) {
	t.Parallel()
	invalid := controlplanetest.Secrets()
	invalid.RlsScopeKey = "not base64"
	cases := []struct {
		name    string
		prepare func(w *world)
		message string
		report  bool
	}{
		{
			name:    "the control plane fails",
			prepare: func(w *world) { w.cp.FailSecretsNext(1, http.StatusInternalServerError) },
			message: "nothing is applied without the escrowed secrets",
			report:  true,
		},
		{
			name:    "the control plane's values are invalid",
			prepare: func(w *world) { w.cp.SetSecrets(invalid) },
			message: "rlsScopeKey must be standard base64",
			report:  true,
		},
		{
			name:    "rate limited",
			prepare: func(w *world) { w.cp.FailSecretsNext(1, http.StatusTooManyRequests) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := newWorld(t)
			w.health.set(true)
			tc.prepare(w)
			w.loop.Tick(t.Context())

			require.Zero(t, w.releases.applies, "nothing is installed without the escrowed secrets")
			_, err := w.escrowSecret(t)
			require.Error(t, err)
			if tc.report {
				r := w.lastReport(t)
				require.False(t, r.Ready)
				require.Nil(t, r.AppliedRevision)
				require.Contains(t, *r.Message, tc.message)
				require.NotContains(t, *r.Message, invalid.EncryptionKey)
				w.cp.SetSecrets(controlplanetest.Secrets())
			} else {
				require.Empty(t, w.cp.Reports(), "it waits out Retry-After without reporting")
				w.loop.Tick(t.Context())
				require.Equal(t, 1, w.cp.SecretCalls(), "nor calling again before it")
			}

			w.now = w.now.Add(2 * time.Second)
			w.loop.Tick(t.Context())
			require.Equal(t, 1, w.releases.applies)
			_, err = w.escrowSecret(t)
			require.NoError(t, err)
			require.True(t, w.lastReport(t).Ready)
		})
	}
}

func TestARefusedCertificateWhileFetchingSecretsGoesToRecovery(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.cp.FailSecretsNext(1, http.StatusUnauthorized)
	w.loop.Tick(t.Context())

	require.Zero(t, w.releases.applies)
	require.Empty(t, w.cp.Reports())
	require.Error(t, w.loop.Healthy(), "the same bundle cannot enrol again")
}

const inviteHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func specWithInvite(id uuid.UUID, expires time.Time) instanceapi.InstanceSpec {
	spec := controlplanetest.Spec(id)
	spec.FirstAdministrator.Invite = &instanceapi.SpecAdministratorInvite{
		TokenSha256: inviteHash,
		ExpiresAt:   expires,
	}
	return spec
}

func TestInstallsTheFirstAdministratorsInviteAndReportsIt(t *testing.T) {
	t.Parallel()
	expires := time.Now().Add(7 * 24 * time.Hour).Truncate(time.Second)
	w := newWorldWith(t, specWithInvite(uuid.New(), expires), nil)

	w.loop.Tick(t.Context())
	require.Empty(t, w.admins.calls(), "not before the TMS is ready")
	require.Nil(t, w.lastReport(t).Administrator)

	w.health.set(true)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	calls := w.admins.calls()
	require.Len(t, calls, 1)
	require.Equal(t, inviteHash, calls[0].TokenSHA256)
	require.True(t, calls[0].ExpiresAt.Equal(expires))
	r := w.lastReport(t)
	require.True(t, r.Ready)
	require.NotNil(t, r.Administrator)
	require.False(t, r.Administrator.SetUp)
	require.Equal(t, inviteHash, *r.Administrator.InviteTokenSha256)
	reports := len(w.cp.Reports())

	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Len(t, w.admins.calls(), 1, "asked again at most every minute")
	require.Len(t, w.cp.Reports(), reports, "an unchanged answer waits for the report interval")

	w.admins.set(tms.Administrator{SetUp: true}, nil)
	w.now = w.now.Add(time.Minute)
	w.loop.Tick(t.Context())
	require.Len(t, w.admins.calls(), 2)
	require.Len(t, w.cp.Reports(), reports+1, "a changed answer is reported at once")
	r = w.lastReport(t)
	require.True(t, r.Administrator.SetUp)
	require.Nil(t, r.Administrator.InviteTokenSha256)
}

func TestANewInviteIsInstalledAtOnce(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	expires := time.Now().Add(time.Hour).Truncate(time.Second)
	w := newWorldWith(t, specWithInvite(id, expires), nil)
	w.health.set(true)
	w.loop.Tick(t.Context())
	require.Len(t, w.admins.calls(), 1)

	next := specWithInvite(id, expires.Add(time.Hour))
	next.FirstAdministrator.Invite.TokenSha256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	w.cp.SetSpec(next)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	calls := w.admins.calls()
	require.Len(t, calls, 2, "a new invite does not wait for the recheck")
	require.Equal(t, next.FirstAdministrator.Invite.TokenSha256, calls[1].TokenSHA256)
	require.Equal(t, next.FirstAdministrator.Invite.TokenSha256,
		*w.lastReport(t).Administrator.InviteTokenSha256)
}

func TestAnInviteFailureIsOmittedAndRetried(t *testing.T) {
	t.Parallel()
	w := newWorldWith(t, specWithInvite(uuid.New(), time.Now().Add(time.Hour)), nil)
	w.health.set(true)
	w.admins.set(tms.Administrator{}, errors.New("the TMS answered 503"))

	w.loop.Tick(t.Context())
	r := w.lastReport(t)
	require.True(t, r.Ready, "a failed invite never holds the report back")
	require.Nil(t, r.Administrator)
	require.Equal(t, 1, w.releases.applies)

	w.admins.set(tms.Administrator{}, nil)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Len(t, w.admins.calls(), 2, "retried on the next tick")
	require.Equal(t, inviteHash, *w.lastReport(t).Administrator.InviteTokenSha256)
}

func TestNoInviteMeansNoCall(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	w.loop.Tick(t.Context())
	w.now = w.now.Add(2 * time.Minute)
	w.loop.Tick(t.Context())
	require.Empty(t, w.admins.calls())
	require.Nil(t, w.lastReport(t).Administrator)
}

func TestTheInviteWaitsWhileSuspended(t *testing.T) {
	t.Parallel()
	spec := specWithInvite(uuid.New(), time.Now().Add(time.Hour))
	spec.Lifecycle = instanceapi.Suspended
	w := newWorldWith(t, spec, nil)
	w.health.set(true)
	w.loop.Tick(t.Context())
	require.True(t, w.lastReport(t).Ready)
	require.Empty(t, w.admins.calls())
	require.Nil(t, w.lastReport(t).Administrator)
}

func specWithFeatures(id uuid.UUID, features ...string) instanceapi.InstanceSpec {
	spec := controlplanetest.Spec(id)
	spec.Features = append([]string{}, features...)
	return spec
}

func TestSendsTheSpecsFeaturesToTheTMSOnEveryTick(t *testing.T) {
	t.Parallel()
	w := newWorldWith(t, specWithFeatures(uuid.New(), "tms.accounting", "ai.desk"), nil)

	w.loop.Tick(t.Context())
	require.Empty(t, w.features.calls(), "not before the TMS is ready")

	w.health.set(true)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, [][]string{{"tms.accounting", "ai.desk"}}, w.features.calls())

	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(
		t,
		[][]string{{"tms.accounting", "ai.desk"}, {"tms.accounting", "ai.desk"}},
		w.features.calls(),
		"sent again so a restarted or restored TMS has it",
	)
}

func TestThePlanURLAndAddOnEndingsAreSentWithTheFeatures(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	spec := specWithFeatures(id, "tms.accounting")
	ends := time.Date(2026, 12, 14, 0, 0, 0, 0, time.UTC)
	spec.AddonEndings = &[]instanceapi.SpecAddonEnding{{Addon: "ACC", EndsAt: ends}}
	w := newWorldWith(t, spec, nil)
	w.health.set(true)
	w.loop.Tick(t.Context())

	sent, ok := w.features.lastSent()
	require.True(t, ok)
	require.Equal(t, spec.Account.PlanUrl, sent.PlanURL)
	require.Equal(t, []tms.AddonEnding{{Addon: "ACC", EndsAt: ends}}, sent.AddonEndings)

	next := specWithFeatures(id, "tms.accounting")
	next.Account = nil
	w.cp.SetSpec(next)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())

	sent, ok = w.features.lastSent()
	require.True(t, ok)
	require.Empty(t, sent.PlanURL, "a spec without an account sends no plan URL")
	require.NotNil(t, sent.AddonEndings)
	require.Empty(
		t,
		sent.AddonEndings,
		"a spec without endings sends an empty list, which clears the TMS's",
	)
}

func TestTheFinalBackupDaysReachTheTMSWithTheFeatures(t *testing.T) {
	t.Parallel()
	spec := specWithFeatures(uuid.New(), "tms.accounting")
	spec.Account.FinalBackupDays = 7
	w := newWorldWith(t, spec, nil)
	w.health.set(true)
	w.loop.Tick(t.Context())

	sent, ok := w.features.lastSent()
	require.True(t, ok)
	require.Equal(t, 7, sent.FinalBackupDays)
}

func TestThePlansEndReachesTheTMSAndAKeptPlanClearsIt(t *testing.T) {
	t.Parallel()
	w := newWorldWith(t, specWithFeatures(uuid.New(), "tms.accounting"), nil)
	w.health.set(true)
	w.loop.Tick(t.Context())

	sent, ok := w.features.lastSent()
	require.True(t, ok)
	require.Nil(t, sent.PlanEndsAt, "a plan nobody set to end is stated as not ending")

	ends := time.Date(2026, 12, 14, 0, 0, 0, 0, time.UTC)
	w.cp.SetPlanEnd(&ends)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())

	sent, ok = w.features.lastSent()
	require.True(t, ok)
	require.NotNil(
		t,
		sent.PlanEndsAt,
		"the end reaches the TMS on the next tick, with no new revision",
	)
	require.True(t, sent.PlanEndsAt.Equal(ends))

	w.cp.SetPlanEnd(nil)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())

	sent, ok = w.features.lastSent()
	require.True(t, ok)
	require.Nil(t, sent.PlanEndsAt, "a kept plan clears the end")
}

func TestAChangedFeatureListIsSentOnTheNextTick(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	w := newWorldWith(t, specWithFeatures(id, "tms.accounting"), nil)
	w.health.set(true)
	w.loop.Tick(t.Context())

	w.cp.SetSpec(specWithFeatures(id, "tms.accounting", "ai.desk"))
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())

	w.cp.SetSpec(specWithFeatures(id))
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())

	require.Equal(
		t,
		[][]string{{"tms.accounting"}, {"tms.accounting", "ai.desk"}, {}},
		w.features.calls(),
	)
}

func TestNoFeaturesAreSentWhileSuspendedOrDecommissioned(t *testing.T) {
	t.Parallel()
	for _, lifecycle := range []instanceapi.InstanceSpecLifecycle{
		instanceapi.Suspended,
		instanceapi.Decommissioned,
	} {
		t.Run(string(lifecycle), func(t *testing.T) {
			t.Parallel()
			spec := specWithFeatures(uuid.New(), "ai.desk")
			spec.Lifecycle = lifecycle
			w := newWorldWith(t, spec, nil)
			w.health.set(true)
			w.loop.Tick(t.Context())
			w.now = w.now.Add(15 * time.Second)
			w.loop.Tick(t.Context())
			require.True(t, w.lastReport(t).Ready)
			require.Empty(t, w.features.calls())
		})
	}
}

func TestAFeatureFailureIsLoggedAndRetried(t *testing.T) {
	t.Parallel()
	w := newWorldWith(t, specWithFeatures(uuid.New(), "ai.desk"), nil)
	w.health.set(true)
	w.features.fail(errors.New("the TMS answered 503"))

	w.loop.Tick(t.Context())
	r := w.lastReport(t)
	require.True(t, r.Ready, "a failed feature list never holds the report back")
	require.Nil(t, r.Message)
	require.Equal(t, 1, w.releases.applies)
	require.Equal(t, 1, w.usage.count(), "nor the usage relay")

	w.features.fail(nil)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(
		t,
		[][]string{{"ai.desk"}, {"ai.desk"}},
		w.features.calls(),
		"retried on the next tick",
	)
}

func TestReportsTheFeaturesTheTMSHolds(t *testing.T) {
	t.Parallel()
	w := newWorldWith(t, specWithFeatures(uuid.New(), "tms.accounting", "ai.desk"), nil)
	w.features.hold([]string{"tms.accounting"}, nil)

	w.loop.Tick(t.Context())
	require.Nil(t, w.lastReport(t).Features, "not read before the TMS is ready")

	w.health.set(true)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	r := w.lastReport(t)
	require.True(t, r.Ready)
	require.NotNil(t, r.Features)
	require.Equal(t, []string{"tms.accounting"}, *r.Features)

	sent := len(w.cp.Reports())
	w.features.hold([]string{"tms.accounting", "ai.desk"}, nil)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Len(t, w.cp.Reports(), sent+1, "a changed list is reported at once")
	require.Equal(t, []string{"tms.accounting", "ai.desk"}, *w.lastReport(t).Features)
}

func TestAFailedFeatureReadLeavesTheFeaturesOutOfTheReport(t *testing.T) {
	t.Parallel()
	w := newWorldWith(t, specWithFeatures(uuid.New(), "ai.desk"), nil)
	w.health.set(true)
	w.features.hold(nil, errors.New("the TMS answered 503"))

	w.loop.Tick(t.Context())

	r := w.lastReport(t)
	require.True(t, r.Ready, "a failed read never holds the report back")
	require.Nil(t, r.Message)
	require.Nil(t, r.Features)
	require.Equal(t, [][]string{{"ai.desk"}}, w.features.calls())
}

type fakeSystemEmail struct {
	mu      sync.Mutex
	desired []*instanceapi.SpecSystemEmail
	err     error
	held    string
	heldErr error
}

func (f *fakeSystemEmail) Sync(_ context.Context, desired *instanceapi.SpecSystemEmail) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.desired = append(f.desired, desired)
	if f.err != nil {
		return f.err
	}
	f.held = ""
	if desired != nil {
		f.held = desired.Fingerprint
	}
	return nil
}

func (f *fakeSystemEmail) Held(context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.held, f.heldErr
}

func (f *fakeSystemEmail) holds(fingerprint string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.held, f.heldErr = fingerprint, err
}

func (f *fakeSystemEmail) calls() []*instanceapi.SpecSystemEmail {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*instanceapi.SpecSystemEmail(nil), f.desired...)
}

func (f *fakeSystemEmail) fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func specWithSystemEmail(id uuid.UUID, fingerprint string) instanceapi.InstanceSpec {
	spec := controlplanetest.Spec(id)
	if fingerprint != "" {
		spec.SystemEmail = &instanceapi.SpecSystemEmail{Fingerprint: fingerprint}
	}
	return spec
}

func TestBringsTheTMSsSystemEmailSenderToTheSpecOnceReady(t *testing.T) {
	t.Parallel()
	fingerprint := strings.Repeat("a", 64)
	w := newWorldWith(t, specWithSystemEmail(uuid.New(), fingerprint), nil)

	w.loop.Tick(t.Context())
	require.Empty(t, w.email.calls(), "not before the TMS is ready")

	w.health.set(true)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	calls := w.email.calls()
	require.Len(t, calls, 1)
	require.NotNil(t, calls[0])
	require.Equal(t, fingerprint, calls[0].Fingerprint)
}

func TestTheReportNamesTheSystemEmailSenderTheTMSHolds(t *testing.T) {
	t.Parallel()
	fingerprint := strings.Repeat("a", 64)
	w := newWorldWith(t, specWithSystemEmail(uuid.New(), fingerprint), nil)
	w.health.set(true)

	w.loop.Tick(t.Context())
	r := w.lastReport(t)
	require.NotNil(t, r.SystemEmailFingerprint)
	require.Equal(t, fingerprint, *r.SystemEmailFingerprint)

	w.email.fail(errors.New("the TMS answered 503"))
	w.email.holds(strings.Repeat("b", 64), nil)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	r = w.lastReport(t)
	require.NotNil(t, r.SystemEmailFingerprint, "a change in what is held is reported at once")
	require.Equal(
		t,
		strings.Repeat("b", 64),
		*r.SystemEmailFingerprint,
		"what the TMS holds, not what the spec asks for",
	)
}

func TestTheReportLeavesOutASenderItCouldNotReadOrThatIsNotHeld(t *testing.T) {
	t.Parallel()
	w := newWorldWith(t, specWithSystemEmail(uuid.New(), ""), nil)
	w.health.set(true)

	w.loop.Tick(t.Context())
	require.Nil(t, w.lastReport(t).SystemEmailFingerprint, "no sender is held")

	w.email.holds("", errors.New("the TMS answered 503"))
	w.now = w.now.Add(2 * time.Minute)
	w.loop.Tick(t.Context())
	r := w.lastReport(t)
	require.True(t, r.Ready, "a failed read never holds the report back")
	require.Nil(t, r.SystemEmailFingerprint)

	w.email.holds("not-a-fingerprint", nil)
	w.now = w.now.Add(2 * time.Minute)
	w.loop.Tick(t.Context())
	require.Nil(
		t,
		w.lastReport(t).SystemEmailFingerprint,
		"a value the control plane would refuse is never sent",
	)
}

func TestAnInstanceWithoutATrenovaSenderAsksForItsRemoval(t *testing.T) {
	t.Parallel()
	w := newWorldWith(t, specWithSystemEmail(uuid.New(), ""), nil)
	w.health.set(true)

	w.loop.Tick(t.Context())

	calls := w.email.calls()
	require.Len(t, calls, 1)
	require.Nil(t, calls[0])
}

func TestNoSystemEmailIsSentWhileSuspended(t *testing.T) {
	t.Parallel()
	spec := specWithSystemEmail(uuid.New(), strings.Repeat("b", 64))
	spec.Lifecycle = instanceapi.Suspended
	w := newWorldWith(t, spec, nil)
	w.health.set(true)

	w.loop.Tick(t.Context())

	require.Empty(t, w.email.calls())
}

func TestASystemEmailFailureIsLoggedAndRetried(t *testing.T) {
	t.Parallel()
	w := newWorldWith(t, specWithSystemEmail(uuid.New(), strings.Repeat("c", 64)), nil)
	w.health.set(true)
	w.email.fail(errors.New("the TMS answered 503"))

	w.loop.Tick(t.Context())
	r := w.lastReport(t)
	require.True(t, r.Ready, "a failed system email sync never holds the report back")
	require.Equal(t, 1, w.usage.count(), "nor the usage relay")

	w.email.fail(nil)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Len(t, w.email.calls(), 2, "retried on the next tick")
}

func TestALoopWithoutItsCommandsIsRefused(t *testing.T) {
	t.Parallel()
	_, err := reconcile.New(&reconcile.Params{
		Logger: slog.New(slog.DiscardHandler), Namespace: "trenova-system",
		Poll: 15 * time.Second, Report: time.Minute,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Commands")
	require.Contains(t, err.Error(), "EmailEvents")
	require.Contains(t, err.Error(), "Operator")
	require.Contains(t, err.Error(), "Databases")
}

func apiDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name: "trenova-api", Namespace: "trenova",
		Labels: map[string]string{
			"app.kubernetes.io/name":      "trenova",
			"app.kubernetes.io/instance":  "trenova",
			"app.kubernetes.io/component": "api",
		},
	}}
}

func restartAPI(w *world, t *testing.T) instanceapi.Command {
	t.Helper()
	_, err := w.kube.AppsV1().Deployments("trenova").
		Create(t.Context(), apiDeployment(), metav1.CreateOptions{})
	require.NoError(t, err)
	component := "api"
	cmd := instanceapi.Command{
		Id: uuid.New(), Kind: instanceapi.Restart, Component: &component, IssuedAt: time.Now(),
	}
	w.cp.AddCommand(&cmd)
	return cmd
}

func TestCarriesOutTheControlPlanesCommandsOnEveryTick(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	cmd := restartAPI(w, t)
	w.health.set(true)

	w.loop.Tick(t.Context())

	require.Equal(t, 1, w.cp.CommandListings())
	answer, ok := w.cp.CommandAnswers()[cmd.Id]
	require.True(t, ok, "the command is answered on the tick that listed it")
	require.Equal(t, instanceapi.CommandResultStatusSucceeded, answer.Status)

	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 2, w.cp.CommandListings(), "an unchanged report does not stop the asking")
}

func TestCommandsAreCarriedOutWhenTheReportCannotBeSent(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	cmd := restartAPI(w, t)
	w.cp.FailStatusNext(1)
	w.releases.fail = errors.New("the API server is unreachable")

	w.loop.Tick(t.Context())

	require.Empty(t, w.cp.Reports())
	_, ok := w.cp.CommandAnswers()[cmd.Id]
	require.True(t, ok, "a broken instance is exactly when staff need its commands")
}

func TestCommandsWaitOutTheControlPlane(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.cp.UnavailableNext(1)
	w.loop.Tick(t.Context())
	require.Zero(t, w.cp.CommandListings())

	w.now = w.now.Add(2 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 1, w.cp.CommandListings())
}

func TestTheReportNamesTheOperatorsVersion(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	w.loop.Tick(t.Context())

	r := w.lastReport(t)
	require.NotNil(t, r.Operator)
	require.Equal(t, "0.11.0", r.Operator.Version)
	require.Nil(t, r.Operator.Upgrade)
	require.Equal(t, []string{"0.11.0"}, w.operator.wants)
}

func TestTrenovaWaitsForTheOperatorOfItsRelease(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.health.set(true)
	w.loop.Tick(t.Context())
	require.Equal(t, 1, w.releases.applies)

	spec := controlplanetest.Spec(w.cp.InstanceID)
	spec.Release.Version = "0.12.0"
	spec.Plan.Vcpu, spec.Plan.MemoryGiB = 16, 64
	w.cp.SetSpec(spec)
	w.operator.set(
		selfupgrade.PhaseUpgrading,
		"the operator is being upgraded from 0.11.0 to 0.12.0",
	)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())

	require.Equal(t, 1, w.releases.applies, "nothing is applied before the operator runs 0.12.0")
	r := w.lastReport(t)
	require.Equal(t, int64(1), *r.AppliedRevision, "revision 2 is not applied yet")
	require.True(t, r.Ready, "Trenova keeps running as it was")
	require.Contains(t, *r.Message, "until the operator runs 0.12.0")
	require.NotNil(t, r.Operator.Upgrade)
	require.Equal(t, "0.12.0", r.Operator.Upgrade.Version)
	require.Equal(t, instanceapi.OperatorUpgradePhase("upgrading"), r.Operator.Upgrade.Phase)

	w.operator.set(selfupgrade.PhaseFailed, "the registry may not have that version")
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	r = w.lastReport(t)
	require.Equal(t, instanceapi.OperatorUpgradePhase("failed"), r.Operator.Upgrade.Phase)
	require.Contains(t, *r.Operator.Message, "registry")
	require.Equal(t, 1, w.releases.applies)

	w.operator.mu.Lock()
	w.operator.status = selfupgrade.Status{Version: "0.12.0", Phase: selfupgrade.PhaseCurrent}
	w.operator.mu.Unlock()
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())
	require.Equal(t, 2, w.releases.applies, "the operator of 0.12.0 brings Trenova to it")
	r = w.lastReport(t)
	require.Equal(t, int64(2), *r.AppliedRevision)
	require.Nil(t, r.Operator.Upgrade)
}
