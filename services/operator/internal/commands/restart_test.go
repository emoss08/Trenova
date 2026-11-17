package commands

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/yaml"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

var chartDir = filepath.Join("..", "..", "..", "..", "deploy", "helm", "trenova")

func readyDeployment(component string) *appsv1.Deployment {
	d := deployment(component)
	d.Status = appsv1.DeploymentStatus{Replicas: 1, UpdatedReplicas: 1, ReadyReplicas: 1, AvailableReplicas: 1}
	return d
}

func readyStatefulSet(component string) *appsv1.StatefulSet {
	s := statefulSet(component)
	s.Status = appsv1.StatefulSetStatus{Replicas: 1, ReadyReplicas: 1, UpdatedReplicas: 1}
	return s
}

func quick(r *Runner, wait time.Duration) {
	r.tierWait, r.tierPoll = wait, 5*time.Millisecond
}

func patched(kube *fake.Clientset) []string {
	var out []string
	for _, a := range kube.Actions() {
		if p, ok := a.(k8stesting.PatchAction); ok {
			out = append(out, strings.TrimPrefix(p.GetName(), "trenova-"))
		}
	}
	return out
}

func answer(t *testing.T, r *Runner, cp *fakeControlPlane, cmd instanceapi.Command) instanceapi.CommandResult {
	t.Helper()
	require.Eventually(t, func() bool {
		if r.Run(t.Context()) != nil {
			return false
		}
		return len(cp.answers[cmd.Id]) > 0
	}, 5*time.Second, 5*time.Millisecond)
	return cp.answers[cmd.Id][0]
}

func TestRestartingEveryPartGoesTierByTierAndAnUnknownPartLast(t *testing.T) {
	t.Parallel()
	cmd := restart("")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	r, kube := runner(t, cp,
		readyDeployment("client"), readyDeployment("mystery"), readyDeployment("api"), readyDeployment("worker"),
		readyDeployment("gtc"), readyDeployment("temporal"), readyStatefulSet("redis"),
		readyStatefulSet("meilisearch"),
	)
	quick(r, time.Second)

	got := answer(t, r, cp, cmd)

	require.Equal(t, instanceapi.CommandResultStatusSucceeded, got.Status, deref(got.Message))
	assert.Equal(t, "restarted 8 workload(s)", *got.Message)
	assert.Equal(t,
		[]string{"meilisearch", "redis", "temporal", "gtc", "api", "worker", "client", "mystery"},
		patched(kube),
	)
	set, err := kube.AppsV1().StatefulSets(ns).Get(t.Context(), "trenova-redis", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "2026-11-12T09:00:00Z", set.Spec.Template.Annotations[restartedAt])
}

func TestATierWaitsUntilTheOneBeforeIsReady(t *testing.T) {
	t.Parallel()
	cmd := restart("")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	r, kube := runner(t, cp, readyStatefulSet("meilisearch"), deployment("gtc"), readyDeployment("api"))
	quick(r, 5*time.Second)

	require.NoError(t, r.Run(t.Context()))
	require.Eventually(t, func() bool { return slices.Contains(patched(kube), "gtc") }, time.Second, time.Millisecond)
	assert.Never(
		t,
		func() bool { return slices.Contains(patched(kube), "api") },
		100*time.Millisecond,
		5*time.Millisecond,
		"the API restarted before search indexing was ready",
	)

	again := restart("")
	cp.commands = append(cp.commands, again)
	require.NoError(t, r.Run(t.Context()))
	assert.Empty(t, cp.answers[cmd.Id], "answered before the last tier")
	assert.Empty(t, cp.answers[again.Id], "a second restart of every part ran beside the first")

	gtc := readyDeployment("gtc")
	_, err := kube.AppsV1().Deployments(ns).UpdateStatus(t.Context(), gtc, metav1.UpdateOptions{})
	require.NoError(t, err)

	got := answer(t, r, cp, cmd)
	require.Equal(t, instanceapi.CommandResultStatusSucceeded, got.Status, deref(got.Message))
	assert.Equal(t, []string{"meilisearch", "gtc", "api"}, patched(kube)[:3])
	assert.Equal(t, instanceapi.CommandResultStatusSucceeded, answer(t, r, cp, again).Status)
}

func TestATierThatNeverComesBackIsNamedAndTheRestStillRestart(t *testing.T) {
	t.Parallel()
	cmd := restart("")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	r, kube := runner(t, cp, readyStatefulSet("redis"), deployment("gtc"), readyDeployment("api"),
		readyDeployment("client"))
	quick(r, 30*time.Millisecond)

	got := answer(t, r, cp, cmd)

	require.Equal(t, instanceapi.CommandResultStatusFailed, got.Status)
	assert.Contains(t, *got.Message, "restarted 4 workload(s) in order, but gtc did not become ready")
	assert.Equal(t, []string{"redis", "gtc", "api", "client"}, patched(kube))
}

func TestRestartingEveryPartLeavesTheDatabase(t *testing.T) {
	t.Parallel()
	cmd := restart("")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	r, kube := runner(t, cp, readyStatefulSet(componentPostgres), readyDeployment("api"))
	quick(r, time.Second)

	got := answer(t, r, cp, cmd)

	require.Equal(t, instanceapi.CommandResultStatusSucceeded, got.Status)
	assert.Equal(t, []string{"api"}, patched(kube))
}

func TestRestartingEveryPartWithNothingToRestartFails(t *testing.T) {
	t.Parallel()
	cmd := restart("")
	cp := &fakeControlPlane{commands: []instanceapi.Command{cmd}}
	r, _ := runner(t, cp)

	require.NoError(t, r.Run(t.Context()))

	require.Len(t, cp.answers[cmd.Id], 1)
	assert.Equal(t, instanceapi.CommandResultStatusFailed, cp.answers[cmd.Id][0].Status)
}

func TestEveryWorkloadTheChartRendersHasARestartTier(t *testing.T) {
	t.Parallel()
	ch, err := loader.Load(chartDir)
	require.NoError(t, err)
	on := map[string]any{"enabled": true}
	vals, err := chartutil.ToRenderValues(ch, map[string]any{
		"domain":        "trenova.example.com",
		"storageDomain": "storage.trenova.example.com",
		"docling":       on,
		"clamav":        on,
		"embedding":     on,
		"logs":          on,
		"minio":         on,
		"client":        on,
	}, chartutil.ReleaseOptions{Name: release, Namespace: ns, Revision: 1, IsInstall: true}, chartutil.DefaultCapabilities)
	require.NoError(t, err)
	out, err := engine.Render(ch, vals)
	require.NoError(t, err)

	var seen []string
	for name, body := range out {
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		for _, doc := range strings.Split(body, "\n---") {
			var m struct {
				Kind     string `json:"kind"`
				Metadata struct {
					Labels map[string]string `json:"labels"`
				} `json:"metadata"`
			}
			require.NoError(t, yaml.Unmarshal([]byte(doc), &m), name)
			if m.Kind != "Deployment" && m.Kind != "StatefulSet" {
				continue
			}
			component := m.Metadata.Labels[labelComponent]
			seen = append(seen, component)
			assert.Less(t, tierOf(component), len(restartOrder),
				"%s %q has no place in restartOrder: say which tier it restarts in", m.Kind, component)
		}
	}
	for _, want := range []string{"redis", "meilisearch", "temporal", "minio", "clamav", "docling", "embedding",
		"loki", "gtc", "alloy", "api", "worker", "imaging", "gotenberg", "client"} {
		assert.Contains(t, seen, want, "the chart no longer renders %q with every part on", want)
	}
	assert.Less(t, tierOf("meilisearch"), tierOf("gtc"), "search indexing waits on search")
	assert.Less(t, tierOf("redis"), tierOf("gtc"), "search indexing waits on the cache")
}
