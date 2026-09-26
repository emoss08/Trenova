package stringutils

import "strings"

func MarkdownTableCell(text string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(text, "|", `\|`)), " ")
}
