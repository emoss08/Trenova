package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	goErrors "errors"
	"fmt"
	"hash"
	"maps"
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
	"github.com/emoss08/trenova/shared/maputils"
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
	ctxEnvKey          = "__ctx"
)

type evaluationTimeoutKey struct{}

// WithEvaluationTimeout caps how long a single evaluation may run, overriding
// the engine's default ceiling for the calls made with this context. An
// interactive preview wants a short leash; a batch re-rate can afford the
// default. Non-positive durations are ignored.
func WithEvaluationTimeout(ctx context.Context, timeout time.Duration) context.Context {
	if timeout <= 0 {
		return ctx
	}

	return context.WithValue(ctx, evaluationTimeoutKey{}, timeout)
}

func evaluationTimeoutFor(ctx context.Context) time.Duration {
	if timeout, ok := ctx.Value(evaluationTimeoutKey{}).(time.Duration); ok && timeout > 0 {
		return timeout
	}

	return evaluationTimeout
}

type Params struct {
	fx.In

	Registry   *schema.Registry
	Resolver   *resolver.Resolver
	EnvBuilder *EnvironmentBuilder

	// CompileCacheSize bounds the compiled-program cache; zero means the
	// default. One entry per distinct expression and schema shape is enough
	// for a tenant's live templates, so the default is generous.
	CompileCacheSize int `optional:"true"`
}

type Engine struct {
	registry    *schema.Registry
	resolver    *resolver.Resolver
	envBuilder  *EnvironmentBuilder
	cache       *lru.Cache[string, *CompiledExpression]
	exprOptions []expr.Option
}

func NewEngine(p Params) (*Engine, error) {
	size := p.CompileCacheSize
	if size <= 0 {
		size = compileCacheSize
	}
	cache, err := lru.New[string, *CompiledExpression](size)
	if err != nil {
		return nil, fmt.Errorf("failed to create compile cache: %w", err)
	}

	return &Engine{
		registry:   p.Registry,
		resolver:   p.Resolver,
		envBuilder: p.EnvBuilder,
		cache:      cache,
		exprOptions: append(
			BuiltinFunctions(),
			expr.MaxNodes(maxExpressionNodes),
			expr.WithContext(ctxEnvKey),
		),
	}, nil
}

var (
	// ErrEvaluationTimeout is the engine's own deadline. It is deliberately
	// not the caller's context error: an HTTP request that was still alive
	// while a formula ran past its budget has a formula problem, not a
	// timed-out request, and the two must not be told apart by guesswork.
	ErrEvaluationTimeout    = goErrors.New("formula evaluation exceeded its time budget")
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

	env, resolveFailures, err := e.envBuilder.BuildWithProvided(
		req.Entity,
		req.Template.SchemaID,
		req.Provided,
		req.Variables,
	)
	if err != nil {
		return nil, errors.NewSchemaError(req.Template.SchemaID, "build environment", err)
	}
	sources := provenanceForSchema(definition)
	sources.markAll(req.Provided, formulatypes.ValueSourceProvided)
	sources.markAll(req.Variables, formulatypes.ValueSourceInput)

	mergeVariables(env, req.Overrides)
	sources.markAll(req.Overrides, formulatypes.ValueSourceOverride)

	sources.markPaths(e.applyVariableDefaults(req.Template, env), formulatypes.ValueSourceDefault)

	recorder := newLookupRecorder(req.Lookup)
	shape := e.shapeFor(ctx, definition, env, recorder)
	result, err := e.evaluateShaped(ctx, req.Template.Expression, env, shape, resolveFailures)
	if err != nil {
		return nil, err
	}
	if err = rejectBooleanAmount(req.Template.Expression, result); err != nil {
		return nil, err
	}

	result.Breakdown = e.evaluateBreakdowns(ctx, req.Template.BreakdownDefinitions, env, shape)
	result.Receipt = &formulatypes.Receipt{
		Variables: receiptVariables(env, sources),
		Lookups:   recorder.entries,
		RawAmount: result.Value,
	}

	return result, nil
}

// ErrBooleanAmount is returned when a charge expression evaluates to true or
// false. The engine can turn a boolean into one or zero, and for a predicate
// that is the point; for a price it is a bug with a dollar sign on it.
var ErrBooleanAmount = goErrors.New(
	"expression produced true/false rather than an amount; " +
		"write a conditional such as condition ? amount : 0",
)

func rejectBooleanAmount(expression string, result *formulatemplatetypes.EvaluationResult) error {
	if _, isBool := result.RawValue.(bool); !isBool {
		return nil
	}

	return errors.NewComputeError(expression, "expression", ErrBooleanAmount)
}

func (e *Engine) EvaluateExpression(
	ctx context.Context,
	req *formulatemplatetypes.ExpressionEvaluationRequest,
) (*formulatemplatetypes.EvaluationResult, error) {
	for key := range req.Variables {
		if isReservedName(key) {
			return nil, errors.NewVariableError(key, req.SchemaID, ErrReservedVariableName)
		}
	}

	env, resolveFailures, err := e.envBuilder.BuildWithProvided(
		req.Entity,
		req.SchemaID,
		req.Provided,
		req.Variables,
	)
	if err != nil {
		return nil, errors.NewSchemaError(req.SchemaID, "build environment", err)
	}

	var sources provenance
	if definition, ok := e.registry.Get(req.SchemaID); ok {
		sources = provenanceForSchema(definition)
	} else {
		sources = make(provenance, len(req.Variables)+len(req.Provided))
	}
	sources.markAll(req.Provided, formulatypes.ValueSourceProvided)
	sources.markAll(req.Variables, formulatypes.ValueSourceInput)

	recorder := newLookupRecorder(req.Lookup)
	var definition *formulatypes.Definition
	if known, ok := e.registry.Get(req.SchemaID); ok {
		definition = known
	}
	shape := e.shapeFor(ctx, definition, env, recorder)
	result, err := e.evaluateShaped(ctx, req.Expression, env, shape, resolveFailures)
	if err != nil {
		return nil, err
	}
	if !req.AllowBoolean {
		if err = rejectBooleanAmount(req.Expression, result); err != nil {
			return nil, err
		}
	}

	result.Breakdown = e.evaluateBreakdowns(ctx, req.Breakdowns, env, shape)
	result.Receipt = &formulatypes.Receipt{
		Variables: receiptVariables(env, sources),
		Lookups:   recorder.entries,
		RawAmount: result.Value,
	}

	return result, nil
}

func (e *Engine) EvaluateWithEnv(
	ctx context.Context,
	req *formulatemplatetypes.EnvEvaluationRequest,
) (*formulatemplatetypes.EvaluationResult, error) {
	recorder := newLookupRecorder(req.Lookup)
	result, err := e.evaluateProgram(ctx, req.Expression, req.Env, recorder, nil)
	if err != nil {
		return nil, err
	}
	if err = rejectBooleanAmount(req.Expression, result); err != nil {
		return nil, err
	}

	sources := make(provenance, len(req.Env))
	sources.markAll(req.Env, formulatypes.ValueSourceSample)
	result.Receipt = &formulatypes.Receipt{
		Variables: receiptVariables(req.Env, sources),
		Lookups:   recorder.entries,
		RawAmount: result.Value,
	}

	return result, nil
}

func (e *Engine) evaluateProgram(
	ctx context.Context,
	expression string,
	env map[string]any,
	lookup formulatemplatetypes.RateTableLookup,
	resolveFailures map[string]error,
) (*formulatemplatetypes.EvaluationResult, error) {
	shape := e.shapeFor(ctx, nil, env, lookup)
	return e.evaluateShaped(ctx, expression, env, shape, resolveFailures)
}

// evaluateShaped compiles against the evaluation's shape — computed once, so
// the main expression and every breakdown line share the schema-level cache
// key — and runs against the real environment.
func (e *Engine) evaluateShaped(
	ctx context.Context,
	expression string,
	env map[string]any,
	shape *compileShape,
	resolveFailures map[string]error,
) (*formulatemplatetypes.EvaluationResult, error) {
	injectLookupFunctions(env, shape.lookup)

	env[ctxEnvKey] = ctx
	defer delete(env, ctxEnvKey)

	compiled, err := e.compileShaped(expression, shape)
	if err != nil {
		if missing := missingFieldError(expression, env, resolveFailures); missing != nil {
			return nil, missing
		}
		return nil, withResolveFailures(err, resolveFailures)
	}

	output, err := e.run(ctx, compiled.program, env)
	if err != nil {
		// The program was compiled against the schema's declared types, so a
		// record that lacks a nullable value fails here rather than at
		// compile time. It is the same authoring problem with the same fix.
		if missing := missingFieldError(expression, env, resolveFailures); missing != nil {
			return nil, missing
		}
		return nil, errors.NewComputeError(
			expression,
			"expression",
			withResolveFailures(err, resolveFailures),
		)
	}

	// The injected context is deleted before the copy so the variables a
	// caller sees are exactly what the formula saw.
	delete(env, ctxEnvKey)
	result := &formulatemplatetypes.EvaluationResult{
		RawValue:  output,
		Variables: maputils.WithoutFuncValues(env),
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
	shape *compileShape,
) []formulatemplatetypes.BreakdownAmount {
	if len(definitions) == 0 {
		return nil
	}

	items := make([]formulatemplatetypes.BreakdownAmount, 0, len(definitions))
	defer shape.recorder().setScope(mainExpressionScope)

	for _, def := range definitions {
		if def == nil {
			continue
		}

		item := formulatemplatetypes.BreakdownAmount{
			Name:  def.Name,
			Label: def.Label,
		}

		shape.recorder().setScope(def.Name)
		result, err := e.evaluateShaped(ctx, def.Expression, env, shape, nil)
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
// run executes a compiled program under the evaluation deadline.
//
// The VM gets its own copy of the environment. The caller keeps using — and
// mutating — its map the moment run returns, and on a timeout it returns while
// the VM goroutine may still be reading; a shared map at that point is a
// concurrent read and write, which the runtime treats as fatal rather than as
// an error anyone can recover from. Copying a few dozen entries costs far
// less than the evaluation itself.
func (e *Engine) run(
	ctx context.Context,
	program *vm.Program,
	env map[string]any,
) (any, error) {
	parent := ctx
	timeout := evaluationTimeoutFor(ctx)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	vmEnv := make(map[string]any, len(env)+1)
	maps.Copy(vmEnv, env)
	vmEnv[ctxEnvKey] = ctx

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

		value, err := vm.Run(program, vmEnv)
		resultCh <- outcome{value: value, err: err}
	}()

	select {
	case <-ctx.Done():
		if parentErr := parent.Err(); parentErr != nil {
			return nil, parentErr
		}
		return nil, fmt.Errorf("%w (%s)", ErrEvaluationTimeout, timeout)
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

type ValidationOutcome struct {
	Err     error
	Warning string
}

func (e *Engine) ValidateExpressionDetailed(
	ctx context.Context,
	expression string,
	env map[string]any,
) ValidationOutcome {
	injectLookupFunctions(env, StubLookup{})

	env[ctxEnvKey] = ctx
	defer delete(env, ctxEnvKey)

	compiled, err := e.Compile(expression, env)
	if err != nil {
		return ValidationOutcome{Err: errors.NewSchemaError(expression, "validate", err)}
	}

	output, err := e.run(ctx, compiled.program, env)
	if err != nil {
		if goErrors.Is(err, context.DeadlineExceeded) || goErrors.Is(err, ErrEvaluationTimeout) {
			return ValidationOutcome{Err: errors.NewSchemaError(expression, "validate", err)}
		}
		return ValidationOutcome{Warning: err.Error()}
	}

	if err = validateResultType(output); err != nil {
		return ValidationOutcome{Err: errors.NewSchemaError(expression, "validate", err)}
	}

	return ValidationOutcome{}
}

func (e *Engine) ValidateExpressionWithEnv(
	ctx context.Context,
	expression string,
	env map[string]any,
) error {
	return e.ValidateExpressionDetailed(ctx, expression, env).Err
}

func (e *Engine) GetEnvironmentBuilder() *EnvironmentBuilder {
	return e.envBuilder
}

func (e *Engine) ClearCache() {
	e.cache.Purge()
}

func (e *Engine) CacheLen() int {
	return e.cache.Len()
}

func (e *Engine) compileOptions(env map[string]any) []expr.Option {
	options := make([]expr.Option, len(e.exprOptions), len(e.exprOptions)+1)
	copy(options, e.exprOptions)
	return append(options, expr.Env(env))
}

func (e *Engine) applyVariableDefaults(
	template *formulatemplate.FormulaTemplate,
	env map[string]any,
) []string {
	filled := make([]string, 0, len(template.VariableDefinitions))
	for _, varDef := range template.VariableDefinitions {
		if _, exists := env[varDef.Name]; !exists && varDef.DefaultValue != nil {
			env[varDef.Name] = varDef.DefaultValue
			filled = append(filled, varDef.Name)
		}
	}
	return filled
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
		if key == ctxEnvKey {
			writeLenPrefixed(digest, "context.Context")
			continue
		}
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

// compileShape is what an evaluation compiles against: the environment with
// every schema-nullable path set to its declared type's zero, so one program
// serves records with and without the value, plus that environment's
// signature hashed once. A nil at compile time types the path as unknown and
// rejects arithmetic outright, while the declared zero compiles arithmetic,
// coalesce, ?? and == nil alike; at run time expr's arithmetic is dynamic, so
// a record that really lacks the value still fails the way it always did.
type compileShape struct {
	env       map[string]any
	signature string
	lookup    formulatemplatetypes.RateTableLookup
}

func (s *compileShape) recorder() *lookupRecorder {
	if rec, ok := s.lookup.(*lookupRecorder); ok {
		return rec
	}
	return newLookupRecorder(s.lookup)
}

func (e *Engine) shapeFor(
	ctx context.Context,
	definition *formulatypes.Definition,
	env map[string]any,
	lookup formulatemplatetypes.RateTableLookup,
) *compileShape {
	shaped := make(map[string]any, len(env)+lookupFunctionCount+1)
	maps.Copy(shaped, env)
	injectLookupFunctions(shaped, lookup)
	shaped[ctxEnvKey] = ctx

	if definition != nil {
		for path, fieldType := range nullablePaths(definition.Properties, "") {
			shapePath(shaped, path, declaredZero(fieldType))
		}
	}

	digest := sha256.New()
	writeEnvSignature(digest, shaped)

	return &compileShape{
		env:       shaped,
		signature: hex.EncodeToString(digest.Sum(nil)),
		lookup:    lookup,
	}
}

// declaredZero is the compile-time stand-in for a nullable field: the zero of
// its declared type, matching what a validation environment would hold.
func declaredZero(fieldType string) any {
	switch fieldType {
	case "boolean":
		return false
	case "string":
		return ""
	case "integer":
		return int64(0)
	case "datetime":
		return time.Time{}
	default:
		return 0.0
	}
}

// shapePath sets a dotted path in a copy of the environment without touching
// the nested maps the real environment still owns.
func shapePath(env map[string]any, path string, value any) {
	segments := strings.Split(path, ".")
	current := env
	for _, segment := range segments[:len(segments)-1] {
		nested, ok := current[segment].(map[string]any)
		if !ok {
			return
		}
		cloned := make(map[string]any, len(nested))
		maps.Copy(cloned, nested)
		current[segment] = cloned
		current = cloned
	}
	if _, present := current[segments[len(segments)-1]]; present {
		current[segments[len(segments)-1]] = value
	}
}

func (e *Engine) compileShaped(
	expression string,
	shape *compileShape,
) (*CompiledExpression, error) {
	digest := sha256.New()
	writeLenPrefixed(digest, expression)
	writeLenPrefixed(digest, shape.signature)
	cacheKey := hex.EncodeToString(digest.Sum(nil))

	if cached, ok := e.cache.Get(cacheKey); ok {
		return cached, nil
	}

	program, err := expr.Compile(expression, e.compileOptions(shape.env)...)
	if err != nil {
		return nil, errors.NewSchemaError(expression, "compile", err)
	}

	compiled := &CompiledExpression{program: program, expression: expression}
	e.cache.Add(cacheKey, compiled)

	return compiled, nil
}
