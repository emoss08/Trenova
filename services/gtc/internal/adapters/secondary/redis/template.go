package redis

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/emoss08/gtc/internal/core/domain"
)

const wildcardSentinel = "\x00"

type Template struct {
	pattern  string
	tmpl     *template.Template
	wildcard *template.Template
}

func ParseTemplate(pattern string) (*Template, error) {
	tmpl, err := template.New("projection").Funcs(templateFuncs()).Parse(pattern)
	if err != nil {
		return nil, fmt.Errorf("parse template %q: %w", pattern, err)
	}

	wildcard, err := template.New("projection_wildcard").Funcs(wildcardFuncs()).Parse(pattern)
	if err != nil {
		return nil, fmt.Errorf("parse wildcard template %q: %w", pattern, err)
	}

	return &Template{pattern: pattern, tmpl: tmpl, wildcard: wildcard}, nil
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"field": func(name string, data map[string]any) string {
			if data == nil {
				return ""
			}
			if value, ok := data[name]; ok {
				return fmt.Sprintf("%v", value)
			}
			return ""
		},
		"value": func(name string, newData map[string]any, oldData map[string]any) string {
			if value, ok := lookupField(name, newData); ok {
				return value
			}
			if value, ok := lookupField(name, oldData); ok {
				return value
			}
			return ""
		},
		"key": func(names []string, newData map[string]any, oldData map[string]any) string {
			parts := make([]string, 0, len(names))
			for _, name := range names {
				if value, ok := lookupField(name, newData); ok {
					parts = append(parts, value)
					continue
				}
				if value, ok := lookupField(name, oldData); ok {
					parts = append(parts, value)
				}
			}
			return strings.Join(parts, ":")
		},
	}
}

func wildcardFuncs() template.FuncMap {
	return template.FuncMap{
		"field": func(string, map[string]any) string {
			return wildcardSentinel
		},
		"value": func(string, map[string]any, map[string]any) string {
			return wildcardSentinel
		},
		"key": func([]string, map[string]any, map[string]any) string {
			return wildcardSentinel
		},
	}
}

func lookupField(name string, data map[string]any) (string, bool) {
	if data == nil {
		return "", false
	}
	if value, ok := data[name]; ok {
		return fmt.Sprintf("%v", value), true
	}
	return "", false
}

type templateData struct {
	Schema      string
	Table       string
	PrimaryKeys []string
	New         map[string]any
	Old         map[string]any
	Meta        domain.RecordMetadata
}

func newTemplateData(record domain.SourceRecord, primaryKeys []string) templateData {
	return templateData{
		Schema:      record.Schema,
		Table:       record.Table,
		PrimaryKeys: primaryKeys,
		New:         record.NewData,
		Old:         record.OldData,
		Meta:        record.Metadata,
	}
}

func (t *Template) Execute(record domain.SourceRecord, primaryKeys []string) (string, error) {
	var buf bytes.Buffer
	if err := t.tmpl.Execute(&buf, newTemplateData(record, primaryKeys)); err != nil {
		return "", fmt.Errorf("execute template %q: %w", t.pattern, err)
	}

	return buf.String(), nil
}

func (t *Template) WildcardPattern(
	record domain.SourceRecord,
	primaryKeys []string,
) (string, error) {
	var buf bytes.Buffer
	if err := t.wildcard.Execute(&buf, newTemplateData(record, primaryKeys)); err != nil {
		return "", fmt.Errorf("execute wildcard template %q: %w", t.pattern, err)
	}

	pattern, hasLiteral := globPattern(buf.String())
	if !hasLiteral {
		return "", fmt.Errorf(
			"template %q renders no literal key content to anchor a wildcard delete",
			t.pattern,
		)
	}

	return pattern, nil
}

func globPattern(rendered string) (string, bool) {
	var builder strings.Builder
	builder.Grow(len(rendered))
	hasLiteral := false

	for _, char := range rendered {
		switch char {
		case rune(0):
			builder.WriteByte('*')
		case '*', '?', '[', ']', '\\':
			builder.WriteByte('\\')
			builder.WriteRune(char)
			hasLiteral = true
		default:
			builder.WriteRune(char)
			hasLiteral = true
		}
	}

	return builder.String(), hasLiteral
}
