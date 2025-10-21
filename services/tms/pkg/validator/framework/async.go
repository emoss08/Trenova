package framework

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/sourcegraph/conc/pool"
)

type AsyncValidationRule interface {
	ValidationRule
	Timeout() time.Duration
	CanRunAsync() bool
}

type AsyncRule struct {
	rule     ValidationRule
	timeout  time.Duration
	canAsync bool
}

func NewAsyncRule(rule ValidationRule, timeout time.Duration) *AsyncRule {
	return &AsyncRule{
		rule:     rule,
		timeout:  timeout,
		canAsync: true,
	}
}

func (ar *AsyncRule) Stage() ValidationStage {
	return ar.rule.Stage()
}

func (ar *AsyncRule) Priority() ValidationPriority {
	return ar.rule.Priority()
}

func (ar *AsyncRule) Validate(ctx context.Context, multiErr *errortypes.MultiError) error {
	if ar.timeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, ar.timeout)
		defer cancel()
		ctx = timeoutCtx
	}

	done := make(chan error, 1)
	go func() {
		done <- ar.rule.Validate(ctx, multiErr)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("validation timed out after %v", ar.timeout)
	}
}

func (ar *AsyncRule) Timeout() time.Duration {
	return ar.timeout
}

func (ar *AsyncRule) CanRunAsync() bool {
	return ar.canAsync
}

type DatabaseValidationRule struct {
	*ConcreteRule
	queryFunc func(context.Context) error
	timeout   time.Duration
}

func NewDatabaseValidationRule(
	name string,
	queryFunc func(context.Context) error,
) *DatabaseValidationRule {
	rule := &DatabaseValidationRule{
		ConcreteRule: NewConcreteRule(name),
		queryFunc:    queryFunc,
		timeout:      5 * time.Second,
	}
	rule.stage = ValidationStageDataIntegrity
	rule.priority = ValidationPriorityHigh

	rule.WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
		return rule.executeDatabaseQuery(ctx, multiErr)
	})

	return rule
}

func (dr *DatabaseValidationRule) WithTimeout(timeout time.Duration) *DatabaseValidationRule {
	dr.timeout = timeout
	return dr
}

func (dr *DatabaseValidationRule) executeDatabaseQuery(
	ctx context.Context,
	multiErr *errortypes.MultiError,
) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, dr.timeout)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- dr.queryFunc(timeoutCtx)
	}()

	select {
	case err := <-errChan:
		if err != nil {
			multiErr.Add("", errortypes.ErrSystemError,
				fmt.Sprintf("Database validation failed: %v", err))
		}
		return nil
	case <-timeoutCtx.Done():
		multiErr.Add("", errortypes.ErrSystemError,
			fmt.Sprintf("Database validation timed out after %v", dr.timeout))
		return nil
	}
}

type AsyncValidationEngine struct {
	*ValidationEngine
	asyncRules []AsyncValidationRule
	asyncMu    sync.RWMutex
}

func NewAsyncValidationEngine(config *EngineConfig) *AsyncValidationEngine {
	return &AsyncValidationEngine{
		ValidationEngine: NewValidationEngine(config),
		asyncRules:       make([]AsyncValidationRule, 0),
	}
}

func (ave *AsyncValidationEngine) AddAsyncRule(rule AsyncValidationRule) *AsyncValidationEngine {
	ave.asyncMu.Lock()
	defer ave.asyncMu.Unlock()

	ave.asyncRules = append(ave.asyncRules, rule)
	ave.AddRule(rule)
	return ave
}

func (ave *AsyncValidationEngine) ValidateAsync(ctx context.Context) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	ruleGroups := ave.groupRulesByStageAndPriority()

	for stage := ValidationStageBasic; stage <= ValidationStageCompliance; stage++ {
		if ave.config.FailFast && multiErr.HasErrors() {
			break
		}

		for priority := ValidationPriorityHigh; priority <= ValidationPriorityLow; priority++ {
			if ave.config.FailFast && multiErr.HasErrors() {
				break
			}

			key := RuleKey{Stage: stage, Priority: priority}
			if rules, exists := ruleGroups[key]; exists && len(rules) > 0 {
				ave.executeAsyncRuleGroup(ctx, rules, multiErr)
			}
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (ave *AsyncValidationEngine) groupRulesByStageAndPriority() map[RuleKey][]ValidationRule {
	ave.mu.RLock()
	ave.asyncMu.RLock()
	defer ave.mu.RUnlock()
	defer ave.asyncMu.RUnlock()

	groups := make(map[RuleKey][]ValidationRule)

	for key, rules := range ave.ruleIndex {
		groups[key] = append(groups[key], rules...)
	}

	return groups
}

func (ave *AsyncValidationEngine) executeAsyncRuleGroup(
	ctx context.Context,
	rules []ValidationRule,
	multiErr *errortypes.MultiError,
) {
	var errMu sync.Mutex
	var asyncRules []AsyncValidationRule
	var syncRules []ValidationRule

	for _, rule := range rules {
		if asyncRule, ok := rule.(AsyncValidationRule); ok && asyncRule.CanRunAsync() {
			asyncRules = append(asyncRules, asyncRule)
		} else {
			syncRules = append(syncRules, rule)
		}
	}

	for _, rule := range syncRules {
		if ave.config.FailFast && multiErr.HasErrors() {
			return
		}

		if err := rule.Validate(ctx, multiErr); err != nil {
			multiErr.Add("system", errortypes.ErrSystemError, err.Error())
		}
	}

	if len(asyncRules) > 0 {
		p := pool.New()
		if ave.config.MaxParallel > 0 {
			p = p.WithMaxGoroutines(ave.config.MaxParallel)
		}

		for _, rule := range asyncRules {
			p.Go(func() {
				tempErr := errortypes.NewMultiError()

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
}

type ParallelValidator struct {
	validators []func(context.Context, *errortypes.MultiError) error
	maxWorkers int
}

func NewParallelValidator(maxWorkers int) *ParallelValidator {
	return &ParallelValidator{
		validators: make([]func(context.Context, *errortypes.MultiError) error, 0),
		maxWorkers: maxWorkers,
	}
}

func (pv *ParallelValidator) Add(
	validator func(context.Context, *errortypes.MultiError) error,
) *ParallelValidator {
	pv.validators = append(pv.validators, validator)
	return pv
}

func (pv *ParallelValidator) Validate(ctx context.Context) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	var errMu sync.Mutex

	p := pool.New()
	if pv.maxWorkers > 0 {
		p = p.WithMaxGoroutines(pv.maxWorkers)
	}

	for _, validator := range pv.validators {
		p.Go(func() {
			tempErr := errortypes.NewMultiError()

			if err := validator(ctx, tempErr); err != nil {
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

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
