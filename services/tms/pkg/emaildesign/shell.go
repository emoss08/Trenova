package emaildesign

import (
	"html"
	"math"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/templatediag"
	"github.com/emoss08/trenova/pkg/templatesanitize"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/imageutils"
)

const contentField = "bodyHtml"

type Brand struct {
	Name string
	Mark string
}

type Message struct {
	Locale  i18n.Locale
	Subject string
	Brand   Brand
	Content string
}

const documentHead = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" ` +
	`"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">`

const documentStyle = `<style>
  :root { color-scheme: light only; }
  body { margin: 0; padding: 0; }
  a { color: ` + ink + `; }
  @media (max-width: 620px) {
    .px { padding-left: 24px !important; padding-right: 24px !important; }
    .h1 { font-size: 26px !important; }
    .fl { width: 104px !important; min-width: 104px !important; }
    .dc { font-size: 13px !important; line-height: 20px !important; padding-left: 6px !important; padding-right: 6px !important; }
  }
</style>`

func Compose(m *Message) (string, templatediag.Diagnostics) {
	diags := templatesanitize.HTML(contentField, m.Content, templatesanitize.RenderedEmailOptions())
	if diags.HasErrors() {
		return "", diags
	}
	diags = append(diags, warnings(Check(m.Content))...)

	pre, rest := preheaderMarker.lift(m.Content)
	foot, rest := footerMarker.lift(rest)

	lang := html.EscapeString(languageOf(m.Locale))

	var b strings.Builder
	b.Grow(len(m.Content) + 6<<10)
	b.WriteString(documentHead + "\n")
	b.WriteString(`<html dir="ltr" lang="` + lang + `"><head>`)
	b.WriteString(`<meta content="width=device-width" name="viewport" />`)
	b.WriteString(`<meta content="text/html; charset=UTF-8" http-equiv="Content-Type" />`)
	b.WriteString(`<meta name="x-apple-disable-message-reformatting" />`)
	b.WriteString(`<meta content="IE=edge" http-equiv="X-UA-Compatible" />`)
	b.WriteString(
		`<meta content="telephone=no,address=no,email=no,date=no,url=no" name="format-detection" />`,
	)
	b.WriteString(`<title>` + html.EscapeString(strings.TrimSpace(m.Subject)) + `</title>`)
	b.WriteString(documentStyle)
	b.WriteString(`</head><body dir="ltr" lang="` + lang + `" style="background-color:` + paper +
		`;margin:0;padding:0">`)

	if pre != "" {
		b.WriteString(
			`<div style="margin:0;padding:0;display:none;max-height:0;overflow:hidden;mso-hide:all;` +
				`font-size:1px;line-height:1px;color:` + paper + `;opacity:0">` + pre + `</div>`,
		)
	}

	b.WriteString(
		`<table ` + tableAttrs + ` bgcolor="` + paper + `" style="margin:0;padding:0;width:100%;` +
			`background-color:` + paper + `"><tr><td align="center" style="margin:0;padding:32px 12px 40px">`,
	)
	b.WriteString(
		`<!--[if mso]><table role="presentation" width="600" align="center" cellpadding="0" ` +
			`cellspacing="0" border="0"><tr><td><![endif]-->`,
	)
	b.WriteString(
		`<table ` + tableAttrs + ` align="center" style="margin:0 auto;padding:0;width:100%;` +
			`max-width:600px">`,
	)

	b.WriteString(header(m.Brand))
	b.WriteString(
		`<tr><td style="margin:0;padding:0;background-color:` + white + `;border:1px solid ` + ink +
			`"><table ` + tableAttrs + ` style="margin:0;padding:0">`,
	)
	b.WriteString(cardRows(rest))
	b.WriteString(`</table></td></tr>`)

	if foot != "" {
		b.WriteString(`<tr><td style="margin:0;padding:20px 4px 0;font-family:` + sans +
			`;font-size:13px;line-height:1.6;color:` + muted + `"><p style="margin:0;padding:0">` +
			foot + `</p></td></tr>`)
	}

	b.WriteString(
		`</table><!--[if mso]></td></tr></table><![endif]--></td></tr></table></body></html>`,
	)

	return b.String(), diags
}

func languageOf(locale i18n.Locale) string {
	if locale.IsValid() {
		return string(locale)
	}
	return string(i18n.Default)
}

func header(brand Brand) string {
	name := html.EscapeString(strings.TrimSpace(brand.Name))
	if name == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString(`<tr><td style="margin:0;padding:0 4px 20px"><table border="0" cellpadding="0" ` +
		`cellspacing="0" role="presentation" style="margin:0;padding:0"><tr>`)
	if mark := markImage(brand.Mark); mark != "" {
		b.WriteString(
			`<td style="margin:0;padding:0 10px 0 0;vertical-align:middle">` + mark + `</td>`,
		)
	}
	b.WriteString(`<td style="margin:0;padding:0;vertical-align:middle;font-family:` + sans +
		`;font-size:19px;line-height:1.2;font-weight:600;letter-spacing:-0.4px;color:` + ink + `">` +
		name + `</td></tr></table></td></tr>`)
	return b.String()
}

func markImage(dataURI string) string {
	if !imageutils.IsDataURI(dataURI) {
		return ""
	}
	img, err := imageutils.DecodeDataURI(dataURI, markMaxBytes)
	if err != nil || img.Width <= 0 || img.Height <= 0 {
		return ""
	}
	w, h := imageutils.FittedSize(float64(img.Width), float64(img.Height), markMaxWidth, markHeight)
	width := strconv.Itoa(int(math.Round(w)))
	height := strconv.Itoa(int(math.Round(h)))

	return `<img alt="" width="` + width + `" height="` + height + `" src="` +
		html.EscapeString(
			imageutils.ToDataURI(img),
		) + `" style="display:block;outline:none;border:0;` +
		`text-decoration:none;width:` + width + `px;height:` + height + `px;border-radius:7px" />`
}

func cardRows(content string) string {
	segments := splitBands(content)
	var b strings.Builder
	for i, seg := range segments {
		last := i == len(segments)-1
		if seg.band {
			b.WriteString(bandCell(seg.html, last))
			continue
		}
		top := cardPadding
		if i > 0 {
			top = bandPadding
		}
		bottom := cardPadding - flowGap
		if !last {
			bottom = bandPadding - flowGap
		}
		b.WriteString(
			`<tr><td class="px" style="margin:0;padding:` + strconv.Itoa(top) + `px 40px ` +
				strconv.Itoa(
					bottom,
				) + `px;font-family:` + sans + `;font-size:16px;line-height:1.6;color:` + ink +
				`">` + seg.html + `</td></tr>`,
		)
	}
	return b.String()
}
