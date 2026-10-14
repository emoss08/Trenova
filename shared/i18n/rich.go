package i18n

import "strings"

type RichPiece struct {
	Tag  string
	Text string
}

type tagKind int

const (
	tagNone tagKind = iota
	tagOpen
	tagClose
	tagSelfClosing
)

func TranslateRich(locale Locale, message string, tags []string, args ...any) []RichPiece {
	if message == "" {
		return nil
	}
	if !locale.IsValid() {
		locale = Default
	}

	translated := lookup(locale, message)
	if pieces, ok := parseRich(translated, tags); ok {
		return formatPieces(locale, pieces, args)
	}
	if pieces, ok := parseRich(message, tags); ok {
		return formatPieces(Default, pieces, args)
	}

	return []RichPiece{{Text: format(locale, stripTags(translated, tags), args)}}
}

func formatPieces(locale Locale, pieces []RichPiece, args []any) []RichPiece {
	for idx := range pieces {
		if pieces[idx].Text != "" {
			pieces[idx].Text = format(locale, pieces[idx].Text, args)
		}
	}

	return pieces
}

func parseRich(template string, tags []string) ([]RichPiece, bool) {
	pieces := make([]RichPiece, 0, 2*len(tags)+1)
	var text strings.Builder
	flush := func() {
		if text.Len() > 0 {
			pieces = append(pieces, RichPiece{Text: text.String()})
			text.Reset()
		}
	}

	rest := template
	for {
		idx := strings.IndexByte(rest, '<')
		if idx < 0 {
			text.WriteString(rest)
			break
		}
		text.WriteString(rest[:idx])
		rest = rest[idx:]

		name, kind, width := matchTag(rest, tags)
		switch kind {
		case tagNone:
			text.WriteByte('<')
			rest = rest[1:]
		case tagClose:
			return nil, false
		case tagSelfClosing:
			flush()
			pieces = append(pieces, RichPiece{Tag: name})
			rest = rest[width:]
		case tagOpen:
			flush()
			closing := "</" + name + ">"
			end := strings.Index(rest[width:], closing)
			if end < 0 {
				return nil, false
			}
			inner := rest[width : width+end]
			if containsTag(inner, tags) {
				return nil, false
			}
			pieces = append(pieces, RichPiece{Tag: name, Text: inner})
			rest = rest[width+end+len(closing):]
		}
	}
	flush()

	return pieces, true
}

func matchTag(s string, tags []string) (string, tagKind, int) {
	for _, name := range tags {
		if name == "" {
			continue
		}
		for _, form := range [...]struct {
			text string
			kind tagKind
		}{
			{"<" + name + ">", tagOpen},
			{"</" + name + ">", tagClose},
			{"<" + name + "/>", tagSelfClosing},
			{"<" + name + " />", tagSelfClosing},
		} {
			if strings.HasPrefix(s, form.text) {
				return name, form.kind, len(form.text)
			}
		}
	}

	return "", tagNone, 0
}

func containsTag(s string, tags []string) bool {
	for idx := strings.IndexByte(s, '<'); idx >= 0; {
		if _, kind, _ := matchTag(s[idx:], tags); kind != tagNone {
			return true
		}
		next := strings.IndexByte(s[idx+1:], '<')
		if next < 0 {
			return false
		}
		idx += next + 1
	}

	return false
}

func stripTags(s string, tags []string) string {
	var out strings.Builder
	out.Grow(len(s))
	for len(s) > 0 {
		idx := strings.IndexByte(s, '<')
		if idx < 0 {
			out.WriteString(s)
			break
		}
		out.WriteString(s[:idx])
		s = s[idx:]
		if _, kind, width := matchTag(s, tags); kind != tagNone {
			s = s[width:]
			continue
		}
		out.WriteByte('<')
		s = s[1:]
	}

	return out.String()
}
