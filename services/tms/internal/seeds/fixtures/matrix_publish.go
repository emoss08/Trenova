package fixtures

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix/ratematrixdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixcell"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixversion"
	"github.com/emoss08/trenova/shared/pulid"
)

type seededMatrix struct {
	matrix     *ratematrix.RateMatrix
	dimensions []*ratematrix.RateMatrixDimension
	cells      []*ratematrixcell.RateMatrixCell
}

func insertPublishedMatrix(ctx context.Context, db *keel.DB, seeded *seededMatrix) error {
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

	if err := ratematrixdb.RateMatrixes.Create(ctx, db, matrix); err != nil {
		return fmt.Errorf("insert rate matrix %s: %w", matrix.Code, err)
	}

	published := ratematrixversion.CopyPublished(matrix, seeded.cells, 1, pulid.Nil)
	if err := ratematrixdb.RateMatrixVersions.Create(ctx, db, published.Version); err != nil {
		return fmt.Errorf("insert rate matrix version %s: %w", matrix.Code, err)
	}

	dimensions := slices.Concat(seeded.dimensions, published.Dimensions)
	if err := ratematrixdb.RateMatrixDimensions.CreateMany(ctx, db, dimensions); err != nil {
		return fmt.Errorf("insert rate matrix dimensions %s: %w", matrix.Code, err)
	}

	cells := slices.Concat(seeded.cells, published.Cells)
	if len(cells) == 0 {
		return nil
	}
	if err := ratematrixdb.RateMatrixCells.CreateMany(ctx, db, cells); err != nil {
		return fmt.Errorf("insert rate matrix cells %s: %w", matrix.Code, err)
	}
	return nil
}
