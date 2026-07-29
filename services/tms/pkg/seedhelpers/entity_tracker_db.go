package seedhelpers

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

// trackBatchSize keeps a tracking insert inside Postgres's bind-parameter cap.
const trackBatchSize = 2000

type SeedCreatedEntity struct {
	bun.BaseModel `bun:"table:seed_created_entities,alias:sce"`

	ID        int      `bun:"id,pk,autoincrement"`
	SeedName  string   `bun:"seed_name,notnull"`
	TableName string   `bun:"table_name,notnull"`
	EntityID  pulid.ID `bun:"entity_id,notnull"`
	CreatedAt int64    `bun:"created_at,notnull"`
}

// PersistentEntityTracker records what a seed created so a later rollback can
// find it. It takes a bun.IDB rather than a *bun.DB because seeds run inside a
// transaction: writing the tracking rows on the same handle is what makes them
// roll back with the data they describe.
type PersistentEntityTracker struct {
	db bun.IDB
}

func NewPersistentEntityTracker(db bun.IDB) *PersistentEntityTracker {
	return &PersistentEntityTracker{db: db}
}

func (t *PersistentEntityTracker) Track(
	ctx context.Context,
	table string,
	id pulid.ID,
	seedName string,
) error {
	return t.TrackBatch(ctx, []TrackedEntity{{Table: table, ID: id, SeedName: seedName}})
}

// TrackBatch writes a whole seed's tracking rows in one statement. Seeds create
// thousands of entities, and a round trip apiece would dominate their runtime.
func (t *PersistentEntityTracker) TrackBatch(
	ctx context.Context,
	tracked []TrackedEntity,
) error {
	if len(tracked) == 0 {
		return nil
	}

	now := timeutils.NowUnix()

	// Chunked because every row costs four bind parameters and Postgres caps a
	// statement at 65535 of them.
	for chunk := range slices.Chunk(tracked, trackBatchSize) {
		entities := make([]SeedCreatedEntity, 0, len(chunk))
		for _, entity := range chunk {
			entities = append(entities, SeedCreatedEntity{
				SeedName:  entity.SeedName,
				TableName: entity.Table,
				EntityID:  entity.ID,
				CreatedAt: now,
			})
		}

		_, err := t.db.NewInsert().
			Model(&entities).
			On("CONFLICT (seed_name, table_name, entity_id) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("insert seed created entities: %w", err)
		}
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
