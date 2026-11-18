package values

import "github.com/emoss08/trenova/operator/internal/instanceapi"

type Capacity struct {
	VCPU      int64
	MemoryGiB int64
	DiskGiB   int64
}

func (c Capacity) Set() bool {
	return c.VCPU > 0 || c.MemoryGiB > 0 || c.DiskGiB > 0
}

func (c Capacity) Fit(base *instanceapi.SpecPlan) instanceapi.SpecPlan {
	plan := *base
	plan.Vcpu = smaller(plan.Vcpu, c.VCPU)
	plan.MemoryGiB = smaller(plan.MemoryGiB, c.MemoryGiB)
	plan.DiskGiB = smaller(plan.DiskGiB, c.DiskGiB)
	return plan
}

func smaller(plan, capacity int64) int64 {
	if capacity > 0 && capacity < plan {
		return capacity
	}
	return plan
}
