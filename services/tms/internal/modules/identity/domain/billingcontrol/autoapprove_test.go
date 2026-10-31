package billingcontrol

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestQueueAutoApprovesOnlyWhatTheModeAndCapAllow(t *testing.T) {
	t.Parallel()

	capped := decimal.NewNullDecimal(decimal.RequireFromString("5000"))
	tests := []struct {
		name    string
		mode    QueueAutoApproveMode
		cap     decimal.NullDecimal
		total   string
		agrees  bool
		approve bool
	}{
		{"off never approves", QueueAutoApproveModeOff, decimal.NullDecimal{}, "10", true, false},
		{
			"clear checks approve",
			QueueAutoApproveModeWhenChecksClear,
			decimal.NullDecimal{},
			"10",
			false,
			true,
		},
		{
			"the cap itself is allowed",
			QueueAutoApproveModeWhenChecksClear,
			capped,
			"5000",
			false,
			true,
		},
		{
			"above the cap waits",
			QueueAutoApproveModeWhenChecksClear,
			capped,
			"5000.01",
			false,
			false,
		},
		{
			"Nova's agreement is required",
			QueueAutoApproveModeWhenAgentAgrees,
			decimal.NullDecimal{},
			"10",
			false,
			false,
		},
		{
			"Nova agrees",
			QueueAutoApproveModeWhenAgentAgrees,
			decimal.NullDecimal{},
			"10",
			true,
			true,
		},
		{"an unset mode never approves", "", decimal.NullDecimal{}, "10", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bc := &BillingControl{QueueAutoApproveMode: tt.mode, QueueAutoApproveMaxAmount: tt.cap}
			assert.Equal(
				t,
				tt.approve,
				bc.AutoApproves(decimal.RequireFromString(tt.total), tt.agrees),
			)
		})
	}
}
