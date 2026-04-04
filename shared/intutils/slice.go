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
