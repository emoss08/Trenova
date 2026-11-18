package values_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

func TestPostgreSQLStopsAfterAShortWaitForClients(t *testing.T) {
	t.Parallel()
	s := spec(plans[0], instanceapi.Active)
	s.Restore = &instanceapi.SpecRestore{
		Id:         uuid.New(),
		Generation: "r1a2b3c4d",
		TargetTime: time.Date(2026, 11, 14, 9, 30, 0, 0, time.UTC),
	}
	r, err := values.Render(&s, base(), operatorNS, &testStorage)
	require.NoError(t, err)
	cluster := renderStorageChart(t, r)["trenova/templates/postgres.yaml"]
	require.Equal(t, 2, countLines(cluster, "  smartShutdownTimeout: 15"), "the live cluster and the restore beside it")
	require.Equal(t, 2, countLines(cluster, "  stopDelay: 300"))

	ch, err := loader.Load(chartDir)
	require.NoError(t, err)
	b := base()
	b["postgres"] = map[string]any{"smartShutdownTimeout": 300, "stopDelay": 300}
	r, err = values.Render(ptr(spec(plans[0], instanceapi.Active)), b, operatorNS, &values.Storage{})
	require.NoError(t, err)
	require.NoError(t, chartutil.ValidateAgainstSchema(ch, r.Values))
	vals, err := chartutil.ToRenderValues(ch, r.Values, chartutil.ReleaseOptions{
		Name: "trenova", Namespace: "trenova", Revision: 1, IsInstall: true,
	}, chartutil.DefaultCapabilities)
	require.NoError(t, err)
	_, err = engine.Render(ch, vals)
	require.ErrorContains(t, err, "postgres.stopDelay must exceed postgres.smartShutdownTimeout")

	b["postgres"] = map[string]any{"stopDelay": 10}
	r, err = values.Render(ptr(spec(plans[0], instanceapi.Active)), b, operatorNS, &values.Storage{})
	require.NoError(t, err)
	require.Error(t, chartutil.ValidateAgainstSchema(ch, r.Values), "a stop too short for the shutdown checkpoint")
}

func countLines(body, line string) int {
	n := 0
	for _, l := range strings.Split(body, "\n") {
		if l == line {
			n++
		}
	}
	return n
}
