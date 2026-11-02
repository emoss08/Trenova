package listquery

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
)

func TestWindow_AppliesOnlyToAFilterOnItsFields(t *testing.T) {
	t.Parallel()

	w := &Window{Fields: []string{"shipment.actualDeliveryDate"}, Cap: 10}
	on := domaintypes.FieldFilter{Field: "shipment.actualDeliveryDate"}
	other := domaintypes.FieldFilter{Field: "status"}

	assert.True(t, w.appliesTo(&pagination.QueryOptions{FieldFilters: []domaintypes.FieldFilter{other, on}}))
	assert.True(t, w.appliesTo(&pagination.QueryOptions{
		FilterGroups: []domaintypes.FilterGroup{{Filters: []domaintypes.FieldFilter{on}}},
	}), "a window inside a filter group")
	assert.False(t, w.appliesTo(&pagination.QueryOptions{FieldFilters: []domaintypes.FieldFilter{other}}))
	assert.False(t, w.appliesTo(nil))
	assert.False(t, (*Window)(nil).appliesTo(&pagination.QueryOptions{
		FieldFilters: []domaintypes.FieldFilter{on},
	}), "no window configured")
	assert.False(t, (&Window{Fields: w.Fields}).appliesTo(&pagination.QueryOptions{
		FieldFilters: []domaintypes.FieldFilter{on},
	}), "a window without a cap reads as it always did")
}
