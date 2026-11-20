package carriersettlementcontrol

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/settlementcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validControl() *CarrierSettlementControl {
	return &CarrierSettlementControl{
		PayTrigger:         settlementcontrol.PayTriggerShipmentDelivered,
		PayPeriodFrequency: settlementcontrol.PayPeriodFrequencyWeekly,
		PeriodEndDayOfWeek: 6,
		PayDelayDays:       5,
	}
}

func TestCarrierSettlementControl_ValidateAutoAccept(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		autoMatch  bool
		autoAccept bool
		wantErr    bool
	}{
		{"both off passes", false, false, false},
		{"auto match alone passes", true, false, false},
		{"both on passes", true, true, false},
		{"accept without auto match fails", false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := validControl()
			sc.AutoMatchInboundInvoices = tt.autoMatch
			sc.AutoAcceptWithinTolerance = tt.autoAccept

			multiErr := errortypes.NewMultiError()
			sc.Validate(multiErr)

			if !tt.wantErr {
				assert.False(t, multiErr.HasErrors())
				return
			}
			require.True(t, multiErr.HasErrors())
			require.Len(t, multiErr.Errors, 1)
			assert.Equal(t, "autoAcceptWithinTolerance", multiErr.Errors[0].Field())
			assert.Equal(t, errortypes.ErrInvalid, multiErr.Errors[0].Code)
		})
	}
}

func TestNewDefault(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	got := NewDefault(orgID, buID)

	assert.Equal(t, &CarrierSettlementControl{
		OrganizationID:          orgID,
		BusinessUnitID:          buID,
		PayTrigger:              settlementcontrol.PayTriggerShipmentDelivered,
		PayPeriodFrequency:      settlementcontrol.PayPeriodFrequencyWeekly,
		PeriodEndDayOfWeek:      6,
		PayDelayDays:            5,
		RequireSeparateApprover: true,
	}, got)
}

func TestCarrierSettlementControl_ValidateAcceptsSundayAndNoPayDelay(t *testing.T) {
	t.Parallel()

	sc := validControl()
	sc.PeriodEndDayOfWeek = 0
	sc.PayDelayDays = 0

	multiErr := errortypes.NewMultiError()
	sc.Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
	assert.Equal(t, 0, sc.PeriodEndDayOfWeek)
	assert.Equal(t, 0, sc.PayDelayDays)
}
