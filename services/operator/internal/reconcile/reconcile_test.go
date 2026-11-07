package reconcile_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/escrow"
	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/reconcile"
	"github.com/emoss08/trenova/operator/internal/release"
	"github.com/emoss08/trenova/operator/internal/tms"
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
}

func (c *countingReleases) Apply(ctx context.Context, ch *chart.Chart, vals map[string]any) error {
	c.applies++
	if c.fail != nil {
		return c.fail
	}
	return c.Releaser.Apply(ctx, ch, vals)
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

type world struct {
	cp       *controlplanetest.Server
	kube     *fake.Clientset
	loop     *reconcile.Loop
	health   *fakeHealth
	releases *countingReleases
	usage    *fakeUsage
	admins   *fakeAdmins
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
	w := &world{cp: cp, kube: kube, health: &fakeHealth{}, admins: &fakeAdmins{}, now: time.Now()}
	w.usage = &fakeUsage{now: func() time.Time { return w.now }}
	var relayer reconcile.Usage = w.usage
	if relay != nil {
		relayer = relay(
			relayParams{kube: kube, client: client, now: func() time.Time { return w.now }},
		)
	}
	creds := credential.NewManager(&credential.ManagerParams{
		Store: credential.NewStore(credential.StoreParams{
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
		Config: cfg, Name: "trenova", Namespace: "trenova", Logger: logger,
	})}
	w.loop = reconcile.New(&reconcile.Params{
		ControlPlane: client,
		Credentials:  creds,
		Charts:       release.NewSource(release.SourceParams{Path: chartDir}),
		Releases:     w.releases,
		Health:       w.health,
		Escrow: escrow.New(&escrow.Params{
			Client:       kube,
			ControlPlane: client,
			Namespace:    "trenova",
			Name:         values.EscrowSecret,
			Logger:       logger,
		}),
		Usage:          relayer,
		Administrators: w.admins,
		Base:           map[string]any{"clamav": map[string]any{"enabled": true}},
		Namespace:      "trenova-system",
		Logger:         logger,
		Now:            func() time.Time { return w.now },
		Poll:           15 * time.Second,
		Report:         time.Minute,
		UsageEvery:     15 * time.Minute,
	})
	return w
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

	spec := controlplanetest.Spec(w.cp.InstanceID)
	spec.Lifecycle = instanceapi.Suspended
	w.cp.SetSpec(spec)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())

	require.Equal(t, 2, w.releases.applies)
	p := w.health.last()
	require.True(t, p.Stopped)
	require.True(t, p.Hibernated)
	require.Equal(t, 2, p.HelmRevision)
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
	require.False(t, r.Ready)
	require.Equal(t, int64(1), *r.AppliedRevision, "it still runs revision 1")
	require.Contains(t, *r.Message, "schema version")
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
			Tokens: tms.NewSecretTokens(tms.SecretTokensParams{
				Client: p.kube, Namespace: "trenova", Name: "trenova-secrets",
			}),
		})
		require.NoError(t, err)
		return usage.New(usage.Params{ControlPlane: p.client, TMS: client, Now: p.now})
	})
	current := w.now.UTC().Truncate(time.Hour)
	srv.SetReadings([]tms.Reading{
		{Meter: app, Hour: current, Quantity: 12},
		{Meter: "ai.spend_cents", Hour: current, Quantity: 250},
	})
	w.health.set(true)

	w.loop.Tick(t.Context())
	require.True(t, w.lastReport(t).Ready)
	require.Empty(t, w.cp.UsageReports(), "the chart has not written the token yet")

	srv.SetToken("managed-token")
	_, err := kube.CoreV1().Secrets("trenova").Create(t.Context(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "trenova-secrets", Namespace: "trenova"},
		Data:       map[string][]byte{tms.TokenKey: []byte("managed-token")},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	w.now = w.now.Add(time.Minute)
	w.loop.Tick(t.Context())

	reports := w.cp.UsageReports()
	require.Len(t, reports, 1)
	require.Len(t, reports[0].Readings, 1, "ai.spend_cents has no allowance here")
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
