package seedhelpers

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func DeleteTrackedEntities(ctx context.Context, tx bun.Tx, seedName string, sc *SeedContext) error {
	entities, err := sc.GetCreatedEntities(ctx, seedName)
	if err != nil {
		return fmt.Errorf("get created entities: %w", err)
	}

	if len(entities) == 0 {
		return nil
	}

	for i := len(entities) - 1; i >= 0; i-- {
		entity := entities[i]

		_, err := tx.NewDelete().
			Table(entity.Table).
			Where("id = ?", entity.ID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("delete %s %s: %w", entity.Table, entity.ID, err)
		}

		sc.Logger().Info("  Deleted %s: %s", entity.Table, entity.ID)
	}

	if err := sc.DeleteTrackedEntities(ctx, seedName); err != nil {
		return fmt.Errorf("delete tracked entities records: %w", err)
	}

	return nil
}

func DeleteEntitiesByTable(ctx context.Context, tx bun.Tx, table string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	result, err := tx.NewDelete().
		Table(table).
		Where("id IN (?)", bun.List(ids)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete from %s: %w", table, err)
	}

	affected, _ := result.RowsAffected()
	if affected != int64(len(ids)) {
		return fmt.Errorf(
			"expected to delete %d rows from %s, but deleted %d",
			len(ids),
			table,
			affected,
		)
	}

	return nil
}

func VerifyEntityExists(ctx context.Context, tx bun.Tx, table string, id string) (bool, error) {
	count, err := tx.NewSelect().
		Table(table).
		Where("id = ?", id).
		Count(ctx)
	if err != nil {
		return false, fmt.Errorf("check if %s %s exists: %w", table, id, err)
	}

	return count > 0, nil
}
