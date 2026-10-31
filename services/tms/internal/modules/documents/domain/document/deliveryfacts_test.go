package document

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAReceiptBillsAChargeOnlyWithAnAmount(t *testing.T) {
	t.Parallel()

	charge := pulid.MustNew("acc_")
	assert.True(
		t,
		(&DeliveryFacts{ChargeID: charge, Amount: decimal.NewNullDecimal(decimal.NewFromInt(185))}).BillsCharge(),
	)
	assert.False(t, (&DeliveryFacts{ChargeID: charge}).BillsCharge())
	assert.False(
		t,
		(&DeliveryFacts{Amount: decimal.NewNullDecimal(decimal.NewFromInt(185))}).BillsCharge(),
	)
	assert.False(
		t,
		(&DeliveryFacts{ChargeID: charge, Amount: decimal.NewNullDecimal(decimal.Zero)}).BillsCharge(),
	)

	var none *DeliveryFacts
	assert.False(t, none.BillsCharge())
}

func TestDeliveryFactsRefuseWhatTheyCannotHold(t *testing.T) {
	t.Parallel()

	arrived, departed := int64(2000), int64(1000)
	facts := &DeliveryFacts{
		OSD:        OSDKind("Lost"),
		Seal:       strings.Repeat("x", maxDeliveryFactText+1),
		ArrivedAt:  &arrived,
		DepartedAt: &departed,
		ChargeID:   pulid.MustNew("acc_"),
	}
	multiErr := errortypes.NewMultiError()
	facts.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}
	assert.Equal(t, []string{
		"deliveryFacts.osd",
		"deliveryFacts.seal",
		"deliveryFacts.departedAt",
		"deliveryFacts.amount",
	}, fields)
}

func TestNormalizingDropsANoteWithNoExceptionAndALabelWithNoCharge(t *testing.T) {
	t.Parallel()

	facts := &DeliveryFacts{
		OSD:         OSDNone,
		OSDNote:     " 2 pallets short ",
		ChargeLabel: "Lumper",
		SignedBy:    " D. Ortiz ",
	}
	facts.Normalize()

	assert.Empty(t, facts.OSDNote)
	assert.Empty(t, facts.ChargeLabel)
	assert.Equal(t, "D. Ortiz", facts.SignedBy)
}
