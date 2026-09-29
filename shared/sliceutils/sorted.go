package sliceutils

import (
	"cmp"
	"slices"
)

func SortedNonNil[T any, K cmp.Ordered](items []*T, key func(*T) K) []*T {
	out := make([]*T, 0, len(items))
	for _, item := range items {
		if item != nil {
			out = append(out, item)
		}
	}
	slices.SortStableFunc(out, func(a, b *T) int {
		return cmp.Compare(key(a), key(b))
	})

	return out
}
