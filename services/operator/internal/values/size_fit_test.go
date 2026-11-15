package values

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"

	"github.com/emoss08/trenova/operator/internal/controlplanetest"
)

// Requests outside the chart's workloads, as the node runs them: k3s's
// CoreDNS and metrics-server (100m and 70Mi each), the operator (50m, 96Mi)
// and cloudflared (10m, 32Mi). The release Job is sized, so it is counted
// with the chart.
const (
	outsideMillicore = 100 + 100 + 50 + 10
	outsideMiB       = 70 + 70 + 96 + 32
	// kubelet's default hard eviction threshold keeps this much memory out of
	// a node's allocatable.
	evictionMiB = 100
)

var sizedChart = filepath.Join("..", "..", "..", "..", "deploy", "helm", "trenova")

var localCap = node{vcpu: 2, memoryGiB: 10, diskGiB: 40}

type node struct {
	vcpu      int64
	memoryGiB int64
	diskGiB   int64
	clamav    bool
	storage   bool
}

func (n node) String() string {
	return fmt.Sprintf(
		"%d vCPU, %d GiB, ClamAV %t, storage of ours %t",
		n.vcpu, n.memoryGiB, n.clamav, n.storage,
	)
}

func nodes() []node {
	var out []node
	for _, n := range []node{
		{vcpu: minVCPU, memoryGiB: minMemoryGiB, diskGiB: minDiskGiB},
		{vcpu: minVCPU, memoryGiB: minMemoryGiBScan, diskGiB: minDiskGiB, clamav: true},
		localCap,
		{vcpu: 4, memoryGiB: 16, diskGiB: 50},
		{vcpu: 4, memoryGiB: 16, diskGiB: 50, clamav: true},
		{vcpu: 8, memoryGiB: 32, diskGiB: 200},
		{vcpu: 8, memoryGiB: 32, diskGiB: 200, clamav: true},
		{vcpu: 16, memoryGiB: 64, diskGiB: 500},
		{vcpu: 16, memoryGiB: 64, diskGiB: 500, clamav: true},
	} {
		withStorage := n
		withStorage.storage = true
		out = append(out, n, withStorage)
	}
	return out
}

type pod struct {
	component string
	cpu       int64
	mem       int64
	count     int64
	surge     int64
}

func TestEveryMachineFitsItsWorkloadsAndARollout(t *testing.T) {
	t.Parallel()
	var surging []string
	for _, s := range shares(true) {
		if s.surge {
			surging = append(surging, s.path[len(s.path)-1])
		}
	}
	slices.Sort(surging)
	for _, n := range nodes() {
		t.Run(n.String(), func(t *testing.T) {
			t.Parallel()
			var cpu, mem, surgeCPU, surgeMem int64
			var rolled []string
			for _, p := range rendered(t, n) {
				cpu += p.cpu * p.count
				mem += p.mem * p.count
				surgeCPU = max(surgeCPU, p.cpu*p.surge)
				surgeMem = max(surgeMem, p.mem*p.surge)
				if p.surge > 0 {
					rolled = append(rolled, p.component)
				}
			}
			slices.Sort(rolled)
			require.Equal(t, surging, rolled, "the workloads that start a new pod beside the old one")

			if got, have := mem+surgeMem+outsideMiB, n.memoryGiB*1024-evictionMiB; got > have {
				t.Errorf("pods request %dMi and the largest new pod of a rollout %dMi more, of %dMi allocatable", mem+outsideMiB, surgeMem, have)
			}
			if got, have := cpu+surgeCPU+outsideMillicore, n.vcpu*1000; got > have {
				t.Errorf("pods request %dm and the largest new pod of a rollout %dm more, of %dm", cpu+outsideMillicore, surgeCPU, have)
			}
			if got := cpu + surgeCPU + outsideMillicore; got > n.vcpu*1000-headroomMillicore/2 {
				t.Errorf("less than half the CPU headroom is left during a rollout (%dm requested)", got)
			}
		})
	}
}

func rendered(t *testing.T, n node) []pod {
	t.Helper()
	spec := controlplanetest.Spec(uuid.New())
	spec.Plan.Vcpu, spec.Plan.MemoryGiB, spec.Plan.DiskGiB = n.vcpu, n.memoryGiB, n.diskGiB
	spec.Components = []string{keyDocling}
	var storage Storage
	if n.storage {
		storage = Storage{
			Endpoint:        "https://s3.bhs.io.cloud.ovh.net",
			Region:          "bhs",
			DocumentsBucket: "trenova-docs",
			BackupsPath:     "s3://trenova-backups/pitr/",
		}
	}
	r, err := Render(&spec, map[string]any{
		keyClamAV: map[string]any{"enabled": n.clamav},
		"ingress": map[string]any{"enabled": true, "className": "traefik"},
	}, "trenova-system", storage)
	require.NoError(t, err)
	ch, err := loader.Load(sizedChart)
	require.NoError(t, err)
	vals, err := chartutil.ToRenderValues(ch, r.Values, chartutil.ReleaseOptions{
		Name: "trenova", Namespace: "trenova", Revision: 1, IsInstall: true,
	}, chartutil.DefaultCapabilities)
	require.NoError(t, err)
	out, err := engine.Render(ch, vals)
	require.NoError(t, err)

	var pods []pod
	for name, body := range out {
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		for _, doc := range strings.Split(body, "\n---") {
			var m map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(doc), &m), name)
			if p, ok := workload(t, m); ok {
				pods = append(pods, p)
			}
		}
	}
	require.NotEmpty(t, pods)
	return pods
}

func workload(t *testing.T, m map[string]any) (pod, bool) {
	t.Helper()
	spec, _ := m["spec"].(map[string]any)
	labels, _ := field(m, "metadata", "labels").(map[string]any)
	component, _ := labels["app.kubernetes.io/component"].(string)
	p := pod{component: component, count: 1}
	switch m["kind"] {
	case "Cluster":
		requests, _ := field(spec, "resources", "requests").(map[string]any)
		require.NotNil(t, requests, "PostgreSQL has no requests")
		p.cpu, p.mem = cpuOf(t, requests), memOf(t, requests)
		p.count = int64(number(spec["instances"]))
		return p, true
	case "Job":
	case "Deployment", "StatefulSet":
		p.count = int64(number(spec["replicas"]))
		if m["kind"] == "Deployment" {
			p.surge = surge(spec, p.count)
		}
	default:
		return pod{}, false
	}
	podSpec, _ := field(spec, "template", "spec").(map[string]any)
	containers, _ := podSpec["containers"].([]any)
	for _, c := range containers {
		requests, _ := field(c, "resources", "requests").(map[string]any)
		require.NotNil(t, requests, "%s: a container has no requests", component)
		p.cpu += cpuOf(t, requests)
		p.mem += memOf(t, requests)
	}
	inits, _ := podSpec["initContainers"].([]any)
	for _, c := range inits {
		requests, _ := field(c, "resources", "requests").(map[string]any)
		p.cpu = max(p.cpu, cpuOf(t, requests))
		p.mem = max(p.mem, memOf(t, requests))
	}
	return p, true
}

func surge(spec map[string]any, replicas int64) int64 {
	if field(spec, "strategy", "type") == "Recreate" || replicas == 0 {
		return 0
	}
	switch v := field(spec, "strategy", "rollingUpdate", "maxSurge").(type) {
	case float64:
		return int64(v)
	case string:
		percent, _ := strconv.ParseFloat(strings.TrimSuffix(v, "%"), 64)
		return int64(math.Ceil(float64(replicas) * percent / 100))
	default:
		return int64(math.Ceil(float64(replicas) * 0.25))
	}
}

func field(v any, path ...string) any {
	for _, k := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

func number(v any) float64 {
	f, _ := v.(float64)
	return f
}

func cpuOf(t *testing.T, requests map[string]any) int64 {
	t.Helper()
	if requests["cpu"] == nil {
		return 0
	}
	q, err := resource.ParseQuantity(fmt.Sprint(requests["cpu"]))
	require.NoError(t, err)
	return q.MilliValue()
}

func memOf(t *testing.T, requests map[string]any) int64 {
	t.Helper()
	if requests[keyMemory] == nil {
		return 0
	}
	q, err := resource.ParseQuantity(fmt.Sprint(requests[keyMemory]))
	require.NoError(t, err)
	return (q.Value() + 1<<20 - 1) >> 20
}

// requests adds up every resources.requests the sizing set, in millicores and MiB.
func requests(t *testing.T, v any) (cpu, mem int64) {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		return 0, 0
	}
	for k, x := range m {
		if k == "requests" {
			r, _ := x.(map[string]any)
			cpu += quantity(t, r["cpu"], "m")
			mem += quantity(t, r[keyMemory], "Mi")
			continue
		}
		c, mm := requests(t, x)
		cpu, mem = cpu+c, mem+mm
	}
	return cpu, mem
}

func quantity(t *testing.T, v any, unit string) int64 {
	t.Helper()
	s, _ := v.(string)
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimSuffix(s, unit), 10, 64)
	if err != nil || !strings.HasSuffix(s, unit) {
		t.Fatalf("unexpected quantity %q, want …%s", s, unit)
	}
	return n
}

func TestEveryComponentHasAShareAndTheReleaseJobIsSized(t *testing.T) {
	t.Parallel()
	out, err := size(sizeInput{vcpu: localCap.vcpu, memoryGiB: localCap.memoryGiB, diskGiB: localCap.diskGiB})
	require.NoError(t, err)
	_, ok := out[keyTMS].(map[string]any)["release"]
	require.True(t, ok, "an unsized release Job would request the chart's default, which nothing counts")
	for _, component := range []string{keyDocling, keyMinio} {
		c, _ := out[component].(map[string]any)
		require.Contains(t, c, "resources", "%s has no share; turning it on would divide the machine again", component)
	}
}

func TestAPlanTooSmallForEveryComponentIsRefused(t *testing.T) {
	t.Parallel()
	for _, p := range []sizeInput{
		{vcpu: 2, memoryGiB: minMemoryGiB - 1, diskGiB: 40},
		{vcpu: 2, memoryGiB: minMemoryGiBScan - 1, diskGiB: 40, clamav: true},
		{vcpu: 1, memoryGiB: minMemoryGiB, diskGiB: 40},
		{vcpu: 2, memoryGiB: minMemoryGiB, diskGiB: 39},
	} {
		if _, err := size(p); err == nil {
			t.Errorf("%+v: sized, but every component it could run does not fit", p)
		}
	}
	for _, p := range []sizeInput{
		{vcpu: 2, memoryGiB: minMemoryGiB - 1, diskGiB: 40},
		{vcpu: 2, memoryGiB: minMemoryGiBScan - 1, diskGiB: 40, clamav: true},
	} {
		out := divide(p)
		_, mem := requests(t, out)
		var surgeMem int64
		for _, s := range shares(p.clamav) {
			if !s.surge {
				continue
			}
			c := out
			for _, k := range s.path {
				c, _ = c[k].(map[string]any)
			}
			_, m := requests(t, c)
			surgeMem = max(surgeMem, m)
		}
		if mem+surgeMem+outsideMiB <= p.memoryGiB*1024-evictionMiB {
			t.Errorf("%+v: everything and a rollout fit one GiB below the minimum; the minimum is too high", p)
		}
	}
}
