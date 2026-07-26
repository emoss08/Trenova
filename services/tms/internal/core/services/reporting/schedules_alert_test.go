package reporting

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/stretchr/testify/assert"
)

// Editing anything else on a schedule must not re-arm a suppressed alert, so
// equality has to compare the threshold value rather than its address.
func TestScheduleAlertsEqualComparesValuesNotPointers(t *testing.T) {
	first, second := 100_000.0, 100_000.0
	different := 50_000.0

	base := func(value *float64) *report.ScheduleAlert {
		return &report.ScheduleAlert{
			Operator: dbtype.OpLessThan, ColumnID: "c_revenue", Value: value,
		}
	}

	assert.True(t, scheduleAlertsEqual(base(&first), base(&second)),
		"same threshold through different pointers is the same condition")
	assert.False(t, scheduleAlertsEqual(base(&first), base(&different)))
	assert.False(t, scheduleAlertsEqual(base(&first), base(nil)))
	assert.True(t, scheduleAlertsEqual(base(nil), base(nil)))
	assert.True(t, scheduleAlertsEqual(nil, nil))
	assert.False(t, scheduleAlertsEqual(base(&first), nil))
}
