package values

import (
	"fmt"
	"math"
	"strconv"
)

const (
	reservedMiB       = 2048
	reservedMillicore = 500
	minMemoryGiB      = 8
	minMemoryGiBScan  = 10
	clamavRequestMiB  = 3072
	clamavLimitMiB    = 4096
	minVCPU           = 2
	minDiskGiB        = 40
	postgresDiskShare = 0.35
)

type sizeInput struct {
	vcpu      int64
	memoryGiB int64
	diskGiB   int64
	docling   bool
	clamav    bool
	minio     bool
}

type share struct {
	path     []string
	memory   float64
	cpu      float64
	floorMiB float64
	fixedMiB float64
	limitMiB float64
	on       bool
}

func size(in sizeInput) (map[string]any, error) {
	minMemory := int64(minMemoryGiB)
	if in.clamav {
		minMemory = minMemoryGiBScan
	}
	if in.vcpu < minVCPU || in.memoryGiB < minMemory || in.diskGiB < minDiskGiB {
		return nil, fmt.Errorf(
			"%w: a plan needs at least %d vCPU, %d GiB of memory and %d GiB of disk",
			ErrInvalidSpec, minVCPU, minMemory, minDiskGiB,
		)
	}
	shares := []share{
		{path: []string{keyPostgres}, memory: 28, cpu: 20, floorMiB: 1024, on: true},
		{path: []string{keyTMS, "api"}, memory: 12, cpu: 15, floorMiB: 512, on: true},
		{path: []string{keyTMS, "worker"}, memory: 12, cpu: 15, floorMiB: 512, on: true},
		{path: []string{"imaging"}, memory: 10, cpu: 15, floorMiB: 512, on: true},
		{path: []string{keyRedis}, memory: 5, cpu: 3, floorMiB: 256, on: true},
		{path: []string{"meilisearch"}, memory: 5, cpu: 5, floorMiB: 256, on: true},
		{path: []string{"temporal"}, memory: 5, cpu: 5, floorMiB: 384, on: true},
		{path: []string{"gotenberg"}, memory: 4, cpu: 4, floorMiB: 256, on: true},
		{path: []string{keyMinio}, memory: 3, cpu: 3, floorMiB: 256, on: in.minio},
		{path: []string{"gtc"}, memory: 2, cpu: 2, floorMiB: 128, on: true},
		{path: []string{"client"}, memory: 1, cpu: 1, floorMiB: 64, on: true},
		{
			path:     []string{"clamav"},
			cpu:      4,
			fixedMiB: clamavRequestMiB,
			limitMiB: clamavLimitMiB,
			on:       in.clamav,
		},
		{path: []string{"docling"}, memory: 15, cpu: 8, floorMiB: 2048, on: in.docling},
	}
	var memTotal, cpuTotal, memFixed float64
	for _, s := range shares {
		if s.on {
			memTotal += s.memory
			cpuTotal += s.cpu
			memFixed += s.fixedMiB
		}
	}
	memAlloc := float64(in.memoryGiB*1024-reservedMiB) - memFixed
	cpuAlloc := float64(in.vcpu*1000 - reservedMillicore)

	out := map[string]any{}
	var pgLimit float64
	for _, s := range shares {
		if !s.on {
			continue
		}
		request := math.Max(memAlloc*s.memory/memTotal, s.floorMiB)
		if s.fixedMiB > 0 {
			request = s.fixedMiB
		}
		limit := math.Max(request*1.5, s.limitMiB)
		limits := map[string]any{keyMemory: mib(limit)}
		switch s.path[0] {
		case keyPostgres:
			limit = request * 1.25
			limits[keyMemory] = mib(limit)
			pgLimit = limit
		case "imaging":
			cores := math.Max(1, math.Floor(float64(in.vcpu)/2))
			limit = math.Max(limit, cores*1024)
			limits = map[string]any{
				"cpu":     strconv.FormatInt(int64(cores), 10),
				keyMemory: mib(limit),
			}
		case keyRedis:
			set(out, []string{keyRedis, "maxMemory"}, fmt.Sprintf("%dmb", int64(limit*0.75)))
		}
		set(out, append(append([]string{}, s.path...), "resources"), map[string]any{
			"requests": map[string]any{
				"cpu":     millicores(cpuAlloc * s.cpu / cpuTotal),
				keyMemory: mib(request),
			},
			"limits": limits,
		})
	}

	pgDiskGiB := math.Max(10, math.Floor(float64(in.diskGiB)*postgresDiskShare))
	set(out, []string{keyPostgres, "storage", "size"}, fmt.Sprintf("%dGi", int64(pgDiskGiB)))
	set(out, []string{keyPostgres, "parameters"}, map[string]any{
		"shared_buffers":         mibBytes(pgLimit / 4),
		"effective_cache_size":   mibBytes(pgLimit * 3 / 4),
		"maintenance_work_mem":   mibBytes(math.Min(pgLimit/16, 1024)),
		"max_wal_size":           mibBytes(clamp(pgDiskGiB*1024/20, 1024, 8192)),
		"max_slot_wal_keep_size": mibBytes(clamp(pgDiskGiB*1024/8, 2048, 16384)),
	})
	return out, nil
}

func set(dst map[string]any, path []string, v any) {
	for _, k := range path[:len(path)-1] {
		next, ok := dst[k].(map[string]any)
		if !ok {
			next = map[string]any{}
			dst[k] = next
		}
		dst = next
	}
	dst[path[len(path)-1]] = v
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(math.Max(v, lo), hi)
}

func mibBytes(n float64) string {
	return fmt.Sprintf("%dMB", int64(math.Floor(n)))
}
