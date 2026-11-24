package toolwords

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentsearch"
)

const longestName = 4

func Named(need string) []string {
	words := strings.FieldsFunc(strings.ToLower(need), notNameRune)
	names := make([]string, 0, len(words)*longestName)
	add := func(name string) {
		if name != "" && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	for idx, word := range words {
		add(word)
		if strings.Contains(word, "_") {
			continue
		}
		for size := 2; size <= longestName && idx+size <= len(words); size++ {
			run := words[idx : idx+size]
			if slices.ContainsFunc(run, func(w string) bool { return strings.Contains(w, "_") }) {
				break
			}
			add(strings.Join(run, "_"))
			add(strings.Join(singulars(run), "_"))
			if size == 2 {
				add(agentsearch.Singularize(run[1]) + "_" + agentsearch.Singularize(run[0]))
			}
		}
	}

	return names
}

func singulars(words []string) []string {
	out := make([]string, 0, len(words))
	for _, word := range words {
		out = append(out, agentsearch.Singularize(word))
	}

	return out
}

func notNameRune(r rune) bool {
	return (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_'
}
