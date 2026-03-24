package validationframework

import (
	"context"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/sourcegraph/conc/pool"
)

type RuleKey struct {
	Stage    ValidationStage
	Priority ValidationPriority
}

type Engine struct {
	ruleIndex map[RuleKey][]ValidationRule
	rules     []ValidationRule
	ctx       *ValidationRuleContext
	config    *EngineConfig
	mu        sync.RWMutex
}

type EngineConfig struct {
	FailFast      bool
	MaxParallel   int
	EnableMetrics bool
	EnableTracing bool
}

func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		FailFast:      false,
		MaxParallel:   10,
		EnableMetrics: false,
		EnableTracing: false,
	}
}

func NewEngine(config *EngineConfig) *Engine {
	if config == nil {
		config = DefaultEngineConfig()
	}

	engine := &Engine{
		ruleIndex: make(map[RuleKey][]ValidationRule),
		rules:     make([]ValidationRule, 0),
		ctx:       &ValidationRuleContext{},
		config:    config,
	}

	return engine
}

func (v *Engine) ForField(field string) *Engine {
	v.ctx.Field = field
	return v
}

func (v *Engine) AtIndex(idx int) *Engine {
	v.ctx.Index = idx
	v.ctx.IsIndexed = true
	return v
}

func (v *Engine) WithParent(parent *errortypes.MultiError) *Engine {
	v.ctx.Parent = parent
	return v
}

func (v *Engine) WithConfig(config *EngineConfig) *Engine {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.config = config
	return v
}

func (v *Engine) AddRule(rule ValidationRule) *Engine {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.rules = append(v.rules, rule)

	key := RuleKey{
		Stage:    rule.Stage(),
		Priority: rule.Priority(),
	}

	if _, exists := v.ruleIndex[key]; !exists {
		v.ruleIndex[key] = make([]ValidationRule, 0)
	}
	v.ruleIndex[key] = append(v.ruleIndex[key], rule)

	return v
}

func (v *Engine) AddRules(rules ...ValidationRule) *Engine {
	for _, rule := range rules {
		v.AddRule(rule)
	}
	return v
}

func (v *Engine) Validate(ctx context.Context) *errortypes.MultiError {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var multiErr *errortypes.MultiError

	if v.ctx.IsIndexed && v.ctx.Parent != nil {
		v.ctx.IndexedErr = v.ctx.Parent.WithIndex(v.ctx.Field, v.ctx.Index)
		v.executeRulesOptimized(ctx, v.ctx.IndexedErr)
		return nil
	}

	if v.ctx.Field != "" && v.ctx.Parent != nil {
		scopedErr := v.ctx.Parent.WithPrefix(v.ctx.Field)
		v.executeRulesOptimized(ctx, scopedErr)
		return nil
	}

	multiErr = errortypes.NewMultiError()

	v.validateWithContext(ctx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Engine) ValidateInto(ctx context.Context, multiErr *errortypes.MultiError) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	v.validateWithContext(ctx, multiErr)
}

func (v *Engine) validateWithContext(
	ctx context.Context,
	multiErr *errortypes.MultiError,
) {
	switch {
	case v.ctx.IsIndexed:
		indexedErr := multiErr.WithIndex(v.ctx.Field, v.ctx.Index)
		v.executeRulesOptimized(ctx, indexedErr)
	case v.ctx.Field != "":
		scopedErr := multiErr.WithPrefix(v.ctx.Field)
		v.executeRulesOptimized(ctx, scopedErr)
	default:
		v.executeRulesOptimized(ctx, multiErr)
	}
}

func (v *Engine) executeRulesOptimized(
	ctx context.Context,
	multiErr *errortypes.MultiError,
) {
	for stage := ValidationStageBasic; stage <= ValidationStageCompliance; stage++ {
		if v.config.FailFast && multiErr.HasErrors() {
			break
		}

		for priority := ValidationPriorityHigh; priority <= ValidationPriorityLow; priority++ {
			if v.config.FailFast && multiErr.HasErrors() {
				break
			}

			key := RuleKey{Stage: stage, Priority: priority}
			rules, exists := v.ruleIndex[key]
			if !exists || len(rules) == 0 {
				continue
			}

			// Set the priority context for all errors added during this priority level
			multiErr.SetPriority(v.priorityToErrorPriority(priority))

			if v.config.MaxParallel > 1 && len(rules) > 1 {
				v.executeRulesParallel(ctx, rules, multiErr, priority)
			} else {
				v.executeRulesSequential(ctx, rules, multiErr, priority)
			}
		}
	}
}

func (v *Engine) executeRulesSequential(
	ctx context.Context,
	rules []ValidationRule,
	multiErr *errortypes.MultiError,
	priority ValidationPriority,
) {
	for _, rule := range rules {
		if v.config.FailFast && multiErr.HasErrors() {
			break
		}

		multiErr.SetPriority(v.priorityToErrorPriority(priority))
		if err := rule.Validate(ctx, multiErr); err != nil {
			multiErr.Add("system", errortypes.ErrSystemError, err.Error())
		}
	}
}

func (v *Engine) executeRulesParallel(
	ctx context.Context,
	rules []ValidationRule,
	multiErr *errortypes.MultiError,
	priority ValidationPriority,
) {
	var errMu sync.Mutex
	p := pool.New()
	if v.config.MaxParallel > 0 {
		p = p.WithMaxGoroutines(v.config.MaxParallel)
	}

	for _, rule := range rules {
		p.Go(func() {
			tempErr := errortypes.NewMultiError()
			tempErr.SetPriority(v.priorityToErrorPriority(priority))

			if err := rule.Validate(ctx, tempErr); err != nil {
				errMu.Lock()
				multiErr.Add("system", errortypes.ErrSystemError, err.Error())
				errMu.Unlock()
				return
			}

			if tempErr.HasErrors() {
				errMu.Lock()
				for _, e := range tempErr.Errors {
					multiErr.AddError(e)
				}
				errMu.Unlock()
			}
		})
	}

	p.Wait()
}

func (v *Engine) Clear() *Engine {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.rules = make([]ValidationRule, 0)
	v.ruleIndex = make(map[RuleKey][]ValidationRule)
	v.ctx = &ValidationRuleContext{}

	return v
}

func (v *Engine) RuleCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.rules)
}

func (v *Engine) RulesByStageAndPriority(
	stage ValidationStage,
	priority ValidationPriority,
) []ValidationRule {
	v.mu.RLock()
	defer v.mu.RUnlock()

	key := RuleKey{Stage: stage, Priority: priority}
	rules, exists := v.ruleIndex[key]
	if !exists {
		return nil
	}

	result := make([]ValidationRule, len(rules))
	copy(result, rules)
	return result
}

func (v *Engine) priorityToErrorPriority(
	priority ValidationPriority,
) errortypes.ValidationPriority {
	switch priority {
	case ValidationPriorityHigh:
		return errortypes.PriorityHigh
	case ValidationPriorityMedium:
		return errortypes.PriorityMedium
	case ValidationPriorityLow:
		return errortypes.PriorityLow
	default:
		return errortypes.PriorityHigh
	}
}

func (v *Engine) String() string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return fmt.Sprintf("ValidationEngine{rules: %d, indexed: %d keys, config: %+v}",
		len(v.rules), len(v.ruleIndex), v.config)
}
