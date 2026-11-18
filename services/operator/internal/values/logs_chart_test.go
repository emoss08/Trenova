package values_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"sigs.k8s.io/yaml"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

var operatorChartDir = filepath.Join("..", "..", "..", "..", "deploy", "helm", "trenova-operator")

func renderedDocs(t *testing.T, r values.Rendered) []map[string]any {
	t.Helper()
	var docs []map[string]any
	for name, body := range renderStorageChart(t, r) {
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

func managedLogs(t *testing.T, b map[string]any) values.Rendered {
	t.Helper()
	r, err := values.Render(ptr(spec(plans[0], instanceapi.Active, "docling")), b, operatorNS, values.Storage{})
	require.NoError(t, err)
	return r
}

func component(d map[string]any) string {
	labels, _ := nestedField(d, "metadata", "labels").(map[string]any)
	c, _ := labels["app.kubernetes.io/component"].(string)
	return c
}

func named(t *testing.T, docs []map[string]any, kind, name string) map[string]any {
	t.Helper()
	var found []map[string]any
	for _, d := range docs {
		if d["kind"] == kind && nestedField(d, "metadata", "name") == name {
			found = append(found, d)
		}
	}
	require.Len(t, found, 1, "%s %s", kind, name)
	return found[0]
}

func nestedField(v any, path ...string) any {
	for _, k := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

func podSpec(d map[string]any) map[string]any {
	s, _ := nestedField(d, "spec", "template", "spec").(map[string]any)
	return s
}

func TestAManagedInstanceKeepsItsLogs(t *testing.T) {
	t.Parallel()
	r := managedLogs(t, base())
	require.Equal(t, true, lookup(t, r.Values, []string{"logs"})["enabled"])

	b := base()
	b["logs"] = map[string]any{"enabled": false}
	require.Equal(t, true, lookup(t, managedLogs(t, b).Values, []string{"logs"})["enabled"],
		"the base values cannot turn log keeping off for a managed instance")
}

func TestLogKeepingReadsOnlyItsNamespaceAndKeepsAWeek(t *testing.T) {
	t.Parallel()
	docs := renderedDocs(t, managedLogs(t, base()))

	for _, d := range docs {
		require.NotContains(t, []any{"ClusterRole", "ClusterRoleBinding", "DaemonSet"}, d["kind"],
			"%v", nestedField(d, "metadata", "name"))
		volumes, _ := podSpec(d)["volumes"].([]any)
		for _, v := range volumes {
			vol, _ := v.(map[string]any)
			require.NotContains(t, vol, "hostPath", "%v %v", d["kind"], nestedField(d, "metadata", "name"))
		}
	}

	role := named(t, docs, "Role", "trenova-alloy")
	require.Equal(t, []any{
		map[string]any{"apiGroups": []any{""}, "resources": []any{"pods"}, "verbs": []any{"get", "list", "watch"}},
		map[string]any{"apiGroups": []any{""}, "resources": []any{"pods/log"}, "verbs": []any{"get"}},
	}, role["rules"])
	binding := named(t, docs, "RoleBinding", "trenova-alloy")
	require.Equal(t, "Role", nestedField(binding, "roleRef", "kind"))
	require.Equal(t, []any{map[string]any{
		"kind": "ServiceAccount", "name": "trenova-alloy", "namespace": "trenova",
	}}, binding["subjects"])

	alloy := named(t, docs, "Deployment", "trenova-alloy")
	require.Equal(t, "trenova-alloy", podSpec(alloy)["serviceAccountName"])
	require.Equal(t, true, podSpec(alloy)["automountServiceAccountToken"])
	require.EqualValues(t, 1, nestedField(alloy, "spec", "replicas"))
	require.Equal(t, "Recreate", nestedField(alloy, "spec", "strategy", "type"),
		"two copies would push every line twice")
	pipeline, _ := nestedField(named(t, docs, "ConfigMap", "trenova-alloy"), "data", "config.alloy").(string)
	require.Contains(t, pipeline, `names = ["trenova"]`)
	require.Contains(t, pipeline, `label = "app.kubernetes.io/name=trenova,app.kubernetes.io/instance=trenova"`)
	require.Contains(t, pipeline, `"__meta_kubernetes_pod_label_app_kubernetes_io_component"`)
	require.Contains(t, pipeline, `regex  = "component|pod|container"`)
	require.Contains(t, pipeline, `url = "http://trenova-loki:3100/loki/api/v1/push"`)

	loki := named(t, docs, "Deployment", values.LokiService("trenova"))
	require.EqualValues(t, 1, nestedField(loki, "spec", "replicas"))
	require.Equal(t, false, podSpec(loki)["automountServiceAccountToken"])
	require.Equal(t, "ClusterIP", nestedField(named(t, docs, "Service", "trenova-loki"), "spec", "type"))
	require.Equal(t, "5Gi", nestedField(
		named(t, docs, "PersistentVolumeClaim", "trenova-loki"), "spec", "resources", "requests", "storage",
	))
	config, _ := nestedField(named(t, docs, "ConfigMap", "trenova-loki"), "data", "config.yaml").(string)
	var lokiConfig map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(config), &lokiConfig))
	require.Equal(t, false, lokiConfig["auth_enabled"])
	require.Equal(t, "168h", nestedField(lokiConfig, "limits_config", "retention_period"))
	require.Equal(t, true, nestedField(lokiConfig, "compactor", "retention_enabled"))
	require.Equal(t, "filesystem", nestedField(lokiConfig, "compactor", "delete_request_store"))
	require.Equal(t, false, nestedField(lokiConfig, "analytics", "reporting_enabled"))
	require.Equal(t, "169h", nestedField(lokiConfig, "limits_config", "max_query_length"),
		"a query over the whole week is allowed")

	lokiPolicy := named(t, docs, "NetworkPolicy", "trenova-loki")
	ingress, _ := nestedField(lokiPolicy, "spec", "ingress").([]any)
	require.Len(t, ingress, 1)
	require.Equal(t, []any{
		map[string]any{"podSelector": map[string]any{"matchLabels": map[string]any{
			"app.kubernetes.io/name":      "trenova",
			"app.kubernetes.io/instance":  "trenova",
			"app.kubernetes.io/component": "alloy",
		}}},
		map[string]any{"namespaceSelector": map[string]any{"matchLabels": map[string]any{
			"kubernetes.io/metadata.name": operatorNS,
		}}},
	}, nestedField(ingress[0], "from"))
	require.Equal(t, []any{}, nestedField(lokiPolicy, "spec", "egress"), "Loki opens no connections")

	alloyPolicy := named(t, docs, "NetworkPolicy", "trenova-alloy")
	require.Equal(t, []any{}, nestedField(alloyPolicy, "spec", "ingress"))
	egress, _ := nestedField(alloyPolicy, "spec", "egress").([]any)
	require.Len(t, egress, 3, "Loki, the Kubernetes API and DNS")

	for _, d := range docs {
		switch d["kind"] {
		case "Deployment", "StatefulSet", "Job":
		default:
			continue
		}
		for _, list := range []string{"initContainers", "containers"} {
			containers, _ := podSpec(d)[list].([]any)
			for _, c := range containers {
				requests, _ := nestedField(c, "resources", "requests").(map[string]any)
				require.NotEmpty(t, requests, "%s: %v has no requests", component(d), nestedField(c, "name"))
			}
		}
	}
}

func TestEveryWorkloadIsLabelledWithItsComponent(t *testing.T) {
	t.Parallel()
	s := spec(plans[0], instanceapi.Active, "docling")
	r, err := values.Render(&s, base(), operatorNS, values.Storage{})
	require.NoError(t, err)
	docs := renderedDocs(t, r)

	seen := map[string]bool{}
	for _, d := range docs {
		switch d["kind"] {
		case "Deployment", "StatefulSet", "Job":
			labels, _ := nestedField(d, "spec", "template", "metadata", "labels").(map[string]any)
			c, _ := labels["app.kubernetes.io/component"].(string)
			require.NotEmpty(t, c, "%v %v", d["kind"], nestedField(d, "metadata", "name"))
			require.Equal(t, "trenova", labels["app.kubernetes.io/instance"])
			require.Equal(t, "trenova", labels["app.kubernetes.io/name"])
			seen[c] = true
		case "Cluster":
			labels, _ := nestedField(d, "spec", "inheritedMetadata", "labels").(map[string]any)
			require.Equal(t, map[string]any{
				"app.kubernetes.io/name":      "trenova",
				"app.kubernetes.io/instance":  "trenova",
				"app.kubernetes.io/component": "postgres",
			}, labels, "CloudNativePG gives its pods the cluster's inherited labels")
			seen["postgres"] = true
		}
	}
	for _, c := range []string{"api", "worker", "release", "postgres", "loki", "alloy", "imaging", "temporal"} {
		require.True(t, seen[c], "%s is not rendered with its component label", c)
	}
}

func TestWithoutLogKeepingNoneOfItRenders(t *testing.T) {
	t.Parallel()
	ch, err := loader.Load(chartDir)
	require.NoError(t, err)
	logs, _ := ch.Values["logs"].(map[string]any)
	require.Equal(t, false, logs["enabled"], "a self-hosted install keeps no logs unless it asks")

	r := managedLogs(t, base())
	r.Values["logs"].(map[string]any)["enabled"] = false
	for _, d := range renderedDocs(t, r) {
		require.NotContains(t, []string{"loki", "alloy"}, component(d),
			"%v %v", d["kind"], nestedField(d, "metadata", "name"))
		require.NotContains(t, []any{"Role", "RoleBinding", "ServiceAccount", "PersistentVolumeClaim"}, d["kind"],
			"%v", nestedField(d, "metadata", "name"))
	}
}

func TestTheOperatorIsToldWhereLokiIs(t *testing.T) {
	t.Parallel()
	for release, url := range map[string]string{
		"trenova": "http://" + values.LokiService("trenova") + ".trenova.svc:3100",
		"acme":    "http://" + values.LokiService("acme") + ".trenova.svc:3100",
		"a-very-long-release-name-that-goes-on-and-on": "http://" +
			values.LokiService("a-very-long-release-name-that-goes-on-and-on") + ".trenova.svc:3100",
	} {
		ch, err := loader.Load(operatorChartDir)
		require.NoError(t, err)
		vals, err := chartutil.ToRenderValues(ch, map[string]any{
			"target": map[string]any{"release": release, "namespace": "trenova"},
		}, chartutil.ReleaseOptions{
			Name: "trenova-operator", Namespace: operatorNS, Revision: 1, IsInstall: true,
		}, chartutil.DefaultCapabilities)
		require.NoError(t, err)
		out, err := engine.Render(ch, vals)
		require.NoError(t, err)
		require.Regexp(t,
			`- name: TRENOVA_OPERATOR_LOKI_URL\s+value: "`+strings.ReplaceAll(url, ".", `\.`)+`"`,
			template(t, out, "templates/deployment.yaml"), release)
	}
}

func TestNoReadinessOrLivenessProbeRunsOnTheOneSecondDefault(t *testing.T) {
	t.Parallel()
	s := spec(plans[0], instanceapi.Active, "docling")
	r, err := values.Render(&s, base(), operatorNS, values.Storage{})
	require.NoError(t, err)

	probes := 0
	for _, d := range renderedDocs(t, r) {
		if d["kind"] != "Deployment" && d["kind"] != "StatefulSet" {
			continue
		}
		containers, _ := podSpec(d)["containers"].([]any)
		for _, c := range containers {
			container, _ := c.(map[string]any)
			for _, kind := range []string{"readinessProbe", "livenessProbe"} {
				probe, ok := container[kind].(map[string]any)
				if !ok {
					continue
				}
				probes++
				where := fmt.Sprintf("%v %v %v", nestedField(d, "metadata", "name"), container["name"], kind)
				timeout, _ := probe["timeoutSeconds"].(float64)
				require.GreaterOrEqual(t, timeout, float64(3), where)
				if kind == "readinessProbe" {
					threshold, set := probe["failureThreshold"].(float64)
					require.True(t, !set || threshold >= 3, where)
				}
			}
		}
	}
	require.NotZero(t, probes)
}
