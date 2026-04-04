package seeder

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/fatih/color"
	"github.com/uptrace/bun"
)

type Engine struct {
	db              *bun.DB
	registry        *Registry
	tracker         SeedTracker
	reporter        ProgressReporter
	rollbackTracker *RollbackTracker
	cfg             *config.Config
}

func NewEngine(db *bun.DB, registry *Registry, cfg *config.Config) *Engine {
	if db != nil {
		db.RegisterModel(domainregistry.RegisterEntities()...)
	}

	return &Engine{
		db:              db,
		registry:        registry,
		tracker:         NewTracker(db),
		reporter:        NewConsoleReporter(false),
		rollbackTracker: NewRollbackTracker(db),
		cfg:             cfg,
	}
}

func (e *Engine) SetReporter(reporter ProgressReporter) {
	e.reporter = reporter
}

func (e *Engine) SetTracker(tracker SeedTracker) {
	e.tracker = tracker
}

func (e *Engine) Execute(ctx context.Context, opts ExecuteOptions) (*ExecutionReport, error) {
	if err := e.tracker.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize tracker: %w", err)
	}

	if err := e.registry.Validate(); err != nil {
		return nil, fmt.Errorf("registry validation failed: %w", err)
	}

	seeds, err := e.registry.GetExecutionOrder(opts.Environment, opts.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution order: %w", err)
	}

	var toApply []Seed
	var skipped []string

	for _, seed := range seeds {
		applied, err := e.tracker.IsApplied(ctx, seed, opts.Environment)
		if err != nil {
			return nil, fmt.Errorf("failed to check seed status for %s: %w", seed.Name(), err)
		}

		if applied && !opts.Force {
			skipped = append(skipped, seed.Name())
			continue
		}

		toApply = append(toApply, seed)
	}

	if len(toApply) == 0 {
		e.reporter.OnStart(0)
		return &ExecutionReport{
			Skipped: len(skipped),
		}, nil
	}

	if opts.DryRun {
		e.printDryRun(toApply, skipped)
		return &ExecutionReport{
			Applied: len(toApply),
			Skipped: len(skipped),
		}, nil
	}

	if opts.Interactive {
		if !e.confirmExecution(toApply) {
			return nil, ErrUserCancelled
		}
	}

	return e.executeSeeds(ctx, toApply, skipped, opts)
}

func (e *Engine) executeSeeds(
	ctx context.Context,
	seeds []Seed,
	skipped []string,
	opts ExecuteOptions,
) (*ExecutionReport, error) {
	report := &ExecutionReport{
		Results: make([]SeedResult, 0, len(seeds)),
		Skipped: len(skipped),
	}

	startTime := time.Now()
	e.reporter.OnStart(len(seeds))

	if e.cfg != nil {
		ctx = seedhelpers.WithConfig(ctx, e.cfg)
	}

	for _, seed := range seeds {
		e.reporter.OnSeedStart(seed.Name())
		seedStart := time.Now()

		err := e.applySeed(ctx, seed)
		duration := time.Since(seedStart)

		result := SeedResult{
			Name:     seed.Name(),
			Version:  seed.Version(),
			Duration: duration.Milliseconds(),
		}

		if err != nil {
			result.Error = err
			report.Failed++
			report.Results = append(report.Results, result)

			e.reporter.OnSeedError(seed.Name(), err)
			if err := e.tracker.RecordFailure(ctx, seed, opts.Environment, err); err != nil {
				color.Yellow("Warning: failed to record seed failure: %v", err)
			}

			if !opts.IgnoreErrors {
				report.Duration = time.Since(startTime).Milliseconds()
				e.reporter.OnComplete(
					report.Applied,
					report.Skipped,
					report.Failed,
					time.Since(startTime),
				)
				return report, NewSeedError(seed.Name(), PhaseExecution, err)
			}
		} else {
			result.Applied = true
			report.Applied++
			report.Results = append(report.Results, result)

			e.reporter.OnSeedComplete(seed.Name(), duration)
			if err := e.tracker.RecordSuccess(ctx, seed, opts.Environment, duration); err != nil {
				color.Yellow("Warning: failed to record seed success: %v", err)
			}
		}
	}

	report.Duration = time.Since(startTime).Milliseconds()
	e.reporter.OnComplete(report.Applied, report.Skipped, report.Failed, time.Since(startTime))

	return report, nil
}

func (e *Engine) applySeed(ctx context.Context, seed Seed) error {
	return e.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return seed.Run(ctx, tx)
	})
}

func (e *Engine) Rollback(
	ctx context.Context,
	seedName string,
	environment common.Environment,
) error {
	if err := e.rollbackTracker.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize rollback tracker: %w", err)
	}

	seed, exists := e.registry.Get(seedName)
	if !exists {
		return fmt.Errorf("seed %s not found in registry", seedName)
	}

	if !seed.CanRollback() {
		return fmt.Errorf("seed %s does not support rollback", seedName)
	}

	applied, err := e.tracker.IsApplied(ctx, seed, environment)
	if err != nil {
		return fmt.Errorf("check if seed is applied: %w", err)
	}

	if !applied {
		return fmt.Errorf("seed %s has not been applied in %s environment", seedName, environment)
	}

	dependents, err := e.findDependents(seedName)
	if err != nil {
		return fmt.Errorf("find dependent seeds: %w", err)
	}

	if len(dependents) > 0 {
		return fmt.Errorf(
			"cannot rollback %s: the following seeds depend on it: %v",
			seedName,
			dependents,
		)
	}

	color.Yellow("Rolling back seed: %s", seedName)
	start := time.Now()

	err = e.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return seed.Down(ctx, tx)
	})

	duration := time.Since(start)

	if err != nil {
		color.Red("✗ Rollback failed: %v", err)
		if trackErr := e.rollbackTracker.RecordFailure(ctx, seed.Name(), seed.Version(), environment, err.Error()); trackErr != nil {
			color.Yellow("Warning: failed to record rollback failure: %v", trackErr)
		}
		return fmt.Errorf("rollback %s: %w", seedName, err)
	}

	if err := e.rollbackTracker.RecordSuccess(ctx, seed.Name(), seed.Version(), environment, 0, duration); err != nil {
		color.Yellow("Warning: failed to record rollback success: %v", err)
	}

	color.Green("✓ Rolled back %s (%dms)", seedName, duration.Milliseconds())
	return nil
}

func (e *Engine) findDependents(seedName string) ([]string, error) {
	var dependents []string

	for _, seed := range e.registry.All() {
		if slices.Contains(seed.Dependencies(), seedName) {
			dependents = append(dependents, seed.Name())
		}
	}

	return dependents, nil
}

func (e *Engine) printDryRun(toApply []Seed, skipped []string) {
	color.Cyan("\n🔍 Dry run - no changes will be made\n")

	if len(skipped) > 0 {
		color.Yellow("\nSkipped (already applied):")
		for _, name := range skipped {
			fmt.Printf("  - %s\n", name)
		}
	}

	if len(toApply) > 0 {
		color.Green("\nWould apply:")
		for _, seed := range toApply {
			fmt.Printf("  - %s (v%s): %s\n", seed.Name(), seed.Version(), seed.Description())
			if len(seed.Dependencies()) > 0 {
				fmt.Printf("    Dependencies: %s\n", strings.Join(seed.Dependencies(), ", "))
			}
		}
	}

	fmt.Println()
}

func (e *Engine) confirmExecution(seeds []Seed) bool {
	fmt.Printf("\n%s The following %d seed(s) will be applied:\n",
		color.YellowString("?"),
		len(seeds))

	for _, seed := range seeds {
		fmt.Printf("  - %s (v%s)\n", seed.Name(), seed.Version())
	}

	fmt.Printf("\n%s Continue? [y/N]: ", color.YellowString("?"))

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	return response == "y" || response == "yes"
}

func (e *Engine) Status(ctx context.Context) ([]*common.SeedStatus, error) {
	return e.tracker.GetStatus(ctx)
}

func (e *Engine) Registry() *Registry {
	return e.registry
}
