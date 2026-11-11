package values

import (
	"strconv"
	"strings"
	"testing"
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

func TestEverySizedPlanFitsItsNode(t *testing.T) {
	t.Parallel()
	plans := []sizeInput{
		{vcpu: 2, memoryGiB: 8, diskGiB: 40, minio: true},                                // the minimum
		{vcpu: 4, memoryGiB: 16, diskGiB: 50, docling: true, clamav: true, minio: true},  // Sandbox
		{vcpu: 8, memoryGiB: 32, diskGiB: 200, docling: true, clamav: true, minio: true}, // Production
		{vcpu: 16, memoryGiB: 64, diskGiB: 500, docling: true, clamav: true, minio: true},
	}
	for _, p := range plans {
		out, err := size(p)
		if err != nil {
			t.Fatalf("%+v: %v", p, err)
		}
		// An unsized release Job would request the chart's default, which
		// nothing here counts.
		if _, ok := out[keyTMS].(map[string]any)["release"]; !ok {
			t.Fatalf("%d vCPU: the release Job is not sized", p.vcpu)
		}
		cpu, mem := requests(t, out)
		if got, have := cpu+outsideMillicore, p.vcpu*1000; got > have {
			t.Errorf("%d vCPU: pods request %dm of %dm; the last one to arrive cannot be scheduled", p.vcpu, got, have)
		}
		if got, have := mem+outsideMiB, p.memoryGiB*1024-evictionMiB; got > have {
			t.Errorf("%d GiB: pods request %dMi of %dMi allocatable", p.memoryGiB, got, have)
		}
		if cpu+outsideMillicore > p.vcpu*1000-headroomMillicore/2 {
			t.Errorf("%d vCPU: less than half the CPU headroom is left (%dm requested)", p.vcpu, cpu+outsideMillicore)
		}
	}
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
			cpu += quantity(t, r["cpu"], "m", 1)
			mem += quantity(t, r[keyMemory], "Mi", 1)
			continue
		}
		c, mm := requests(t, x)
		cpu, mem = cpu+c, mem+mm
	}
	return cpu, mem
}

func quantity(t *testing.T, v any, unit string, _ int64) int64 {
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
