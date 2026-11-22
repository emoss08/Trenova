package emailpreview

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/emoss08/trenova/shared/i18n"
)

type pageLink struct {
	Locale  string
	Subject string
	HTML    string
	Text    string
}

type pageCase struct {
	Kind  string
	Name  string
	Links []pageLink
}

type pageGroup struct {
	Name  string
	Cases []*pageCase
}

var indexTemplate = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Email preview</title>
<style>
  body { margin: 0; padding: 32px 16px 64px; background: #f7f6f2; color: #121210;
    font-family: Geist, -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif; }
  main { max-width: 880px; margin: 0 auto; }
  h1 { font-size: 28px; letter-spacing: -0.6px; margin: 0 0 8px; }
  p.lead { color: #55544e; margin: 0 0 32px; }
  h2 { font-family: 'Geist Mono', ui-monospace, Menlo, Consolas, monospace; font-size: 12px;
    letter-spacing: 1px; text-transform: uppercase; color: #55544e; margin: 32px 0 10px; }
  table { width: 100%; border-collapse: collapse; border-top: 1px solid #121210; background: #fff; }
  td { padding: 10px 12px; border-bottom: 1px solid #dddbd3; vertical-align: top; font-size: 14px; }
  td.kind { font-family: 'Geist Mono', ui-monospace, Menlo, Consolas, monospace; font-size: 12px; color: #55544e; white-space: nowrap; }
  a { color: #121210; text-underline-offset: 3px; }
  .subject { color: #55544e; }
</style>
</head>
<body>
<main>
<h1>Email preview</h1>
<p class="lead">Every email the TMS sends, rendered from fixed sample data through the same templates and shell as a real send.</p>
{{ range .Groups }}<h2>[ {{ .Name }} ]</h2>
<table>
{{ range .Cases }}<tr>
<td class="kind">{{ .Kind }}</td>
<td><strong>{{ .Name }}</strong>{{ range .Links }}<br><span class="subject">{{ .Locale }}: {{ .Subject }}</span> <a href="{{ .HTML }}">HTML</a> <a href="{{ .Text }}">text</a>{{ end }}</td>
</tr>
{{ end }}</table>
{{ end }}</main>
</body>
</html>
`))

func Write(ctx context.Context, dir string, locales []i18n.Locale) error {
	cases, err := Catalog()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	renderer := NewRenderer()

	groups := make([]*pageGroup, 0, 8)
	byName := make(map[string]*pageGroup, 8)
	for _, c := range cases {
		group, ok := byName[c.Group]
		if !ok {
			group = &pageGroup{Name: c.Group}
			byName[c.Group] = group
			groups = append(groups, group)
		}
		page := &pageCase{Kind: string(c.Kind), Name: c.Name}
		for _, locale := range locales {
			rendered, renderErr := renderer.Render(ctx, c, locale)
			if renderErr != nil {
				return fmt.Errorf("%s in %s: %w", c.Slug, locale, renderErr)
			}
			base := filepath.Join(string(locale), c.Slug)
			if writeErr := writeFile(dir, base+".html", rendered.HTML); writeErr != nil {
				return writeErr
			}
			if writeErr := writeFile(dir, base+".txt", rendered.Text); writeErr != nil {
				return writeErr
			}
			page.Links = append(page.Links, pageLink{
				Locale:  string(locale),
				Subject: rendered.Subject,
				HTML:    filepath.ToSlash(base + ".html"),
				Text:    filepath.ToSlash(base + ".txt"),
			})
		}
		group.Cases = append(group.Cases, page)
	}

	index, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		return err
	}
	if err = indexTemplate.Execute(index, struct{ Groups []*pageGroup }{groups}); err != nil {
		_ = index.Close()
		return err
	}
	return index.Close()
}

func writeFile(dir, name, body string) error {
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o600)
}
