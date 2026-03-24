package testutil

import (
	"context"
	"testing"
	"time"
)

type Container interface {
	Terminate(ctx context.Context) error
}

type TestContext struct {
	T          *testing.T
	Ctx        context.Context
	Cancel     context.CancelFunc
	Containers []Container
}

func NewTestContext(t *testing.T) *TestContext {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	return &TestContext{
		T:          t,
		Ctx:        ctx,
		Cancel:     cancel,
		Containers: make([]Container, 0),
	}
}

func NewTestContextWithTimeout(t *testing.T, timeout time.Duration) *TestContext {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	return &TestContext{
		T:          t,
		Ctx:        ctx,
		Cancel:     cancel,
		Containers: make([]Container, 0),
	}
}

func (tc *TestContext) AddContainer(c Container) {
	tc.Containers = append(tc.Containers, c)
}

func (tc *TestContext) Cleanup() {
	tc.Cancel()
	for _, c := range tc.Containers {
		if err := c.Terminate(tc.Ctx); err != nil {
			tc.T.Logf("failed to terminate container: %v", err)
		}
	}
}

func (tc *TestContext) RegisterCleanup() {
	tc.T.Cleanup(tc.Cleanup)
}

func (tc *TestContext) Deadline() (time.Time, bool) {
	return tc.Ctx.Deadline()
}
