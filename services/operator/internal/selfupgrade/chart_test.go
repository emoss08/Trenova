package selfupgrade

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
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
		if d["kind"] == "ClusterRole" || d["kind"] == "ClusterRoleBinding" {
			require.Equal(t, releaseName+"-telemetry", nested(d, "metadata", "name"),
				"the operator is given nothing cluster-wide but reading the node for telemetry")
		}
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

func TestTheChartLetsTheOperatorReadOnlyWhatTelemetrySamples(t *testing.T) {
	t.Parallel()
	docs := renderOperatorChart(t, nil)

	cluster := one(t, docs, "ClusterRole", releaseName+"-telemetry")
	require.Equal(t, []string{
		"/nodes:get",
		"/nodes/stats:get",
		"metrics.k8s.io/nodes:get",
	}, grants(t, cluster))
	for _, d := range docs {
		if d["kind"] != "ClusterRole" && d["kind"] != "Role" {
			continue
		}
		rules, _ := d["rules"].([]any)
		for _, r := range rules {
			resources, _ := nested(r, "resources").([]any)
			require.NotContains(t, resources, "nodes/proxy",
				"the node proxy reaches the whole kubelet API, exec included")
		}
	}

	binding := one(t, docs, "ClusterRoleBinding", releaseName+"-telemetry")
	require.Equal(t, releaseName+"-telemetry", nested(binding, "roleRef", "name"))
	subjects, _ := binding["subjects"].([]any)
	require.Len(t, subjects, 1)
	subject, _ := subjects[0].(map[string]any)
	require.Equal(t, "ServiceAccount", subject["kind"])
	require.Equal(t, releaseName, subject["name"])
	require.Equal(t, releaseNamespace, subject["namespace"])

	release := one(t, docs, "Role", releaseName+"-release")
	require.Contains(t, grants(t, release), "metrics.k8s.io/pods:list")
	require.Equal(t, "trenova", nested(release, "metadata", "namespace"))

	deployment := one(t, docs, "Deployment", releaseName)
	containers, _ := nested(deployment, "spec", "template", "spec", "containers").([]any)
	require.Len(t, containers, 1)
	container, _ := containers[0].(map[string]any)
	vars, _ := container["env"].([]any)
	node := ""
	for _, v := range vars {
		e, _ := v.(map[string]any)
		if e["name"] == "TRENOVA_OPERATOR_NODE_NAME" {
			node, _ = nested(e, "valueFrom", "fieldRef", "fieldPath").(string)
		}
	}
	require.Equal(t, "spec.nodeName", node, "a suspended instance runs no pod to find its node by")
}

func grants(t *testing.T, role map[string]any) []string {
	t.Helper()
	rules, _ := role["rules"].([]any)
	var out []string
	for _, r := range rules {
		rule, _ := r.(map[string]any)
		require.Nil(t, rule["resourceNames"])
		groups, _ := rule["apiGroups"].([]any)
		resources, _ := rule["resources"].([]any)
		verbs, _ := rule["verbs"].([]any)
		for _, g := range groups {
			for _, res := range resources {
				for _, v := range verbs {
					out = append(out, g.(string)+"/"+res.(string)+":"+v.(string))
				}
			}
		}
	}
	return out
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

const operatorDigest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"

func releaseManifest(version, operatorVersion string) string {
	return `{"schemaVersion": 1, "version": "` + version + `", "migrations": false,
  "commit": "0123456789abcdef0123456789abcdef01234567",
  "images": {"operator": {"repository": "ghcr.io/emoss08/trenova/operator",
    "version": "` + operatorVersion + `", "digest": "` + operatorDigest + `", "changed": false}}}`
}

func TestAReleaseThatDidNotRebuildTheOperatorLeavesItsPodAlone(t *testing.T) {
	t.Parallel()
	before := one(t, renderPublished(t, "0.13.0", releaseManifest("0.13.0", "0.13.0")),
		"Deployment", releaseName)
	after := one(t, renderPublished(t, "0.13.1", releaseManifest("0.13.1", "0.13.0")),
		"Deployment", releaseName)

	require.Equal(t, nested(before, "spec"), nested(after, "spec"),
		"the pod template must not change with the release alone")
	require.Equal(t, "0.13.1", nested(after, "metadata", "annotations", AnnotationVersion))
	containers, _ := nested(after, "spec", "template", "spec", "containers").([]any)
	container, _ := containers[0].(map[string]any)
	require.Equal(t, "ghcr.io/emoss08/trenova/operator:0.13.0@"+operatorDigest, container["image"])

	rebuilt := one(t, renderPublished(t, "0.13.1", releaseManifest("0.13.1", "0.13.1")),
		"Deployment", releaseName)
	require.NotEqual(t, nested(before, "spec"), nested(rebuilt, "spec"))
}

func renderPublished(t *testing.T, version, manifest string) []map[string]any {
	t.Helper()
	ch, err := loader.Load(operatorChart)
	require.NoError(t, err)
	ch.Metadata.Version, ch.Metadata.AppVersion = version, version
	ch.Files = append(ch.Files, &chart.File{Name: "release-manifest.json", Data: []byte(manifest)})
	return renderLoaded(t, ch, nil)
}

func renderOperatorChart(t *testing.T, values map[string]any) []map[string]any {
	t.Helper()
	ch, err := loader.Load(operatorChart)
	require.NoError(t, err)
	return renderLoaded(t, ch, values)
}

func renderLoaded(t *testing.T, ch *chart.Chart, values map[string]any) []map[string]any {
	t.Helper()
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
