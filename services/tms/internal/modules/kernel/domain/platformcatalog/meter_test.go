package platformcatalog_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
)

func TestEveryMeterIsListedOnceWithAName(t *testing.T) {
	t.Parallel()

	seen := map[platformcatalog.MeterKey]bool{}
	for i, meter := range platformcatalog.Meters() {
		assert.False(t, seen[meter.Key], "%s is listed twice", meter.Key)
		seen[meter.Key] = true
		assert.NotEmpty(t, meter.Label, "%s has no name", meter.Key)
		found, position, ok := platformcatalog.LookupMeter(meter.Key)
		assert.True(t, ok)
		assert.Equal(t, i, position)
		assert.Equal(t, meter, found)
	}
	_, _, ok := platformcatalog.LookupMeter("teleport.jumps")
	assert.False(t, ok)
}

func TestLookupAddOnKnowsEverySoldAddOnAndNotTheBase(t *testing.T) {
	t.Parallel()

	for _, addOn := range platformcatalog.AddOns() {
		found, ok := platformcatalog.LookupAddOn(addOn.Key)
		assert.True(t, ok)
		assert.Equal(t, addOn, found)
	}
	_, ok := platformcatalog.LookupAddOn(platformcatalog.AddOnBase)
	assert.False(t, ok)
}
