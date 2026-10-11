package compliancejobs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReminderStep(t *testing.T) {
	tests := []struct {
		daysLeft int64
		want     int64
	}{
		{daysLeft: 45, want: -1},
		{daysLeft: 31, want: -1},
		{daysLeft: 30, want: 30},
		{daysLeft: 29, want: -1},
		{daysLeft: 14, want: 14},
		{daysLeft: 8, want: -1},
		{daysLeft: 7, want: 7},
		{daysLeft: 1, want: -1},
		{daysLeft: 0, want: 0},
		{daysLeft: -3, want: 0},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, reminderStep(tt.daysLeft), "daysLeft=%d", tt.daysLeft)
	}
}
