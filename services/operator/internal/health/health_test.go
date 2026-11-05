package health_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/health"
)

func labels(component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "trenova",
		"app.kubernetes.io/instance":  "trenova",
		"app.kubernetes.io/component": component,
	}
}

func ptr[T any](v T) *T { return &v }

func cnpg(phase string, ready int64, hibernated bool) *unstructured.Unstructured {
	status := map[string]any{"phase": phase, "readyInstances": ready}
	if hibernated {
		status["conditions"] = []any{map[string]any{"type": "cnpg.io/hibernation", "status": "True"}}
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "postgresql.cnpg.io/v1",
		"kind":       "Cluster",
		"metadata":   map[string]any{"name": "trenova-postgres", "namespace": "trenova"},
		"spec":       map[string]any{"instances": int64(1)},
		"status":     status,
	}}
}

func checker(t *testing.T, cluster *unstructured.Unstructured, objs ...runtime.Object) *health.Checker {
	t.Helper()
	scheme := runtime.NewScheme()
	gvr := schema.GroupVersionResource{Group: "postgresql.cnpg.io", Version: "v1", Resource: "clusters"}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{gvr: "ClusterList"}, cluster)
	return health.NewChecker(&health.CheckerParams{
		Kube:      fake.NewClientset(objs...),
		Dynamic:   dyn,
		Namespace: "trenova",
		Release:   "trenova",
		Chart:     "trenova",
	})
}

func byName(comps []health.Component) map[string]health.Component {
	out := map[string]health.Component{}
	for _, c := range comps {
		out[c.Name] = c
	}
	return out
}

func TestReadyWhenEverythingIsUpAndTheJobCompleted(t *testing.T) {
	t.Parallel()
	c := checker(t, cnpg("Cluster in healthy state", 1, false),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "trenova-api", Namespace: "trenova", Labels: labels("api"), Generation: 2},
			Spec:       appsv1.DeploymentSpec{Replicas: ptr(int32(1))},
			Status:     appsv1.DeploymentStatus{ObservedGeneration: 2, Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "trenova-redis", Namespace: "trenova", Labels: labels("redis"), Generation: 1},
			Spec:       appsv1.StatefulSetSpec{Replicas: ptr(int32(1))},
			Status:     appsv1.StatefulSetStatus{ObservedGeneration: 1, Replicas: 1, ReadyReplicas: 1},
		},
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "trenova-release-3", Namespace: "trenova", Labels: labels("release")},
			Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{
				{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
			}},
		},
	)
	comps, err := c.Check(t.Context(), health.CheckParams{HelmRevision: 3})
	require.NoError(t, err)
	got := byName(comps)
	require.Len(t, got, 4)
	for name, comp := range got {
		require.True(t, comp.Ready, "%s: %s", name, comp.Message)
	}
}

func TestExplainsWhatIsNotReady(t *testing.T) {
	t.Parallel()
	c := checker(t, cnpg("Setting up primary", 0, false),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "trenova-worker", Namespace: "trenova", Labels: labels("worker"), Generation: 1},
			Spec:       appsv1.DeploymentSpec{Replicas: ptr(int32(1))},
			Status:     appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, UpdatedReplicas: 1},
		},
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "trenova-release-2", Namespace: "trenova", Labels: labels("release")},
			Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{
				{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Message: "BackoffLimitExceeded"},
			}},
		},
	)
	comps, err := c.Check(t.Context(), health.CheckParams{HelmRevision: 3})
	require.NoError(t, err)
	got := byName(comps)
	require.Equal(t, "0 of 1 replicas available", got["worker"].Message)
	require.Contains(t, got["postgres"].Message, "Setting up primary")
	require.Equal(t, "waiting for the release Job of revision 3", got["release"].Message)

	comps, err = c.Check(t.Context(), health.CheckParams{HelmRevision: 2})
	require.NoError(t, err)
	require.Equal(t, "failed: BackoffLimitExceeded", byName(comps)["release"].Message)
}

func TestStoppedAndHibernatedCountAsReady(t *testing.T) {
	t.Parallel()
	c := checker(t, cnpg("Cluster in healthy state", 0, true),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "trenova-api", Namespace: "trenova", Labels: labels("api"), Generation: 3},
			Spec:       appsv1.DeploymentSpec{Replicas: ptr(int32(0))},
			Status:     appsv1.DeploymentStatus{ObservedGeneration: 3},
		},
	)
	comps, err := c.Check(t.Context(), health.CheckParams{HelmRevision: 4, Stopped: true, Hibernated: true})
	require.NoError(t, err)
	got := byName(comps)
	require.Len(t, got, 2, "a stopped release has no Job to wait for")
	require.True(t, got["api"].Ready)
	require.Equal(t, "stopped", got["api"].Message)
	require.True(t, got["postgres"].Ready)
	require.Equal(t, "hibernated", got["postgres"].Message)
}
