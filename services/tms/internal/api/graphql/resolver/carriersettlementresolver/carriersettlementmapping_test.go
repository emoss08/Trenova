package carriersettlementresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/settlementcontrol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCarrierSettlementControlFromInputTakesHoldUntilInvoiceMatchedAsSent(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	for _, hold := range []bool{true, false} {
		input := &gqlmodel.UpdateCarrierSettlementControlInput{
			Version:                 3,
			PayTrigger:              settlementcontrol.PayTriggerShipmentDelivered,
			PayPeriodFrequency:      settlementcontrol.PayPeriodFrequencyWeekly,
			HoldUntilInvoiceMatched: hold,
		}

		got, err := carrierSettlementControlFromInput(input, tenant)
		require.NoError(t, err)
		assert.Equal(t, hold, got.HoldUntilInvoiceMatched)
	}
}

func TestCarrierSettlementControlFromInputKeepsSundayAndNoPayDelay(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	input := &gqlmodel.UpdateCarrierSettlementControlInput{
		Version:            3,
		PayTrigger:         settlementcontrol.PayTriggerShipmentDelivered,
		PayPeriodFrequency: settlementcontrol.PayPeriodFrequencyWeekly,
		PeriodEndDayOfWeek: 0,
		PayDelayDays:       0,
	}

	got, err := carrierSettlementControlFromInput(input, tenant)
	require.NoError(t, err)
	assert.Equal(t, 0, got.PeriodEndDayOfWeek)
	assert.Equal(t, 0, got.PayDelayDays)
	assert.Equal(t, tenant.OrgID, got.OrganizationID)
	assert.Equal(t, tenant.BuID, got.BusinessUnitID)
	assert.Equal(t, int64(3), got.Version)
}
