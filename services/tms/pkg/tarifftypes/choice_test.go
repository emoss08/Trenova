package tarifftypes_test

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const choices = `[{"label":"Linehaul","kind":"best","range":{"from":0,"to":90},"chosen":"Flat",
  "options":[
    {"label":"Per mile","range":{"from":10,"to":40},"status":"lost",
     "subtotal":{"$q":"1175","u":"$"},"refusal":null},
    {"label":"Flat","range":{"from":41,"to":60},"status":"chosen",
     "subtotal":{"$q":"900","u":"$"},"refusal":null},
    {"label":"Volume","range":{"from":61,"to":88},"status":"refused","subtotal":null,
     "refusal":{"message":"Volume rates start at 6,000 lb","range":{"from":62,"to":80}}}]},
 {"label":"Floor","kind":"worst","range":null,"chosen":null,
  "options":[
    {"label":"A","range":null,"status":"refused","subtotal":null,
     "refusal":{"message":{"message":"The input {0} has no value and no default","args":["rate"]},"range":null}}]}]`

func decodeChoices(t *testing.T, text string) []rules.TariffChoice {
	t.Helper()
	var wire []rules.TariffChoice
	require.NoError(t, sonic.Unmarshal([]byte(text), &wire))
	return wire
}

func TestChoicesFromReadsEveryOption(t *testing.T) {
	t.Parallel()

	out, err := tarifftypes.ChoicesFrom(&tarifftypes.ChoicesRequest{
		Choices:  decodeChoices(t, choices),
		Currency: "USD",
		Locale:   i18n.Default,
	})
	require.NoError(t, err)
	require.Len(t, out, 2)

	linehaul := out[0]
	assert.Equal(t, "Linehaul", linehaul.Label)
	assert.Equal(t, tarifftypes.ChoiceBest, linehaul.Kind)
	assert.Equal(t, "USD", linehaul.Currency)
	require.NotNil(t, linehaul.Chosen)
	assert.Equal(t, "Flat", *linehaul.Chosen)
	require.Len(t, linehaul.Options, 3)
	assert.Equal(t, tarifftypes.OptionLost, linehaul.Options[0].Status)
	require.NotNil(t, linehaul.Options[0].Subtotal)
	assert.Equal(t, "1175", linehaul.Options[0].Subtotal.String())
	assert.Equal(t, tarifftypes.OptionChosen, linehaul.Options[1].Status)
	assert.Equal(t, "900", linehaul.Options[1].Subtotal.String())
	assert.Empty(t, linehaul.Options[1].Refusal)
	assert.Equal(t, tarifftypes.OptionRefused, linehaul.Options[2].Status)
	assert.Nil(t, linehaul.Options[2].Subtotal)
	assert.Equal(t, "Volume rates start at 6,000 lb", linehaul.Options[2].Refusal)

	floor := out[1]
	assert.Equal(t, tarifftypes.ChoiceWorst, floor.Kind)
	assert.Nil(t, floor.Chosen)
	assert.Equal(t, "The input rate has no value and no default", floor.Options[0].Refusal)
}

func TestChoicesFromTranslatesAModuleRefusal(t *testing.T) {
	t.Parallel()

	out, err := tarifftypes.ChoicesFrom(&tarifftypes.ChoicesRequest{
		Choices:  decodeChoices(t, choices),
		Currency: "USD",
		Locale:   i18n.ES,
	})
	require.NoError(t, err)
	assert.Equal(t,
		"La entrada rate no tiene valor ni valor predeterminado",
		out[1].Options[0].Refusal,
	)
}

func TestChoicesFromAnswersEmptyForNoChoices(t *testing.T) {
	t.Parallel()

	out, err := tarifftypes.ChoicesFrom(&tarifftypes.ChoicesRequest{Currency: "USD"})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Empty(t, out)
}

func TestChoicesFromRefusesAnUnknownShape(t *testing.T) {
	t.Parallel()

	for name, text := range map[string]string{
		"kind":     `[{"label":"A","kind":"cheapest","chosen":null,"options":[]}]`,
		"status":   `[{"label":"A","kind":"best","chosen":null,"options":[{"label":"x","status":"won","subtotal":null}]}]`,
		"subtotal": `[{"label":"A","kind":"best","chosen":"x","options":[{"label":"x","status":"chosen","subtotal":{"$q":"3","u":"lb"}}]}]`,
	} {
		_, err := tarifftypes.ChoicesFrom(&tarifftypes.ChoicesRequest{
			Choices:  decodeChoices(t, text),
			Currency: "USD",
		})
		assert.ErrorIs(t, err, tarifftypes.ErrBillShape, name)
	}
}
