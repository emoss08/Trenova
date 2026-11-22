package emaildesign

import (
	"fmt"
	"html"
	"html/template"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

func textOf(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case template.URL:
		return strings.TrimSpace(string(v))
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func joinLines(values []any) string {
	lines := make([]string, 0, len(values))
	for _, value := range values {
		switch v := value.(type) {
		case []string:
			for _, line := range v {
				if line = strings.TrimSpace(line); line != "" {
					lines = append(lines, line)
				}
			}
		default:
			if line := textOf(v); line != "" {
				lines = append(lines, line)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func escaped(value any) string {
	return html.EscapeString(textOf(value))
}

func escapedLines(value any) string {
	return stringutils.TextToHTMLBreaks(strings.ReplaceAll(textOf(value), "\r\n", "\n"))
}

func safeURL(value any) string {
	raw := textOf(value)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https", "http", "mailto":
		return raw
	default:
		return ""
	}
}

func anchor(href, label, style string) string {
	return `<a href="` + html.EscapeString(
		href,
	) + `" target="_blank" rel="noopener noreferrer" style="` +
		style + `">` + label + `</a>`
}

//nolint:gosec // every caller builds the markup from escaped text and constant styles.
func trusted(markup string) template.HTML {
	return template.HTML(markup)
}
