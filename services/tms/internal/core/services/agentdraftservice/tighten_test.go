package agentdraftservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckTightenedKeepsAShorterVersionWithEveryPlaceholder(t *testing.T) {
	t.Parallel()

	result, err := checkTightened(
		"You help {{user.name}} at {{organization}}. You should always, in every case, look the shipment up first.",
		"Help {{user.name}} at {{organization}}. Look the shipment up first.",
	)

	require.NoError(t, err)
	assert.Equal(t, "Help {{user.name}} at {{organization}}. Look the shipment up first.", result.Instructions)
	assert.True(t, result.Changed)
}

func TestCheckTightenedSaysWhenNothingChanged(t *testing.T) {
	t.Parallel()

	result, err := checkTightened("Look the shipment up first.", "  Look the shipment up first.\n")

	require.NoError(t, err)
	assert.False(t, result.Changed)
}

func TestCheckTightenedRefusesALostPlaceholder(t *testing.T) {
	t.Parallel()

	_, err := checkTightened(
		"Greet {{user.name}} and mention {{today}}.",
		"Greet the person and mention {{today}}.",
	)

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "{{user.name}}")
}

func TestCheckTightenedRefusesAnInventedPlaceholder(t *testing.T) {
	t.Parallel()

	_, err := checkTightened("Greet the person.", "Greet {{user.name}}.")

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "{{user.name}}")
}

func TestCheckTightenedRefusesAnEmptyOrOverlongReply(t *testing.T) {
	t.Parallel()

	_, err := checkTightened("Look the shipment up first.", "   ")
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))

	_, err = checkTightened(
		"Look the shipment up first.",
		strings.Repeat("x", agentdefinition.MaxInstructionsRunes+1),
	)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestParseTightenReplyRefusesEmptyAndUnreadableText(t *testing.T) {
	t.Parallel()

	_, err := parseTightenReply("")
	require.Error(t, err)

	_, err = parseTightenReply("Here you go: shorter text")
	require.Error(t, err)

	text, err := parseTightenReply(`{"instructions":"Look it up first."}`)
	require.NoError(t, err)
	assert.Equal(t, "Look it up first.", text)
}
