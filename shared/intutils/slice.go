package intutils

func IntSliceValue(v any) []int {
	items, ok := v.([]any)
	if !ok {
		if direct, ok := v.([]int); ok {
			return append([]int{}, direct...)
		}
		return []int{}
	}
	out := make([]int, 0, len(items))
	for _, item := range items {
		out = append(out, IntValue(item))
	}
	return out
}

func Ints[T ~int8 | ~int16 | ~int32 | ~int64](items []T) []int {
	out := make([]int, len(items))
	for idx, item := range items {
		out[idx] = int(item)
	}
	return out
}
