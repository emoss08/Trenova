package seeder

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/uptrace/bun"
)

type MockSeedOption func(*MockSeed)

type MockSeed struct {
	mu           sync.Mutex
	name         string
	version      string
	description  string
	environments []common.Environment
	dependencies []string
	runFunc      func(context.Context, bun.Tx) error
	runCalls     int
	runArgs      []runCallArgs
	downFunc     func(context.Context, bun.Tx) error
	downCalls    int
	canRollback  bool
}

type runCallArgs struct {
	Ctx context.Context
	Tx  bun.Tx
}

func NewMockSeed(name string, opts ...MockSeedOption) *MockSeed {
	m := &MockSeed{
		name:         name,
		version:      "1.0.0",
		description:  "Mock seed: " + name,
		environments: []common.Environment{common.EnvDevelopment, common.EnvTest},
		dependencies: []string{},
		runFunc:      func(context.Context, bun.Tx) error { return nil },
		runArgs:      make([]runCallArgs, 0),
		downFunc:     func(context.Context, bun.Tx) error { return nil },
		canRollback:  false,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func WithVersion(v string) MockSeedOption {
	return func(m *MockSeed) {
		m.version = v
	}
}

func WithDescription(d string) MockSeedOption {
	return func(m *MockSeed) {
		m.description = d
	}
}

func WithEnvironments(envs ...common.Environment) MockSeedOption {
	return func(m *MockSeed) {
		m.environments = envs
	}
}

func WithDependencies(deps ...string) MockSeedOption {
	return func(m *MockSeed) {
		m.dependencies = deps
	}
}

func WithRunFunc(fn func(context.Context, bun.Tx) error) MockSeedOption {
	return func(m *MockSeed) {
		m.runFunc = fn
	}
}

func WithRunError(err error) MockSeedOption {
	return func(m *MockSeed) {
		m.runFunc = func(context.Context, bun.Tx) error { return err }
	}
}

func WithDownFunc(fn func(context.Context, bun.Tx) error) MockSeedOption {
	return func(m *MockSeed) {
		m.downFunc = fn
	}
}

func WithDownError(err error) MockSeedOption {
	return func(m *MockSeed) {
		m.downFunc = func(context.Context, bun.Tx) error { return err }
	}
}

func WithCanRollback(canRollback bool) MockSeedOption {
	return func(m *MockSeed) {
		m.canRollback = canRollback
	}
}

func (m *MockSeed) Name() string {
	return m.name
}

func (m *MockSeed) Version() string {
	return m.version
}

func (m *MockSeed) Description() string {
	return m.description
}

func (m *MockSeed) Environments() []common.Environment {
	return m.environments
}

func (m *MockSeed) Dependencies() []string {
	return m.dependencies
}

func (m *MockSeed) Run(ctx context.Context, tx bun.Tx) error {
	m.mu.Lock()
	m.runCalls++
	m.runArgs = append(m.runArgs, runCallArgs{Ctx: ctx, Tx: tx})
	m.mu.Unlock()
	return m.runFunc(ctx, tx)
}

func (m *MockSeed) Down(ctx context.Context, tx bun.Tx) error {
	m.mu.Lock()
	m.downCalls++
	m.mu.Unlock()
	return m.downFunc(ctx, tx)
}

func (m *MockSeed) CanRollback() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.canRollback
}

func (m *MockSeed) RunCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.runCalls
}

func (m *MockSeed) DownCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.downCalls
}

func (m *MockSeed) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runCalls = 0
	m.runArgs = make([]runCallArgs, 0)
	m.downCalls = 0
}

type MockReporter struct {
	mu            sync.Mutex
	StartCalls    []int
	SeedStarts    []string
	SeedSkips     []SeedSkipCall
	SeedCompletes []SeedCompleteCall
	SeedErrors    []SeedErrorCall
	CompleteCalls []CompleteCall
}

type SeedSkipCall struct {
	Name   string
	Reason string
}

type SeedCompleteCall struct {
	Name     string
	Duration time.Duration
}

type SeedErrorCall struct {
	Name string
	Err  error
}

type CompleteCall struct {
	Applied  int
	Skipped  int
	Failed   int
	Duration time.Duration
}

func NewMockReporter() *MockReporter {
	return &MockReporter{
		StartCalls:    make([]int, 0),
		SeedStarts:    make([]string, 0),
		SeedSkips:     make([]SeedSkipCall, 0),
		SeedCompletes: make([]SeedCompleteCall, 0),
		SeedErrors:    make([]SeedErrorCall, 0),
		CompleteCalls: make([]CompleteCall, 0),
	}
}

func (r *MockReporter) OnStart(total int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.StartCalls = append(r.StartCalls, total)
}

func (r *MockReporter) OnSeedStart(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.SeedStarts = append(r.SeedStarts, name)
}

func (r *MockReporter) OnSeedSkip(name, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.SeedSkips = append(r.SeedSkips, SeedSkipCall{Name: name, Reason: reason})
}

func (r *MockReporter) OnSeedComplete(name string, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.SeedCompletes = append(r.SeedCompletes, SeedCompleteCall{Name: name, Duration: duration})
}

func (r *MockReporter) OnSeedError(name string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.SeedErrors = append(r.SeedErrors, SeedErrorCall{Name: name, Err: err})
}

func (r *MockReporter) OnComplete(applied, skipped, failed int, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.CompleteCalls = append(r.CompleteCalls, CompleteCall{
		Applied:  applied,
		Skipped:  skipped,
		Failed:   failed,
		Duration: duration,
	})
}

func (r *MockReporter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.StartCalls = make([]int, 0)
	r.SeedStarts = make([]string, 0)
	r.SeedSkips = make([]SeedSkipCall, 0)
	r.SeedCompletes = make([]SeedCompleteCall, 0)
	r.SeedErrors = make([]SeedErrorCall, 0)
	r.CompleteCalls = make([]CompleteCall, 0)
}

type SeedInterface interface {
	Name() string
	Version() string
	Description() string
	Environments() []common.Environment
	Dependencies() []string
}

type MockTracker struct {
	mu                 sync.Mutex
	InitializeFunc     func(context.Context) error
	IsAppliedFunc      func(context.Context, SeedInterface, common.Environment) (bool, error)
	RecordSuccessFunc  func(context.Context, SeedInterface, common.Environment, time.Duration) error
	RecordFailureFunc  func(context.Context, SeedInterface, common.Environment, error) error
	GetStatusFunc      func(context.Context) ([]*common.SeedStatus, error)
	InitializeCalls    int
	IsAppliedCalls     []IsAppliedCall
	RecordSuccessCalls []RecordSuccessCall
	RecordFailureCalls []RecordFailureCall
	appliedSeeds       map[string]bool
}

type IsAppliedCall struct {
	Name    string
	Version string
	Env     common.Environment
}

type RecordSuccessCall struct {
	Name     string
	Version  string
	Env      common.Environment
	Duration time.Duration
}

type RecordFailureCall struct {
	Name string
	Env  common.Environment
	Err  error
}

func NewMockTracker() *MockTracker {
	return &MockTracker{
		IsAppliedCalls:     make([]IsAppliedCall, 0),
		RecordSuccessCalls: make([]RecordSuccessCall, 0),
		RecordFailureCalls: make([]RecordFailureCall, 0),
		appliedSeeds:       make(map[string]bool),
	}
}

func (t *MockTracker) Initialize(ctx context.Context) error {
	t.mu.Lock()
	t.InitializeCalls++
	t.mu.Unlock()
	if t.InitializeFunc != nil {
		return t.InitializeFunc(ctx)
	}
	return nil
}

func (t *MockTracker) IsApplied(
	ctx context.Context,
	seed SeedInterface,
	env common.Environment,
) (bool, error) {
	t.mu.Lock()
	t.IsAppliedCalls = append(
		t.IsAppliedCalls,
		IsAppliedCall{Name: seed.Name(), Version: seed.Version(), Env: env},
	)
	key := seed.Name() + ":" + seed.Version() + ":" + string(env)
	applied := t.appliedSeeds[key]
	t.mu.Unlock()
	if t.IsAppliedFunc != nil {
		return t.IsAppliedFunc(ctx, seed, env)
	}
	return applied, nil
}

func (t *MockTracker) RecordSuccess(
	ctx context.Context,
	seed SeedInterface,
	env common.Environment,
	duration time.Duration,
) error {
	t.mu.Lock()
	t.RecordSuccessCalls = append(t.RecordSuccessCalls, RecordSuccessCall{
		Name:     seed.Name(),
		Version:  seed.Version(),
		Env:      env,
		Duration: duration,
	})
	key := seed.Name() + ":" + seed.Version() + ":" + string(env)
	t.appliedSeeds[key] = true
	t.mu.Unlock()
	if t.RecordSuccessFunc != nil {
		return t.RecordSuccessFunc(ctx, seed, env, duration)
	}
	return nil
}

func (t *MockTracker) RecordFailure(
	ctx context.Context,
	seed SeedInterface,
	env common.Environment,
	seedErr error,
) error {
	t.mu.Lock()
	t.RecordFailureCalls = append(t.RecordFailureCalls, RecordFailureCall{
		Name: seed.Name(),
		Env:  env,
		Err:  seedErr,
	})
	t.mu.Unlock()
	if t.RecordFailureFunc != nil {
		return t.RecordFailureFunc(ctx, seed, env, seedErr)
	}
	return nil
}

func (t *MockTracker) GetStatus(ctx context.Context) ([]*common.SeedStatus, error) {
	if t.GetStatusFunc != nil {
		return t.GetStatusFunc(ctx)
	}
	return []*common.SeedStatus{}, nil
}

func (t *MockTracker) MarkApplied(name, version string, env common.Environment) {
	t.mu.Lock()
	defer t.mu.Unlock()
	key := name + ":" + version + ":" + string(env)
	t.appliedSeeds[key] = true
}

func (t *MockTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.InitializeCalls = 0
	t.IsAppliedCalls = make([]IsAppliedCall, 0)
	t.RecordSuccessCalls = make([]RecordSuccessCall, 0)
	t.RecordFailureCalls = make([]RecordFailureCall, 0)
	t.appliedSeeds = make(map[string]bool)
}
