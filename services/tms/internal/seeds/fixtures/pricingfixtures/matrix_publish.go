package pricingfixtures

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix/ratematrixdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixcell"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixcell/ratematrixcelldb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixversion"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixversion/ratematrixversiondb"
	"github.com/emoss08/trenova/shared/pulid"
)

type SeededMatrix struct {
	Matrix     *ratematrix.RateMatrix
	Dimensions []*ratematrix.RateMatrixDimension
	Cells      []*ratematrixcell.RateMatrixCell
}

func InsertPublishedMatrix(ctx context.Context, db *keel.DB, seeded *SeededMatrix) error {
	matrix := seeded.Matrix
	matrix.Status = formulatemplate.StatusActive
	matrix.PublishedVersionNumber = 1
	matrix.HasDraftChanges = false
	matrix.DraftCellCount = int64(len(seeded.Cells))
	matrix.DraftEditorIDs = []pulid.ID{}
	matrix.Dimensions = seeded.Dimensions
	if matrix.UpdatedAt != 0 {
		publishedAt := matrix.UpdatedAt
		matrix.PublishedAt = &publishedAt
	}

	if err := ratematrixdb.RateMatrixes.Create(ctx, db, matrix); err != nil {
		return fmt.Errorf("insert rate matrix %s: %w", matrix.Code, err)
	}

	published := ratematrixversion.CopyPublished(matrix, seeded.Cells, 1, pulid.Nil)
	if err := ratematrixversiondb.RateMatrixVersions.Create(ctx, db, published.Version); err != nil {
		return fmt.Errorf("insert rate matrix version %s: %w", matrix.Code, err)
	}

	dimensions := slices.Concat(seeded.Dimensions, published.Dimensions)
	if err := ratematrixdb.RateMatrixDimensions.CreateMany(ctx, db, dimensions); err != nil {
		return fmt.Errorf("insert rate matrix dimensions %s: %w", matrix.Code, err)
	}

	cells := slices.Concat(seeded.Cells, published.Cells)
	if len(cells) == 0 {
		return nil
	}
	if err := ratematrixcelldb.RateMatrixCells.CreateMany(ctx, db, cells); err != nil {
		return fmt.Errorf("insert rate matrix cells %s: %w", matrix.Code, err)
	}
	return nil
}
