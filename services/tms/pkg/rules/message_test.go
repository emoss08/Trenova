package rules

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage_DecodesTextCountAndNestedArguments(t *testing.T) {
	t.Parallel()

	var message Message
	err := sonic.Unmarshal([]byte(`{"message": "{0} and {1}, {2}", "args": [
		"text", 3, {"message": "inner {0}", "args": [{"message": "deepest"}]}]}`), &message)
	require.NoError(t, err)

	assert.Equal(t, "{0} and {1}, {2}", message.Template)
	require.Len(t, message.Args, 3)
	assert.Equal(t, "text", message.Args[0])
	assert.Equal(t, int64(3), message.Args[1])
	assert.Equal(t, Message{
		Template: "inner {0}",
		Args:     []any{Message{Template: "deepest", Args: []any{}}},
	}, message.Args[2])
	assert.Equal(t, "text and 3, inner deepest", message.String())
}

func TestMessage_RefusesAnArgumentThatIsNoneOfTheThree(t *testing.T) {
	t.Parallel()

	var message Message
	err := sonic.Unmarshal([]byte(`{"message": "{0}", "args": [1.5]}`), &message)
	require.ErrorIs(t, err, errMessageArg)
}

func TestMessage_TranslatesNestedArgumentsInTheReadersLocale(t *testing.T) {
	t.Parallel()

	message := Message{
		Template: "Row {0}: {1}",
		Args:     []any{int64(2), Message{Template: "Bill-to customer is required"}},
	}

	assert.Equal(t, "Row 2: Bill-to customer is required", message.String())
	assert.Equal(t, "Row 2: El cliente facturable es obligatorio", message.Localize(i18n.ES))

	ctx := i18n.WithLocale(t.Context(), i18n.ES)
	assert.Equal(
		t,
		"Row 2: El cliente facturable es obligatorio",
		errortypes.LocalizedText(ctx, message.ValidationError("rows", errortypes.ErrInvalid)),
	)
}
