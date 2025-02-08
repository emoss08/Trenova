package assignmentvalidator

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB       db.Connection
	MoveRepo repositories.ShipmentMoveRepository
}

type Validator struct {
	db       db.Connection
	moveRepo repositories.ShipmentMoveRepository
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		db:       p.DB,
		moveRepo: p.MoveRepo,
	}
}

func (v *Validator) Validate(ctx context.Context, a *shipment.Assignment) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// a.Validate(ctx, multiErr)
	v.validateAssignmentCriteria(ctx, a, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) validateAssignmentCriteria(ctx context.Context, a *shipment.Assignment, multiErr *errors.MultiError) {
	move, err := v.moveRepo.GetByID(ctx, repositories.GetMoveByIDOptions{
		MoveID: a.ShipmentMoveID,
		OrgID:  a.OrganizationID,
		BuID:   a.BusinessUnitID,
	})
	if err != nil {
		multiErr.Add(
			"move",
			errors.ErrSystemError,
			fmt.Sprintf("failed to get move: %s", err.Error()),
		)
		return
	}

	if !assignableMoveStatuses[move.Status] {
		multiErr.Add(
			"__all__",
			errors.ErrInvalid,
			fmt.Sprintf("Cannot assign to a move that is in the `%s` status", move.Status),
		)

		return
	}
}
