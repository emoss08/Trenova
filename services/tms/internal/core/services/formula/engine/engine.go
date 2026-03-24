package engine

import (
	goErrors "errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/services/formula/errors"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Registry   *schema.Registry
	Resolver   *resolver.Resolver
	EnvBuilder *EnvironmentBuilder
}

type Engine struct {
	registry    *schema.Registry
	resolver    *resolver.Resolver
	envBuilder  *EnvironmentBuilder
	cache       sync.Map
	exprOptions []expr.Option
}

func NewEngine(p Params) *Engine {
	return &Engine{
		registry:    p.Registry,
		resolver:    p.Resolver,
		envBuilder:  p.EnvBuilder,
		exprOptions: BuiltinFunctions(),
	}
}

var (
	ErrTemplateNil    = goErrors.New("template is nil")
	ErrSchemaNotFound = goErrors.New("schema not found")
)

type CompliedExpression struct {
	program    *vm.Program
	expression string
}

func (e *Engine) Compile(
	expression string,
	env map[string]any,
) (*CompliedExpression, error) {
	cacheKey := compileCacheKey(expression, env)
	if cached, ok := e.cache.Load(cacheKey); ok {
		return cached.(*CompliedExpression), nil //nolint:errcheck // ignore error because we know the type is correct
	}

	options := make([]expr.Option, len(e.exprOptions), len(e.exprOptions)+1)
	copy(options, e.exprOptions)
	options = append(options, expr.Env(env))

	program, err := expr.Compile(expression, options...)
	if err != nil {
		return nil, errors.NewSchemaError(expression, "compile", err)
	}

	compiled := &CompliedExpression{
		program:    program,
		expression: expression,
	}

	e.cache.Store(cacheKey, compiled)

	return compiled, nil
}

func (e *Engine) Evaluate(
	req *formulatemplatetypes.EvaluationRequest,
) (*formulatemplatetypes.EvaluationResult, error) {
	if req.Template == nil {
		return nil, errors.NewSchemaError("", "evaluate", ErrTemplateNil)
	}

	env, err := e.envBuilder.BuildWithVariables(
		req.Entity,
		req.Template.SchemaID,
		req.Variables,
	)
	if err != nil {
		return nil, errors.NewSchemaError(req.Template.SchemaID, "build environment", err)
	}

	e.applyVariableDefaults(req.Template, env)

	compiled, err := e.Compile(req.Template.Expression, env)
	if err != nil {
		return nil, err
	}

	output, err := vm.Run(compiled.program, env)
	if err != nil {
		return nil, errors.NewComputeError(req.Template.Expression, "expression", err)
	}

	result := &formulatemplatetypes.EvaluationResult{
		RawValue:  output,
		Variables: env,
	}

	result.Value, err = e.toDecimal(output)
	if err != nil {
		return nil, errors.NewTransformError("expression result", "decimal", output, err)
	}

	return result, nil
}

func (e *Engine) EvaluateExpression(
	expression string,
	entity any,
	schemaID string,
	variables map[string]any,
) (*formulatemplatetypes.EvaluationResult, error) {
	env, err := e.envBuilder.BuildWithVariables(entity, schemaID, variables)
	if err != nil {
		return nil, errors.NewSchemaError(schemaID, "build environment", err)
	}

	compiled, err := e.Compile(expression, env)
	if err != nil {
		return nil, err
	}

	output, err := vm.Run(compiled.program, env)
	if err != nil {
		return nil, errors.NewComputeError(expression, "expression", err)
	}

	result := &formulatemplatetypes.EvaluationResult{
		RawValue:  output,
		Variables: env,
	}

	result.Value, err = e.toDecimal(output)
	if err != nil {
		return nil, errors.NewTransformError("expression result", "decimal", output, err)
	}

	return result, nil
}

func (e *Engine) EvaluateWithEnv(
	expression string,
	env map[string]any,
) (*formulatemplatetypes.EvaluationResult, error) {
	compiled, err := e.Compile(expression, env)
	if err != nil {
		return nil, err
	}

	output, err := vm.Run(compiled.program, env)
	if err != nil {
		return nil, errors.NewComputeError(expression, "expression", err)
	}

	result := &formulatemplatetypes.EvaluationResult{
		RawValue:  output,
		Variables: env,
	}

	result.Value, err = e.toDecimal(output)
	if err != nil {
		return nil, errors.NewTransformError("expression result", "decimal", output, err)
	}

	return result, nil
}

func (e *Engine) ValidateExpression(expression, schemaID string) error {
	env, err := e.envBuilder.BuildValidationEnvironment(schemaID, nil)
	if err != nil {
		return errors.NewSchemaError(schemaID, "get", err)
	}

	options := make([]expr.Option, len(e.exprOptions), len(e.exprOptions)+1)
	copy(options, e.exprOptions)
	options = append(options, expr.Env(env))

	_, err = expr.Compile(expression, options...)
	if err != nil {
		return errors.NewSchemaError(expression, "validate", err)
	}

	return nil
}

func (e *Engine) ValidateExpressionWithEnv(expression string, env map[string]any) error {
	options := make([]expr.Option, len(e.exprOptions), len(e.exprOptions)+1)
	copy(options, e.exprOptions)
	options = append(options, expr.Env(env))

	_, err := expr.Compile(expression, options...)
	if err != nil {
		return errors.NewSchemaError(expression, "validate", err)
	}

	return nil
}

func (e *Engine) GetEnvironmentBuilder() *EnvironmentBuilder {
	return e.envBuilder
}

func (e *Engine) ClearCache() {
	e.cache = sync.Map{}
}

func (e *Engine) applyVariableDefaults(
	template *formulatemplate.FormulaTemplate,
	env map[string]any,
) {
	for _, varDef := range template.VariableDefinitions {
		if _, exists := env[varDef.Name]; !exists && varDef.DefaultValue != nil {
			env[varDef.Name] = varDef.DefaultValue
		}
	}
}

func (e *Engine) toDecimal(value any) (decimal.Decimal, error) {
	switch v := value.(type) {
	case float64:
		return decimal.NewFromFloat(v), nil
	case float32:
		return decimal.NewFromFloat32(v), nil
	case int:
		return decimal.NewFromInt(int64(v)), nil
	case int64:
		return decimal.NewFromInt(v), nil
	case int32:
		return decimal.NewFromInt(int64(v)), nil
	case decimal.Decimal:
		return v, nil
	case bool:
		if v {
			return decimal.NewFromInt(1), nil
		}
		return decimal.NewFromInt(0), nil
	default:
		return decimal.Zero, fmt.Errorf("cannot convert %T to decimal", value)
	}
}

func compileCacheKey(expression string, env map[string]any) string {
	return expression + "::" + envSignature(env)
}

func envSignature(env map[string]any) string {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+valueSignature(env[key]))
	}

	return strings.Join(parts, ",")
}

func valueSignature(value any) string {
	if value == nil {
		return "nil"
	}

	switch value.(type) {
	case map[string]any:
		return "map{" + envSignature(value.(map[string]any)) + "}"
	case []any:
		items := value.([]any)
		itemSignatures := make([]string, len(items))
		for i, item := range items {
			itemSignatures[i] = valueSignature(item)
		}
		return "slice[" + strings.Join(itemSignatures, ",") + "]"
	default:
		return reflect.TypeOf(value).String()
	}
}
