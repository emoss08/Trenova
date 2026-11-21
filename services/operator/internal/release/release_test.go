package release_test

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/kube"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/release"
)

var chartDir = filepath.Join("..", "..", "..", "..", "deploy", "helm", "trenova")

func setup(t *testing.T) (*action.Configuration, *release.Releaser, *chart.Chart) {
	t.Helper()
	return setupOn(t, fake.NewClientset(), &kubefake.PrintingKubeClient{Out: io.Discard})
}

func setupOn(
	t *testing.T,
	cluster kubernetes.Interface,
	helmKube kube.Interface,
) (*action.Configuration, *release.Releaser, *chart.Chart) {
	t.Helper()
	cfg := &action.Configuration{
		Releases:     storage.Init(driver.NewMemory()),
		KubeClient:   helmKube,
		Capabilities: capabilities(),
		Log:          func(string, ...any) {},
	}
	r := release.NewReleaser(release.ReleaserParams{
		Config:    cfg,
		Kube:      cluster,
		Name:      "trenova",
		Namespace: "trenova",
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	ch, err := release.NewSource(release.SourceParams{Path: chartDir, Provenance: release.Unverified{}}).Load("ignored for a local chart")
	require.NoError(t, err)
	return cfg, r, ch
}

func capabilities() *chartutil.Capabilities {
	c := chartutil.DefaultCapabilities.Copy()
	c.KubeVersion = chartutil.KubeVersion{Version: "v1.31.5", Major: "1", Minor: "31"}
	return c
}

func vals(domain string) map[string]any {
	return map[string]any{"domain": domain, "storageDomain": "files." + domain}
}

func TestInstallsThenOnlyUpgradesOnAChange(t *testing.T) {
	t.Parallel()
	_, r, ch := setup(t)

	d, err := r.Deployed()
	require.NoError(t, err)
	require.False(t, d.Exists)
	need, err := release.NeedsApply(&d, ch, vals("a.example.com"))
	require.NoError(t, err)
	require.True(t, need)

	require.NoError(t, r.Apply(t.Context(), ch, vals("a.example.com")))
	d, err = r.Deployed()
	require.NoError(t, err)
	require.Equal(t, helmrelease.StatusDeployed, d.Status)
	require.Equal(t, 1, d.Revision)

	need, err = release.NeedsApply(&d, ch, vals("a.example.com"))
	require.NoError(t, err)
	require.False(t, need, "the same values and chart need nothing")

	need, err = release.NeedsApply(&d, ch, vals("b.example.com"))
	require.NoError(t, err)
	require.True(t, need)
	require.NoError(t, r.Apply(t.Context(), ch, vals("b.example.com")))
	d, err = r.Deployed()
	require.NoError(t, err)
	require.Equal(t, 2, d.Revision)
	require.Equal(t, "b.example.com", d.Config["domain"])
}

func TestNumericValuesCompareByValue(t *testing.T) {
	t.Parallel()
	_, r, ch := setup(t)
	v := vals("a.example.com")
	v["tms"] = map[string]any{"api": map[string]any{"replicas": 2}}
	require.NoError(t, r.Apply(t.Context(), ch, v))
	d, err := r.Deployed()
	require.NoError(t, err)
	v["tms"] = map[string]any{"api": map[string]any{"replicas": float64(2)}}
	need, err := release.NeedsApply(&d, ch, v)
	require.NoError(t, err)
	require.False(t, need)
}

func TestRecoversAnInterruptedUpgrade(t *testing.T) {
	t.Parallel()
	cfg, r, ch := setup(t)
	require.NoError(t, r.Apply(t.Context(), ch, vals("a.example.com")))

	stuck, err := cfg.Releases.Last("trenova")
	require.NoError(t, err)
	next := *stuck
	info := *stuck.Info
	next.Info = &info
	next.Version = 2
	next.Info.Status = helmrelease.StatusPendingUpgrade
	require.NoError(t, cfg.Releases.Create(&next))

	require.NoError(t, r.Apply(t.Context(), ch, vals("b.example.com")))
	d, err := r.Deployed()
	require.NoError(t, err)
	require.Equal(t, helmrelease.StatusDeployed, d.Status)
	require.Equal(t, "b.example.com", d.Config["domain"])
}

func TestRemovesAnInstallThatNeverDeployed(t *testing.T) {
	t.Parallel()
	cfg, r, ch := setup(t)
	require.NoError(t, r.Apply(t.Context(), ch, vals("a.example.com")))
	rel, err := cfg.Releases.Last("trenova")
	require.NoError(t, err)
	rel.Info.Status = helmrelease.StatusPendingInstall
	require.NoError(t, cfg.Releases.Update(rel))

	require.NoError(t, r.Apply(t.Context(), ch, vals("c.example.com")))
	d, err := r.Deployed()
	require.NoError(t, err)
	require.Equal(t, helmrelease.StatusDeployed, d.Status)
	require.Equal(t, "c.example.com", d.Config["domain"])
}

type waitingKube struct {
	*kubefake.PrintingKubeClient
	waitedForDelete bool
}

func (k *waitingKube) WaitForDelete(resources kube.ResourceList, timeout time.Duration) error {
	k.waitedForDelete = true
	return k.PrintingKubeClient.WaitForDelete(resources, timeout)
}

func claim(name, instance string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: "trenova",
		Labels:    map[string]string{"app.kubernetes.io/instance": instance},
	}}
}

func claimNames(t *testing.T, cluster kubernetes.Interface) []string {
	t.Helper()
	list, err := cluster.CoreV1().PersistentVolumeClaims("trenova").List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	names := make([]string, 0, len(list.Items))
	for i := range list.Items {
		names = append(names, list.Items[i].Name)
	}
	return names
}

func TestAnInstallThatFailedPartwayLeavesNothingForTheNextOne(t *testing.T) {
	t.Parallel()
	cluster := fake.NewClientset(
		claim("data-trenova-meilisearch-0", "trenova"),
		claim("trenova-loki", "trenova"),
		claim("data-other-0", "other"),
	)
	helmKube := &waitingKube{PrintingKubeClient: &kubefake.PrintingKubeClient{Out: io.Discard}}
	cfg, r, ch := setupOn(t, cluster, helmKube)
	require.NoError(t, r.Apply(t.Context(), ch, vals("a.example.com")))
	rel, err := cfg.Releases.Last("trenova")
	require.NoError(t, err)
	rel.Info.Status = helmrelease.StatusFailed
	require.NoError(t, cfg.Releases.Update(rel))

	require.NoError(t, r.Apply(t.Context(), ch, vals("a.example.com")))

	require.True(t, helmKube.waitedForDelete, "the failed install's resources are gone before the next one starts")
	require.Equal(t, []string{"data-other-0"}, claimNames(t, cluster),
		"the volumes a release that never deployed left behind are removed, and no one else's")
	d, err := r.Deployed()
	require.NoError(t, err)
	require.Equal(t, helmrelease.StatusDeployed, d.Status)
}

func TestAReleaseThatDeployedKeepsItsVolumesWhenAChangeIsInterrupted(t *testing.T) {
	t.Parallel()
	cluster := fake.NewClientset(
		claim("data-trenova-meilisearch-0", "trenova"),
		claim("trenova-loki", "trenova"),
	)
	cfg, r, ch := setupOn(t, cluster, &kubefake.PrintingKubeClient{Out: io.Discard})
	require.NoError(t, r.Apply(t.Context(), ch, vals("a.example.com")))

	stuck, err := cfg.Releases.Last("trenova")
	require.NoError(t, err)
	next := *stuck
	info := *stuck.Info
	next.Info = &info
	next.Version = 2
	next.Info.Status = helmrelease.StatusPendingUpgrade
	require.NoError(t, cfg.Releases.Create(&next))

	require.NoError(t, r.Apply(t.Context(), ch, vals("b.example.com")))
	require.ElementsMatch(t, []string{"data-trenova-meilisearch-0", "trenova-loki"}, claimNames(t, cluster))
}

func TestACleanupThatWasInterruptedIsFinishedBeforeInstallingAgain(t *testing.T) {
	t.Parallel()
	for _, status := range []helmrelease.Status{
		helmrelease.StatusUninstalling,
		helmrelease.StatusUninstalled,
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()
			cluster := fake.NewClientset(claim("data-trenova-meilisearch-0", "trenova"))
			cfg, r, ch := setupOn(t, cluster, &kubefake.PrintingKubeClient{Out: io.Discard})
			require.NoError(t, r.Apply(t.Context(), ch, vals("a.example.com")))
			rel, err := cfg.Releases.Last("trenova")
			require.NoError(t, err)
			rel.Info.Status = status
			require.NoError(t, cfg.Releases.Update(rel))

			require.NoError(t, r.Apply(t.Context(), ch, vals("a.example.com")))

			require.Empty(t, claimNames(t, cluster))
			d, err := r.Deployed()
			require.NoError(t, err)
			require.Equal(t, helmrelease.StatusDeployed, d.Status)
			require.Equal(t, 1, d.Revision, "the release that never deployed is forgotten, not upgraded")
		})
	}
}

func TestAReleaseRemovedByHandAfterItDeployedIsLeftAlone(t *testing.T) {
	t.Parallel()
	cluster := fake.NewClientset(claim("data-trenova-meilisearch-0", "trenova"))
	cfg, r, ch := setupOn(t, cluster, &kubefake.PrintingKubeClient{Out: io.Discard})
	require.NoError(t, r.Apply(t.Context(), ch, vals("a.example.com")))
	require.NoError(t, r.Apply(t.Context(), ch, vals("b.example.com")))
	rel, err := cfg.Releases.Last("trenova")
	require.NoError(t, err)
	rel.Info.Status = helmrelease.StatusUninstalled
	require.NoError(t, cfg.Releases.Update(rel))

	require.Error(t, r.Apply(t.Context(), ch, vals("c.example.com")))

	require.Equal(t, []string{"data-trenova-meilisearch-0"}, claimNames(t, cluster),
		"volumes that held a deployed release's data are never removed")
	d, err := r.Deployed()
	require.NoError(t, err)
	require.Equal(t, helmrelease.StatusUninstalled, d.Status, "it is not rolled back into existence")
}

func TestTheReleaseJobIsTheOneTheManifestNames(t *testing.T) {
	t.Parallel()
	manifest := `---
# Source: trenova/templates/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: trenova-config
  labels:
    app.kubernetes.io/component: config
---
# Source: trenova/templates/release-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: trenova-release-4
  labels:
    app.kubernetes.io/component: release
`
	name, err := release.ReleaseJob(manifest)
	require.NoError(t, err)
	require.Equal(t, "trenova-release-4", name,
		"a later revision that changed nothing the Job runs keeps the Job of revision 4")

	name, err = release.ReleaseJob("---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n")
	require.NoError(t, err)
	require.Empty(t, name, "a stopped release renders no Job")
}

func TestANewChartVersionIsAppliedEvenWithTheSameValues(t *testing.T) {
	t.Parallel()
	ch := &chart.Chart{Metadata: &chart.Metadata{Name: "trenova", Version: "0.13.1"}}
	d := release.Deployed{
		Exists: true, Status: helmrelease.StatusDeployed, ChartVersion: "0.13.0",
		Config: vals("a.example.com"),
	}
	need, err := release.NeedsApply(&d, ch, vals("a.example.com"))
	require.NoError(t, err)
	require.True(t, need, "a release that carries every digest forward still moves the chart")
}
