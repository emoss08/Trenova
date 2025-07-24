/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package common

import (
	"context"
	"fmt"

	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

// BulkInsert performs efficient bulk insertion
func BulkInsert[T any](ctx context.Context, tx bun.Tx, items []T, batchSize int) error {
	if len(items) == 0 {
		return nil
	}

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		_, err := tx.NewInsert().
			Model(&batch).
			Exec(ctx)
		if err != nil {
			return eris.Wrapf(err, "bulk insert batch %d-%d", i, end)
		}
	}

	return nil
}

// BulkInsertWithReturning performs bulk insertion and returns the created entities
func BulkInsertWithReturning[T any](
	ctx context.Context,
	tx bun.Tx,
	items []T,
	batchSize int,
) error {
	if len(items) == 0 {
		return nil
	}

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		_, err := tx.NewInsert().
			Model(&batch).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return eris.Wrapf(err, "bulk insert batch %d-%d", i, end)
		}
	}

	return nil
}

// BulkUpdate performs bulk updates with returning
func BulkUpdate[T any](ctx context.Context, tx bun.Tx, items []T) error {
	if len(items) == 0 {
		return nil
	}

	values := tx.NewValues(&items)

	_, err := tx.NewUpdate().
		With("_data", values).
		Model((*T)(nil)).
		TableExpr("_data").
		Set("updated_at = _data.updated_at").
		Where("id = _data.id").
		Returning("*").
		Exec(ctx)

	return err
}

// BulkOperationResult tracks results of bulk operations
type BulkOperationResult struct {
	Successful int
	Failed     int
	Errors     []error
}

// AddError adds an error to the result and increments failed count
func (r *BulkOperationResult) AddError(err error, batchSize int) {
	r.Failed += batchSize
	r.Errors = append(r.Errors, err)
}

// AddSuccess increments the successful count
func (r *BulkOperationResult) AddSuccess(count int) {
	r.Successful += count
}

// HasErrors returns true if there are any errors
func (r *BulkOperationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// ProcessInBatches processes items in batches with error tracking
func ProcessInBatches[T any](
	ctx context.Context,
	items []T,
	batchSize int,
	processor func(context.Context, []T) error,
) (*BulkOperationResult, error) {
	result := &BulkOperationResult{
		Errors: make([]error, 0),
	}

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		if err := processor(ctx, batch); err != nil {
			result.AddError(fmt.Errorf("batch %d-%d: %w", i, end, err), len(batch))
		} else {
			result.AddSuccess(len(batch))
		}
	}

	return result, nil
}
