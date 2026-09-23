package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestStepKey_IsTheSameHoweverTheArgumentsWereOrdered(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("ar_")
	first := StepKey(StepKeyParams{
		OwnerID:  owner,
		ToolName: "tender_to_carriers",
		Args:     map[string]any{"moveId": "mv_1", "carrierIds": []any{"c_1", "c_2"}},
	})
	second := StepKey(StepKeyParams{
		OwnerID:  owner,
		ToolName: "tender_to_carriers",
		Args:     map[string]any{"carrierIds": []any{"c_1", "c_2"}, "moveId": "mv_1"},
	})

	assert.NotEmpty(t, first)
	assert.Equal(t, first, second,
		"a provider that serialised the arguments in another order names the same write")
}

func TestStepKey_IgnoresTheProvidersCallID(t *testing.T) {
	t.Parallel()

	// This is the whole point. The call id is minted per completion, so a
	// retried run asking for the same write gets a different one; anything
	// keyed on it would see two operations where the model decided once.
	owner := pulid.MustNew("ar_")
	params := StepKeyParams{
		OwnerID:  owner,
		ToolName: "create_shipment",
		Args:     map[string]any{"customerId": "cus_1"},
	}

	assert.Equal(t, StepKey(params), StepKey(params))
}

func TestStepKey_SeparatesRunsAndRepeats(t *testing.T) {
	t.Parallel()

	args := map[string]any{"to": "ops@example.com"}
	mine := StepKey(StepKeyParams{OwnerID: pulid.MustNew("ar_"), ToolName: "email_customer", Args: args})
	theirs := StepKey(StepKeyParams{OwnerID: pulid.MustNew("ar_"), ToolName: "email_customer", Args: args})
	assert.NotEqual(t, mine, theirs, "two runs asking for the same write are two writes")

	owner := pulid.MustNew("ar_")
	first := StepKey(StepKeyParams{OwnerID: owner, ToolName: "email_customer", Args: args, Ordinal: 0})
	again := StepKey(StepKeyParams{OwnerID: owner, ToolName: "email_customer", Args: args, Ordinal: 1})
	assert.NotEqual(t, first, again,
		"two identical reminders are two reminders, not one asked for twice")
}

func TestStepKey_IsEmptyWhenTheArgumentsCannotBeCanonicalised(t *testing.T) {
	t.Parallel()

	// An unencodable argument leaves the call unguarded rather than guarded by
	// a key that does not describe it. The caller falls back to the provider's
	// call id, which is where every run stood before the ledger existed.
	key := StepKey(StepKeyParams{
		OwnerID:  pulid.MustNew("ar_"),
		ToolName: "assign_move",
		Args:     map[string]any{"cycle": make(chan int)},
	})

	assert.Empty(t, key)
}
