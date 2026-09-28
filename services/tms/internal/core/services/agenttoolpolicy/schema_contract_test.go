package agenttoolpolicy_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
)

// domainOwnedSchemas are the subtrees of a tool's schema whose shape another
// domain declares and checks itself, keyed by "tool:path", with the reason.
// Everything under one is left to that domain; nothing else is exempt.
var domainOwnedSchemas = map[string]string{
	"create_report:definition": "report.DefinitionJSONSchema is the report domain's own " +
		"schema; the report compiler refuses a definition it cannot compile",
	"update_report:definition":  "as create_report",
	"preview_report:definition": "as create_report",
	"run_report:parameters": "keyed by the parameters each saved report declares; the " +
		"report compiler checks every value against its declaration",
	"preview_report:parameters": "as run_report",
	"test_formula_expression:variables": "keyed by the variables each formula declares; " +
		"the formula engine checks every value against its declared type",
	"test_formula_expression:scenarios[].variables": "as test_formula_expression:variables",
	"propose_formula:scenarios[].variables":         "as test_formula_expression:variables",
}

func domainOwned(tool, path string) bool {
	for key := range domainOwnedSchemas {
		owner, prefix, _ := strings.Cut(key, ":")
		if owner == tool && (path == prefix || strings.HasPrefix(path, prefix+".") ||
			strings.HasPrefix(path, prefix+"[")) {
			return true
		}
	}

	return false
}

type schemaTool interface {
	Name() string
	ParamSchema() map[string]any
}

func registeredSchemas(t *testing.T) []schemaTool {
	t.Helper()

	tools := buildRegistered(t)
	out := make([]schemaTool, 0, len(tools.Queries)+len(tools.Actions))
	for _, tool := range tools.Queries {
		out = append(out, tool)
	}
	for _, tool := range tools.Actions {
		out = append(out, tool)
	}

	return out
}

func isObject(node map[string]any) bool {
	if _, declares := node[toolschema.KeyProperties]; declares {
		return true
	}
	switch declared := node[toolschema.KeyType].(type) {
	case string:
		return declared == toolschema.TypeObject
	case []string:
		for _, name := range declared {
			if name == toolschema.TypeObject {
				return true
			}
		}
	case []any:
		for _, name := range declared {
			if name == toolschema.TypeObject {
				return true
			}
		}
	}

	return false
}

func closed(node map[string]any) bool {
	switch additional := node[toolschema.KeyAdditionalProperties].(type) {
	case bool:
		return !additional
	case map[string]any:
		return true
	default:
		return false
	}
}

func enumValues(raw any) ([]string, bool) {
	switch values := raw.(type) {
	case []string:
		return values, true
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			out = append(out, fmt.Sprint(value))
		}

		return out, true
	default:
		return nil, false
	}
}

// Every object a tool's schema declares, at any depth, says what it takes
// and refuses the rest. An open object is where a model's invented field
// used to vanish: moves[].type was sent, never read, and never refused.
func TestEveryObjectInAToolSchemaIsClosed(t *testing.T) {
	t.Parallel()

	for _, tool := range registeredSchemas(t) {
		toolschema.Walk(tool.ParamSchema(), func(path string, node map[string]any) {
			if !isObject(node) || domainOwned(tool.Name(), path) {
				return
			}
			if !closed(node) {
				t.Errorf("%s: the object at %q takes anything; declare its fields and "+
					"set additionalProperties: false", tool.Name(), pathOrRoot(path))
			}
		})
	}
}

// Every enum a tool's schema lists names the registered source it was taken
// from, and lists exactly that source's values, so a list cannot drift from
// the domain that owns it: the comment priorities used to omit Urgent.
func TestEveryEnumNamesItsSource(t *testing.T) {
	t.Parallel()

	for _, tool := range registeredSchemas(t) {
		toolschema.Walk(tool.ParamSchema(), func(path string, node map[string]any) {
			listed, hasEnum := enumValues(node[toolschema.KeyEnum])
			if !hasEnum || domainOwned(tool.Name(), path) {
				return
			}
			source, named := node[toolschema.KeyEnumOf].(string)
			if !named || source == "" {
				t.Errorf("%s: the enum at %q names no source; build it with "+
					"agenttoolschema.Enum", tool.Name(), pathOrRoot(path))

				return
			}
			registered, found := agenttoolschema.Registered(source)
			if !found {
				t.Errorf("%s: the enum at %q names %q, which is not a registered source",
					tool.Name(), pathOrRoot(path), source)

				return
			}
			assert.Equal(t, registered, listed,
				"%s: the enum at %q differs from its source %q",
				tool.Name(), pathOrRoot(path), source)
		})
	}
}

func pathOrRoot(path string) string {
	if path == "" {
		return "(root)"
	}

	return path
}
