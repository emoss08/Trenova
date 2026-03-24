package redis

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/emoss08/gtc/internal/core/domain"
)

type Template struct {
	pattern string
	tmpl    *template.Template
}

func ParseTemplate(pattern string) (*Template, error) {
	tmpl, err := template.New("projection").Funcs(template.FuncMap{
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
	}).Parse(pattern)
	if err != nil {
		return nil, fmt.Errorf("parse template %q: %w", pattern, err)
	}

	return &Template{pattern: pattern, tmpl: tmpl}, nil
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

func (t *Template) Execute(record domain.SourceRecord, primaryKeys []string) (string, error) {
	data := struct {
		Schema      string
		Table       string
		PrimaryKeys []string
		New         map[string]any
		Old         map[string]any
		Meta        domain.RecordMetadata
	}{
		Schema:      record.Schema,
		Table:       record.Table,
		PrimaryKeys: primaryKeys,
		New:         record.NewData,
		Old:         record.OldData,
		Meta:        record.Metadata,
	}

	var buf bytes.Buffer
	if err := t.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %q: %w", t.pattern, err)
	}

	return buf.String(), nil
}
