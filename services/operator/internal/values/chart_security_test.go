package values_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

func renderChart(t *testing.T, b map[string]any) (*chart.Chart, map[string]any, map[string]string) {
	t.Helper()
	ch, err := loader.Load(chartDir)
	require.NoError(t, err)
	r, err := values.Render(ptr(spec(plans[0], instanceapi.Active, "docling")), b)
	require.NoError(t, err)
	vals, err := chartutil.ToRenderValues(ch, r.Values, chartutil.ReleaseOptions{
		Name: "trenova", Namespace: "trenova", Revision: 1, IsInstall: true,
	}, chartutil.DefaultCapabilities)
	require.NoError(t, err)
	out, err := engine.Render(ch, vals)
	require.NoError(t, err)
	return ch, r.Values, out
}

func template(t *testing.T, out map[string]string, suffix string) string {
	t.Helper()
	for name, body := range out {
		if strings.HasSuffix(name, suffix) {
			return body
		}
	}
	require.Failf(t, "template not rendered", "%s", suffix)
	return ""
}

func TestChartHasNoModeThatSignsInAsTheOwner(t *testing.T) {
	t.Parallel()
	b := base()
	b["postgres"] = map[string]any{"rlsMode": "off"}
	ch, vals, _ := renderChart(t, b)
	require.ErrorContains(t, chartutil.ValidateAgainstSchema(ch, vals), "rlsMode")
}

func TestApplicationSignsInAsItsOwnRoleInEveryMode(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"enforce", "observe"} {
		b := base()
		b["postgres"] = map[string]any{"rlsMode": mode}
		ch, vals, out := renderChart(t, b)
		require.NoError(t, chartutil.ValidateAgainstSchema(ch, vals), mode)

		config := template(t, out, "templates/configmap.yaml")
		require.Contains(t, config, "user: trenova_app", mode)
		require.Contains(t, config, "user: trenova_system", mode)
		require.Contains(t, template(t, out, "templates/release-job.yaml"), "name: rls-roles", mode)
	}
}
