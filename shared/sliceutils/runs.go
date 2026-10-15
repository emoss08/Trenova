package sliceutils

func Runs[T any, K comparable, V any](items []T, key func(T) K, value func(T) V) [][]V {
	if len(items) == 0 {
		return nil
	}

	runs := make([][]V, 0, len(items))
	start := 0
	for i := 1; i <= len(items); i++ {
		if i < len(items) && key(items[i]) == key(items[start]) {
			continue
		}
		run := make([]V, 0, i-start)
		for _, item := range items[start:i] {
			run = append(run, value(item))
		}
		runs = append(runs, run)
		start = i
	}

	return runs
}
