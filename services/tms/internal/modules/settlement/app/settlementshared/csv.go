package settlementshared

import (
	"strings"
	"time"
)

// WriteCSVField writes value, quoting it when it contains a delimiter, quote,
// or line break (\n or \r) so multi-line values cannot split a row.
func WriteCSVField(sb *strings.Builder, value string) {
	if strings.ContainsAny(value, ",\"\n\r") {
		sb.WriteByte('"')
		sb.WriteString(strings.ReplaceAll(value, "\"", "\"\""))
		sb.WriteByte('"')
		return
	}
	sb.WriteString(value)
}

// FormatCSVDate renders a unix timestamp as a UTC calendar date.
func FormatCSVDate(unix int64) string {
	return time.Unix(unix, 0).UTC().Format("2006-01-02")
}
