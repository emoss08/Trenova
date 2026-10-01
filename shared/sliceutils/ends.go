package sliceutils

func KeepEnds(weights []int, budget int) (head, tail int) {
	front, back := 0, len(weights)-1
	used := 0
	frontOpen, backOpen := true, true

	for front <= back && (frontOpen || backOpen) {
		if frontOpen {
			if used+weights[front] <= budget {
				used += weights[front]
				front++
			} else {
				frontOpen = false
			}
		}
		if front <= back && backOpen {
			if used+weights[back] <= budget {
				used += weights[back]
				back--
			} else {
				backOpen = false
			}
		}
	}

	return front, len(weights) - 1 - back
}
