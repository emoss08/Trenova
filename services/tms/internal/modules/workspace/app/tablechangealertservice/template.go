package tablechangealertservice

import (
	"fmt"
	"regexp"
	"strings"
)

var templatePattern = regexp.MustCompile(`\{\{(\w+(?:\.\w+)?)\}\}`)

func RenderTemplate(
	tmpl string,
	table string,
	operation string,
	recordID string,
	newData map[string]any,
	oldData map[string]any,
	changedFields []string,
) string {
	if tmpl == "" {
		return ""
	}

	return templatePattern.ReplaceAllStringFunc(tmpl, func(match string) string {
		key := match[2 : len(match)-2]

		switch key {
		case "table":
			return table
		case "operation":
			return operation
		case "record_id":
			return recordID
		case "changed_fields":
			return strings.Join(changedFields, ", ")
		default:
			if strings.HasPrefix(key, "new.") {
				field := key[4:]
				v, ok := newData[field]
				if !ok || v == nil {
					return ""
				}
				return fmt.Sprintf("%v", v)
			}
			if strings.HasPrefix(key, "old.") {
				field := key[4:]
				v, ok := oldData[field]
				if !ok || v == nil {
					return ""
				}
				return fmt.Sprintf("%v", v)
			}
			return match
		}
	})
}
