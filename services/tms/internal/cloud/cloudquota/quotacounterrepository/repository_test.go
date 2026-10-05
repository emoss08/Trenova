package quotacounterrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/cloud/catalog"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryCountedFreeDemoMeterHasACounter(t *testing.T) {
	t.Parallel()

	plan, err := platformplan.FreeDemo(nil)
	require.NoError(t, err)

	repo := &repository{}
	for meter, limit := range plan.Limits {
		if limit.Window == platformplan.WindowPerItem {
			assert.False(t, repo.Supports(meter), "%s is checked per item and is never counted", meter)
			continue
		}
		assert.True(t, repo.Supports(meter), "%s is limited by the free plan but has no counter", meter)
	}
}

func TestMetersAreSortedAndKnownToTheCatalog(t *testing.T) {
	t.Parallel()

	known := make(map[platformcatalog.MeterKey]struct{})
	for _, meter := range catalog.NewStaticProvider().Meters() {
		known[meter.Key] = struct{}{}
	}

	meters := Meters()
	assert.IsIncreasing(t, meters)
	for _, meter := range meters {
		assert.Contains(t, known, meter, "%s is counted but missing from the platform catalog", meter)
	}
}
