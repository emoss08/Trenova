package emaildesign

import (
	"fmt"
	"html/template"
	"reflect"
	"strconv"
	"strings"
)

type Cell struct {
	Text    string
	Detail  string
	URL     string
	Numeric bool
	Label   bool
	Step    int
	Span    int
}

type Column struct {
	Text    string
	Numeric bool
}

type Fact struct {
	Label  string
	Value  string
	Detail string
}

func cell(value any) Cell { return Cell{Text: textOf(value)} }

func num(value any) Cell { return Cell{Text: textOf(value), Numeric: true} }

func label(value any) Cell { return Cell{Text: textOf(value), Label: true} }

func step(index int) Cell { return Cell{Text: stepNumber(index + 1), Step: index + 1} }

func wide(span int, value any) Cell { return Cell{Text: textOf(value), Span: span} }

func item(title, detail any, href ...any) Cell {
	out := Cell{Text: textOf(title), Detail: textOf(detail)}
	if len(href) > 0 {
		out.URL = safeURL(href[0])
	}
	return out
}

func col(value any) Column { return Column{Text: textOf(value)} }

func numCol(value any) Column { return Column{Text: textOf(value), Numeric: true} }

func fact(name, value any, detail ...any) Fact {
	out := Fact{Label: textOf(name), Value: textOf(value)}
	if len(detail) > 0 {
		out.Detail = textOf(detail[0])
	}
	return out
}

func stepNumber(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

func ruleStyle(last bool) string {
	if last {
		return "border-bottom:1px solid " + ink + ";"
	}
	return "border-bottom:1px solid " + hairline + ";"
}

func valueHTML(text, detail, href string) string {
	body := escapedLines(text)
	if href != "" && body != "" {
		body = anchor(href, body, linkStyle)
	}
	if detail != "" {
		body += `<br><span style="` + detailStyle + `">` + escapedLines(detail) + `</span>`
	}
	return body
}

func facts(entries ...any) template.HTML {
	kept := make([]Fact, 0, len(entries))
	for _, entry := range entries {
		for _, f := range factsFrom(entry) {
			if f.Value != "" {
				kept = append(kept, f)
			}
		}
	}
	if len(kept) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(
		`<table ` + tableAttrs + ` style="margin:0 0 16px;padding:0;border-top:1px solid ` + ink + `">`,
	)
	for i, f := range kept {
		rule := ruleStyle(i == len(kept)-1)
		b.WriteString(`<tr><td class="fl" style="margin:0;padding:12px 12px 12px 0;width:132px;` +
			`min-width:132px;vertical-align:top;` + rule + labelFont + `">` + escaped(f.Label) + `</td>`)
		b.WriteString(`<td style="margin:0;padding:12px 0;vertical-align:top;` + rule +
			`font-size:16px;line-height:26px;font-weight:500;color:` + ink + `;overflow-wrap:anywhere">` +
			valueHTML(f.Value, f.Detail, "") + `</td></tr>`)
	}
	b.WriteString(`</table>`)
	return trusted(b.String())
}

func steps(entries ...any) template.HTML {
	items := make([]string, 0, len(entries))
	for _, cellValue := range flattenCells(entries) {
		if cellValue.Text != "" {
			items = append(items, cellValue.Text)
		}
	}
	if len(items) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(
		`<table ` + tableAttrs + ` style="margin:0 0 16px;padding:0;border-top:1px solid ` + ink + `">`,
	)
	for i, text := range items {
		rule := ruleStyle(i == len(items)-1)
		b.WriteString(
			`<tr>` + stepCell(stepNumber(i+1), rule) + `<td style="margin:0;padding:12px 0;` +
				`vertical-align:top;` + rule + `font-size:16px;line-height:26px;font-weight:500;color:` + ink +
				`">` + escapedLines(text) + `</td></tr>`,
		)
	}
	b.WriteString(`</table>`)
	return trusted(b.String())
}

func stepCell(number, rule string) string {
	return `<td style="margin:0;padding:12px 0;width:44px;min-width:44px;white-space:nowrap;` +
		`vertical-align:top;` + rule + `font-family:` + mono + `;font-size:12px;line-height:26px;color:` +
		muted + `">` + number + `</td>`
}

func tableOpen(columns ...any) template.HTML {
	cols := flattenColumns(columns)
	var b strings.Builder
	b.WriteString(
		`<table ` + tableAttrs + ` style="margin:0;padding:0;border-top:1px solid ` + ink + `">`,
	)
	if len(cols) == 0 {
		return trusted(b.String())
	}
	b.WriteString(`<tr>`)
	for i, c := range cols {
		align := "left"
		if c.Numeric {
			align = "right"
		}
		b.WriteString(`<th scope="col" align="` + align + `" style="margin:0;padding:` +
			cellPadding(i, len(cols)) + `;vertical-align:bottom;text-align:` + align +
			`;font-weight:400;` + labelFont + `">` + escaped(c.Text) + `</th>`)
	}
	b.WriteString(`</tr>`)
	return trusted(b.String())
}

func tableClose() template.HTML {
	return trusted(`</table><table ` + tableAttrs + ` style="margin:0 0 16px;padding:0"><tr>` +
		`<td style="margin:0;padding:0;height:1px;font-size:1px;line-height:1px;border-top:1px solid ` +
		ink + `">&nbsp;</td></tr></table>`)
}

func row(cells ...any) template.HTML {
	return tableRow(flattenCells(cells), false)
}

func totalRow(cells ...any) template.HTML {
	return tableRow(flattenCells(cells), true)
}

func tableRow(cells []Cell, total bool) template.HTML {
	if len(cells) == 0 {
		return ""
	}
	numbered := false
	for _, c := range cells {
		if c.Step > 0 {
			numbered = true
		}
	}

	border := "border-top:1px solid " + hairline + ";"
	switch {
	case total:
		border = "border-top:1px solid " + ink + ";"
	case cells[0].Step == 1:
		border = ""
	}

	font := "font-size:15px;line-height:22px;font-weight:400;"
	switch {
	case numbered:
		font = "font-size:16px;line-height:26px;font-weight:500;"
	case total:
		font = "font-size:15px;line-height:22px;font-weight:600;"
	}

	var b strings.Builder
	b.WriteString(`<tr>`)
	for i, c := range cells {
		if c.Step > 0 {
			b.WriteString(stepCell(escaped(c.Text), border))
			continue
		}
		b.WriteString(dataCell(c, cellPadding(i, len(cells)), border, font))
	}
	b.WriteString(`</tr>`)
	return trusted(b.String())
}

func dataCell(c Cell, padding, border, font string) string {
	attrs := ""
	if c.Span > 1 {
		attrs = ` colspan="` + strconv.Itoa(c.Span) + `"`
	}
	style := "margin:0;padding:" + padding + ";vertical-align:top;" + border
	switch {
	case c.Label:
		style += labelFont + ";"
	case c.Numeric:
		style += font + "color:" + ink + ";text-align:right;font-variant-numeric:tabular-nums;white-space:nowrap;"
		attrs += ` align="right"`
	default:
		style += font + "color:" + ink + ";word-break:break-word;"
	}
	return `<td class="dc"` + attrs + ` style="` + style + `">` + valueHTML(c.Text, c.Detail, c.URL) + `</td>`
}

func cellPadding(index, count int) string {
	left, right := "10px", "10px"
	if index == 0 {
		left = "0"
	}
	if index == count-1 {
		right = "0"
	}
	return "10px " + right + " 10px " + left
}

func flattenCells(values []any) []Cell {
	out := make([]Cell, 0, len(values))
	for _, value := range values {
		switch v := value.(type) {
		case nil:
		case Cell:
			out = append(out, v)
		case []Cell:
			out = append(out, v...)
		case string:
			out = append(out, Cell{Text: strings.TrimSpace(v)})
		case []string:
			for _, s := range v {
				out = append(out, Cell{Text: strings.TrimSpace(s)})
			}
		default:
			out = append(out, reflectedCells(value)...)
		}
	}
	return out
}

func flattenColumns(values []any) []Column {
	out := make([]Column, 0, len(values))
	for _, value := range values {
		switch v := value.(type) {
		case nil:
		case Column:
			out = append(out, v)
		case []Column:
			out = append(out, v...)
		default:
			for _, c := range flattenCells([]any{value}) {
				out = append(out, Column{Text: c.Text, Numeric: c.Numeric})
			}
		}
	}
	return out
}

func factsFrom(entry any) []Fact {
	switch v := entry.(type) {
	case nil:
		return nil
	case Fact:
		return []Fact{v}
	case []Fact:
		return v
	}
	rv := reflect.ValueOf(entry)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil
	}
	out := make([]Fact, 0, rv.Len())
	for i := range rv.Len() {
		element := rv.Index(i)
		for element.Kind() == reflect.Pointer || element.Kind() == reflect.Interface {
			if element.IsNil() {
				break
			}
			element = element.Elem()
		}
		if element.Kind() != reflect.Struct {
			continue
		}
		out = append(
			out,
			Fact{Label: stringField(element, "Label"), Value: stringField(element, "Value")},
		)
	}
	return out
}

func stringField(rv reflect.Value, name string) string {
	field := rv.FieldByName(name)
	if !field.IsValid() || field.Kind() != reflect.String {
		return ""
	}
	return strings.TrimSpace(field.String())
}

func reflectedCells(value any) []Cell {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return []Cell{reflectedCell(rv)}
	}
	out := make([]Cell, 0, rv.Len())
	for i := range rv.Len() {
		out = append(out, reflectedCell(rv.Index(i)))
	}
	return out
}

func reflectedCell(rv reflect.Value) Cell {
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return Cell{}
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return Cell{Text: textOf(fmt.Sprint(rv.Interface()))}
	}
	out := Cell{}
	for _, name := range []string{"Text", "Label"} {
		if field := rv.FieldByName(name); field.IsValid() && field.Kind() == reflect.String {
			out.Text = strings.TrimSpace(field.String())
			break
		}
	}
	if field := rv.FieldByName("Numeric"); field.IsValid() && field.Kind() == reflect.Bool {
		out.Numeric = field.Bool()
	}
	return out
}
