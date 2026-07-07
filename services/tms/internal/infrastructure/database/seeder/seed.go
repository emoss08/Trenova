package seeder

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/uptrace/bun"
)

type Seed interface {
	Name() string
	Version() string
	Description() string
	Environments() []common.Environment
	Dependencies() []string
	Run(ctx context.Context, tx bun.Tx) error
	Down(ctx context.Context, tx bun.Tx) error
	CanRollback() bool
}

// Repeatable is an optional interface a Seed may implement to opt into
// re-running on every seed invocation, even when already recorded as applied.
// Use it for idempotent reconciliation seeds that must pick up newly registered
// state (e.g. syncing role permissions with the resource registry) without
// requiring a version bump each time the registry changes.
type Repeatable interface {
	Repeatable() bool
}

func isRepeatable(seed Seed) bool {
	r, ok := seed.(Repeatable)
	return ok && r.Repeatable()
}

type ExecuteOptions struct {
	Environment  common.Environment
	Target       string
	Force        bool
	IgnoreErrors bool
	DryRun       bool
	Verbose      bool
	Interactive  bool
}

type SeedResult struct {
	Name     string
	Version  string
	Duration int64
	Applied  bool
	Skipped  bool
	Error    error
}

type ExecutionReport struct {
	Results  []SeedResult
	Applied  int
	Skipped  int
	Failed   int
	Duration int64
}

func (r *ExecutionReport) Success() bool {
	return r.Failed == 0
}
