package aianalytics_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/stretchr/testify/assert"
)

func TestPaletteSlotsAreDistinctAndDoNotDependOnOrder(t *testing.T) {
	t.Parallel()

	keys := []string{
		"llama-3.3-70b",
		"gpt-5.1-mini",
		"qwen-2.5-14b",
		"gpt-5.1",
		"claude-sonnet-5",
		"gemini-3-flash",
	}
	slots := aianalytics.AssignPaletteSlots(keys)

	seen := make(map[int]string, len(keys))
	for _, key := range keys {
		slot := slots[key]
		assert.GreaterOrEqual(t, slot, 0)
		assert.Less(t, slot, aianalytics.PaletteSlots)
		assert.NotContains(t, seen, slot, "%s shares a hue with %s", key, seen[slot])
		seen[slot] = key
	}

	reversed := []string{
		"gemini-3-flash",
		"claude-sonnet-5",
		"gpt-5.1",
		"qwen-2.5-14b",
		"gpt-5.1-mini",
		"llama-3.3-70b",
		"gpt-5.1",
	}
	assert.Equal(t, slots, aianalytics.AssignPaletteSlots(reversed))
}

func TestPaletteSlotOfAnUnknownKeyFallsBackToItsHash(t *testing.T) {
	t.Parallel()

	slot := aianalytics.PaletteSlot(map[string]int{}, "mistral-large-3")

	assert.Equal(t, slot, aianalytics.PaletteSlot(nil, "mistral-large-3"))
	assert.Less(t, slot, aianalytics.PaletteSlots)
}
