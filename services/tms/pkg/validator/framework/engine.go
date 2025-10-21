package framework

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

type ValidationEngine struct {
	ruleIndex map[RuleKey][]ValidationRule
	rules     []ValidationRule
	ctx       *ValidationContext
	config    *EngineConfig
	metrics   *ValidationMetrics
	mu        sync.RWMutex
}

type EngineConfig struct {
	FailFast      bool
	MaxParallel   int
	EnableMetrics bool
	EnableTracing bool
}

type ValidationMetrics struct {
	TotalRules    int
	ExecutedRules int
	FailedRules   int
	SkippedRules  int
	TotalDuration int64
	RuleDurations map[string]int64
}

func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		FailFast:      false,
		MaxParallel:   10,
		EnableMetrics: false,
		EnableTracing: false,
	}
}

func NewValidationEngine(config *EngineConfig) *ValidationEngine {
	if config == nil {
		config = DefaultEngineConfig()
	}

	engine := &ValidationEngine{
		ruleIndex: make(map[RuleKey][]ValidationRule),
		rules:     make([]ValidationRule, 0),
		ctx:       &ValidationContext{},
		config:    config,
	}

	if config.EnableMetrics {
		engine.metrics = &ValidationMetrics{
			RuleDurations: make(map[string]int64),
		}
	}

	return engine
}

func (v *ValidationEngine) ForField(field string) *ValidationEngine {
	v.ctx.Field = field
	return v
}

func (v *ValidationEngine) AtIndex(idx int) *ValidationEngine {
	v.ctx.Index = idx
	v.ctx.IsIndexed = true
	return v
}

func (v *ValidationEngine) WithParent(parent *errortypes.MultiError) *ValidationEngine {
	v.ctx.Parent = parent
	return v
}

func (v *ValidationEngine) WithConfig(config *EngineConfig) *ValidationEngine {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.config = config
	if config.EnableMetrics && v.metrics == nil {
		v.metrics = &ValidationMetrics{
			RuleDurations: make(map[string]int64),
		}
	}
	return v
}

func (v *ValidationEngine) AddRule(rule ValidationRule) *ValidationEngine {
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

func (v *ValidationEngine) AddRules(rules ...ValidationRule) *ValidationEngine {
	for _, rule := range rules {
		v.AddRule(rule)
	}
	return v
}

func (v *ValidationEngine) Validate(ctx context.Context) *errortypes.MultiError {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var multiErr *errortypes.MultiError

	if v.ctx.IsIndexed && v.ctx.Parent != nil {
		v.ctx.IndexedErr = v.ctx.Parent.WithIndex(v.ctx.Field, v.ctx.Index)
		v.executeRulesOptimized(ctx, v.ctx.IndexedErr)
		return nil
	}

	multiErr = errortypes.NewMultiError()

	if v.ctx.IsIndexed {
		indexedErr := multiErr.WithIndex(v.ctx.Field, v.ctx.Index)
		v.executeRulesOptimized(ctx, indexedErr)
	} else {
		v.executeRulesOptimized(ctx, multiErr)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *ValidationEngine) ValidateInto(ctx context.Context, multiErr *errortypes.MultiError) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.ctx.IsIndexed {
		indexedErr := multiErr.WithIndex(v.ctx.Field, v.ctx.Index)
		v.executeRulesOptimized(ctx, indexedErr)
	} else {
		v.executeRulesOptimized(ctx, multiErr)
	}
}

func (v *ValidationEngine) executeRulesOptimized(
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

func (v *ValidationEngine) executeRulesSequential(
	ctx context.Context,
	rules []ValidationRule,
	multiErr *errortypes.MultiError,
	priority ValidationPriority,
) {
	for _, rule := range rules {
		if v.config.FailFast && multiErr.HasErrors() {
			break
		}

		// Ensure priority is set for sequential rules too
		multiErr.SetPriority(v.priorityToErrorPriority(priority))
		if err := rule.Validate(ctx, multiErr); err != nil {
			multiErr.Add("system", errortypes.ErrSystemError, err.Error())
		}
	}
}

func (v *ValidationEngine) executeRulesParallel(
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

func (v *ValidationEngine) GetMetrics() *ValidationMetrics {
	if !v.config.EnableMetrics {
		return nil
	}
	return v.metrics
}

func (v *ValidationEngine) Clear() *ValidationEngine {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.rules = make([]ValidationRule, 0)
	v.ruleIndex = make(map[RuleKey][]ValidationRule)
	v.ctx = &ValidationContext{}
	if v.metrics != nil {
		v.metrics = &ValidationMetrics{
			RuleDurations: make(map[string]int64),
		}
	}

	return v
}

func (v *ValidationEngine) RuleCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.rules)
}

func (v *ValidationEngine) RulesByStageAndPriority(
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

func (v *ValidationEngine) priorityToErrorPriority(
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

func (v *ValidationEngine) String() string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return fmt.Sprintf("ValidationEngine{rules: %d, indexed: %d keys, config: %+v}",
		len(v.rules), len(v.ruleIndex), v.config)
}
