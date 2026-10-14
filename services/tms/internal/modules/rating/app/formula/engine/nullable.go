package engine

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/errors"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/expr-lang/expr/ast"
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

	objects := nullableObjects(definition.Properties, "")
	baselineRuns := e.probeRuns(ctx, expression, env, definition)

	warnings := make([]NullableFieldWarning, 0, len(fields))
	for _, field := range fields {
		if !e.breaksWithoutValue(ctx, &nullProbe{
			expression:   expression,
			env:          env,
			field:        field,
			objects:      objects,
			definition:   definition,
			baselineRuns: baselineRuns,
		}) {
			continue
		}
		warnings = append(warnings, NullableFieldWarning{
			Field:      field,
			Type:       candidates[field],
			Suggestion: guardFor(field, candidates[field]),
		})
	}

	return warnings, nil
}

type nullProbe struct {
	expression   string
	env          map[string]any
	field        string
	objects      map[string]formulatypes.Property
	definition   *formulatypes.Definition
	baselineRuns bool
}

func (e *Engine) breaksWithoutValue(ctx context.Context, p *nullProbe) bool {
	parent, nested := nullableParent(p.field, p.objects)
	if !nested {
		probe := cloneAlongPath(p.env, p.field)
		formulatypes.SetNestedValue(probe, p.field, nil)
		probe[ctxEnvKey] = ctx

		_, compileErr := e.Compile(p.expression, probe)
		return compileErr != nil
	}

	if !p.baselineRuns {
		return false
	}

	probe := cloneAlongPath(p.env, parent)
	formulatypes.SetNestedValue(probe, parent, nil)

	return !e.probeRuns(ctx, p.expression, probe, p.definition)
}

func (e *Engine) probeRuns(
	ctx context.Context,
	expression string,
	env map[string]any,
	definition *formulatypes.Definition,
) bool {
	probe := maps.Clone(env)
	shape := e.shapeFor(ctx, definition, probe, StubLookup{})
	_, err := e.runShaped(ctx, expression, probe, shape, nil)
	return err == nil
}

// guardFor is the smallest edit that keeps a formula pricing when the field is
// empty: substitute the type's zero.
func guardFor(field, fieldType string) string {
	zero := guardZero(fieldType)
	if strings.Contains(field, ".") {
		return strings.ReplaceAll(field, ".", "?.") + " ?? " + zero
	}
	return "coalesce(" + field + ", " + zero + ")"
}

func guardZero(fieldType string) string {
	switch fieldType {
	case "boolean":
		return "false"
	case "string":
		return "\"\""
	case "datetime":
		return "now()"
	default:
		return "0"
	}
}

func nullablePaths(properties map[string]formulatypes.Property, prefix string) map[string]string {
	return nullablePathsUnder(properties, prefix, false)
}

func nullablePathsUnder(
	properties map[string]formulatypes.Property,
	prefix string,
	optionalParent bool,
) map[string]string {
	paths := make(map[string]string, len(properties))

	for name, prop := range properties {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		if len(prop.Properties) > 0 {
			_, objectNullable := propertyTypeInfo(prop)
			maps.Copy(
				paths,
				nullablePathsUnder(prop.Properties, path, optionalParent || objectNullable),
			)
			continue
		}

		if fieldType, nullable := propertyTypeInfo(prop); nullable || optionalParent {
			paths[path] = fieldType
		}
	}

	return paths
}

func nullableObjects(
	properties map[string]formulatypes.Property,
	prefix string,
) map[string]formulatypes.Property {
	objects := make(map[string]formulatypes.Property)

	for name := range properties {
		prop := properties[name]
		if len(prop.Properties) == 0 {
			continue
		}

		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		if _, nullable := propertyTypeInfo(prop); nullable {
			objects[path] = prop
		}
		maps.Copy(objects, nullableObjects(prop.Properties, path))
	}

	return objects
}

func nullableParent(field string, objects map[string]formulatypes.Property) (string, bool) {
	for parent := field; ; {
		index := strings.LastIndex(parent, ".")
		if index < 0 {
			return "", false
		}
		parent = parent[:index]
		if _, ok := objects[parent]; ok {
			return parent, true
		}
	}
}

func propertyTypeInfo(prop formulatypes.Property) (string, bool) {
	fieldType, nullable := rawPropertyTypeInfo(prop)
	if prop.IsDateTime() {
		return "datetime", nullable
	}
	return fieldType, nullable
}

func rawPropertyTypeInfo(prop formulatypes.Property) (string, bool) {
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

func ReferencesPath(expression, path string) bool {
	_, ok := referencedPaths(expression)[path]
	return ok
}

// referencedPaths lists every identifier and dotted member path an expression
// names. A parse failure yields nothing: compile validation reports it.
func referencedPaths(expression string) map[string]struct{} {
	tree, err := parseExpression(expression)
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
		if value == nil {
			return nil, true
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

func unresolvedFieldError(
	expressions []string,
	env map[string]any,
	resolveFailures map[string]error,
) error {
	if len(resolveFailures) == 0 {
		return nil
	}

	fields := make([]string, 0, len(resolveFailures))
	causes := make(map[string]error, len(resolveFailures))
	for _, expression := range expressions {
		for path := range referencedPaths(expression) {
			cause, failed := resolveFailures[path]
			if !failed {
				continue
			}
			if _, seen := causes[path]; seen {
				continue
			}
			if value, ok := nestedValue(env, path); ok && value != nil {
				continue
			}
			fields = append(fields, path)
			causes[path] = cause
		}
	}
	if len(fields) == 0 {
		return nil
	}
	sort.Strings(fields)

	return errors.NewUnresolvedFieldError(fields, causes)
}

func breakdownExpressions(
	expression string,
	definitions []*formulatypes.BreakdownDefinition,
) []string {
	expressions := make([]string, 0, len(definitions)+1)
	expressions = append(expressions, expression)
	for _, definition := range definitions {
		if definition != nil {
			expressions = append(expressions, definition.Expression)
		}
	}
	return expressions
}
