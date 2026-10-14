package stringutils

import "strings"

func DistinctShortNames(names []string) []string {
	candidates := make([][]string, len(names))
	level := make([]int, len(names))
	for idx, name := range names {
		candidates[idx] = shortNameCandidates(name)
	}

	for range 3 {
		claims := make(map[string]int, len(names))
		for idx := range names {
			claims[strings.ToLower(candidates[idx][level[idx]])]++
		}
		moved := false
		for idx := range names {
			want := strings.ToLower(candidates[idx][level[idx]])
			if claims[want] > 1 && level[idx] < len(candidates[idx])-1 {
				level[idx]++
				moved = true
			}
		}
		if !moved {
			break
		}
	}

	out := make([]string, len(names))
	for idx := range names {
		out[idx] = candidates[idx][level[idx]]
	}

	return out
}

func shortNameCandidates(name string) []string {
	trimmed := strings.TrimSpace(name)
	words := strings.Fields(trimmed)
	if len(words) < 2 {
		return []string{trimmed}
	}
	first, last := words[0], words[len(words)-1]
	if strings.EqualFold(first, last) {
		return []string{last, trimmed}
	}

	return []string{last, first, trimmed}
}
