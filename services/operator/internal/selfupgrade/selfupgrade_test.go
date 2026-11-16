package selfupgrade_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

	"github.com/emoss08/trenova/operator/internal/selfupgrade"
)

const (
	chartName = "trenova-operator"
	ownNS     = "trenova-system"
)

type rig struct {
	dyn  *dynamicfake.FakeDynamicClient
	kube *fake.Clientset
	now  time.Time
	up   *selfupgrade.Upgrader
}

func helmChart(version string, owners ...metav1.OwnerReference) *unstructured.Unstructured {
	hc := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "helm.cattle.io/v1",
		"kind":       "HelmChart",
		"metadata":   map[string]any{"name": chartName, "namespace": "kube-system"},
		"spec": map[string]any{
			"chart":           "oci://ghcr.io/emoss08/charts/trenova-operator",
			"version":         version,
			"targetNamespace": ownNS,
		},
	}}
	hc.SetOwnerReferences(owners)
	return hc
}

func newRig(t *testing.T, own string, objs ...runtime.Object) *rig {
	t.Helper()
	r := &rig{
		dyn: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
			map[schema.GroupVersionResource]string{selfupgrade.HelmCharts: "HelmChartList"}, objs...),
		kube: fake.NewClientset(),
		now:  time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC),
	}
	up, err := selfupgrade.New(&selfupgrade.Params{
		Dynamic: r.dyn, Kube: r.kube, Version: own, HelmChart: chartName,
		Namespace: ownNS, Deployment: chartName, Logger: slog.New(slog.DiscardHandler),
		Now: func() time.Time { return r.now },
	})
	require.NoError(t, err)
	r.up = up
	return r
}

func (r *rig) chart(t *testing.T) *unstructured.Unstructured {
	t.Helper()
	hc, err := r.dyn.Resource(selfupgrade.HelmCharts).Namespace("kube-system").
		Get(t.Context(), chartName, metav1.GetOptions{})
	require.NoError(t, err)
	return hc
}

func asked(t *testing.T, hc *unstructured.Unstructured) string {
	t.Helper()
	v, _, err := unstructured.NestedString(hc.Object, "spec", "version")
	require.NoError(t, err)
	return v
}

func installJob(version string, failed int32) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "helm-install-" + chartName, Namespace: "kube-system"},
		Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "helm",
				Env:  []corev1.EnvVar{{Name: "VERSION", Value: version}},
			}},
		}}},
		Status: batchv1.JobStatus{Failed: failed},
	}
}

func TestAnOperatorOnItsReleaseChangesNothing(t *testing.T) {
	t.Parallel()
	r := newRig(t, "0.12.2", helmChart("0.12.2"))

	st := r.up.Converge(t.Context(), "0.12.2")

	assert.Equal(t, selfupgrade.PhaseCurrent, st.Phase)
	assert.False(t, st.Holds())
	assert.Equal(t, "0.12.2", asked(t, r.chart(t)))
	assert.Empty(t, st.Message)
}

func TestANewReleaseUpgradesTheOperatorFirst(t *testing.T) {
	t.Parallel()
	r := newRig(t, "0.12.2", helmChart("0.12.2"))

	st := r.up.Converge(t.Context(), "0.13.0")

	assert.Equal(t, selfupgrade.PhaseUpgrading, st.Phase)
	assert.True(t, st.Holds(), "Trenova waits for the operator of its release")
	hc := r.chart(t)
	assert.Equal(t, "0.13.0", asked(t, hc))
	assert.Equal(t, "2026-11-15T09:00:00Z", hc.GetAnnotations()[selfupgrade.AnnotationStarted])

	r.now = r.now.Add(time.Minute)
	st = r.up.Converge(t.Context(), "0.13.0")
	assert.Equal(t, selfupgrade.PhaseUpgrading, st.Phase)
	assert.Contains(t, st.Message, "since 2026-11-15T09:00:00Z")
}

func TestTheNewOperatorSettlesItsHelmChart(t *testing.T) {
	t.Parallel()
	hc := helmChart("0.13.0")
	hc.SetAnnotations(map[string]string{selfupgrade.AnnotationStarted: "2026-11-15T08:59:00Z"})
	r := newRig(t, "0.13.0", hc)

	st := r.up.Converge(t.Context(), "0.13.0")

	assert.Equal(t, selfupgrade.PhaseCurrent, st.Phase)
	assert.NotContains(t, r.chart(t).GetAnnotations(), selfupgrade.AnnotationStarted)
}

func TestAVersionTheRegistryDoesNotHaveIsReportedNotLoopedOn(t *testing.T) {
	t.Parallel()
	r := newRig(t, "0.12.2", helmChart("0.12.2"))
	r.up.Converge(t.Context(), "0.13.0")
	_, err := r.kube.BatchV1().Jobs("kube-system").
		Create(t.Context(), installJob("0.13.0", 3), metav1.CreateOptions{})
	require.NoError(t, err)

	st := r.up.Converge(t.Context(), "0.13.0")

	assert.Equal(t, selfupgrade.PhaseFailed, st.Phase)
	assert.True(t, st.Holds())
	assert.Contains(t, st.Message, "registry may not have that version")
	hc := r.chart(t)
	assert.Equal(t, "0.12.2", asked(t, hc), "the operator it runs is asked for again")
	assert.Equal(t, "0.13.0", hc.GetAnnotations()[selfupgrade.AnnotationFailed])

	st = r.up.Converge(t.Context(), "0.13.0")
	assert.Equal(t, selfupgrade.PhaseFailed, st.Phase)
	assert.Equal(t, "0.12.2", asked(t, r.chart(t)), "the failed version is not asked for again")

	st = r.up.Converge(t.Context(), "0.13.1")
	assert.Equal(t, selfupgrade.PhaseUpgrading, st.Phase, "a new release is tried")
	assert.Equal(t, "0.13.1", asked(t, r.chart(t)))
}

func TestAnInstallJobStillRetryingIsWaitedFor(t *testing.T) {
	t.Parallel()
	r := newRig(t, "0.12.2", helmChart("0.12.2"))
	r.up.Converge(t.Context(), "0.13.0")
	_, err := r.kube.BatchV1().Jobs("kube-system").
		Create(t.Context(), installJob("0.13.0", 1), metav1.CreateOptions{})
	require.NoError(t, err)

	st := r.up.Converge(t.Context(), "0.13.0")

	assert.Equal(t, selfupgrade.PhaseUpgrading, st.Phase)
	assert.Equal(t, "0.13.0", asked(t, r.chart(t)))
}

func TestANewOperatorThatNeverBecomesReadyIsRolledBack(t *testing.T) {
	t.Parallel()
	r := newRig(t, "0.12.2", helmChart("0.12.2"))
	r.up.Converge(t.Context(), "0.13.0")
	_, err := r.kube.AppsV1().Deployments(ownNS).Create(t.Context(), &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: chartName, Namespace: ownNS,
			Labels: map[string]string{"app.kubernetes.io/version": "0.13.0"},
		},
		Status: appsv1.DeploymentStatus{Conditions: []appsv1.DeploymentCondition{{
			Type:    appsv1.DeploymentProgressing,
			Status:  corev1.ConditionFalse,
			Reason:  "ProgressDeadlineExceeded",
			Message: `ReplicaSet "trenova-operator-7d9" has timed out progressing.`,
		}}},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	st := r.up.Converge(t.Context(), "0.13.0")

	assert.Equal(t, selfupgrade.PhaseFailed, st.Phase)
	assert.Contains(t, st.Message, "never became ready")
	assert.Equal(t, "0.12.2", asked(t, r.chart(t)))
}

func TestAnUpgradeThatTakesTooLongIsRolledBack(t *testing.T) {
	t.Parallel()
	r := newRig(t, "0.12.2", helmChart("0.12.2"))
	r.up.Converge(t.Context(), "0.13.0")

	r.now = r.now.Add(selfupgrade.DefaultDeadline + time.Minute)
	st := r.up.Converge(t.Context(), "0.13.0")

	assert.Equal(t, selfupgrade.PhaseFailed, st.Phase)
	assert.Contains(t, st.Message, "did not take over")
	assert.Equal(t, "0.12.2", asked(t, r.chart(t)))
}

func TestAnOperatorNotInstalledByHelmControllerIsLeftAlone(t *testing.T) {
	t.Parallel()
	r := newRig(t, "0.0.0-dev")

	st := r.up.Converge(t.Context(), "0.11.0-ci")

	assert.Equal(t, selfupgrade.PhaseUnmanaged, st.Phase)
	assert.False(t, st.Holds())
}

func TestAVersionPinnedByTheNodesManifestIsReported(t *testing.T) {
	t.Parallel()
	r := newRig(t, "0.12.2", helmChart("0.12.0", metav1.OwnerReference{
		APIVersion: "k3s.cattle.io/v1", Kind: "Addon", Name: "trenova-charts", UID: "addon",
	}))

	st := r.up.Converge(t.Context(), "0.12.2")

	assert.Equal(t, selfupgrade.PhaseCurrent, st.Phase)
	assert.Contains(t, st.Message, "trenova-charts pins the operator's version")
	assert.Equal(t, "0.12.2", asked(t, r.chart(t)), "the HelmChart names the version it runs")
}

func TestAnUpgraderWithoutItsVersionIsRefused(t *testing.T) {
	t.Parallel()
	_, err := selfupgrade.New(&selfupgrade.Params{Logger: slog.New(slog.DiscardHandler)})
	require.Error(t, err)
	_, err = selfupgrade.New(&selfupgrade.Params{
		Version: "0.12.2", HelmChart: chartName, Logger: slog.New(slog.DiscardHandler),
	})
	require.Error(t, err, "a managed upgrader needs its clients")
}
