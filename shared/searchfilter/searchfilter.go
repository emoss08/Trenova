package searchfilter

import (
	"regexp"
	"strings"
)

var fieldName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)

var quoteEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

func Quote(value string) string {
	return `"` + quoteEscaper.Replace(value) + `"`
}

func Eq(field, value string) string {
	if !fieldName.MatchString(field) {
		panic("searchfilter: invalid field name " + field)
	}
	return field + " = " + Quote(value)
}

func And(conditions ...string) string {
	parts := make([]string, 0, len(conditions))
	for _, condition := range conditions {
		if condition != "" {
			parts = append(parts, "("+condition+")")
		}
	}
	return strings.Join(parts, " AND ")
}
