package development

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type seededMatrix struct {
	matrix     *ratematrix.RateMatrix
	dimensions []*ratematrix.RateMatrixDimension
	cells      []*ratematrix.RateMatrixCell
}

func insertPublishedMatrix(ctx context.Context, tx bun.Tx, seeded *seededMatrix) error {
	matrix := seeded.matrix
	matrix.Status = formulatemplate.StatusActive
	matrix.PublishedVersionNumber = 1
	matrix.HasDraftChanges = false
	matrix.DraftCellCount = int64(len(seeded.cells))
	matrix.DraftEditorIDs = []pulid.ID{}
	matrix.Dimensions = seeded.dimensions
	if matrix.UpdatedAt != 0 {
		publishedAt := matrix.UpdatedAt
		matrix.PublishedAt = &publishedAt
	}

	if _, err := tx.NewInsert().Model(matrix).Exec(ctx); err != nil {
		return fmt.Errorf("insert rate matrix %s: %w", matrix.Code, err)
	}

	published := ratematrix.CopyPublished(matrix, seeded.cells, 1, pulid.Nil)
	if _, err := tx.NewInsert().Model(published.Version).Exec(ctx); err != nil {
		return fmt.Errorf("insert rate matrix version %s: %w", matrix.Code, err)
	}

	dimensions := slices.Concat(seeded.dimensions, published.Dimensions)
	if _, err := tx.NewInsert().Model(&dimensions).Exec(ctx); err != nil {
		return fmt.Errorf("insert rate matrix dimensions %s: %w", matrix.Code, err)
	}

	cells := slices.Concat(seeded.cells, published.Cells)
	if len(cells) == 0 {
		return nil
	}
	if _, err := tx.NewInsert().Model(&cells).Exec(ctx); err != nil {
		return fmt.Errorf("insert rate matrix cells %s: %w", matrix.Code, err)
	}
	return nil
}
