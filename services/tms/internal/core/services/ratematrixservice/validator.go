package ratematrixservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// maxCellsPerReplace bounds one upload. A four-axis class tariff is large but
// finite; anything past this is a mistake in the sheet rather than a tariff,
// and letting it through would tie up a connection long enough to matter.
const maxCellsPerReplace = 250_000

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*ratematrix.RateMatrix]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{validator: newBuilder(p.DB).Build()}
}

func newBuilder(
	db *postgres.Connection,
) *validationframework.TenantedValidatorBuilder[*ratematrix.RateMatrix] {
	builder := validationframework.
		NewTenantedValidatorBuilder[*ratematrix.RateMatrix]().
		WithModelName("Rate Matrix")

	if db == nil {
		return builder
	}

	return builder.
		WithUniquenessChecker(
			validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return db.DB() }),
		).
		WithUniqueField(
			"code",
			"code",
			"A rate matrix with this code already exists in your organization",
			func(m *ratematrix.RateMatrix) any { return m.Code },
		)
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}

// ValidateCells checks a replacement grid against the axes it is meant to fill.
//
// Each cell is checked against the matrix's own dimensions rather than in
// isolation, because a cell is only well formed relative to them: an exact axis
// needs a key, a banded axis needs a lower bound, and a cell missing either is
// one the lookup can never reach.
func (v *Validator) ValidateCells(
	matrix *ratematrix.RateMatrix,
	req *repositories.ReplaceRateMatrixCellsRequest,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if len(req.Cells) > maxCellsPerReplace {
		multiErr.Add(
			"cells",
			errortypes.ErrInvalid,
			"A rate matrix cannot be loaded with more than 250,000 cells at once",
		)

		return multiErr
	}

	dimensions := matrix.OrderedDimensions()

	for index, cell := range req.Cells {
		if cell == nil {
			continue
		}

		cell.RateMatrixID = matrix.ID
		cell.OrganizationID = req.TenantInfo.OrgID
		cell.BusinessUnitID = req.TenantInfo.BuID

		cell.ValidateAgainst(dimensions, multiErr.WithIndex("cells", index))
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
