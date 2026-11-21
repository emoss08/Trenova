package telemetry_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"

	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/telemetry"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const (
	ns   = "trenova"
	node = "node-1"
	gib  = int64(1 << 30)
	mib  = int64(1 << 20)
)

var minute = time.Date(2026, 11, 16, 12, 30, 0, 0, time.UTC)

func params() *health.CheckParams { return &health.CheckParams{ReleaseJob: "trenova-release-1"} }

func ptr[T any](v T) *T { return &v }

func chartLabels(component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "trenova",
		"app.kubernetes.io/instance":  "trenova",
		"app.kubernetes.io/component": component,
	}
}

func selectorFor(component string) *metav1.LabelSelector {
	return &metav1.LabelSelector{MatchLabels: map[string]string{
		"app.kubernetes.io/instance":  "trenova",
		"app.kubernetes.io/component": component,
	}}
}

func limited(cpu, memory string) corev1.ResourceRequirements {
	if cpu == "" {
		return corev1.ResourceRequirements{}
	}
	return corev1.ResourceRequirements{Limits: corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse(cpu),
		corev1.ResourceMemory: resource.MustParse(memory),
	}}
}

func readyDeployment(component string, replicas int32, image string, res corev1.ResourceRequirements) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "trenova-" + component, Namespace: ns, Labels: chartLabels(component), Generation: 1,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr(replicas),
			Selector: selectorFor(component),
			Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{
				{Name: component, Image: image, Resources: res},
			}}},
		},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 1, Replicas: replicas, UpdatedReplicas: replicas,
			AvailableReplicas: replicas, ReadyReplicas: replicas,
		},
	}
}

func runningPod(
	name, component string,
	restarts int32,
	image string,
	res corev1.ResourceRequirements,
	extra map[string]string,
) *corev1.Pod {
	l := chartLabels(component)
	for k, v := range extra {
		l[k] = v
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: l},
		Spec: corev1.PodSpec{
			NodeName:   node,
			Containers: []corev1.Container{{Name: component, Image: image, Resources: res}},
		},
		Status: corev1.PodStatus{
			Phase:             corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{Name: component, Ready: true, RestartCount: restarts}},
		},
	}
}

func postgresCluster() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "postgresql.cnpg.io/v1",
		"kind":       "Cluster",
		"metadata":   map[string]any{"name": "trenova-postgres", "namespace": ns},
		"spec":       map[string]any{"instances": int64(1)},
		"status": map[string]any{
			"phase":          "Cluster in healthy state",
			"readyInstances": int64(1),
			"image":          "ghcr.io/cloudnative-pg/postgresql:17.5",
		},
	}}
}

func standardObjects() []runtime.Object {
	apiLimits := limited("500m", "1Gi")
	return []runtime.Object{
		readyDeployment("api", 2, "ghcr.io/emoss08/trenova-api:1.4.2@sha256:abc", apiLimits),
		readyDeployment("client", 1, "ghcr.io/emoss08/trenova-client:1.4.2", limited("", "")),
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "trenova-meilisearch", Namespace: ns, Labels: chartLabels("meilisearch"), Generation: 1},
			Spec: appsv1.StatefulSetSpec{
				Replicas: ptr(int32(1)),
				Selector: selectorFor("meilisearch"),
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{
					{Name: "meilisearch", Image: "getmeili/meilisearch:v1.15.2"},
				}}},
			},
			Status: appsv1.StatefulSetStatus{ObservedGeneration: 1, Replicas: 1, ReadyReplicas: 1},
		},
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "trenova-release-1", Namespace: ns, Labels: chartLabels("release")},
			Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{
				{Name: "release", Image: "ghcr.io/emoss08/trenova-api:1.4.2"},
			}}}},
			Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{
				{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
			}},
		},
		runningPod("trenova-api-a", "api", 1, "ghcr.io/emoss08/trenova-api:1.4.2", apiLimits, nil),
		runningPod("trenova-api-b", "api", 2, "ghcr.io/emoss08/trenova-api:1.4.2", apiLimits, nil),
		runningPod("trenova-client-a", "client", 0, "ghcr.io/emoss08/trenova-client:1.4.2", limited("", ""), nil),
		runningPod("trenova-meilisearch-0", "meilisearch", 0, "getmeili/meilisearch:v1.15.2", limited("1", "2Gi"), nil),
		runningPod("trenova-postgres-1", "postgres", 4, "ghcr.io/cloudnative-pg/postgresql:17.5", limited("2", "4Gi"),
			map[string]string{"cnpg.io/cluster": "trenova-postgres"}),
		runningPod("trenova-loki-0", "loki", 0, "grafana/loki:3.5.0", limited("1", "1Gi"), nil),
		machineNode(),
	}
}

func machineNode() *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: node},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("16Gi"),
			},
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeHostName, Address: node},
				{Type: corev1.NodeInternalIP, Address: "10.0.0.5"},
			},
			DaemonEndpoints: corev1.NodeDaemonEndpoints{KubeletEndpoint: corev1.DaemonEndpoint{Port: 10250}},
		},
	}
}

type fakeSummaries struct {
	mu         sync.Mutex
	err        error
	asked      []string
	sharedDisk bool
	dataDisk   bool
}

func (f *fakeSummaries) Summary(_ context.Context, kubelet string) (telemetry.Summary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, kubelet)
	if f.err != nil {
		return telemetry.Summary{}, f.err
	}
	used := func(n int64) *uint64 { v := uint64(n); return &v }
	logsVolume := telemetry.SummaryVolume{
		Name: "storage", PVCRef: &telemetry.PodRef{Name: "trenova-loki", Namespace: ns},
		CapacityBytes: used(20 * gib), UsedBytes: used(2 * gib),
	}
	if f.sharedDisk {
		logsVolume.CapacityBytes, logsVolume.UsedBytes = used(100*gib), used(10*gib)
	}
	machine := telemetry.SummaryNode{FS: &telemetry.FSStats{
		CapacityBytes: used(100 * gib), UsedBytes: used(10 * gib),
	}}
	if f.dataDisk {
		machine.Runtime = &telemetry.SummaryRuntime{ImageFS: &telemetry.FSStats{
			CapacityBytes: used(500 * gib), AvailableBytes: used(460 * gib), UsedBytes: used(6 * gib),
		}}
		logsVolume.CapacityBytes, logsVolume.UsedBytes = used(500*gib), used(40*gib)
	}
	return telemetry.Summary{
		Node: machine,
		Pods: []telemetry.SummaryPod{
			{
				PodRef: telemetry.PodRef{Name: "trenova-postgres-1", Namespace: ns},
				Volumes: []telemetry.SummaryVolume{
					{Name: "pgdata", PVCRef: &telemetry.PodRef{Name: "trenova-postgres-1", Namespace: ns}, UsedBytes: used(5 * gib)},
					{Name: "scratch", UsedBytes: used(gib)},
				},
			},
			{
				PodRef: telemetry.PodRef{Name: "trenova-meilisearch-0", Namespace: ns},
				Volumes: []telemetry.SummaryVolume{
					{Name: "data", PVCRef: &telemetry.PodRef{Name: "data-trenova-meilisearch-0", Namespace: ns}, UsedBytes: used(gib)},
				},
			},
			{
				PodRef:  telemetry.PodRef{Name: "trenova-loki-0", Namespace: ns},
				Volumes: []telemetry.SummaryVolume{logsVolume},
			},
			{
				PodRef: telemetry.PodRef{Name: "trenova-loki-0", Namespace: "elsewhere"},
				Volumes: []telemetry.SummaryVolume{
					{Name: "pgdata", PVCRef: &telemetry.PodRef{Name: "other", Namespace: "elsewhere"}, UsedBytes: used(50 * gib)},
				},
			},
		},
	}, nil
}

type fakeData struct {
	mu     sync.Mutex
	answer tms.DataUsage
	err    error
	calls  int
}

func (f *fakeData) DataUsage(context.Context) (tms.DataUsage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.answer, f.err
}

func (f *fakeData) set(answer tms.DataUsage, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.answer, f.err = answer, err
}

func (f *fakeData) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

const kubeletAt = "10.0.0.5:10250"

type world struct {
	kube      *fake.Clientset
	metrics   *metricsfake.Clientset
	summaries *fakeSummaries
	data      *fakeData
	logs      *logCounter
	checker   *health.Checker
	sampler   *telemetry.Sampler
}

func podUsage(name, cpu, memory string) metricsv1beta1.PodMetrics {
	return metricsv1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Containers: []metricsv1beta1.ContainerMetrics{{Name: "main", Usage: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpu),
			corev1.ResourceMemory: resource.MustParse(memory),
		}}},
	}
}

func newWorld(t *testing.T, ownNode string, objs ...runtime.Object) *world {
	t.Helper()
	scheme := runtime.NewScheme()
	gvr := schema.GroupVersionResource{Group: "postgresql.cnpg.io", Version: "v1", Resource: "clusters"}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		scheme, map[schema.GroupVersionResource]string{gvr: "ClusterList"}, postgresCluster(),
	)
	w := &world{
		kube:      fake.NewClientset(objs...),
		metrics:   metricsfake.NewSimpleClientset(),
		summaries: &fakeSummaries{},
		data: &fakeData{answer: tms.DataUsage{
			MeasuredAt:     minute,
			DatabaseBytes:  ptr(5 * gib),
			DocumentsBytes: ptr(3 * gib),
			SearchBytes:    ptr(gib),
		}},
		logs: &logCounter{},
	}
	w.metrics.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, &metricsv1beta1.PodMetricsList{Items: []metricsv1beta1.PodMetrics{
			podUsage("trenova-api-a", "250m", "300Mi"),
			podUsage("trenova-api-b", "150m", "200Mi"),
			podUsage("trenova-client-a", "10m", "50Mi"),
			podUsage("trenova-meilisearch-0", "20m", "400Mi"),
			podUsage("trenova-postgres-1", "300m", "1Gi"),
		}}, nil
	})
	w.metrics.PrependReactor("get", "nodes", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, &metricsv1beta1.NodeMetrics{
			ObjectMeta: metav1.ObjectMeta{Name: node},
			Usage: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1200m"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		}, nil
	})
	w.checker = health.NewChecker(&health.CheckerParams{
		Kube: w.kube, Dynamic: dyn, Namespace: ns, Release: "trenova", Chart: "trenova",
	})
	w.sampler = telemetry.NewSampler(&telemetry.SamplerParams{
		Kube:      w.kube,
		Metrics:   w.metrics,
		Parts:     w.checker,
		Summaries: w.summaries,
		Data:      w.data,
		Namespace: ns,
		OwnNode:   ownNode,
		Logger:    slog.New(w.logs),
	})
	return w
}

func (w *world) metricsDown() {
	down := func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("the server is currently unable to handle the request")
	}
	w.metrics.PrependReactor("list", "pods", down)
	w.metrics.PrependReactor("get", "nodes", down)
}

func partsByName(s instanceapi.TelemetrySample) map[string]instanceapi.PartUsage {
	out := map[string]instanceapi.PartUsage{}
	for _, p := range s.Parts {
		out[p.Name] = p
	}
	return out
}

func TestAPartWithTwoCopiesSumsRestartsAndUsage(t *testing.T) {
	t.Parallel()
	w := newWorld(t, "", standardObjects()...)

	api := partsByName(w.sampler.Sample(t.Context(), minute, params()))["api"]

	require.True(t, api.Ready)
	require.Equal(t, int32(2), *api.Replicas)
	require.Equal(t, int32(2), *api.ReadyReplicas)
	require.Equal(t, int64(3), *api.Restarts)
	require.Equal(t, int64(400), *api.CpuMillicores)
	require.Equal(t, 500*mib, *api.MemoryBytes)
	require.Equal(t, int64(1000), *api.CpuLimitMillicores)
	require.Equal(t, 2*gib, *api.MemoryLimitBytes)
	require.Equal(t, "1.4.2", *api.Version, "the tag, without the registry, the name or the digest")
}

func startedAt(objs []runtime.Object, at map[string]time.Time) []runtime.Object {
	for _, o := range objs {
		pod, ok := o.(*corev1.Pod)
		if !ok {
			continue
		}
		if when, named := at[pod.Name]; named {
			pod.Status.ContainerStatuses[0].State.Running = &corev1.ContainerStateRunning{
				StartedAt: metav1.NewTime(when),
			}
		}
	}
	return objs
}

func TestAPartSaysWhenItsNewestContainerStarted(t *testing.T) {
	t.Parallel()
	older, newer := minute.Add(-3*time.Hour), minute.Add(-10*time.Minute)
	w := newWorld(t, "", startedAt(standardObjects(), map[string]time.Time{
		"trenova-api-a": older,
		"trenova-api-b": newer,
	})...)

	parts := partsByName(w.sampler.Sample(t.Context(), minute, params()))

	require.NotNil(t, parts["api"].StartedAt)
	require.True(t, newer.Equal(*parts["api"].StartedAt), "the newest of its copies, so any copy starting again moves it")
	require.Nil(t, parts["client"].StartedAt, "none of its containers is known to be running")
}

func TestAPartWithNoLimitOmitsTheLimit(t *testing.T) {
	t.Parallel()
	w := newWorld(t, "", standardObjects()...)

	client := partsByName(w.sampler.Sample(t.Context(), minute, params()))["client"]

	require.Nil(t, client.CpuLimitMillicores)
	require.Nil(t, client.MemoryLimitBytes)
	require.Equal(t, int64(10), *client.CpuMillicores)
	require.Equal(t, int64(0), *client.Restarts)
}

func TestTheMetricsAPIDownOmitsUsageButKeepsReadiness(t *testing.T) {
	t.Parallel()
	w := newWorld(t, "", standardObjects()...)
	w.metricsDown()

	sample := w.sampler.Sample(t.Context(), minute, params())

	api := partsByName(sample)["api"]
	require.True(t, api.Ready)
	require.Equal(t, int32(2), *api.Replicas)
	require.Equal(t, int64(3), *api.Restarts)
	require.Equal(t, int64(1000), *api.CpuLimitMillicores)
	require.Nil(t, api.CpuMillicores, "never sent as zero")
	require.Nil(t, api.MemoryBytes)
	require.NotNil(t, sample.Machine)
	require.Nil(t, sample.Machine.CpuMillicores)
	require.Nil(t, sample.Machine.MemoryBytes)
	require.Equal(t, int64(4000), *sample.Machine.CpuCapacityMillicores)
	require.Equal(t, 10*gib, *sample.Machine.DiskBytes)
}

func TestTheStatsSummaryDownOmitsDiskAndData(t *testing.T) {
	t.Parallel()
	w := newWorld(t, "", standardObjects()...)
	w.summaries.err = errors.New("the node proxy answered 403")

	sample := w.sampler.Sample(t.Context(), minute, params())

	require.NotNil(t, sample.Machine)
	require.Nil(t, sample.Machine.DiskBytes)
	require.Nil(t, sample.Machine.DiskCapacityBytes)
	require.Nil(t, sample.Data.LogsBytes)
	require.Equal(t, 5*gib, *sample.Data.DatabaseBytes, "Trenova's figures do not need the kubelet")
	require.Equal(t, int64(1200), *sample.Machine.CpuMillicores)
	require.Equal(t, 16*gib, *sample.Machine.MemoryCapacityBytes)
}

func TestTheMachineAndTheDataAreRead(t *testing.T) {
	t.Parallel()
	w := newWorld(t, "", standardObjects()...)

	sample := w.sampler.Sample(t.Context(), minute, params())

	require.Equal(t, minute, sample.At)
	require.Equal(t, instanceapi.MachineUsage{
		CpuMillicores:         ptr(int64(1200)),
		CpuCapacityMillicores: ptr(int64(4000)),
		MemoryBytes:           ptr(8 * gib),
		MemoryCapacityBytes:   ptr(16 * gib),
		DiskBytes:             ptr(10 * gib),
		DiskCapacityBytes:     ptr(100 * gib),
	}, *sample.Machine)
	require.Equal(t, instanceapi.DataUsage{
		DatabaseBytes:  ptr(5 * gib),
		DocumentsBytes: ptr(3 * gib),
		SearchBytes:    ptr(gib),
		LogsBytes:      ptr(2 * gib),
	}, *sample.Data, "Trenova's figures, and Loki's volume for the logs")
	require.Equal(t, []string{kubeletAt}, w.summaries.asked)
}

func TestAVolumeOnTheNodesOwnDiskIsNotMeasuredAsLogs(t *testing.T) {
	t.Parallel()
	w := newWorld(t, "", standardObjects()...)
	w.summaries.sharedDisk = true

	sample := w.sampler.Sample(t.Context(), minute, params())

	require.Nil(t, sample.Data.LogsBytes,
		"a volume that is a directory on the node's disk reports the whole disk, which is not what the logs take")
	require.Equal(t, 10*gib, *sample.Machine.DiskBytes)
	require.Equal(t, 5*gib, *sample.Data.DatabaseBytes)
}

func TestTheDiskReadIsTheOneHoldingTheInstancesData(t *testing.T) {
	t.Parallel()
	w := newWorld(t, "", standardObjects()...)
	w.summaries.dataDisk = true

	sample := w.sampler.Sample(t.Context(), minute, params())

	require.Equal(t, 500*gib, *sample.Machine.DiskCapacityBytes,
		"the volume the images and the claims are on, not the disk the machine boots from")
	require.Equal(t, 40*gib, *sample.Machine.DiskBytes,
		"what is no longer free on it, not only what the images take")
	require.Nil(t, sample.Data.LogsBytes,
		"a claim that is a directory on that volume reports the whole volume")
}

func TestThePartsAreNamedAsTheStatusReportNamesThem(t *testing.T) {
	t.Parallel()
	w := newWorld(t, "", standardObjects()...)

	comps, err := w.checker.Check(t.Context(), *params())
	require.NoError(t, err)
	sample := w.sampler.Sample(t.Context(), minute, params())

	var reported, sampled []string
	for _, c := range comps {
		reported = append(reported, c.Name)
	}
	for _, p := range sample.Parts {
		sampled = append(sampled, p.Name)
	}
	require.Equal(t, []string{"api", "client", "meilisearch", "postgres", "release"}, reported)
	require.Equal(t, reported, sampled)

	parts := partsByName(sample)
	require.Equal(t, "17.5", *parts["postgres"].Version)
	require.Equal(t, int64(4), *parts["postgres"].Restarts)
	require.Equal(t, int64(2000), *parts["postgres"].CpuLimitMillicores)
	require.True(t, parts["release"].Ready)
	require.Nil(t, parts["release"].Replicas, "a Job has no copies that should run")
}

func TestBeforeTheLoopObservedTheReleaseTheSampleHasNoParts(t *testing.T) {
	t.Parallel()
	w := newWorld(t, "", standardObjects()...)

	sample := w.sampler.Sample(t.Context(), minute, nil)

	require.NotNil(t, sample.Parts)
	require.Empty(t, sample.Parts)
	require.NotNil(t, sample.Machine)
}

func TestAnInstanceRunningNoPodsIsSampledOnTheOperatorsNode(t *testing.T) {
	t.Parallel()
	w := newWorld(t, node, machineNode())
	w.data.set(tms.DataUsage{}, tms.ErrNoDataUsage)

	sample := w.sampler.Sample(t.Context(), minute, &health.CheckParams{Stopped: true, Hibernated: true})

	require.Equal(t, []string{kubeletAt}, w.summaries.asked)
	require.Equal(t, int64(4000), *sample.Machine.CpuCapacityMillicores)
	pg := partsByName(sample)["postgres"]
	require.Equal(t, int32(0), *pg.Replicas, "a hibernated database should run no copies")
	require.Nil(t, pg.CpuLimitMillicores)
	require.Nil(t, sample.Data)
}
