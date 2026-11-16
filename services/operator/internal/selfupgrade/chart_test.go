package selfupgrade

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"sigs.k8s.io/yaml"
)

const (
	releaseName      = "trenova-operator"
	releaseNamespace = "trenova-system"
	envHelmChart     = "TRENOVA_OPERATOR_HELMCHART"
)

var operatorChart = filepath.Join("..", "..", "..", "..", "deploy", "helm", "trenova-operator")

func TestTheChartLetsTheOperatorMoveOnlyItsOwnHelmChart(t *testing.T) {
	t.Parallel()
	docs := renderOperatorChart(t, nil)
	for _, d := range docs {
		require.NotContains(t, []any{"ClusterRole", "ClusterRoleBinding"}, d["kind"],
			"the operator is given nothing cluster-wide")
	}

	role := one(t, docs, "Role", releaseName+"-self")
	require.Equal(t, defaultChartsSpace, nested(role, "metadata", "namespace"))
	rules, _ := role["rules"].([]any)
	require.Len(t, rules, 2)
	granted := map[string][]any{}
	for _, r := range rules {
		rule, _ := r.(map[string]any)
		names, _ := rule["resourceNames"].([]any)
		require.Len(t, names, 1, "a rule without a name reaches every object of its kind")
		resources, _ := rule["resources"].([]any)
		require.Len(t, resources, 1)
		verbs, _ := rule["verbs"].([]any)
		granted[resources[0].(string)+"/"+names[0].(string)] = verbs
	}
	require.Equal(t, map[string][]any{
		"helmcharts/" + releaseName:        {"get", "patch"},
		"jobs/helm-install-" + releaseName: {"get"},
	}, granted)

	binding := one(t, docs, "RoleBinding", releaseName+"-self")
	subjects, _ := binding["subjects"].([]any)
	require.Len(t, subjects, 1)
	subject, _ := subjects[0].(map[string]any)
	require.Equal(t, "ServiceAccount", subject["kind"])
	require.Equal(t, releaseNamespace, subject["namespace"])

	require.Equal(t, releaseName, operatorEnv(t, docs)[envHelmChart],
		"helm-controller names the release after its HelmChart")
}

func TestTheChartRollsTheOperatorOnePodAtATime(t *testing.T) {
	t.Parallel()
	deployment := one(t, renderOperatorChart(t, nil), "Deployment", releaseName)
	require.EqualValues(t, 1, nested(deployment, "spec", "replicas"))
	require.Equal(t, "RollingUpdate", nested(deployment, "spec", "strategy", "type"))
	require.EqualValues(t, 1, nested(deployment, "spec", "strategy", "rollingUpdate", "maxSurge"))
	require.EqualValues(t, 0, nested(deployment, "spec", "strategy", "rollingUpdate", "maxUnavailable"))
	require.NotNil(t, nested(deployment, "spec", "progressDeadlineSeconds"),
		"without a deadline a new operator that never becomes ready is never rolled back")
}

func TestWithSelfUpgradeOffTheOperatorReachesNothingInHelmControllersNamespace(t *testing.T) {
	t.Parallel()
	docs := renderOperatorChart(t, map[string]any{"selfUpgrade": map[string]any{"enabled": false}})
	for _, d := range docs {
		require.NotEqual(t, defaultChartsSpace, nested(d, "metadata", "namespace"),
			"%v %v", d["kind"], nested(d, "metadata", "name"))
	}
	require.NotContains(t, operatorEnv(t, docs), envHelmChart)
}

func renderOperatorChart(t *testing.T, values map[string]any) []map[string]any {
	t.Helper()
	ch, err := loader.Load(operatorChart)
	require.NoError(t, err)
	vals, err := chartutil.ToRenderValues(ch, values, chartutil.ReleaseOptions{
		Name: releaseName, Namespace: releaseNamespace, Revision: 1, IsInstall: true,
	}, chartutil.DefaultCapabilities)
	require.NoError(t, err)
	out, err := engine.Render(ch, vals)
	require.NoError(t, err)

	var docs []map[string]any
	for name, body := range out {
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		for _, doc := range strings.Split(body, "\n---") {
			var m map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(doc), &m), name)
			if m["kind"] != nil {
				docs = append(docs, m)
			}
		}
	}
	require.NotEmpty(t, docs)
	return docs
}

func one(t *testing.T, docs []map[string]any, kind, name string) map[string]any {
	t.Helper()
	var found []map[string]any
	for _, d := range docs {
		if d["kind"] == kind && nested(d, "metadata", "name") == name {
			found = append(found, d)
		}
	}
	require.Len(t, found, 1, "%s %s", kind, name)
	return found[0]
}

func operatorEnv(t *testing.T, docs []map[string]any) map[string]any {
	t.Helper()
	containers, _ := nested(one(t, docs, "Deployment", releaseName), "spec", "template", "spec", "containers").([]any)
	require.Len(t, containers, 1)
	container, _ := containers[0].(map[string]any)
	env := map[string]any{}
	vars, _ := container["env"].([]any)
	for _, v := range vars {
		e, _ := v.(map[string]any)
		env[e["name"].(string)] = e["value"]
	}
	return env
}

func nested(v any, path ...string) any {
	for _, k := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}
