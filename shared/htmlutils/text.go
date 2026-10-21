// Package htmlutils reads HTML documents into forms the rest of the system can
// search, classify and show.
package htmlutils

import (
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ToText renders an HTML document as plain text.
//
// Block elements end their line, list items are bulleted, and the contents of
// script, style, head, template and noscript are dropped, because none of them
// is text a reader of the message ever saw. So is anything the markup hides
// from the reader: an element with the hidden attribute, aria-hidden="true",
// or an inline style of display:none, visibility:hidden, font-size:0 or
// opacity:0. Text hidden that way is how a message carries instructions to a
// model that the person reading it never sees. Runs of whitespace collapse to
// one space and runs of blank lines to one, so the result reads the way the
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
			w.start(&token, tt == html.SelfClosingTagToken)
		case html.EndTagToken:
			w.end(&token)
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
	open      []openElement
	hidden    int
	pendingSp bool
	newlines  int
}

type openElement struct {
	name   string
	hidden bool
}

var hiddenStyles = []string{
	"display:none",
	"visibility:hidden",
	"visibility:collapse",
	"font-size:0",
	"opacity:0",
}

// hiddenByMarkup reports whether an element's attributes keep it from being
// shown: the hidden attribute, aria-hidden="true", or an inline style that
// does not draw it.
func hiddenByMarkup(token *html.Token) bool {
	for _, attr := range token.Attr {
		switch strings.ToLower(attr.Key) {
		case "hidden":
			return true
		case "aria-hidden":
			if strings.EqualFold(strings.TrimSpace(attr.Val), "true") {
				return true
			}
		case "style":
			if hiddenStyle(attr.Val) {
				return true
			}
		}
	}
	return false
}

func hiddenStyle(style string) bool {
	compact := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, strings.ToLower(style))
	for declaration := range strings.SplitSeq(compact, ";") {
		declaration = strings.TrimSuffix(declaration, "!important")
		for _, hidden := range hiddenStyles {
			if declaration == hidden || strings.HasPrefix(declaration, hidden) &&
				zeroValue(strings.TrimPrefix(declaration, hidden)) {
				return true
			}
		}
	}
	return false
}

// zeroValue reports whether what follows "font-size:0" or "opacity:0" keeps
// the value zero: a unit ("0px", "0em", "0%") or more zeros ("0.0"), not "0.5".
func zeroValue(rest string) bool {
	rest = strings.TrimLeft(rest, "0")
	if strings.HasPrefix(rest, ".") {
		rest = strings.TrimLeft(rest[1:], "0")
	}
	for _, r := range rest {
		if r >= '1' && r <= '9' || r == '.' {
			return false
		}
	}
	return true
}

var voidElements = []atom.Atom{
	atom.Area, atom.Base, atom.Br, atom.Col, atom.Embed, atom.Hr, atom.Img, atom.Input,
	atom.Link, atom.Meta, atom.Source, atom.Track, atom.Wbr,
}

func voidElement(a atom.Atom) bool {
	return slices.Contains(voidElements, a)
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

func (w *textWriter) start(token *html.Token, selfClosing bool) {
	a := token.DataAtom
	hidden := hiddenElement(a) || hiddenByMarkup(token)
	if !selfClosing && !voidElement(a) {
		w.open = append(w.open, openElement{name: token.Data, hidden: hidden})
		if hidden {
			w.hidden++
		}
	}
	if hidden || w.hidden > 0 {
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

// end closes the innermost open element of the same name, and every element
// left open inside it, as a browser does for markup that never closed them.
// An end tag with nothing open to close is ignored.
func (w *textWriter) end(token *html.Token) {
	match := -1
	for i := len(w.open) - 1; i >= 0; i-- {
		if w.open[i].name == token.Data {
			match = i
			break
		}
	}
	if match < 0 {
		return
	}
	for _, closed := range w.open[match:] {
		if closed.hidden {
			w.hidden--
		}
	}
	w.open = w.open[:match]
	if w.hidden > 0 {
		return
	}

	a := token.DataAtom
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
