package seeder

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type SeedTracker interface {
	Initialize(ctx context.Context) error
	IsApplied(ctx context.Context, seed Seed, env common.Environment) (bool, error)
	RecordSuccess(
		ctx context.Context,
		seed Seed,
		env common.Environment,
		duration time.Duration,
	) error
	RecordFailure(ctx context.Context, seed Seed, env common.Environment, seedErr error) error
	GetStatus(ctx context.Context) ([]*common.SeedStatus, error)
}

type SeedStatus string

const (
	SeedStatusActive   SeedStatus = "Active"
	SeedStatusInactive SeedStatus = "Inactive"
	SeedStatusOrphaned SeedStatus = "Orphaned"
)

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
	DurationMs  int64              `bun:"duration_ms"`
}

type Tracker struct {
	db *bun.DB
}

func NewTracker(db *bun.DB) *Tracker {
	return &Tracker{db: db}
}

func (t *Tracker) Initialize(ctx context.Context) error {
	query := `
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
			duration_ms BIGINT,
			UNIQUE(name, version, environment)
		);

		CREATE INDEX IF NOT EXISTS idx_seed_history_name ON seed_history(name);
		CREATE INDEX IF NOT EXISTS idx_seed_history_environment ON seed_history(environment);
		CREATE INDEX IF NOT EXISTS idx_seed_history_applied_at ON seed_history(applied_at);
		CREATE INDEX IF NOT EXISTS idx_seed_history_status ON seed_history(status);
	`

	_, err := t.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to initialize seed tracking table: %w", err)
	}

	return nil
}

func (t *Tracker) IsApplied(ctx context.Context, seed Seed, env common.Environment) (bool, error) {
	exists, err := t.db.NewSelect().
		Model((*SeedRecord)(nil)).
		Where("name = ?", seed.Name()).
		Where("version = ?", seed.Version()).
		Where("environment = ?", env).
		Where("status = ?", SeedStatusActive).
		Exists(ctx)

	return exists, err
}

func (t *Tracker) RecordSuccess(
	ctx context.Context,
	seed Seed,
	env common.Environment,
	duration time.Duration,
) error {
	record := &SeedRecord{
		ID:          pulid.MustNew("seed_"),
		Name:        seed.Name(),
		Version:     seed.Version(),
		Environment: env,
		Checksum:    t.calculateChecksum(seed),
		AppliedAt:   time.Now().Unix(),
		AppliedBy:   "system",
		Status:      SeedStatusActive,
		Details:     make(map[string]any),
		DurationMs:  duration.Milliseconds(),
	}

	_, err := t.db.NewInsert().
		Model(record).
		On("CONFLICT (name, version, environment) DO UPDATE").
		Set("applied_at = EXCLUDED.applied_at").
		Set("applied_by = EXCLUDED.applied_by").
		Set("checksum = EXCLUDED.checksum").
		Set("status = EXCLUDED.status").
		Set("duration_ms = EXCLUDED.duration_ms").
		Set("error = NULL").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to record seed success: %w", err)
	}

	return nil
}

func (t *Tracker) RecordFailure(
	ctx context.Context,
	seed Seed,
	env common.Environment,
	seedErr error,
) error {
	record := &SeedRecord{
		ID:          pulid.MustNew("seed_"),
		Name:        seed.Name(),
		Version:     seed.Version(),
		Environment: env,
		Checksum:    t.calculateChecksum(seed),
		AppliedAt:   time.Now().Unix(),
		AppliedBy:   "system",
		Status:      SeedStatusInactive,
		Details:     make(map[string]any),
		Error:       seedErr.Error(),
	}

	_, err := t.db.NewInsert().
		Model(record).
		On("CONFLICT (name, version, environment) DO UPDATE").
		Set("applied_at = EXCLUDED.applied_at").
		Set("status = EXCLUDED.status").
		Set("error = EXCLUDED.error").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to record seed failure: %w", err)
	}

	return nil
}

func (t *Tracker) GetStatus(ctx context.Context) ([]*common.SeedStatus, error) {
	var records []SeedRecord
	err := t.db.NewSelect().
		Model(&records).
		OrderExpr("applied_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get seed history: %w", err)
	}

	status := make([]*common.SeedStatus, 0, len(records))
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

func (t *Tracker) MarkOrphaned(ctx context.Context, name string, env common.Environment) error {
	_, err := t.db.NewUpdate().
		Model((*SeedRecord)(nil)).
		Set("status = ?", SeedStatusOrphaned).
		Where("name = ?", name).
		Where("environment = ?", env).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to mark seed as orphaned: %w", err)
	}

	return nil
}

func (t *Tracker) calculateChecksum(seed Seed) string {
	data := fmt.Sprintf("%s:%s:%s", seed.Name(), seed.Version(), seed.Description())
	sum := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", sum)
}
