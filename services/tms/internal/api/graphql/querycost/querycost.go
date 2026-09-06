package querycost

import (
	"math"
	"sort"
	"strings"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/vektah/gqlparser/v2/ast"
)

const (
	MaxOperationCost  = 750_000
	MaxOperationDepth = 12

	ComplexityLimitErrorCode = "COMPLEXITY_LIMIT_EXCEEDED"
	DepthLimitErrorCode      = "OPERATION_DEPTH_LIMIT_EXCEEDED"

	UnpaginatedListWeight = 5

	maxCost             = math.MaxInt32
	maxFragmentDepth    = 64
	connectionPageKey   = "first"
	introspectionPrefix = "__"
)

type Index struct {
	connectionTypes  map[string]struct{}
	connectionFields map[string]struct{}
	listFields       map[string]struct{}
	listWeight       int
}

func NewIndex(schema *ast.Schema) *Index {
	idx := &Index{
		connectionTypes:  make(map[string]struct{}),
		connectionFields: make(map[string]struct{}),
		listFields:       make(map[string]struct{}),
		listWeight:       UnpaginatedListWeight,
	}
	if schema == nil {
		return idx
	}

	for _, def := range schema.Types {
		if isConnectionType(def) {
			idx.connectionTypes[def.Name] = struct{}{}
		}
	}

	for _, def := range schema.Types {
		if def.Kind != ast.Object && def.Kind != ast.Interface {
			continue
		}
		for _, field := range def.Fields {
			key := def.Name + "." + field.Name
			if _, ok := idx.connectionTypes[namedType(field.Type)]; ok {
				idx.connectionFields[key] = struct{}{}
				continue
			}
			if isListType(field.Type) {
				idx.listFields[key] = struct{}{}
			}
		}
	}

	return idx
}

func (i *Index) Complexity(
	typeName, fieldName string,
	childComplexity int,
	args map[string]any,
) (int, bool) {
	if _, ok := i.connectionTypes[typeName]; ok {
		return 0, false
	}

	key := typeName + "." + fieldName
	if _, ok := i.connectionFields[key]; ok {
		return scale(childComplexity, pagination.ClampLimit(pageSize(args))), true
	}
	if _, ok := i.listFields[key]; ok {
		return scale(childComplexity, i.listWeight), true
	}

	return 0, false
}

func (i *Index) IsConnectionField(typeName, fieldName string) bool {
	_, ok := i.connectionFields[typeName+"."+fieldName]
	return ok
}

func (i *Index) IsListField(typeName, fieldName string) bool {
	_, ok := i.listFields[typeName+"."+fieldName]
	return ok
}

func (i *Index) IsConnectionType(typeName string) bool {
	_, ok := i.connectionTypes[typeName]
	return ok
}

func Depth(op *ast.OperationDefinition) int {
	if op == nil {
		return 0
	}

	return selectionDepth(op.SelectionSet, 0, 0)
}

func selectionDepth(set ast.SelectionSet, current, spreads int) int {
	if spreads > maxFragmentDepth {
		return current
	}

	deepest := current
	for _, selection := range set {
		switch s := selection.(type) {
		case *ast.Field:
			if strings.HasPrefix(s.Name, introspectionPrefix) {
				continue
			}
			depth := current + 1
			if len(s.SelectionSet) > 0 {
				depth = selectionDepth(s.SelectionSet, depth, spreads)
			}
			if depth > deepest {
				deepest = depth
			}
		case *ast.FragmentSpread:
			if s.Definition == nil {
				continue
			}
			if depth := selectionDepth(s.Definition.SelectionSet, current, spreads+1); depth > deepest {
				deepest = depth
			}
		case *ast.InlineFragment:
			if depth := selectionDepth(s.SelectionSet, current, spreads+1); depth > deepest {
				deepest = depth
			}
		}
	}

	return deepest
}

func isConnectionType(def *ast.Definition) bool {
	if def == nil || def.Kind != ast.Object {
		return false
	}
	if def.Fields.ForName("pageInfo") == nil {
		return false
	}

	return def.Fields.ForName("edges") != nil || def.Fields.ForName("nodes") != nil
}

func isListType(t *ast.Type) bool {
	for cur := t; cur != nil; cur = cur.Elem {
		if cur.Elem != nil {
			return true
		}
	}

	return false
}

func namedType(t *ast.Type) string {
	for cur := t; cur != nil; cur = cur.Elem {
		if cur.Elem == nil {
			return cur.NamedType
		}
	}

	return ""
}

func scale(childComplexity, multiplier int) int {
	child := childComplexity
	if child < 1 {
		child = 1
	}
	if multiplier < 1 {
		multiplier = 1
	}
	if child > maxCost/multiplier {
		return maxCost
	}

	return child * multiplier
}

func pageSize(args map[string]any) int {
	if value, ok := numeric(args[connectionPageKey]); ok {
		return value
	}

	keys := make([]string, 0, len(args))
	for key := range args {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		nested, ok := args[key].(map[string]any)
		if !ok {
			continue
		}
		if value, ok := numeric(nested[connectionPageKey]); ok {
			return value
		}
	}

	return pagination.DefaultLimit
}

func numeric(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int32:
		return int(value), true
	case int64:
		return clampInt64(value), true
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, false
		}
		return clampInt64(int64(value)), true
	case interface{ Int64() (int64, error) }:
		parsed, err := value.Int64()
		if err != nil {
			return 0, false
		}
		return clampInt64(parsed), true
	default:
		return 0, false
	}
}

func clampInt64(value int64) int {
	if value > maxCost {
		return maxCost
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}

	return int(value)
}
