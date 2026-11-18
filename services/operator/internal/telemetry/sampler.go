package telemetry

import (
	"context"
	"log/slog"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const (
	readTimeout    = 5 * time.Second
	summaryTimeout = 10 * time.Second
	maxParts       = 50
	maxReplicas    = 1000
	maxVersion     = 128
	labelComponent = "app.kubernetes.io/component"
	componentLogs  = "loki"

	defaultKubeletPort = 10250
)

var partName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

type Parts interface {
	Parts(ctx context.Context, p health.CheckParams) ([]health.Part, error)
}

type Sampler struct {
	kube      kubernetes.Interface
	metrics   metrics.Interface
	parts     Parts
	summaries Summaries
	trenova   *trenovaData
	namespace string
	ownNode   string
	logger    *slog.Logger
}

type SamplerParams struct {
	Kube      kubernetes.Interface
	Metrics   metrics.Interface
	Parts     Parts
	Summaries Summaries
	Data      DataSource
	Namespace string
	OwnNode   string
	Logger    *slog.Logger
}

func NewSampler(p *SamplerParams) *Sampler {
	return &Sampler{
		kube:      p.Kube,
		metrics:   p.Metrics,
		parts:     p.Parts,
		summaries: p.Summaries,
		trenova:   &trenovaData{source: p.Data, logger: p.Logger},
		namespace: p.Namespace,
		ownNode:   p.OwnNode,
		logger:    p.Logger,
	}
}

type usage struct {
	cpu    int64
	memory int64
}

type reading struct {
	pods      []corev1.Pod
	podsRead  bool
	usage     map[string]usage
	usageRead bool
}

func (s *Sampler) Sample(
	ctx context.Context,
	at time.Time,
	params *health.CheckParams,
) instanceapi.TelemetrySample {
	sample := instanceapi.TelemetrySample{At: at, Parts: []instanceapi.PartUsage{}}
	r := s.read(ctx)
	if params != nil {
		sample.Parts = s.sampleParts(ctx, *params, &r)
	}
	data := s.trenova.read(ctx, at)
	if node := s.node(&r); node != "" {
		sample.Machine = s.sampleMachine(ctx, node, &r, &data)
	} else {
		s.logger.Warn(
			"telemetry: the instance's node is not known; the sample leaves the machine out",
		)
	}
	if data != (instanceapi.DataUsage{}) {
		sample.Data = &data
	}
	return sample
}

func (s *Sampler) sampleMachine(
	ctx context.Context,
	node string,
	r *reading,
	data *instanceapi.DataUsage,
) *instanceapi.MachineUsage {
	machine, kubelet := s.machine(ctx, node)
	if summary, ok := s.summary(ctx, node, kubelet); ok {
		if fs := summary.Node.FS; fs != nil {
			machine.DiskBytes = byteCount(fs.UsedBytes, 0)
			machine.DiskCapacityBytes = byteCount(fs.CapacityBytes, 1)
		}
		if r.podsRead {
			data.LogsBytes = s.logs(&summary, r.pods)
		}
	}
	if machine == (instanceapi.MachineUsage{}) {
		return nil
	}
	return &machine
}

func (s *Sampler) read(ctx context.Context) reading {
	var r reading
	run, cancel := context.WithTimeout(ctx, readTimeout)
	pods, err := s.kube.CoreV1().Pods(s.namespace).List(run, metav1.ListOptions{})
	cancel()
	if err != nil {
		s.logger.Warn(
			"telemetry: cannot list the instance's pods; the sample leaves restarts, limits, usage and data out",
			"error",
			err,
		)
	} else {
		r.pods, r.podsRead = pods.Items, true
	}
	run, cancel = context.WithTimeout(ctx, readTimeout)
	used, err := s.metrics.MetricsV1beta1().
		PodMetricses(s.namespace).
		List(run, metav1.ListOptions{})
	cancel()
	if err != nil {
		s.logger.Warn(
			"telemetry: cannot read the pods' metrics; the sample leaves the parts' usage out",
			"error",
			err,
		)
		return r
	}
	r.usage, r.usageRead = make(map[string]usage, len(used.Items)), true
	for i := range used.Items {
		r.usage[used.Items[i].Name] = podUsage(&used.Items[i])
	}
	return r
}

func podUsage(m *metricsv1beta1.PodMetrics) usage {
	var u usage
	for _, c := range m.Containers {
		u.cpu += c.Usage.Cpu().MilliValue()
		u.memory += c.Usage.Memory().Value()
	}
	return u
}

func (s *Sampler) sampleParts(
	ctx context.Context,
	params health.CheckParams,
	r *reading,
) []instanceapi.PartUsage {
	run, cancel := context.WithTimeout(ctx, readTimeout)
	parts, err := s.parts.Parts(run, params)
	cancel()
	if err != nil {
		s.logger.Warn("telemetry: cannot read the instance's parts; the sample leaves them out",
			"error", err)
		return []instanceapi.PartUsage{}
	}
	out := make([]instanceapi.PartUsage, 0, min(len(parts), maxParts))
	for i := range parts {
		if len(out) == maxParts {
			break
		}
		if !partName.MatchString(parts[i].Name) {
			s.logger.Warn("telemetry: a part's name does not fit the contract; it is left out",
				"part", parts[i].Name)
			continue
		}
		out = append(out, partUsage(&parts[i], r))
	}
	return out
}

func partUsage(p *health.Part, r *reading) instanceapi.PartUsage {
	out := instanceapi.PartUsage{
		Name:          p.Name,
		Ready:         p.Ready,
		Replicas:      count(p.Replicas),
		ReadyReplicas: count(p.ReadyReplicas),
	}
	var pods []*corev1.Pod
	if r.podsRead && p.Pods != nil && !p.Pods.Empty() {
		for i := range r.pods {
			if p.Pods.Matches(labels.Set(r.pods[i].Labels)) {
				pods = append(pods, &r.pods[i])
			}
		}
	}
	image := p.Image
	if r.podsRead {
		restarts := int64(0)
		for _, pod := range pods {
			restarts += podRestarts(pod)
		}
		out.Restarts = &restarts
		out.StartedAt = newestStart(pods)
		out.CpuLimitMillicores, out.MemoryLimitBytes = limits(pods)
		if image == "" && len(pods) > 0 && len(pods[0].Spec.Containers) > 0 {
			image = pods[0].Spec.Containers[0].Image
		}
	}
	if r.podsRead && r.usageRead {
		out.CpuMillicores, out.MemoryBytes = partInUse(pods, r.usage)
	}
	if tag := imageTag(image); tag != "" {
		out.Version = &tag
	}
	return out
}

func count(n *int32) *int32 {
	if n == nil {
		return nil
	}
	v := min(max(*n, 0), maxReplicas)
	return &v
}

func podRestarts(p *corev1.Pod) int64 {
	var n int64
	for i := range p.Status.InitContainerStatuses {
		n += int64(p.Status.InitContainerStatuses[i].RestartCount)
	}
	for i := range p.Status.ContainerStatuses {
		n += int64(p.Status.ContainerStatuses[i].RestartCount)
	}
	return n
}

func newestStart(pods []*corev1.Pod) *time.Time {
	var newest *time.Time
	for _, p := range pods {
		if !running(p) {
			continue
		}
		for i := range p.Status.ContainerStatuses {
			r := p.Status.ContainerStatuses[i].State.Running
			if r == nil || r.StartedAt.IsZero() {
				continue
			}
			if at := r.StartedAt.UTC(); newest == nil || at.After(*newest) {
				newest = &at
			}
		}
	}
	return newest
}

func running(p *corev1.Pod) bool {
	return p.Status.Phase != corev1.PodSucceeded && p.Status.Phase != corev1.PodFailed &&
		p.DeletionTimestamp == nil
}

func limits(pods []*corev1.Pod) (cpu, memory *int64) {
	var cpuSum, memSum int64
	cpuSet, memSet, counted := true, true, false
	for _, p := range pods {
		if !running(p) {
			continue
		}
		counted = true
		for i := range p.Spec.Containers {
			c := &p.Spec.Containers[i]
			if q, ok := c.Resources.Limits[corev1.ResourceCPU]; ok && !q.IsZero() {
				cpuSum += q.MilliValue()
			} else {
				cpuSet = false
			}
			if q, ok := c.Resources.Limits[corev1.ResourceMemory]; ok && !q.IsZero() {
				memSum += q.Value()
			} else {
				memSet = false
			}
		}
	}
	if !counted {
		return nil, nil
	}
	if cpuSet && cpuSum > 0 {
		cpu = &cpuSum
	}
	if memSet && memSum > 0 {
		memory = &memSum
	}
	return cpu, memory
}

func partInUse(pods []*corev1.Pod, used map[string]usage) (cpu, memory *int64) {
	var sum usage
	for _, p := range pods {
		if !running(p) || p.Status.Phase != corev1.PodRunning {
			continue
		}
		u, ok := used[p.Name]
		if !ok {
			return nil, nil
		}
		sum.cpu += u.cpu
		sum.memory += u.memory
	}
	return &sum.cpu, &sum.memory
}

func imageTag(image string) string {
	if at := strings.IndexByte(image, '@'); at >= 0 {
		image = image[:at]
	}
	slash := strings.LastIndexByte(image, '/')
	colon := strings.LastIndexByte(image, ':')
	if colon <= slash || colon == len(image)-1 {
		return ""
	}
	tag := image[colon+1:]
	if len(tag) > maxVersion {
		return ""
	}
	return tag
}

func (s *Sampler) node(r *reading) string {
	seen := map[string]int{}
	best := ""
	for i := range r.pods {
		n := r.pods[i].Spec.NodeName
		if n == "" || !running(&r.pods[i]) {
			continue
		}
		seen[n]++
		if best == "" || seen[n] > seen[best] || (seen[n] == seen[best] && n < best) {
			best = n
		}
	}
	if best != "" {
		return best
	}
	return s.ownNode
}

func (s *Sampler) machine(
	ctx context.Context,
	node string,
) (usage instanceapi.MachineUsage, kubeletAddr string) {
	m := instanceapi.MachineUsage{}
	kubelet := ""
	run, cancel := context.WithTimeout(ctx, readTimeout)
	got, err := s.kube.CoreV1().Nodes().Get(run, node, metav1.GetOptions{})
	cancel()
	if err != nil {
		s.logger.Warn(
			"telemetry: cannot read the node; the sample leaves the machine's capacity and disk out",
			"node",
			node,
			"error",
			err,
		)
	} else {
		if q, ok := got.Status.Capacity[corev1.ResourceCPU]; ok && q.MilliValue() > 0 {
			v := q.MilliValue()
			m.CpuCapacityMillicores = &v
		}
		if q, ok := got.Status.Capacity[corev1.ResourceMemory]; ok && q.Value() > 0 {
			v := q.Value()
			m.MemoryCapacityBytes = &v
		}
		kubelet = kubeletAddress(got)
	}
	run, cancel = context.WithTimeout(ctx, readTimeout)
	used, err := s.metrics.MetricsV1beta1().NodeMetricses().Get(run, node, metav1.GetOptions{})
	cancel()
	if err != nil {
		s.logger.Warn(
			"telemetry: cannot read the node's metrics; the sample leaves the machine's usage out",
			"node",
			node,
			"error",
			err,
		)
	} else {
		cpu, mem := used.Usage.Cpu().MilliValue(), used.Usage.Memory().Value()
		m.CpuMillicores, m.MemoryBytes = &cpu, &mem
	}
	return m, kubelet
}

func kubeletAddress(n *corev1.Node) string {
	port := n.Status.DaemonEndpoints.KubeletEndpoint.Port
	if port <= 0 {
		port = defaultKubeletPort
	}
	for _, a := range n.Status.Addresses {
		if a.Type == corev1.NodeInternalIP && a.Address != "" {
			return net.JoinHostPort(a.Address, strconv.Itoa(int(port)))
		}
	}
	return ""
}

func (s *Sampler) summary(ctx context.Context, node, kubelet string) (Summary, bool) {
	if kubelet == "" {
		s.logger.Warn(
			"telemetry: the node's kubelet address is not known; the sample leaves disk out",
			"node",
			node,
		)
		return Summary{}, false
	}
	run, cancel := context.WithTimeout(ctx, summaryTimeout)
	defer cancel()
	summary, err := s.summaries.Summary(run, kubelet)
	if err != nil {
		s.logger.Warn(
			"telemetry: cannot read the kubelet's stats summary; the sample leaves disk out",
			"node",
			node,
			"error",
			err,
		)
		return Summary{}, false
	}
	return summary, true
}

func (s *Sampler) logs(summary *Summary, pods []corev1.Pod) *int64 {
	component := make(map[string]string, len(pods))
	for i := range pods {
		component[pods[i].Name] = pods[i].Labels[labelComponent]
	}
	var sum int64
	found := false
	seen := map[string]bool{}
	for _, p := range summary.Pods {
		if p.PodRef.Namespace != s.namespace || component[p.PodRef.Name] != componentLogs {
			continue
		}
		for _, v := range p.Volumes {
			if v.PVCRef == nil || v.UsedBytes == nil || seen[v.PVCRef.Name] {
				continue
			}
			if sharesNodeFilesystem(summary.Node.FS, v) {
				continue
			}
			seen[v.PVCRef.Name] = true
			sum += clampBytes(*v.UsedBytes)
			found = true
		}
	}
	if !found {
		return nil
	}
	return &sum
}

func sharesNodeFilesystem(node *FSStats, v SummaryVolume) bool {
	return node != nil && node.CapacityBytes != nil && v.CapacityBytes != nil &&
		*v.CapacityBytes == *node.CapacityBytes
}

func byteCount(n *uint64, least int64) *int64 {
	if n == nil {
		return nil
	}
	v := clampBytes(*n)
	if v < least {
		return nil
	}
	return &v
}

func clampBytes(n uint64) int64 {
	const most = uint64(1<<63 - 1)
	return int64(min(n, most))
}
