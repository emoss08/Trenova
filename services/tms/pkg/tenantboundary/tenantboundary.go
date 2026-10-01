package tenantboundary

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/shared/pulid"
)

const (
	GinContextKey       = "trenova.tenantboundary"
	maxViolationsPerRun = 8
)

type Source string

const (
	SourceRequestBody    Source = "request_body"
	SourcePath           Source = "path"
	SourceDatabasePolicy Source = "database_policy"
)

type Violation struct {
	Source         Source
	Field          string
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

type Tracker struct {
	mu         sync.Mutex
	violations []Violation
}

func NewTracker() *Tracker {
	return &Tracker{}
}

func (t *Tracker) add(v Violation) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.violations) < maxViolationsPerRun {
		t.violations = append(t.violations, v)
	}
}

func (t *Tracker) Violations() []Violation {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]Violation(nil), t.violations...)
}

type trackerKey struct{}

func With(ctx context.Context, tracker *Tracker) context.Context {
	return context.WithValue(ctx, trackerKey{}, tracker)
}

func From(ctx context.Context) (*Tracker, bool) {
	if ctx == nil {
		return nil, false
	}

	if tracker, ok := ctx.Value(trackerKey{}).(*Tracker); ok && tracker != nil {
		return tracker, true
	}

	tracker, ok := ctx.Value(GinContextKey).(*Tracker)
	return tracker, ok && tracker != nil
}

func Report(ctx context.Context, v Violation) bool {
	tracker, ok := From(ctx)
	if !ok {
		return false
	}

	tracker.add(v)
	return true
}
