package setutils

func Jaccard[T comparable](a, b map[T]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}

	small, large := a, b
	if len(small) > len(large) {
		small, large = large, small
	}

	intersection := 0
	for item := range small {
		if _, ok := large[item]; ok {
			intersection++
		}
	}

	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

func Union[T comparable](into, from map[T]struct{}) map[T]struct{} {
	if into == nil {
		into = make(map[T]struct{}, len(from))
	}
	for item := range from {
		into[item] = struct{}{}
	}

	return into
}
