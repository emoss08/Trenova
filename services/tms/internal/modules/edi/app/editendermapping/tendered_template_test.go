package editendermapping

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestATenderedRateIsRatedByHand(t *testing.T) {
	mapped := pulid.MustNew("ft_")

	tendered := &editransfer.LoadTenderPayload{
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(1850)),
	}
	require.True(t, TenderedTemplate(tendered, mapped).IsNil(),
		"the receiver bills the tendered rate, so no tariff re-rates it")

	unpriced := &editransfer.LoadTenderPayload{}
	require.Equal(t, mapped, TenderedTemplate(unpriced, mapped),
		"a tender without a rate is priced by the mapped tariff")
}
