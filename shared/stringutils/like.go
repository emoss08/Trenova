package stringutils

import "strings"

var likePatternEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func EscapeLikePattern(term string) string {
	return likePatternEscaper.Replace(term)
}
