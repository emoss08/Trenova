package runpattern

import (
	"regexp"
	"sort"
	"strings"
)

// From builds the -run expression that selects exactly the named top-level
// tests. Everything the tool claims — that narrowing ran the right tests, that a
// mutant was judged by its covering tests, that verify exercised the complement —
// rides on this one string, which is why there is a single implementation.
func From(tests []string) string {
	quoted := make([]string, 0, len(tests))
	for _, test := range tests {
		quoted = append(quoted, regexp.QuoteMeta(test))
	}
	sort.Strings(quoted)

	return "^(" + strings.Join(quoted, "|") + ")$"
}
