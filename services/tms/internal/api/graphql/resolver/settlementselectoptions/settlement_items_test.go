package settlementselectoptions

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPayCodeSelectOptionItem(t *testing.T) {
	t.Parallel()

	entity := &paycode.PayCode{
		ID:                    pulid.MustNew("payc_"),
		Code:                  "LH",
		Name:                  "Linehaul",
		Direction:             paycode.PayCodeDirectionEarning,
		Taxable:               true,
		CountsTowardGuarantee: true,
		CreatedAt:             1780415883,
	}

	item := payCodeSelectOptionItem(entity)

	assert.Equal(t, "LH", item.Option.Label)
	require.NotNil(t, item.Option.Description)
	assert.Equal(t, "Linehaul", *item.Option.Description)
	assert.Equal(t, "Linehaul", item.Option.Meta["name"])
	assert.Equal(t, string(paycode.PayCodeDirectionEarning), item.Option.Meta["direction"])
	assert.Equal(t, true, item.Option.Meta["taxable"])
	assert.Equal(t, int64(1780415883), item.Cursor.At)
}

func TestPayProfileSelectOptionItem(t *testing.T) {
	t.Parallel()

	item := payProfileSelectOptionItem(&driverpay.PayProfile{
		ID:             pulid.MustNew("dpp_"),
		Name:           "Owner Operator 2026",
		Description:    "Percentage of linehaul",
		Classification: driverpay.PayeeClassificationOwnerOperator,
		CurrencyCode:   "USD",
		CreatedAt:      1780415884,
	})

	assert.Equal(t, "Owner Operator 2026", item.Option.Label)
	assert.Equal(
		t,
		string(driverpay.PayeeClassificationOwnerOperator),
		item.Option.Meta["classification"],
	)
	assert.Equal(t, "USD", item.Option.Meta["currencyCode"])
}
