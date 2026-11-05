package release_test

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"

	"github.com/emoss08/trenova/operator/internal/release"
)

var chartDir = filepath.Join("..", "..", "..", "..", "deploy", "helm", "trenova")

func setup(t *testing.T) (*action.Configuration, *release.Releaser, *chart.Chart) {
	t.Helper()
	cfg := &action.Configuration{
		Releases:     storage.Init(driver.NewMemory()),
		KubeClient:   &kubefake.PrintingKubeClient{Out: io.Discard},
		Capabilities: capabilities(),
		Log:          func(string, ...any) {},
	}
	r := release.NewReleaser(release.ReleaserParams{
		Config:    cfg,
		Name:      "trenova",
		Namespace: "trenova",
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	ch, err := release.NewSource(release.SourceParams{Path: chartDir}).Load("ignored for a local chart")
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
	need, err := release.NeedsApply(d, ch, vals("a.example.com"))
	require.NoError(t, err)
	require.True(t, need)

	require.NoError(t, r.Apply(t.Context(), ch, vals("a.example.com")))
	d, err = r.Deployed()
	require.NoError(t, err)
	require.Equal(t, helmrelease.StatusDeployed, d.Status)
	require.Equal(t, 1, d.Revision)

	need, err = release.NeedsApply(d, ch, vals("a.example.com"))
	require.NoError(t, err)
	require.False(t, need, "the same values and chart need nothing")

	need, err = release.NeedsApply(d, ch, vals("b.example.com"))
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
	need, err := release.NeedsApply(d, ch, v)
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
