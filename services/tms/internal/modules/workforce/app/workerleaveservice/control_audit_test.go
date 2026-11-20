package workerleaveservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestControlAuditComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change func(*workerleave.LeaveControl)
		want   string
	}{
		{
			name:   "nothing changed",
			change: func(*workerleave.LeaveControl) {},
			want:   "Leave settings saved with no changes",
		},
		{
			name: "only the measurement method changed",
			change: func(c *workerleave.LeaveControl) {
				c.MeasurementMethod = workerleave.MeasureCalendarYear
			},
			want: "Leave is now measured by calendar year",
		},
		{
			name: "one setting changed and the method did not",
			change: func(c *workerleave.LeaveControl) {
				c.WorkweekHours = decimal.RequireFromString("37.5")
			},
			want: "Leave settings updated: the workweek hours changed",
		},
		{
			name: "several settings changed and the method did not",
			change: func(c *workerleave.LeaveControl) {
				c.EntitlementWeeks = decimal.NewFromInt(16)
				c.EligibilityMonths = 6
				c.CertificationDueDays = 20
			},
			want: "Leave settings updated: the entitlement weeks, the months of service to " +
				"qualify and the days to return a certification changed",
		},
		{
			name: "the method and a setting changed",
			change: func(c *workerleave.LeaveControl) {
				c.MeasurementMethod = workerleave.MeasureHireAnniversary
				c.MilitaryCaregiverWeeks = decimal.NewFromInt(30)
				c.EligibilityHours = 1000
			},
			want: "Leave is now measured by twelve months from the hire anniversary, and " +
				"the military caregiver weeks and the hours worked to qualify changed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			previous := workerleave.DefaultLeaveControl()
			current := workerleave.DefaultLeaveControl()
			tt.change(current)

			assert.Equal(t, tt.want, controlAuditComment(previous, current))
		})
	}
}
