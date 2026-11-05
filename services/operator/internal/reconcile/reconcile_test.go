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
	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/reconcile"
	"github.com/emoss08/trenova/operator/internal/release"
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
	return []health.Component{{Name: "api", Ready: true}, {Name: "release", Message: "running"}}, nil
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

type world struct {
	cp       *controlplanetest.Server
	loop     *reconcile.Loop
	health   *fakeHealth
	releases *countingReleases
	now      time.Time
}

func newWorld(t *testing.T) *world {
	t.Helper()
	cp := controlplanetest.New(t, controlplanetest.Spec(uuid.New()))
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
	client, err := controlplane.New(cp.URL, &http.Client{Transport: transport, Timeout: 10 * time.Second})
	require.NoError(t, err)
	w := &world{cp: cp, health: &fakeHealth{}, now: time.Now()}
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
		Base:         map[string]any{"clamav": map[string]any{"enabled": true}},
		Logger:       logger,
		Now:          func() time.Time { return w.now },
		Poll:         15 * time.Second,
		Report:       time.Minute,
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
