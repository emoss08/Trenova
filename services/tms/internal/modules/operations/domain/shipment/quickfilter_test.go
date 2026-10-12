package shipment

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestQuickFilterSpecValidate(t *testing.T) {
	t.Parallel()

	end := 60
	early := 30
	hour := 24
	tests := []struct {
		name  string
		spec  QuickFilterSpec
		valid bool
	}{
		{name: "plain filter", spec: Quick(QuickFilterLate), valid: true},
		{name: "unknown filter", spec: Quick(QuickFilter("Nope")), valid: false},
		{
			name: "plain filter with params",
			spec: QuickFilterSpec{Filter: QuickFilterLate, Hour: &end},
		},
		{name: "delivery hour", spec: DeliveryHourFilter(9), valid: true},
		{name: "delivery hour missing", spec: QuickFilterSpec{Filter: QuickFilterDeliveryHour}},
		{
			name: "delivery hour out of range",
			spec: QuickFilterSpec{Filter: QuickFilterDeliveryHour, Hour: &hour},
		},
		{name: "pickup window", spec: PickupWindowFilter(0, &end), valid: true},
		{name: "open pickup window", spec: PickupWindowFilter(360, nil), valid: true},
		{
			name: "pickup window missing start",
			spec: QuickFilterSpec{Filter: QuickFilterPickupWindow},
		},
		{name: "pickup window negative", spec: PickupWindowFilter(-5, nil)},
		{name: "pickup window inverted", spec: PickupWindowFilter(90, &early)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			multiErr := errortypes.NewMultiError()
			tt.spec.Validate(multiErr)
			assert.Equal(t, !tt.valid, multiErr.HasErrors())
		})
	}
}

func TestValidateQuickFiltersCapsTheList(t *testing.T) {
	t.Parallel()

	specs := make([]QuickFilterSpec, MaxQuickFilters+1)
	for i := range specs {
		specs[i] = Quick(QuickFilterLate)
	}
	multiErr := errortypes.NewMultiError()
	ValidateQuickFilters(specs, multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestQuickFiltersNeed(t *testing.T) {
	t.Parallel()

	margin, detention := QuickFiltersNeed([]QuickFilterSpec{Quick(QuickFilterLowMargin)})
	assert.True(t, margin)
	assert.False(t, detention)

	margin, detention = QuickFiltersNeed([]QuickFilterSpec{Quick(QuickFilterDetention)})
	assert.False(t, margin)
	assert.True(t, detention)

	for _, filter := range CountableQuickFilters() {
		assert.True(t, filter.IsValid())
	}
}
