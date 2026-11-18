package values_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

func TestCapacityFitsThePlanToASmallerMachine(t *testing.T) {
	t.Parallel()
	plan := instanceapi.SpecPlan{Key: "production_plus", Name: "Production Plus", Vcpu: 16, MemoryGiB: 64, DiskGiB: 500}

	require.Equal(t, plan, values.Capacity{}.Fit(&plan))
	require.False(t, values.Capacity{}.Set())

	local := values.Capacity{VCPU: 2, MemoryGiB: 10, DiskGiB: 40}
	fit := local.Fit(&plan)
	require.True(t, local.Set())
	require.Equal(t, int64(2), fit.Vcpu)
	require.Equal(t, int64(10), fit.MemoryGiB)
	require.Equal(t, int64(40), fit.DiskGiB)
	require.Equal(t, plan.Key, fit.Key)

	small := instanceapi.SpecPlan{Vcpu: 2, MemoryGiB: 8, DiskGiB: 40}
	require.Equal(t, small, values.Capacity{VCPU: 4, MemoryGiB: 16, DiskGiB: 100}.Fit(&small))

	s := spec(local.Fit(&plan), instanceapi.Active)
	b := base()
	b["clamav"] = map[string]any{"enabled": false}
	r, err := values.Render(&s, b, operatorNS, &values.Storage{})
	require.NoError(t, err)
	require.Equal(t, "14Gi", lookup(t, r.Values, []string{"postgres", "storage"})["size"])
}

func TestFittingAPlanLeavesTheCallersPlanAlone(t *testing.T) {
	t.Parallel()
	plan := instanceapi.SpecPlan{Key: "production_plus", Vcpu: 16, MemoryGiB: 64, DiskGiB: 500}
	want := plan

	fit := values.Capacity{VCPU: 2, MemoryGiB: 10, DiskGiB: 40}.Fit(&plan)

	require.Equal(t, int64(2), fit.Vcpu)
	require.Equal(t, want, plan)
}
