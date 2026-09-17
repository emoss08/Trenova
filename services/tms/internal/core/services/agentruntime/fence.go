package agentruntime

import (
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxToolResultChars = 12000
	untrustedOpenTag   = "<untrusted_data>"
	untrustedCloseTag  = "</untrusted_data>"
)

func FenceToolResult(toolName, payload string) string {
	truncated := payload
	if len(truncated) > maxToolResultChars {
		truncated = truncated[:maxToolResultChars] + "\n…(truncated)"
	}

	var builder strings.Builder
	builder.WriteString("Result from ")
	builder.WriteString(toolName)
	builder.WriteString(":\n")
	builder.WriteString(untrustedOpenTag)
	builder.WriteString("\n")
	builder.WriteString(stringutils.NeutralizeCloseTag(truncated, untrustedCloseTag))
	builder.WriteString("\n")
	builder.WriteString(untrustedCloseTag)

	return builder.String()
}
