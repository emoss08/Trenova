package framework

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
)

type MockValidationEngine struct {
	*ValidationEngine
	executedRules []string
	shouldFail    map[string]error
	delays        map[string]time.Duration
	mu            sync.RWMutex
}

func NewMockValidationEngine() *MockValidationEngine {
	return &MockValidationEngine{
		ValidationEngine: NewValidationEngine(DefaultEngineConfig()),
		executedRules:    make([]string, 0),
		shouldFail:       make(map[string]error),
		delays:           make(map[string]time.Duration),
	}
}

func (mve *MockValidationEngine) SetRuleShouldFail(ruleName string, err error) {
	mve.mu.Lock()
	defer mve.mu.Unlock()
	mve.shouldFail[ruleName] = err
}

func (mve *MockValidationEngine) SetRuleDelay(ruleName string, delay time.Duration) {
	mve.mu.Lock()
	defer mve.mu.Unlock()
	mve.delays[ruleName] = delay
}

func (mve *MockValidationEngine) GetExecutedRules() []string {
	mve.mu.RLock()
	defer mve.mu.RUnlock()
	result := make([]string, len(mve.executedRules))
	copy(result, mve.executedRules)
	return result
}

func (mve *MockValidationEngine) Reset() {
	mve.mu.Lock()
	defer mve.mu.Unlock()
	mve.executedRules = make([]string, 0)
	mve.shouldFail = make(map[string]error)
	mve.delays = make(map[string]time.Duration)
	mve.Clear()
}

func (mve *MockValidationEngine) AddMockRule(
	name string,
	stage ValidationStage,
	priority ValidationPriority,
) {
	rule := NewConcreteRule(name).
		WithStage(stage).
		WithPriority(priority).
		WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
			mve.mu.Lock()
			mve.executedRules = append(mve.executedRules, name)
			delay := mve.delays[name]
			shouldFail := mve.shouldFail[name]
			mve.mu.Unlock()

			if delay > 0 {
				time.Sleep(delay)
			}

			if shouldFail != nil {
				return shouldFail
			}

			return nil
		})

	mve.AddRule(rule)
}

type ValidationTestCase struct {
	Name           string
	Setup          func()
	Validate       func(context.Context) *errortypes.MultiError
	ExpectedErrors []ExpectedError
	ExpectSuccess  bool
}

type ExpectedError struct {
	Field   string
	Code    errortypes.ErrorCode
	Message string
}

func (tc *ValidationTestCase) Run(t *testing.T) {
	t.Run(tc.Name, func(t *testing.T) {
		if tc.Setup != nil {
			tc.Setup()
		}

		ctx := context.Background()
		multiErr := tc.Validate(ctx)

		if tc.ExpectSuccess {
			if multiErr != nil && multiErr.HasErrors() {
				t.Errorf("Expected validation to succeed but got errors: %v", multiErr)
			}
			return
		}

		if multiErr == nil || !multiErr.HasErrors() {
			t.Error("Expected validation to fail but it succeeded")
			return
		}

		for _, expected := range tc.ExpectedErrors {
			found := false
			for _, err := range multiErr.Errors {
				if err.Field == expected.Field &&
					err.Code == expected.Code &&
					(expected.Message == "" || strings.Contains(err.Message, expected.Message)) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error not found: field=%s, code=%s, message=%s",
					expected.Field, expected.Code, expected.Message)
			}
		}
	})
}

type ValidationTestSuite struct {
	Name      string
	TestCases []ValidationTestCase
}

func (ts *ValidationTestSuite) Run(t *testing.T) {
	t.Run(ts.Name, func(t *testing.T) {
		for _, tc := range ts.TestCases {
			tc.Run(t)
		}
	})
}

type RuleExecutionTracker struct {
	executions []RuleExecution
	mu         sync.RWMutex
}

type RuleExecution struct {
	RuleName  string
	Stage     ValidationStage
	Priority  ValidationPriority
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Error     error
}

func NewRuleExecutionTracker() *RuleExecutionTracker {
	return &RuleExecutionTracker{
		executions: make([]RuleExecution, 0),
	}
}

func (ret *RuleExecutionTracker) Track(execution RuleExecution) {
	ret.mu.Lock()
	defer ret.mu.Unlock()
	ret.executions = append(ret.executions, execution)
}

func (ret *RuleExecutionTracker) GetExecutions() []RuleExecution {
	ret.mu.RLock()
	defer ret.mu.RUnlock()
	result := make([]RuleExecution, len(ret.executions))
	copy(result, ret.executions)
	return result
}

func (ret *RuleExecutionTracker) GetExecutionOrder() []string {
	ret.mu.RLock()
	defer ret.mu.RUnlock()

	names := make([]string, len(ret.executions))
	for i, exec := range ret.executions {
		names[i] = exec.RuleName
	}
	return names
}

func (ret *RuleExecutionTracker) Reset() {
	ret.mu.Lock()
	defer ret.mu.Unlock()
	ret.executions = make([]RuleExecution, 0)
}

type TrackedRule struct {
	rule    ValidationRule
	name    string
	tracker *RuleExecutionTracker
}

func NewTrackedRule(name string, rule ValidationRule, tracker *RuleExecutionTracker) *TrackedRule {
	return &TrackedRule{
		rule:    rule,
		name:    name,
		tracker: tracker,
	}
}

func (tr *TrackedRule) Stage() ValidationStage {
	return tr.rule.Stage()
}

func (tr *TrackedRule) Priority() ValidationPriority {
	return tr.rule.Priority()
}

func (tr *TrackedRule) Validate(ctx context.Context, multiErr *errortypes.MultiError) error {
	execution := RuleExecution{
		RuleName:  tr.name,
		Stage:     tr.rule.Stage(),
		Priority:  tr.rule.Priority(),
		StartTime: time.Now(),
	}

	err := tr.rule.Validate(ctx, multiErr)

	execution.EndTime = time.Now()
	execution.Duration = execution.EndTime.Sub(execution.StartTime)
	execution.Error = err

	tr.tracker.Track(execution)

	return err
}

type ValidationAssertions struct {
	t *testing.T
}

func NewValidationAssertions(t *testing.T) *ValidationAssertions {
	return &ValidationAssertions{t: t}
}

func (va *ValidationAssertions) AssertNoErrors(multiErr *errortypes.MultiError) {
	if multiErr != nil && multiErr.HasErrors() {
		va.t.Errorf("Expected no errors but got: %v", multiErr)
	}
}

func (va *ValidationAssertions) AssertHasError(
	multiErr *errortypes.MultiError,
	field string,
	code errortypes.ErrorCode,
) {
	if multiErr == nil || !multiErr.HasErrors() {
		va.t.Error("Expected errors but got none")
		return
	}

	found := false
	for _, err := range multiErr.Errors {
		if err.Field == field && err.Code == code {
			found = true
			break
		}
	}

	if !found {
		va.t.Errorf("Expected error with field=%s and code=%s not found", field, code)
	}
}

func (va *ValidationAssertions) AssertErrorCount(multiErr *errortypes.MultiError, expected int) {
	actual := 0
	if multiErr != nil {
		actual = len(multiErr.Errors)
	}

	if actual != expected {
		va.t.Errorf("Expected %d errors but got %d", expected, actual)
	}
}

func (va *ValidationAssertions) AssertErrorMessage(
	multiErr *errortypes.MultiError,
	field string,
	contains string,
) {
	if multiErr == nil || !multiErr.HasErrors() {
		va.t.Error("Expected errors but got none")
		return
	}

	found := false
	for _, err := range multiErr.Errors {
		if err.Field == field && strings.Contains(err.Message, contains) {
			found = true
			break
		}
	}

	if !found {
		va.t.Errorf(
			"Expected error message containing '%s' for field '%s' not found",
			contains,
			field,
		)
	}
}

type TestDataGenerator struct {
	seed int64
}

func NewTestDataGenerator(seed int64) *TestDataGenerator {
	return &TestDataGenerator{seed: seed}
}

func (tdg *TestDataGenerator) GenerateString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[i%len(charset)]
	}
	return string(result)
}

func (tdg *TestDataGenerator) GenerateEmail() string {
	return fmt.Sprintf("test%d@example.com", tdg.seed)
}

func (tdg *TestDataGenerator) GenerateURL() string {
	return fmt.Sprintf("https://example.com/test/%d", tdg.seed)
}

func (tdg *TestDataGenerator) GeneratePhone() string {
	return fmt.Sprintf("555-%04d-%04d", tdg.seed%10000, (tdg.seed+1)%10000)
}
