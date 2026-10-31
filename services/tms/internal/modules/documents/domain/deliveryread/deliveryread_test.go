package deliveryread

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhatNovaReadsFromAReceiptBillsOnlyAChargeInTheCatalog(t *testing.T) {
	t.Parallel()

	lumper := Charge{ID: pulid.MustNew("acc_"), Label: "Lumper Fee"}
	charges := map[string]Charge{lumper.ID.String(): lumper}
	amount := " 185.004 "

	facts := Facts(&Answer{
		Amount:    &amount,
		Reference: "LMP-5521",
		ChargeID:  lumper.ID.String(),
		OSD:       "Sideways",
	}, charges, 1_800_000_000)

	require.NotNil(t, facts)
	assert.Equal(t, document.FactsSourceAgent, facts.Source)
	assert.Equal(t, "185", facts.Amount.Decimal.String())
	assert.Equal(t, lumper.ID, facts.ChargeID)
	assert.Equal(t, "Lumper Fee", facts.ChargeLabel)
	assert.Equal(t, int64(1_800_000_000), facts.UpdatedAt)
	assert.Empty(t, facts.OSD, "an exception Nova invents is dropped")
	assert.Equal(t, []string{"amount", "reference", "chargeId"}, facts.ReadFields)

	unknown := Facts(
		&Answer{Amount: &amount, ChargeID: pulid.MustNew("acc_").String()},
		charges,
		1_800_000_000,
	)
	require.NotNil(t, unknown)
	assert.True(t, unknown.ChargeID.IsNil(), "a charge outside the catalog is never billed")
}

func TestADocumentNovaReadNothingFromIsLeftAlone(t *testing.T) {
	t.Parallel()

	assert.Nil(t, Facts(&Answer{}, nil, 1))
}

func TestSameFactsIgnoreWhenTheyWereRecorded(t *testing.T) {
	t.Parallel()

	a := &document.DeliveryFacts{SignedBy: "D. Ortiz", UpdatedAt: 1}
	b := &document.DeliveryFacts{SignedBy: "D. Ortiz", UpdatedAt: 2}
	assert.True(t, Same(a, b))
	assert.False(t, Same(nil, b))
	assert.False(t, Same(a, &document.DeliveryFacts{SignedBy: "M. Lee"}))
}

func TestFactsAPersonRecordedAreKept(t *testing.T) {
	t.Parallel()

	assert.True(t, Kept(&document.DeliveryFacts{Source: document.FactsSourcePerson}))
	assert.False(t, Kept(&document.DeliveryFacts{Source: document.FactsSourceAgent}))
	assert.False(t, Kept(nil))
}
