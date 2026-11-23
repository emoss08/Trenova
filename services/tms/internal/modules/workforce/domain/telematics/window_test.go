package telematics_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
)

func TestRequireProviderWindow(t *testing.T) {
	t.Parallel()
	const start = int64(1_800_000_000)
	month := int64(telematics.MaxProviderWindowDays * timeutils.SecondsPerDay)

	assert.NoError(t, telematics.RequireProviderWindow(start, start+timeutils.SecondsPerDay))
	assert.NoError(t, telematics.RequireProviderWindow(start, start+month))
	assert.Error(t, telematics.RequireProviderWindow(start, start+month+1))
	assert.Error(t, telematics.RequireProviderWindow(start, start))
	assert.Error(t, telematics.RequireProviderWindow(0, start))
}

func TestRequireProviderDays(t *testing.T) {
	t.Parallel()

	assert.NoError(t, telematics.RequireProviderDays("2026-11-17", "2026-11-23"))
	assert.NoError(t, telematics.RequireProviderDays("2026-11-01", "2026-12-01"))
	assert.Error(t, telematics.RequireProviderDays("2026-11-01", "2026-12-02"))
	assert.Error(t, telematics.RequireProviderDays("2026-11-23", "2026-11-17"))
	assert.Error(t, telematics.RequireProviderDays("11/17/2026", "2026-11-23"))
}
