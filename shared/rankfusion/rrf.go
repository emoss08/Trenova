package rankfusion

const DefaultK = 60

func Reciprocal[K comparable](k int, lists ...[]K) map[K]float64 {
	if k <= 0 {
		k = DefaultK
	}

	size := 0
	for _, list := range lists {
		size += len(list)
	}

	scores := make(map[K]float64, size)
	for _, list := range lists {
		seen := make(map[K]struct{}, len(list))
		rank := 0
		for _, key := range list {
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			rank++
			scores[key] += 1 / float64(k+rank)
		}
	}

	return scores
}
