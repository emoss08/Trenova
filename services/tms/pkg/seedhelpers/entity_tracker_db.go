package seedhelpers

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type SeedCreatedEntity struct {
	bun.BaseModel `bun:"table:seed_created_entities,alias:sce"`

	ID        int      `bun:"id,pk,autoincrement"`
	SeedName  string   `bun:"seed_name,notnull"`
	TableName string   `bun:"table_name,notnull"`
	EntityID  pulid.ID `bun:"entity_id,notnull"`
	CreatedAt int64    `bun:"created_at,notnull"`
}

type PersistentEntityTracker struct {
	db *bun.DB
}

func NewPersistentEntityTracker(db *bun.DB) *PersistentEntityTracker {
	return &PersistentEntityTracker{db: db}
}

func (t *PersistentEntityTracker) Track(
	ctx context.Context,
	table string,
	id pulid.ID,
	seedName string,
) error {
	entity := &SeedCreatedEntity{
		SeedName:  seedName,
		TableName: table,
		EntityID:  id,
		CreatedAt: time.Now().Unix(),
	}

	_, err := t.db.NewInsert().
		Model(entity).
		On("CONFLICT (seed_name, table_name, entity_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert seed created entity: %w", err)
	}

	return nil
}

func (t *PersistentEntityTracker) GetBySeed(
	ctx context.Context,
	seedName string,
) ([]TrackedEntity, error) {
	var entities []SeedCreatedEntity

	err := t.db.NewSelect().
		Model(&entities).
		Where("seed_name = ?", seedName).
		Order("id ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get created entities for seed %s: %w", seedName, err)
	}

	result := make([]TrackedEntity, len(entities))
	for i, entity := range entities {
		result[i] = TrackedEntity{
			Table:    entity.TableName,
			ID:       entity.EntityID,
			SeedName: entity.SeedName,
		}
	}

	return result, nil
}

func (t *PersistentEntityTracker) GetAll(ctx context.Context) ([]TrackedEntity, error) {
	var entities []SeedCreatedEntity

	err := t.db.NewSelect().
		Model(&entities).
		Order("id ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all created entities: %w", err)
	}

	result := make([]TrackedEntity, len(entities))
	for i, entity := range entities {
		result[i] = TrackedEntity{
			Table:    entity.TableName,
			ID:       entity.EntityID,
			SeedName: entity.SeedName,
		}
	}

	return result, nil
}

func (t *PersistentEntityTracker) DeleteBySeed(ctx context.Context, seedName string) error {
	_, err := t.db.NewDelete().
		Model((*SeedCreatedEntity)(nil)).
		Where("seed_name = ?", seedName).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete tracked entities for seed %s: %w", seedName, err)
	}

	return nil
}

func (t *PersistentEntityTracker) Count(ctx context.Context) (int, error) {
	count, err := t.db.NewSelect().
		Model((*SeedCreatedEntity)(nil)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count created entities: %w", err)
	}

	return count, nil
}

func (t *PersistentEntityTracker) CountBySeed(ctx context.Context, seedName string) (int, error) {
	count, err := t.db.NewSelect().
		Model((*SeedCreatedEntity)(nil)).
		Where("seed_name = ?", seedName).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count created entities for seed %s: %w", seedName, err)
	}

	return count, nil
}

func (t *PersistentEntityTracker) Clear(ctx context.Context) error {
	_, err := t.db.NewDelete().
		Model((*SeedCreatedEntity)(nil)).
		Where("1=1").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("clear all tracked entities: %w", err)
	}

	return nil
}
