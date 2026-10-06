// Package htmlutils reads HTML documents into forms the rest of the system can
// search, classify and show.
package htmlutils

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ToText renders an HTML document as plain text.
//
// Block elements end their line, list items are bulleted, and the contents of
// script, style, head, template and noscript are dropped, because none of them
// is text a reader of the message ever saw. Runs of whitespace collapse to one
// space and runs of blank lines to one, so the result reads the way the
// message rendered rather than the way its markup was indented.
func ToText(document string) string {
	if strings.TrimSpace(document) == "" {
		return ""
	}

	w := &textWriter{}
	tokenizer := html.NewTokenizer(strings.NewReader(document))
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}

		token := tokenizer.Token()
		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			w.start(token.DataAtom, tt == html.SelfClosingTagToken)
		case html.EndTagToken:
			w.end(token.DataAtom)
		case html.TextToken:
			if w.hidden == 0 {
				w.text(token.Data)
			}
		default:
		}
	}

	return w.String()
}

type textWriter struct {
	b         strings.Builder
	hidden    int
	pendingSp bool
	newlines  int
}

func hiddenElement(a atom.Atom) bool {
	switch a {
	case atom.Script, atom.Style, atom.Head, atom.Template, atom.Noscript, atom.Title:
		return true
	default:
		return false
	}
}

func blockElement(a atom.Atom) bool {
	switch a {
	case atom.P, atom.Div, atom.Section, atom.Article, atom.Header, atom.Footer,
		atom.Blockquote, atom.Pre, atom.Table, atom.Tr, atom.Ul, atom.Ol, atom.Li,
		atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Hr, atom.Dl,
		atom.Dt, atom.Dd, atom.Address, atom.Form, atom.Fieldset, atom.Main,
		atom.Nav, atom.Aside, atom.Figure, atom.Figcaption, atom.Center:
		return true
	default:
		return false
	}
}

func (w *textWriter) start(a atom.Atom, selfClosing bool) {
	if hiddenElement(a) {
		if !selfClosing {
			w.hidden++
		}

		return
	}
	if w.hidden > 0 {
		return
	}

	switch {
	case a == atom.Br:
		w.lineBreak(1)
	case a == atom.Li:
		w.lineBreak(1)
		w.write("- ")
	case a == atom.Td || a == atom.Th:
		w.pendingSp = w.b.Len() > 0 && w.newlines == 0
	case blockElement(a):
		w.lineBreak(1)
	}
}

func (w *textWriter) end(a atom.Atom) {
	if hiddenElement(a) {
		if w.hidden > 0 {
			w.hidden--
		}

		return
	}
	if w.hidden > 0 {
		return
	}

	switch a {
	case atom.P, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6,
		atom.Blockquote, atom.Pre, atom.Table, atom.Ul, atom.Ol:
		w.lineBreak(2)
	default:
		if blockElement(a) {
			w.lineBreak(1)
		}
	}
}

func (w *textWriter) text(data string) {
	fields := strings.Fields(data)
	if len(fields) == 0 {
		if data != "" && w.b.Len() > 0 && w.newlines == 0 {
			w.pendingSp = true
		}

		return
	}

	leading := data[0] == ' ' || data[0] == '\t' || data[0] == '\n' || data[0] == '\r'
	if leading && w.b.Len() > 0 && w.newlines == 0 {
		w.pendingSp = true
	}

	w.write(strings.Join(fields, " "))

	last := data[len(data)-1]
	w.pendingSp = last == ' ' || last == '\t' || last == '\n' || last == '\r'
}

func (w *textWriter) write(s string) {
	if w.pendingSp && w.newlines == 0 && w.b.Len() > 0 {
		w.b.WriteByte(' ')
	}
	w.pendingSp = false
	w.newlines = 0
	w.b.WriteString(s)
}

func (w *textWriter) lineBreak(n int) {
	w.pendingSp = false
	if w.b.Len() == 0 {
		return
	}
	for w.newlines < n {
		w.b.WriteByte('\n')
		w.newlines++
	}
}

func (w *textWriter) String() string {
	return strings.TrimSpace(w.b.String())
}
