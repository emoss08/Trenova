package engine

import (
	goErrors "errors"
	"fmt"
	"sort"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
)

var ErrDisabledBuiltin = goErrors.New("function is not available in formulas")

const (
	DisabledNondeterministic = "nondeterministic"
	DisabledSerialization    = "serialization"
	DisabledTextBuilding     = "textBuilding"
)

type DisabledFunction struct {
	Name        string `json:"name"`
	Reason      string `json:"reason"`
	Replacement string `json:"replacement,omitempty"`
	message     string
}

var disabledBuiltins = map[string]DisabledFunction{
	"now": {
		Name:        "now",
		Reason:      DisabledNondeterministic,
		Replacement: RatingTimeVariable,
		message: "a formula must price a shipment the same way every time it runs; " +
			"use ratingTime, the date the contract terms are read at",
	},
	"toJSON": {
		Name:    "toJSON",
		Reason:  DisabledSerialization,
		message: "formulas price shipments and do not serialize them",
	},
	"fromJSON": {
		Name:    "fromJSON",
		Reason:  DisabledSerialization,
		message: "formulas read the shipment's fields directly and have nothing to parse",
	},
	"repeat": {
		Name:    "repeat",
		Reason:  DisabledTextBuilding,
		message: "formulas compare codes and do not build text",
	},
}

func DisabledFunctions() []DisabledFunction {
	names := make([]string, 0, len(disabledBuiltins))
	for name := range disabledBuiltins {
		names = append(names, name)
	}
	sort.Strings(names)

	disabled := make([]DisabledFunction, 0, len(names))
	for _, name := range names {
		disabled = append(disabled, disabledBuiltins[name])
	}
	return disabled
}

func disabledBuiltinOptions() []expr.Option {
	names := make([]string, 0, len(disabledBuiltins))
	for name := range disabledBuiltins {
		names = append(names, name)
	}
	sort.Strings(names)

	options := make([]expr.Option, 0, len(names))
	for _, name := range names {
		options = append(options, expr.DisableBuiltin(name))
	}
	return options
}

func disabledBuiltinUse(expression string) error {
	name := disabledBuiltinIn(expression)
	if name == "" {
		return nil
	}

	return fmt.Errorf("%w: %s() — %s", ErrDisabledBuiltin, name, disabledBuiltins[name].message)
}

func disabledBuiltinIn(expression string) string {
	tree, err := parseExpression(expression)
	if err != nil {
		return ""
	}

	visitor := &disabledBuiltinVisitor{}
	ast.Walk(&tree.Node, visitor)
	return visitor.name
}

type disabledBuiltinVisitor struct {
	name string
}

//nolint:gocritic // expr ast.Visitor requires the pointer signature
func (v *disabledBuiltinVisitor) Visit(node *ast.Node) {
	if v.name != "" {
		return
	}

	var name string
	switch typed := (*node).(type) {
	case *ast.BuiltinNode:
		name = typed.Name
	case *ast.CallNode:
		callee, ok := typed.Callee.(*ast.IdentifierNode)
		if !ok {
			return
		}
		name = callee.Value
	default:
		return
	}

	if _, disabled := disabledBuiltins[name]; disabled {
		v.name = name
	}
}

var goTypeNames = strings.NewReplacer(
	"map[string]interface {}", "group of fields",
	"[]interface {}", "list",
	"interface {}", "any value",
	"map[string]any", "group of fields",
	"[]any", "list",
	"[]string", "list of text",
	"[]float64", "list of numbers",
	"*time.Location", "time zone",
	"time.Time", "date",
	"time.Duration", "duration",
	"float64", "number",
	"float32", "number",
	"int64", "number",
	"int32", "number",
)

type languageError struct {
	err error
}

func (e *languageError) Error() string {
	return goTypeNames.Replace(e.err.Error())
}

func (e *languageError) Unwrap() error {
	return e.err
}

func humanized(err error) error {
	if err == nil {
		return nil
	}
	return &languageError{err: err}
}
