package health_test

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

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

func pod(name string, lbls map[string]string, created int64, status corev1.PodStatus) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: "trenova", Labels: lbls,
			CreationTimestamp: metav1.Unix(created, 0),
		},
		Status: status,
	}
}

func TestNamesWhyPostgresIsNotReady(t *testing.T) {
	t.Parallel()
	initdb := map[string]string{"cnpg.io/cluster": "trenova-postgres", "cnpg.io/jobRole": "initdb"}
	c := checker(t, cnpg("Setting up primary", 0, false),
		pod("trenova-postgres-1-initdb-old", initdb, 100, corev1.PodStatus{Phase: corev1.PodPending}),
		pod("trenova-postgres-1-initdb-new", initdb, 200, corev1.PodStatus{
			Phase: corev1.PodPending,
			InitContainerStatuses: []corev1.ContainerStatus{{
				Name: "bootstrap-controller",
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{
					Reason:  "ImagePullBackOff",
					Message: "Back-off pulling image \"ghcr.io/emoss08/trenova/postgres:0.11.0\"\nmore detail",
				}},
			}},
		}),
	)
	comps, err := c.Check(t.Context(), health.CheckParams{HelmRevision: 1, Stopped: true})
	require.NoError(t, err)
	require.Equal(t,
		`pod trenova-postgres-1-initdb-new: bootstrap-controller ImagePullBackOff: Back-off pulling image "ghcr.io/emoss08/trenova/postgres:0.11.0"; Setting up primary (0 of 1 instances ready)`,
		byName(comps)["postgres"].Message)
}

func TestNamesUnschedulablePodsAndPendingClaims(t *testing.T) {
	t.Parallel()
	cluster := map[string]string{"cnpg.io/cluster": "trenova-postgres"}
	c := checker(t, cnpg("Setting up primary", 0, false),
		pod("trenova-postgres-1", cluster, 100, corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{{
				Type: corev1.PodScheduled, Status: corev1.ConditionFalse, Reason: "Unschedulable",
				Message: "0/1 nodes are available: pod has unbound immediate PersistentVolumeClaims.",
			}},
		}),
	)
	comps, err := c.Check(t.Context(), health.CheckParams{HelmRevision: 1, Stopped: true})
	require.NoError(t, err)
	require.Contains(t, byName(comps)["postgres"].Message,
		"pod trenova-postgres-1: Unschedulable: 0/1 nodes are available: pod has unbound immediate PersistentVolumeClaims.")

	// No pod yet, only a claim that cannot be provisioned.
	c = checker(t, cnpg("Setting up primary", 0, false),
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "trenova-postgres-1", Namespace: "trenova", Labels: cluster},
			Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "e1", Namespace: "trenova"},
			InvolvedObject: corev1.ObjectReference{Kind: "PersistentVolumeClaim", Name: "trenova-postgres-1"},
			Type:           corev1.EventTypeWarning, Reason: "ProvisioningFailed",
			Message: `storageclass.storage.k8s.io "fast" not found`,
		},
	)
	comps, err = c.Check(t.Context(), health.CheckParams{HelmRevision: 1, Stopped: true})
	require.NoError(t, err)
	require.Contains(t, byName(comps)["postgres"].Message,
		`volume claim trenova-postgres-1: ProvisioningFailed: storageclass.storage.k8s.io "fast" not found`)
}

func TestNamesTheLatestWarningWhenContainersAreStillCreating(t *testing.T) {
	t.Parallel()
	selector := map[string]string{"app.kubernetes.io/component": "api"}
	c := checker(t, cnpg("Cluster in healthy state", 1, false),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "trenova-api", Namespace: "trenova", Labels: labels("api"), Generation: 1},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptr(int32(1)), Selector: &metav1.LabelSelector{MatchLabels: selector},
			},
			Status: appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, UpdatedReplicas: 1},
		},
		pod("trenova-api-abc", selector, 100, corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:  "api",
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ContainerCreating"}},
			}},
		}),
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "old", Namespace: "trenova"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "trenova-api-abc"},
			Type:           corev1.EventTypeWarning, Reason: "FailedScheduling", Message: "earlier",
			LastTimestamp: metav1.Unix(100, 0),
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "new", Namespace: "trenova"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "trenova-api-abc"},
			Type:           corev1.EventTypeWarning, Reason: "FailedMount",
			Message:       `MountVolume.SetUp failed for volume "certs": secret "imaging-client" not found`,
			LastTimestamp: metav1.Unix(200, 0),
		},
	)
	comps, err := c.Check(t.Context(), health.CheckParams{HelmRevision: 1, Stopped: true})
	require.NoError(t, err)
	require.Equal(t,
		`pod trenova-api-abc: FailedMount: MountVolume.SetUp failed for volume "certs": secret "imaging-client" not found; 0 of 1 replicas available`,
		byName(comps)["api"].Message)
}

func TestNamesCrashingContainersAndKeepsTheCauseWithinTheLimit(t *testing.T) {
	t.Parallel()
	selector := map[string]string{"app.kubernetes.io/component": "worker"}
	long := strings.Repeat("x", 300)
	c := checker(t, cnpg("Cluster in healthy state", 1, false),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "trenova-worker", Namespace: "trenova", Labels: labels("worker"), Generation: 1},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptr(int32(1)), Selector: &metav1.LabelSelector{MatchLabels: selector},
			},
			Status: appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, UpdatedReplicas: 1},
		},
		pod("trenova-worker-abc", selector, 100, corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "worker",
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{
					Reason: "CrashLoopBackOff", Message: "back-off 5m0s restarting failed container " + long,
				}},
			}},
		}),
	)
	comps, err := c.Check(t.Context(), health.CheckParams{HelmRevision: 1, Stopped: true})
	require.NoError(t, err)
	msg := byName(comps)["worker"].Message
	require.True(t, strings.HasPrefix(msg, "pod trenova-worker-abc: worker CrashLoopBackOff: back-off"), msg)
	require.LessOrEqual(t, len([]rune(msg)), 200)
}

func TestLeavesReadyComponentsAlone(t *testing.T) {
	t.Parallel()
	selector := map[string]string{"app.kubernetes.io/component": "api"}
	c := checker(t, cnpg("Cluster in healthy state", 1, false),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "trenova-api", Namespace: "trenova", Labels: labels("api"), Generation: 1},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptr(int32(1)), Selector: &metav1.LabelSelector{MatchLabels: selector},
			},
			Status: appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1},
		},
		// An old pod still failing does not make a ready component unready.
		pod("trenova-api-old", selector, 100, corev1.PodStatus{
			Phase: corev1.PodFailed, Reason: "Evicted", Message: "node was low on memory",
		}),
	)
	comps, err := c.Check(t.Context(), health.CheckParams{HelmRevision: 1, Stopped: true})
	require.NoError(t, err)
	require.True(t, byName(comps)["api"].Ready)
	require.Empty(t, byName(comps)["api"].Message)
}

func TestTrimsBetweenCharacters(t *testing.T) {
	t.Parallel()
	selector := map[string]string{"app.kubernetes.io/component": "worker"}
	c := checker(t, cnpg("Cluster in healthy state", 1, false),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "trenova-worker", Namespace: "trenova", Labels: labels("worker"), Generation: 1},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptr(int32(1)), Selector: &metav1.LabelSelector{MatchLabels: selector},
			},
			Status: appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, UpdatedReplicas: 1},
		},
		pod("trenova-worker-abc", selector, 100, corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "worker",
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{
					Reason: "CreateContainerConfigError", Message: strings.Repeat("“é”", 100),
				}},
			}},
		}),
	)
	comps, err := c.Check(t.Context(), health.CheckParams{HelmRevision: 1, Stopped: true})
	require.NoError(t, err)
	msg := byName(comps)["worker"].Message
	require.True(t, utf8.ValidString(msg))
	require.Equal(t, 200, utf8.RuneCountInString(msg))
}

func TestCleanMakesMessagesTheInstanceAPIAccepts(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, want string }{
		{"install failed:\n  timed out waiting for the condition\n", "install failed: timed out waiting for the condition"},
		{"a\tb\r\nc  d", "a b c d"},
		{"\x00\x1bstart", "start"},
		{"", ""},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, health.Clean(tc.in, 200))
	}
	long := health.Clean(strings.Repeat("“é”\n", 300), 500)
	require.True(t, utf8.ValidString(long))
	require.Equal(t, 500, utf8.RuneCountInString(long))
	require.False(t, strings.ContainsFunc(long, unicode.IsControl), "the control plane refuses control characters")
}
