package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type seedRowOnce[T any] struct {
	seedName string
	table    string
	row      *T
	id       func() pulid.ID
	where    func(*bun.SelectQuery) *bun.SelectQuery
}

func insertSeedRow(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	seedName, table string,
	id pulid.ID,
	model any,
) error {
	if _, err := tx.NewInsert().Model(model).Exec(ctx); err != nil {
		return fmt.Errorf("insert %s: %w", table, err)
	}
	if err := sc.TrackCreated(ctx, table, id, seedName); err != nil {
		return fmt.Errorf("track %s: %w", table, err)
	}

	return nil
}

func insertSeedRowOnce[T any](
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	req *seedRowOnce[T],
) (bool, error) {
	exists, err := tx.NewSelect().Model((*T)(nil)).Apply(req.where).Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("find %s: %w", req.table, err)
	}
	if exists {
		return false, nil
	}
	if _, err = tx.NewInsert().Model(req.row).Exec(ctx); err != nil {
		return false, fmt.Errorf("insert %s: %w", req.table, err)
	}
	if err = sc.TrackCreated(ctx, req.table, req.id(), req.seedName); err != nil {
		return false, fmt.Errorf("track %s: %w", req.table, err)
	}

	return true, nil
}
