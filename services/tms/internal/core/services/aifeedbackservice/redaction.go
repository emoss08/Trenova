package aifeedbackservice

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

var jsonPairPattern = regexp.MustCompile(
	`"([A-Za-z_][A-Za-z0-9_]*)"\s*:\s*("(?:[^"\\]|\\.)*"|-?\d+(?:\.\d+)?)`,
)

type toolResult struct {
	name    string
	content string
}

type toolResourceLookup interface {
	Resource(name string) (permission.Resource, bool)
}

type sensitivityLookup interface {
	GetFieldSensitivity(resource, field string) permission.FieldSensitivity
}

type toolResources struct {
	tools   services.AgentToolRegistry
	queries services.AgentQueryToolRegistry
}

func newToolResources(
	tools services.AgentToolRegistry,
	queries services.AgentQueryToolRegistry,
) toolResources {
	return toolResources{tools: tools, queries: queries}
}

func (t toolResources) Resource(name string) (permission.Resource, bool) {
	if t.queries != nil {
		if tool, ok := t.queries.Get(name); ok {
			return tool.Policy().Resource, true
		}
	}
	if t.tools != nil {
		if tool, ok := t.tools.Get(name); ok {
			return tool.Policy().Resource, true
		}
	}

	return "", false
}

type redactor struct {
	tools    toolResourceLookup
	registry sensitivityLookup
}

func newRedactor(tools toolResourceLookup) *redactor {
	return &redactor{tools: tools, registry: permission.NewRegistry()}
}

func (r *redactor) restrictedValues(results []toolResult) []string {
	if r == nil || r.tools == nil || r.registry == nil {
		return nil
	}

	values := make([]string, 0, len(results))
	for _, result := range results {
		resource, ok := r.tools.Resource(result.name)
		if !ok || resource == "" {
			continue
		}

		body := jsonBody(result.content)
		if body == "" {
			continue
		}

		var payload any
		if err := sonic.UnmarshalString(body, &payload); err == nil {
			r.walk(resource, payload, false, &values)
			continue
		}
		r.scan(resource, body, &values)
	}

	return values
}

func (r *redactor) restricted(resource permission.Resource, field string) bool {
	return r.registry.GetFieldSensitivity(resource.String(), field).Level() >=
		permission.SensitivityRestricted.Level()
}

func (r *redactor) walk(
	resource permission.Resource,
	node any,
	restricted bool,
	out *[]string,
) {
	switch value := node.(type) {
	case map[string]any:
		for key, child := range value {
			r.walk(resource, child, restricted || r.restricted(resource, key), out)
		}
	case []any:
		for _, child := range value {
			r.walk(resource, child, restricted, out)
		}
	case string:
		if restricted {
			*out = append(*out, value)
		}
	case float64:
		if restricted {
			*out = append(*out, strconv.FormatFloat(value, 'f', -1, 64))
		}
	case int64:
		if restricted {
			*out = append(*out, strconv.FormatInt(value, 10))
		}
	}
}

func (r *redactor) scan(resource permission.Resource, body string, out *[]string) {
	for _, match := range jsonPairPattern.FindAllStringSubmatch(body, -1) {
		if !r.restricted(resource, match[1]) {
			continue
		}
		raw := match[2]
		if strings.HasPrefix(raw, `"`) {
			if unquoted, err := strconv.Unquote(raw); err == nil {
				raw = unquoted
			} else {
				raw = strings.Trim(raw, `"`)
			}
		}
		*out = append(*out, raw)
	}
}

func jsonBody(content string) string {
	start := strings.IndexAny(content, "{[")
	if start < 0 {
		return ""
	}
	end := strings.LastIndexAny(content, "}]")
	if end < start {
		return content[start:]
	}

	return content[start : end+1]
}
