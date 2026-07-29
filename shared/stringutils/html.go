package stringutils

import (
	"html"
	"strings"
)

// TextToHTMLParagraph escapes plain text and turns newlines into line breaks,
// producing a single paragraph safe to place in an HTML document.
//
// Escaping happens before any markup is added, so text that arrives from a
// customer record — a company name containing an ampersand, a memo containing a
// tag — cannot inject markup into a document or an email body.
func TextToHTMLParagraph(text string) string {
	if text == "" {
		return ""
	}
	escaped := html.EscapeString(text)
	return "<p>" + strings.ReplaceAll(escaped, "\n", "<br>") + "</p>"
}

// TextToHTMLBreaks escapes plain text and turns newlines into line breaks
// without wrapping it in a paragraph, for callers that supply their own block
// element. Escaping happens first, as above.
func TextToHTMLBreaks(text string) string {
	if text == "" {
		return ""
	}
	return strings.ReplaceAll(html.EscapeString(text), "\n", "<br>")
}
