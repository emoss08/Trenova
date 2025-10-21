package seeder

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/fatih/color"
	"github.com/uptrace/bun"
)

type SeedStatus string

const (
	SeedStatusActive   SeedStatus = "Active"
	SeedStatusInactive SeedStatus = "Inactive"
	SeedStatusOrphaned SeedStatus = "Orphaned"
)

type SeedRunner interface {
	Name() string
	Version() string
	Description() string
	Environment() []common.Environment
	Run(ctx context.Context, db *bun.DB) error
}

type SeedRecord struct {
	bun.BaseModel `bun:"table:seed_history,alias:sh"`

	ID          pulid.ID           `bun:"id,pk"`
	Name        string             `bun:"name,notnull"`
	Version     string             `bun:"version,notnull"`
	Environment common.Environment `bun:"environment,notnull"`
	Checksum    string             `bun:"checksum,notnull"`
	AppliedAt   int64              `bun:"applied_at,notnull"`
	AppliedBy   string             `bun:"applied_by,notnull"`
	Status      SeedStatus         `bun:"status,notnull,type:seed_status_enum"`
	Details     map[string]any     `bun:"details,type:jsonb"`
	Error       string             `bun:"error"`
	Notes       string             `bun:"notes"`
}

type Seeder struct {
	db          *bun.DB
	config      *common.DatabaseConfig
	runners     []SeedRunner
	runnerIndex map[string]int // Track runner positions for deduplication
	reporter    common.ProgressReporter
}

func NewSeeder(config *common.DatabaseConfig) *Seeder {
	return &Seeder{
		db:          config.DB,
		config:      config,
		runners:     make([]SeedRunner, 0),
		runnerIndex: make(map[string]int),
		reporter:    common.NewConsoleProgressReporter(),
	}
}

func (s *Seeder) RegisterRunner(runner SeedRunner) {
	if _, exists := s.runnerIndex[runner.Name()]; exists {
		idx := s.runnerIndex[runner.Name()]
		s.runners[idx] = runner
	} else {
		s.runners = append(s.runners, runner)
		s.runnerIndex[runner.Name()] = len(s.runners) - 1
	}
}

func (s *Seeder) SetProgressReporter(reporter common.ProgressReporter) {
	s.reporter = reporter
}

func (s *Seeder) Initialize(ctx context.Context) error {
	query := `
		-- Create seed status enum if it doesn't exist
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'seed_status_enum') THEN
				CREATE TYPE seed_status_enum AS ENUM ('Active', 'Inactive', 'Orphaned');
			END IF;
		END $$;

		CREATE TABLE IF NOT EXISTS seed_history (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			version VARCHAR(50) NOT NULL,
			environment VARCHAR(50) NOT NULL,
			checksum VARCHAR(32) NOT NULL,
			applied_at BIGINT NOT NULL,
			applied_by VARCHAR(255) NOT NULL,
			status seed_status_enum NOT NULL DEFAULT 'Active',
			details JSONB,
			error TEXT,
			notes TEXT,
			UNIQUE(name, version, environment)
		);

		CREATE INDEX IF NOT EXISTS idx_seed_history_name ON seed_history(name);
		CREATE INDEX IF NOT EXISTS idx_seed_history_environment ON seed_history(environment);
		CREATE INDEX IF NOT EXISTS idx_seed_history_applied_at ON seed_history(applied_at);
		CREATE INDEX IF NOT EXISTS idx_seed_history_status ON seed_history(status);
	`

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create seed history table: %w", err)
	}

	color.Green("✓ Seed tracking table initialized")
	return nil
}

func (s *Seeder) Seed(
	ctx context.Context,
	opts common.OperationOptions,
) (*common.OperationResult, error) {
	result := &common.OperationResult{
		Type:      common.OpSeed,
		StartTime: time.Now(),
		Details:   make(map[string]any),
	}

	if err := s.Initialize(ctx); err != nil {
		result.Error = err
		result.Success = false
		result.Message = fmt.Sprintf("Failed to initialize seed tracking: %v", err)
		return result, err
	}

	var runnersToExecute []SeedRunner
	for _, runner := range s.runners {
		if !s.supportsEnvironment(runner, opts.Environment) {
			continue
		}

		applied, err := s.isSeedApplied(ctx, runner, opts.Environment)
		if err != nil {
			result.Error = err
			result.Success = false
			result.Message = fmt.Sprintf("Failed to check seed status: %v", err)
			return result, err
		}

		if applied && !opts.Force {
			color.Yellow("→ Skipping %s (already applied)", runner.Name())
			continue
		}

		runnersToExecute = append(runnersToExecute, runner)
	}

	if len(runnersToExecute) == 0 {
		result.Success = true
		result.Message = "No seeds to apply"
		result.EndTime = time.Now()
		color.Yellow("→ No seeds to apply")
		return result, nil
	}

	color.Cyan("Seeds to apply:")
	for _, runner := range runnersToExecute {
		fmt.Printf("  - %s (v%s): %s\n", runner.Name(), runner.Version(), runner.Description())
	}

	if opts.DryRun {
		result.Success = true
		result.Message = fmt.Sprintf("Dry run: Would apply %d seeds", len(runnersToExecute))
		result.EndTime = time.Now()
		color.Blue("→ Dry run completed (no changes made)")
		return result, nil
	}

	if opts.Interactive && !opts.Force {
		fmt.Printf("\n%s Apply %d seed(s)? [y/N]: ",
			color.YellowString("?"),
			len(runnersToExecute))

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			result.Success = false
			result.Message = "Seeding cancelled by user"
			result.EndTime = time.Now()
			color.Red("✗ Seeding cancelled")
			return result, nil
		}
	}

	s.reporter.Start(len(runnersToExecute), "Applying seeds...")

	appliedCount := 0
	failedCount := 0

	for i, runner := range runnersToExecute {
		s.reporter.Update(i+1, fmt.Sprintf("Applying %s...", runner.Name()))

		if err := s.applySeed(ctx, runner, opts); err != nil {
			failedCount++
			color.Red("✗ Failed to apply %s: %v", runner.Name(), err)

			if !opts.Force {
				result.Error = err
				result.Success = false
				result.Message = fmt.Sprintf("Failed to apply seed %s: %v", runner.Name(), err)
				result.EndTime = time.Now()
				return result, err
			}
		} else {
			appliedCount++
			color.Green("✓ Applied %s (v%s)", runner.Name(), runner.Version())
		}
	}

	result.Success = failedCount == 0
	result.Details["applied"] = appliedCount
	result.Details["failed"] = failedCount
	result.Message = fmt.Sprintf("Applied %d seeds (%d failed)", appliedCount, failedCount)
	result.EndTime = time.Now()

	s.reporter.Complete("Seeding completed")

	if failedCount > 0 {
		color.Yellow("→ Seeding completed with %d failures", failedCount)
	} else {
		color.Green("✓ All seeds applied successfully")
	}

	return result, nil
}

func (s *Seeder) Status(ctx context.Context) ([]*common.SeedStatus, error) {
	var records []SeedRecord
	err := s.db.NewSelect().
		Model(&records).
		OrderExpr("applied_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get seed history: %w", err)
	}

	var status []*common.SeedStatus
	for _, record := range records {
		status = append(status, &common.SeedStatus{
			Name:        record.Name,
			Version:     record.Version,
			AppliedAt:   time.Unix(record.AppliedAt, 0),
			Checksum:    record.Checksum,
			Environment: record.Environment,
			Status:      string(record.Status),
		})
	}

	return status, nil
}

func (s *Seeder) supportsEnvironment(runner SeedRunner, env common.Environment) bool {
	for _, supported := range runner.Environment() {
		if supported == env {
			return true
		}
	}
	return false
}

func (s *Seeder) isSeedApplied(
	ctx context.Context,
	runner SeedRunner,
	env common.Environment,
) (bool, error) {
	exists, err := s.db.NewSelect().
		Model((*SeedRecord)(nil)).
		Where("name = ?", runner.Name()).
		Where("version = ?", runner.Version()).
		Where("environment = ?", env).
		Where("status = ?", SeedStatusActive).
		Exists(ctx)

	return exists, err
}

func (s *Seeder) applySeed(
	ctx context.Context,
	runner SeedRunner,
	opts common.OperationOptions,
) error {
	checksum := s.calculateChecksum(runner)

	record := &SeedRecord{
		ID:          pulid.MustNew("seed_"),
		Name:        runner.Name(),
		Version:     runner.Version(),
		Environment: opts.Environment,
		Checksum:    checksum,
		AppliedAt:   time.Now().Unix(),
		AppliedBy:   "system", // TODO: get from context
		Status:      SeedStatusActive,
		Details:     make(map[string]any),
	}

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := runner.Run(ctx, s.db); err != nil {
			record.Status = SeedStatusInactive
			record.Error = err.Error()
			if record.ID == "" {
				record.ID = pulid.MustNew("seed_")
			}

			if _, insertErr := tx.NewInsert().Model(record).Exec(ctx); insertErr != nil {
				color.Yellow("Warning: Failed to save seed record: %v", insertErr)
			}

			return fmt.Errorf("seed application failed: %w", err)
		}

		if _, err := tx.NewInsert().
			Model(record).
			On("CONFLICT (name, version, environment) DO UPDATE").
			Set("applied_at = EXCLUDED.applied_at").
			Set("applied_by = EXCLUDED.applied_by").
			Set("details = EXCLUDED.details").
			Set("checksum = EXCLUDED.checksum").
			Set("status = EXCLUDED.status").
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to save seed record: %w", err)
		}

		return nil
	})

	return err
}

func (s *Seeder) calculateChecksum(runner SeedRunner) string {
	data := map[string]string{
		"name":        runner.Name(),
		"version":     runner.Version(),
		"description": runner.Description(),
	}

	jsonData, _ := json.Marshal(data)
	sum := md5.Sum(jsonData)
	return fmt.Sprintf("%x", sum)
}
