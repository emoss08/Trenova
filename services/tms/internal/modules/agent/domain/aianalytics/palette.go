package aianalytics

import (
	"cmp"
	"hash/fnv"
	"slices"
)

const PaletteSlots = 12

func paletteHash(key string) uint32 {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(key))

	return hasher.Sum32()
}

func AssignPaletteSlots(keys []string) map[string]int {
	ordered := slices.Clone(keys)
	slices.SortFunc(ordered, func(left, right string) int {
		return cmp.Or(cmp.Compare(paletteHash(left), paletteHash(right)), cmp.Compare(left, right))
	})
	ordered = slices.Compact(ordered)

	out := make(map[string]int, len(ordered))
	taken := make([]bool, PaletteSlots)
	for _, key := range ordered {
		slot := int(paletteHash(key) % PaletteSlots)
		for step := 0; step < PaletteSlots && taken[slot]; step++ {
			slot = (slot + 1) % PaletteSlots
		}
		taken[slot] = true
		out[key] = slot
	}

	return out
}

func PaletteSlot(slots map[string]int, key string) int {
	if slot, ok := slots[key]; ok {
		return slot
	}

	return int(paletteHash(key) % PaletteSlots)
}
