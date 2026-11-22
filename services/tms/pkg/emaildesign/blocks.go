package emaildesign

import (
	"html"
	"html/template"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/i18n"
)

const highlightTag = "hl"

func kicker(text any) template.HTML {
	return kickerAt("0 0 20px", muted, text)
}

func section(text any) template.HTML {
	return kickerAt("16px 0 10px", muted, text)
}

func kickerAt(margin, color string, text any) template.HTML {
	words := escaped(text)
	if words == "" {
		return ""
	}
	return trusted(
		`<p style="margin:` + margin + `;` + kickerFont + color + `">[ ` + words + ` ]</p>`,
	)
}

func headingFor(locale i18n.Locale) func(message string, args ...any) template.HTML {
	return func(message string, args ...any) template.HTML {
		pieces := i18n.TranslateRich(locale, message, []string{highlightTag}, args...)
		var b strings.Builder
		highlighted := false
		for i := range pieces {
			piece := pieces[i]
			if piece.Text == "" {
				continue
			}
			if piece.Tag == highlightTag && !highlighted {
				highlighted = true
				closing := ""
				if i+1 < len(pieces) {
					closing, pieces[i+1].Text = splitClosingPunctuation(pieces[i+1].Text)
				}
				b.WriteString(`<span style="background-color:` + amber + `;color:` + headingInk +
					`;padding:0 6px;-webkit-box-decoration-break:clone;box-decoration-break:clone">` +
					html.EscapeString(piece.Text+closing) + `</span>`)
				continue
			}
			b.WriteString(html.EscapeString(piece.Text))
		}
		if strings.TrimSpace(b.String()) == "" {
			return ""
		}
		return trusted(
			`<h1 class="h1" style="margin:0 0 28px;padding:0;font-size:30px;line-height:1.2;` +
				`font-weight:600;letter-spacing:-1px;color:` + headingInk + `;overflow-wrap:anywhere">` +
				b.String() + `</h1>`,
		)
	}
}

func splitClosingPunctuation(text string) (closing, rest string) {
	end := 0
	for i, r := range text {
		if !strings.ContainsRune(".?!。？！", r) {
			break
		}
		end = i + utf8.RuneLen(r)
	}
	return text[:end], text[end:]
}

func plainHeadingFor(locale i18n.Locale) func(message string, args ...any) string {
	return func(message string, args ...any) string {
		pieces := i18n.TranslateRich(locale, message, []string{highlightTag}, args...)
		var b strings.Builder
		for _, piece := range pieces {
			b.WriteString(piece.Text)
		}
		return strings.TrimSpace(b.String())
	}
}

func para(text ...any) template.HTML {
	body := escapedLines(joinLines(text))
	if body == "" {
		return ""
	}
	return trusted(`<p style="` + paragraphStyle + `">` + body + `</p>`)
}

func signoff(lines ...any) template.HTML {
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if text := escapedLines(line); text != "" {
			kept = append(kept, text)
		}
	}
	if len(kept) == 0 {
		return ""
	}
	return trusted(`<p style="` + paragraphStyle + `">` + strings.Join(kept, "<br>") + `</p>`)
}

func link(href, label any) template.HTML {
	target := safeURL(href)
	text := escaped(label)
	if target == "" || text == "" {
		return ""
	}
	return trusted(`<p style="` + paragraphStyle + `">` + anchor(target, text, linkStyle) + `</p>`)
}

func boxed(cellAttrs, cellStyle, body string) template.HTML {
	if body == "" {
		return ""
	}
	return trusted(
		`<table ` + tableAttrs + ` style="margin:0 0 16px;padding:0"><tr><td` + cellAttrs +
			` style="` + cellStyle + `">` + body + `</td></tr></table>`,
	)
}

func quote(text ...any) template.HTML {
	return boxed("", "margin:0;padding:0 0 0 16px;border-left:3px solid "+ink+
		";font-size:16px;line-height:1.6;color:"+ink, escapedLines(joinLines(text)))
}

func code(text any) template.HTML {
	return boxed(` bgcolor="`+paper+`"`, "margin:0;padding:12px 14px;background-color:"+paper+
		";border:1px solid "+hairline+";font-family:"+mono+
		";font-size:14px;line-height:22px;color:"+ink+";word-break:break-all", escaped(text))
}

func notice(text ...any) template.HTML {
	return boxed(
		` bgcolor="`+noticeFill+`"`,
		"margin:0;padding:16px 18px;background-color:"+noticeFill+
			";border:1px solid "+ink+";font-size:16px;line-height:1.6;color:"+ink,
		escapedLines(joinLines(text)),
	)
}

func action(label, href any, context ...any) template.HTML {
	target := safeURL(href)
	text := escaped(label)
	if target == "" || text == "" {
		return ""
	}

	var b strings.Builder
	if len(context) > 1 {
		b.WriteString(string(kickerAt("0 0 12px", bandInk, context[1])))
	}
	if len(context) > 0 {
		if sentence := escapedLines(context[0]); sentence != "" {
			b.WriteString(
				`<p style="margin:0 0 20px;padding:0;font-size:16px;line-height:1.6;color:` +
					headingInk + `">` + sentence + `</p>`,
			)
		}
	}
	b.WriteString(`<table border="0" cellpadding="0" cellspacing="0" role="presentation" ` +
		`style="margin:0;padding:0"><tr><td align="center" bgcolor="#121210" ` +
		`style="margin:0;padding:14px 28px;background-color:` + ink + `">`)
	b.WriteString(anchor(target, text, "display:inline-block;font-family:"+sans+
		";font-size:16px;line-height:20px;font-weight:600;color:"+white+";text-decoration:none"))
	b.WriteString(`</td></tr></table>`)

	return trusted(bandMarker.wrap(b.String()))
}

func bandCell(inner string, last bool) string {
	closing := "border-bottom:1px solid " + ink + ";"
	if last {
		closing = ""
	}
	return `<tr><td class="px" bgcolor="` + amber + `" style="margin:0;padding:32px 40px;background-color:` +
		amber + `;border-top:1px solid ` + ink + `;` + closing + `font-family:` + sans + `">` +
		inner + `</td></tr>`
}

func preheader(text any) template.HTML {
	words := escaped(text)
	if words == "" {
		return ""
	}
	return trusted(preheaderMarker.wrap(words))
}

func footer(text any, manage ...any) template.HTML {
	words := escaped(text)
	if words == "" {
		return ""
	}
	if len(manage) > 1 {
		label := escaped(manage[0])
		if target := safeURL(manage[1]); target != "" && label != "" {
			words += " " + anchor(target, label,
				"color:"+muted+";text-decoration:underline;text-underline-offset:3px")
		}
	}
	return trusted(footerMarker.wrap(words))
}
