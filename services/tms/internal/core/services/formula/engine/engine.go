package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	goErrors "errors"
	"fmt"
	"hash"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/services/formula/errors"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

const (
	compileCacheSize   = 1024
	maxExpressionNodes = 1_000
	evaluationTimeout  = 5 * time.Second
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
	cache       *lru.Cache[string, *CompiledExpression]
	exprOptions []expr.Option
}

func NewEngine(p Params) (*Engine, error) {
	cache, err := lru.New[string, *CompiledExpression](compileCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create compile cache: %w", err)
	}

	return &Engine{
		registry:    p.Registry,
		resolver:    p.Resolver,
		envBuilder:  p.EnvBuilder,
		cache:       cache,
		exprOptions: append(BuiltinFunctions(), expr.MaxNodes(maxExpressionNodes)),
	}, nil
}

var (
	ErrTemplateNil          = goErrors.New("template is nil")
	ErrSchemaNotFound       = goErrors.New("schema not found")
	ErrNonFiniteResult      = goErrors.New("expression produced a non-finite number")
	ErrNullResult           = goErrors.New("expression produced a null decimal")
	ErrNonNumericResult     = goErrors.New("expression must produce a numeric result")
	ErrVariableShadowsField = goErrors.New(
		"variable is not declared in the template and shadows a schema field",
	)
)

type CompiledExpression struct {
	program    *vm.Program
	expression string
}

func (e *Engine) Compile(
	expression string,
	env map[string]any,
) (*CompiledExpression, error) {
	cacheKey := compileCacheKey(expression, env)
	if cached, ok := e.cache.Get(cacheKey); ok {
		return cached, nil
	}

	program, err := expr.Compile(expression, e.compileOptions(env)...)
	if err != nil {
		return nil, errors.NewSchemaError(expression, "compile", err)
	}

	compiled := &CompiledExpression{
		program:    program,
		expression: expression,
	}

	e.cache.Add(cacheKey, compiled)

	return compiled, nil
}

func (e *Engine) Evaluate(
	ctx context.Context,
	req *formulatemplatetypes.EvaluationRequest,
) (*formulatemplatetypes.EvaluationResult, error) {
	if req.Template == nil {
		return nil, errors.NewSchemaError("", "evaluate", ErrTemplateNil)
	}

	definition, ok := e.registry.Get(req.Template.SchemaID)
	if !ok {
		return nil, errors.NewSchemaError(
			req.Template.SchemaID,
			"evaluate",
			fmt.Errorf("%w: %s", ErrSchemaNotFound, req.Template.SchemaID),
		)
	}

	if err := validateVariableKeys(definition, req.Template, req.Variables); err != nil {
		return nil, err
	}

	env, resolveFailures, err := e.envBuilder.BuildWithVariables(
		req.Entity,
		req.Template.SchemaID,
		req.Variables,
	)
	if err != nil {
		return nil, errors.NewSchemaError(req.Template.SchemaID, "build environment", err)
	}

	e.applyVariableDefaults(req.Template, env)

	result, err := e.evaluateProgram(ctx, req.Template.Expression, env, req.Lookup, resolveFailures)
	if err != nil {
		return nil, err
	}

	result.Breakdown = e.evaluateBreakdowns(ctx, req.Template.BreakdownDefinitions, env, req.Lookup)

	return result, nil
}

func (e *Engine) EvaluateExpression(
	ctx context.Context,
	req *formulatemplatetypes.ExpressionEvaluationRequest,
) (*formulatemplatetypes.EvaluationResult, error) {
	env, resolveFailures, err := e.envBuilder.BuildWithVariables(
		req.Entity,
		req.SchemaID,
		req.Variables,
	)
	if err != nil {
		return nil, errors.NewSchemaError(req.SchemaID, "build environment", err)
	}

	result, err := e.evaluateProgram(ctx, req.Expression, env, req.Lookup, resolveFailures)
	if err != nil {
		return nil, err
	}

	result.Breakdown = e.evaluateBreakdowns(ctx, req.Breakdowns, env, req.Lookup)

	return result, nil
}

func (e *Engine) EvaluateWithEnv(
	ctx context.Context,
	expression string,
	env map[string]any,
) (*formulatemplatetypes.EvaluationResult, error) {
	return e.evaluateProgram(ctx, expression, env, nil, nil)
}

func (e *Engine) evaluateProgram(
	ctx context.Context,
	expression string,
	env map[string]any,
	lookup formulatemplatetypes.RateTableLookup,
	resolveFailures map[string]error,
) (*formulatemplatetypes.EvaluationResult, error) {
	injectLookupFunctions(env, lookup)

	compiled, err := e.Compile(expression, env)
	if err != nil {
		return nil, withResolveFailures(err, resolveFailures)
	}

	output, err := e.run(ctx, compiled.program, env)
	if err != nil {
		return nil, errors.NewComputeError(
			expression,
			"expression",
			withResolveFailures(err, resolveFailures),
		)
	}

	result := &formulatemplatetypes.EvaluationResult{
		RawValue:  output,
		Variables: env,
	}

	result.Value, err = e.toDecimal(output)
	if err != nil {
		return nil, errors.NewTransformError(
			"expression result",
			"decimal",
			output,
			withResolveFailures(err, resolveFailures),
		)
	}

	return result, nil
}

func (e *Engine) evaluateBreakdowns(
	ctx context.Context,
	definitions []*formulatypes.BreakdownDefinition,
	env map[string]any,
	lookup formulatemplatetypes.RateTableLookup,
) []formulatemplatetypes.BreakdownAmount {
	if len(definitions) == 0 {
		return nil
	}

	items := make([]formulatemplatetypes.BreakdownAmount, 0, len(definitions))

	for _, def := range definitions {
		if def == nil {
			continue
		}

		item := formulatemplatetypes.BreakdownAmount{
			Name:  def.Name,
			Label: def.Label,
		}

		result, err := e.evaluateProgram(ctx, def.Expression, env, lookup, nil)
		if err != nil {
			item.Error = err.Error()
		} else {
			item.Amount = result.Value
		}

		items = append(items, item)
	}

	return items
}

// vm.Run cannot be interrupted, so a timed-out evaluation goroutine is
// abandoned; its work stays bounded by expr's memory budget and MaxNodes.
func (e *Engine) run(
	ctx context.Context,
	program *vm.Program,
	env map[string]any,
) (any, error) {
	ctx, cancel := context.WithTimeout(ctx, evaluationTimeout)
	defer cancel()

	type outcome struct {
		value any
		err   error
	}

	resultCh := make(chan outcome, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultCh <- outcome{err: fmt.Errorf("evaluation panicked: %v", r)}
			}
		}()

		value, err := vm.Run(program, env)
		resultCh <- outcome{value: value, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultCh:
		return res.value, res.err
	}
}

func (e *Engine) ValidateExpression(ctx context.Context, expression, schemaID string) error {
	env, _, err := e.envBuilder.BuildValidationEnvironment(schemaID, nil)
	if err != nil {
		return errors.NewSchemaError(schemaID, "get", err)
	}

	return e.ValidateExpressionWithEnv(ctx, expression, env)
}

func (e *Engine) ValidateExpressionWithEnv(
	ctx context.Context,
	expression string,
	env map[string]any,
) error {
	injectLookupFunctions(env, nil)

	compiled, err := e.Compile(expression, env)
	if err != nil {
		return errors.NewSchemaError(expression, "validate", err)
	}

	output, err := e.run(ctx, compiled.program, env)
	if err != nil {
		if goErrors.Is(err, context.DeadlineExceeded) {
			return errors.NewSchemaError(expression, "validate", err)
		}
		return nil
	}

	if err = validateResultType(output); err != nil {
		return errors.NewSchemaError(expression, "validate", err)
	}

	return nil
}

func (e *Engine) GetEnvironmentBuilder() *EnvironmentBuilder {
	return e.envBuilder
}

func (e *Engine) ClearCache() {
	e.cache.Purge()
}

func (e *Engine) compileOptions(env map[string]any) []expr.Option {
	options := make([]expr.Option, len(e.exprOptions), len(e.exprOptions)+1)
	copy(options, e.exprOptions)
	return append(options, expr.Env(env))
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
		return decimalFromFloat(v)
	case float32:
		return decimalFromFloat(float64(v))
	case int:
		return decimal.NewFromInt(int64(v)), nil
	case int64:
		return decimal.NewFromInt(v), nil
	case int32:
		return decimal.NewFromInt(int64(v)), nil
	case int16:
		return decimal.NewFromInt(int64(v)), nil
	case int8:
		return decimal.NewFromInt(int64(v)), nil
	case uint:
		return decimal.NewFromUint64(uint64(v)), nil
	case uint64:
		return decimal.NewFromUint64(v), nil
	case uint32:
		return decimal.NewFromInt(int64(v)), nil
	case uint16:
		return decimal.NewFromInt(int64(v)), nil
	case uint8:
		return decimal.NewFromInt(int64(v)), nil
	case decimal.Decimal:
		return v, nil
	case decimal.NullDecimal:
		if !v.Valid {
			return decimal.Zero, ErrNullResult
		}
		return v.Decimal, nil
	case bool:
		if v {
			return decimal.NewFromInt(1), nil
		}
		return decimal.NewFromInt(0), nil
	default:
		return decimal.Zero, fmt.Errorf("cannot convert %T to decimal", value)
	}
}

func decimalFromFloat(value float64) (decimal.Decimal, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return decimal.Zero, fmt.Errorf("%w: %v", ErrNonFiniteResult, value)
	}
	return decimal.NewFromFloat(value), nil
}

func validateResultType(value any) error {
	switch value.(type) {
	case float64, float32,
		int, int64, int32, int16, int8,
		uint, uint64, uint32, uint16, uint8,
		decimal.Decimal, decimal.NullDecimal, bool:
		return nil
	default:
		return fmt.Errorf("%w, got %T", ErrNonNumericResult, value)
	}
}

func validateVariableKeys(
	definition *formulatypes.Definition,
	template *formulatemplate.FormulaTemplate,
	variables map[string]any,
) error {
	if len(variables) == 0 {
		return nil
	}

	declared := make(map[string]struct{}, len(template.VariableDefinitions))
	for _, varDef := range template.VariableDefinitions {
		declared[varDef.Name] = struct{}{}
	}

	fieldRoots := schemaFieldRoots(definition)

	for key := range variables {
		if isReservedName(key) {
			return errors.NewVariableError(key, template.SchemaID, ErrReservedVariableName)
		}

		if _, ok := declared[key]; ok {
			continue
		}

		root, _, _ := strings.Cut(key, ".")
		if _, ok := fieldRoots[root]; ok {
			return errors.NewVariableError(key, template.SchemaID, ErrVariableShadowsField)
		}
	}

	return nil
}

func schemaFieldRoots(definition *formulatypes.Definition) map[string]struct{} {
	roots := make(map[string]struct{}, len(definition.FieldSources))
	for fieldPath := range definition.FieldSources {
		root, _, _ := strings.Cut(fieldPath, ".")
		roots[strings.TrimSuffix(root, "[]")] = struct{}{}
	}
	return roots
}

func withResolveFailures(err error, resolveFailures map[string]error) error {
	if len(resolveFailures) == 0 {
		return err
	}

	paths := make([]string, 0, len(resolveFailures))
	for path := range resolveFailures {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	details := make([]string, 0, len(paths))
	for _, path := range paths {
		details = append(details, path+": "+resolveFailures[path].Error())
	}

	return fmt.Errorf("%w (unresolved fields: %s)", err, strings.Join(details, "; "))
}

func compileCacheKey(expression string, env map[string]any) string {
	digest := sha256.New()
	writeLenPrefixed(digest, expression)
	writeEnvSignature(digest, env)
	return hex.EncodeToString(digest.Sum(nil))
}

func writeLenPrefixed(digest hash.Hash, value string) {
	digest.Write([]byte(strconv.Itoa(len(value))))
	digest.Write([]byte{':'})
	digest.Write([]byte(value))
}

func writeEnvSignature(digest hash.Hash, env map[string]any) {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	digest.Write([]byte{'{'})
	for _, key := range keys {
		writeLenPrefixed(digest, key)
		writeValueSignature(digest, env[key])
	}
	digest.Write([]byte{'}'})
}

// Slice elements are statically typed as any by expr, so only the []any
// marker participates in the cache key; map value types affect compilation
// and are hashed recursively.
func writeValueSignature(digest hash.Hash, value any) {
	switch typed := value.(type) {
	case nil:
		writeLenPrefixed(digest, "nil")
	case map[string]any:
		writeEnvSignature(digest, typed)
	case []any:
		writeLenPrefixed(digest, "[]any")
	default:
		writeLenPrefixed(digest, reflect.TypeOf(value).String())
	}
}
