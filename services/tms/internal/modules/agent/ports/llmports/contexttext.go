package llmports

import (
	"strings"

	"github.com/emoss08/trenova/shared/textfence"
)

const (
	untrustedOpenTag  = "<untrusted_data>"
	untrustedCloseTag = "</untrusted_data>"
)

// Text renders the context, fencing every untrusted section so prompt
// injection carried in customer data cannot be read as instruction.
func (c DelimitedContext) Text() string {
	var builder strings.Builder

	for _, section := range c.Sections {
		title := strings.TrimSpace(section.Title)
		if title != "" {
			builder.WriteString("## ")
			builder.WriteString(title)
			builder.WriteString("\n")
		}

		if section.Trusted {
			builder.WriteString(section.Content)
			builder.WriteString("\n\n")
			continue
		}

		builder.WriteString(untrustedOpenTag)
		builder.WriteString("\n")
		builder.WriteString(textfence.FenceText(section.Content, untrustedCloseTag))
		builder.WriteString("\n")
		builder.WriteString(untrustedCloseTag)
		builder.WriteString("\n\n")
	}

	return strings.TrimSpace(builder.String())
}
