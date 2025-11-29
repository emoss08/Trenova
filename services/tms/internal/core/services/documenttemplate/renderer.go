package documenttemplate

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/infrastructure/gotenberg"
	"github.com/shopspring/decimal"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type Renderer struct {
	gotenberg *gotenberg.Client
}

func NewRenderer(g *gotenberg.Client) *Renderer {
	return &Renderer{
		gotenberg: g,
	}
}

func (r *Renderer) RenderHTML(tmpl *documenttemplate.DocumentTemplate, data any) ([]byte, error) {
	funcMap := r.buildFuncMap()

	htmlTmpl, err := template.New("main").Funcs(funcMap).Parse(tmpl.HTMLContent)
	if err != nil {
		return nil, fmt.Errorf("parse html template: %w", err)
	}

	var buf bytes.Buffer
	if err = htmlTmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute html template: %w", err)
	}

	finalHTML := r.wrapWithCSS(buf.String(), tmpl.CSSContent)

	return []byte(finalHTML), nil
}

func (r *Renderer) RenderHeaderHTML(
	tmpl *documenttemplate.DocumentTemplate,
	data any,
) ([]byte, error) {
	if tmpl.HeaderHTML == "" {
		return nil, nil
	}

	funcMap := r.buildFuncMap()

	headerTmpl, err := template.New("header").Funcs(funcMap).Parse(tmpl.HeaderHTML)
	if err != nil {
		return nil, fmt.Errorf("parse header template: %w", err)
	}

	var buf bytes.Buffer
	if err = headerTmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute header template: %w", err)
	}

	return buf.Bytes(), nil
}

func (r *Renderer) RenderFooterHTML(
	tmpl *documenttemplate.DocumentTemplate,
	data any,
) ([]byte, error) {
	if tmpl.FooterHTML == "" {
		return nil, nil
	}

	funcMap := r.buildFuncMap()

	footerTmpl, err := template.New("footer").Funcs(funcMap).Parse(tmpl.FooterHTML)
	if err != nil {
		return nil, fmt.Errorf("parse footer template: %w", err)
	}

	var buf bytes.Buffer
	if err = footerTmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute footer template: %w", err)
	}

	return buf.Bytes(), nil
}

func (r *Renderer) wrapWithCSS(html, css string) string {
	if css == "" {
		return html
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<style>%s</style>
</head>
<body>%s</body>
</html>`, css, html)
}

func (r *Renderer) buildFuncMap() template.FuncMap {
	printer := message.NewPrinter(language.English)

	return template.FuncMap{
		"formatDate": func(ts int64, layout string) string {
			if ts == 0 {
				return ""
			}
			return time.Unix(ts, 0).Format(layout)
		},
		"formatDateDefault": func(ts int64) string {
			if ts == 0 {
				return ""
			}
			return time.Unix(ts, 0).Format("01/02/2006")
		},
		"formatDateTime": func(ts int64) string {
			if ts == 0 {
				return ""
			}
			return time.Unix(ts, 0).Format("01/02/2006 03:04 PM")
		},
		"formatCurrency": func(amount decimal.Decimal) string {
			f, _ := amount.Float64()
			return printer.Sprintf("$%.2f", f)
		},
		"formatCurrencyFromFloat": func(amount float64) string {
			return printer.Sprintf("$%.2f", amount)
		},
		"formatNumber": func(n any) string {
			switch v := n.(type) {
			case int:
				return printer.Sprintf("%d", v)
			case int64:
				return printer.Sprintf("%d", v)
			case float64:
				return printer.Sprintf("%.2f", v)
			case decimal.Decimal:
				f, _ := v.Float64()
				return printer.Sprintf("%.2f", f)
			default:
				return fmt.Sprintf("%v", n)
			}
		},
		"formatWeight": func(weight decimal.Decimal) string {
			f, _ := weight.Float64()
			return printer.Sprintf("%.0f lbs", f)
		},
		"formatMiles": func(miles decimal.Decimal) string {
			f, _ := miles.Float64()
			return printer.Sprintf("%.1f mi", f)
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"trim":  strings.TrimSpace,
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"addDecimal": func(a, b decimal.Decimal) decimal.Decimal {
			return a.Add(b)
		},
		"subDecimal": func(a, b decimal.Decimal) decimal.Decimal {
			return a.Sub(b)
		},
		"mulDecimal": func(a, b decimal.Decimal) decimal.Decimal {
			return a.Mul(b)
		},
		"divDecimal": func(a, b decimal.Decimal) decimal.Decimal {
			if b.IsZero() {
				return decimal.Zero
			}
			return a.Div(b)
		},
		"eq": func(a, b any) bool {
			return a == b
		},
		"ne": func(a, b any) bool {
			return a != b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"le": func(a, b int) bool {
			return a <= b
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"ge": func(a, b int) bool {
			return a >= b
		},
		"and": func(a, b bool) bool {
			return a && b
		},
		"or": func(a, b bool) bool {
			return a || b
		},
		"not": func(a bool) bool {
			return !a
		},
		"default": func(defaultVal, val any) any {
			if val == nil || val == "" || val == 0 {
				return defaultVal
			}
			return val
		},
		"join": func(sep string, items []string) string {
			return strings.Join(items, sep)
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"hasPrefix": func(s, prefix string) bool {
			return strings.HasPrefix(s, prefix)
		},
		"hasSuffix": func(s, suffix string) bool {
			return strings.HasSuffix(s, suffix)
		},
		"replace": func(s, old, new string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"repeat": func(s string, count int) string {
			return strings.Repeat(s, count)
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"nl2br": func(s string) template.HTML {
			return template.HTML(strings.ReplaceAll(s, "\n", "<br>"))
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"now": func() int64 {
			return time.Now().Unix()
		},
		"seq": func(start, end int) []int {
			result := make([]int, 0, end-start+1)
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
			return result
		},
		"dict": func(values ...any) map[string]any {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				dict[key] = values[i+1]
			}
			return dict
		},
		"list": func(values ...any) []any {
			return values
		},
		"first": func(items any) any {
			switch v := items.(type) {
			case []string:
				if len(v) > 0 {
					return v[0]
				}
			case []any:
				if len(v) > 0 {
					return v[0]
				}
			}
			return nil
		},
		"last": func(items any) any {
			switch v := items.(type) {
			case []string:
				if len(v) > 0 {
					return v[len(v)-1]
				}
			case []any:
				if len(v) > 0 {
					return v[len(v)-1]
				}
			}
			return nil
		},
		"len": func(items any) int {
			switch v := items.(type) {
			case []string:
				return len(v)
			case []any:
				return len(v)
			case string:
				return len(v)
			case map[string]any:
				return len(v)
			default:
				return 0
			}
		},
	}
}

func (r *Renderer) GetPDFOptions(tmpl *documenttemplate.DocumentTemplate) gotenberg.PDFOptions {
	opts := gotenberg.PDFOptions{
		MarginTop:    float64(tmpl.MarginTop) / 25.4,
		MarginBottom: float64(tmpl.MarginBottom) / 25.4,
		MarginLeft:   float64(tmpl.MarginLeft) / 25.4,
		MarginRight:  float64(tmpl.MarginRight) / 25.4,
		Landscape:    tmpl.Orientation == documenttemplate.OrientationLandscape,
	}

	switch tmpl.PageSize {
	case documenttemplate.PageSizeLetter:
		opts.PageWidth = 8.5
		opts.PageHeight = 11
	case documenttemplate.PageSizeA4:
		opts.PageWidth = 8.27
		opts.PageHeight = 11.69
	case documenttemplate.PageSizeLegal:
		opts.PageWidth = 8.5
		opts.PageHeight = 14
	}

	return opts
}
