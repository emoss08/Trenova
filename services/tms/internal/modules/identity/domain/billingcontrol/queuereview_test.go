package billingcontrol

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyQueueReviewPlan_FallsBackToOffWithoutNovasReview(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mode   QueueAutoApproveMode
		onPlan bool
		want   QueueAutoApproveMode
	}{
		{
			"Nova agreeing, on the plan",
			QueueAutoApproveModeWhenAgentAgrees,
			true,
			QueueAutoApproveModeWhenAgentAgrees,
		},
		{
			"Nova agreeing, off the plan",
			QueueAutoApproveModeWhenAgentAgrees,
			false,
			QueueAutoApproveModeOff,
		},
		{
			"clear checks, off the plan",
			QueueAutoApproveModeWhenChecksClear,
			false,
			QueueAutoApproveModeWhenChecksClear,
		},
		{"off, off the plan", QueueAutoApproveModeOff, false, QueueAutoApproveModeOff},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bc := &BillingControl{QueueAutoApproveMode: tt.mode}
			bc.ApplyQueueReviewPlan(tt.onPlan)

			assert.Equal(t, tt.want, bc.QueueAutoApproveMode)
		})
	}
}
