package seeder

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/uptrace/bun"
)

type RollbackTracker struct {
	db *bun.DB
}

func NewRollbackTracker(db *bun.DB) *RollbackTracker {
	return &RollbackTracker{db: db}
}

func (rt *RollbackTracker) Initialize(ctx context.Context) error {
	statements, err := loadSchema(dbdialect.FromBun(rt.db), seedRollbacksSchema)
	if err != nil {
		return err
	}

	for _, statement := range statements {
		if _, execErr := rt.db.ExecContext(ctx, statement); execErr != nil {
			return fmt.Errorf("create seed_rollbacks table: %w", execErr)
		}
	}

	return nil
}

func (rt *RollbackTracker) RecordSuccess(
	ctx context.Context,
	seedName string,
	seedVersion string,
	environment common.Environment,
	entitiesDeleted int,
	duration time.Duration,
) error {
	query := `
		INSERT INTO seed_rollbacks (
			seed_name,
			seed_version,
			environment,
			rolled_back_at,
			entities_deleted,
			duration_ms
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := rt.db.ExecContext(
		ctx,
		query,
		seedName,
		seedVersion,
		string(environment),
		time.Now(),
		entitiesDeleted,
		duration.Milliseconds(),
	)
	if err != nil {
		return fmt.Errorf("record rollback success: %w", err)
	}

	return nil
}

func (rt *RollbackTracker) RecordFailure(
	ctx context.Context,
	seedName string,
	seedVersion string,
	environment common.Environment,
	errorMessage string,
) error {
	query := `
		INSERT INTO seed_rollbacks (
			seed_name,
			seed_version,
			environment,
			rolled_back_at,
			error_message
		) VALUES (?, ?, ?, ?, ?)
	`

	_, err := rt.db.ExecContext(
		ctx,
		query,
		seedName,
		seedVersion,
		string(environment),
		time.Now(),
		errorMessage,
	)
	if err != nil {
		return fmt.Errorf("record rollback failure: %w", err)
	}

	return nil
}

type RollbackHistory struct {
	ID              int
	SeedName        string
	SeedVersion     string
	Environment     string
	RolledBackAt    time.Time
	EntitiesDeleted int
	DurationMs      int64
	ErrorMessage    *string
}

func (rt *RollbackTracker) GetHistory(
	ctx context.Context,
	seedName string,
) ([]RollbackHistory, error) {
	var history []RollbackHistory

	err := rt.db.NewSelect().
		ColumnExpr("id, seed_name, seed_version, environment, rolled_back_at, entities_deleted, duration_ms, error_message").
		TableExpr("seed_rollbacks").
		Where("seed_name = ?", seedName).
		OrderExpr("rolled_back_at DESC").
		Limit(10).
		Scan(ctx, &history)
	if err != nil {
		return nil, fmt.Errorf("get rollback history for %s: %w", seedName, err)
	}

	return history, nil
}

func (rt *RollbackTracker) GetAllHistory(ctx context.Context) ([]RollbackHistory, error) {
	var history []RollbackHistory

	err := rt.db.NewSelect().
		ColumnExpr("id, seed_name, seed_version, environment, rolled_back_at, entities_deleted, duration_ms, error_message").
		TableExpr("seed_rollbacks").
		OrderExpr("rolled_back_at DESC").
		Limit(50).
		Scan(ctx, &history)
	if err != nil {
		return nil, fmt.Errorf("get all rollback history: %w", err)
	}

	return history, nil
}

func (rt *RollbackTracker) LastRollback(
	ctx context.Context,
	seedName string,
) (*RollbackHistory, error) {
	var history RollbackHistory

	err := rt.db.NewSelect().
		ColumnExpr("id, seed_name, seed_version, environment, rolled_back_at, entities_deleted, duration_ms, error_message").
		TableExpr("seed_rollbacks").
		Where("seed_name = ?", seedName).
		OrderExpr("rolled_back_at DESC").
		Limit(1).
		Scan(ctx, &history)
	if err != nil {
		return nil, fmt.Errorf("get last rollback for %s: %w", seedName, err)
	}

	return &history, nil
}
