package engine

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/services/formula/errors"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

// NullableFieldWarning names a schema field that can be empty on a real record
// and that an expression uses without a guard. Such an expression compiles
// and previews fine against sample data, then fails to rate the first
// shipment that arrives without the field.
type NullableFieldWarning struct {
	Field      string
	Type       string
	Suggestion string
}

// UnguardedNullableFields finds the nullable fields an expression would break
// on. For each nullable field the expression references it compiles once more
// against an environment where only that field is nil; a compile failure means
// the field reaches an operator that cannot take nothing.
func (e *Engine) UnguardedNullableFields(
	ctx context.Context,
	expression string,
	schemaID string,
	variables map[string]any,
) ([]NullableFieldWarning, error) {
	definition, ok := e.registry.Get(schemaID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrSchemaNotFound, schemaID)
	}

	referenced := referencedPaths(expression)
	if len(referenced) == 0 {
		return nil, nil
	}

	candidates := nullablePaths(definition.Properties, "")
	fields := make([]string, 0, len(candidates))
	for path := range candidates {
		if _, used := referenced[path]; used {
			fields = append(fields, path)
		}
	}
	if len(fields) == 0 {
		return nil, nil
	}
	sort.Strings(fields)

	env, _, err := e.envBuilder.BuildValidationEnvironment(schemaID, variables)
	if err != nil {
		return nil, err
	}
	injectLookupFunctions(env, StubLookup{})

	warnings := make([]NullableFieldWarning, 0, len(fields))
	for _, field := range fields {
		probe := cloneAlongPath(env, field)
		formulatypes.SetNestedValue(probe, field, nil)
		probe[ctxEnvKey] = ctx

		if _, compileErr := e.Compile(expression, probe); compileErr != nil {
			warnings = append(warnings, NullableFieldWarning{
				Field:      field,
				Type:       candidates[field],
				Suggestion: guardFor(field, candidates[field]),
			})
		}
	}

	return warnings, nil
}

// guardFor is the smallest edit that keeps a formula pricing when the field is
// empty: substitute the type's zero.
func guardFor(field, fieldType string) string {
	switch fieldType {
	case "boolean":
		return "coalesce(" + field + ", false)"
	case "string":
		return "coalesce(" + field + ", \"\")"
	default:
		return "coalesce(" + field + ", 0)"
	}
}

func nullablePaths(properties map[string]formulatypes.Property, prefix string) map[string]string {
	paths := make(map[string]string, len(properties))

	for name, prop := range properties {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		if len(prop.Properties) > 0 {
			maps.Copy(paths, nullablePaths(prop.Properties, path))
			continue
		}

		if fieldType, nullable := propertyTypeInfo(prop); nullable {
			paths[path] = fieldType
		}
	}

	return paths
}

func propertyTypeInfo(prop formulatypes.Property) (string, bool) {
	nullable := prop.Source.Nullable

	switch typed := prop.Type.(type) {
	case string:
		return typed, nullable
	case []any:
		resolved := ""
		for _, entry := range typed {
			value, ok := entry.(string)
			if !ok {
				continue
			}
			if value == "null" {
				nullable = true
				continue
			}
			if resolved == "" {
				resolved = value
			}
		}
		return resolved, nullable
	default:
		return "", nullable
	}
}

// referencedPaths lists every identifier and dotted member path an expression
// names. A parse failure yields nothing: compile validation reports it.
func referencedPaths(expression string) map[string]struct{} {
	tree, err := parser.Parse(expression)
	if err != nil {
		return nil
	}

	visitor := &pathVisitor{paths: make(map[string]struct{}, 8)}
	ast.Walk(&tree.Node, visitor)

	return visitor.paths
}

type pathVisitor struct {
	paths map[string]struct{}
}

//nolint:gocritic // expr ast.Visitor requires the pointer signature
func (v *pathVisitor) Visit(node *ast.Node) {
	switch n := (*node).(type) {
	case *ast.IdentifierNode:
		v.paths[n.Value] = struct{}{}
	case *ast.MemberNode:
		if path, ok := memberPath(n); ok {
			v.paths[path] = struct{}{}
		}
	}
}

func memberPath(node *ast.MemberNode) (string, bool) {
	property, ok := node.Property.(*ast.StringNode)
	if !ok {
		return "", false
	}

	switch base := node.Node.(type) {
	case *ast.IdentifierNode:
		return base.Value + "." + property.Value, true
	case *ast.MemberNode:
		if parent, ok := memberPath(base); ok {
			return parent + "." + property.Value, true
		}
	}

	return "", false
}

// nilReferencedFields names the referenced paths that hold nil in env, which
// is what turns a compile failure on a real record into an explanation. A
// path that failed to resolve is left out: that is a schema or data problem
// with its own message, not a field the author forgot to guard.
func nilReferencedFields(
	expression string,
	env map[string]any,
	resolveFailures map[string]error,
) []string {
	fields := make([]string, 0, 2)
	for path := range referencedPaths(expression) {
		if _, failed := resolveFailures[path]; failed {
			continue
		}
		if value, ok := nestedValue(env, path); ok && value == nil {
			fields = append(fields, path)
		}
	}
	sort.Strings(fields)

	return fields
}

func nestedValue(env map[string]any, path string) (any, bool) {
	segments := strings.Split(path, ".")
	current := env
	for i, segment := range segments {
		value, ok := current[segment]
		if !ok {
			return nil, false
		}
		if i == len(segments)-1 {
			return value, true
		}
		next, isMap := value.(map[string]any)
		if !isMap {
			return nil, false
		}
		current = next
	}

	return nil, false
}

// cloneAlongPath copies env and every nested map on the way to path, so a
// probe can nil one field without disturbing the environment it came from.
func cloneAlongPath(env map[string]any, path string) map[string]any {
	clone := maps.Clone(env)
	segments := strings.Split(path, ".")

	current := clone
	for _, segment := range segments[:len(segments)-1] {
		nested, ok := current[segment].(map[string]any)
		if !ok {
			return clone
		}
		copied := maps.Clone(nested)
		current[segment] = copied
		current = copied
	}

	return clone
}

func missingFieldError(
	expression string,
	env map[string]any,
	resolveFailures map[string]error,
) error {
	fields := nilReferencedFields(expression, env, resolveFailures)
	if len(fields) == 0 {
		return nil
	}

	suggestions := make([]string, 0, len(fields))
	for _, field := range fields {
		suggestions = append(suggestions, guardFor(field, ""))
	}

	return errors.NewMissingFieldError(expression, fields, suggestions)
}
