package trailervalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/rotisserie/eris"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB            db.Connection
	EquipTypeRepo repositories.EquipmentTypeRepository
}

type Validator struct {
	db            db.Connection
	equipTypeRepo repositories.EquipmentTypeRepository
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		db:            p.DB,
		equipTypeRepo: p.EquipTypeRepo,
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	t *trailer.Trailer,
) *errors.MultiError {
	multiErr := errors.NewMultiError()

	t.Validate(ctx, multiErr)

	// Validate uniqueness
	if err := v.ValidateUniqueness(ctx, valCtx, t, multiErr); err != nil {
		multiErr.Add("uniqueness", errors.ErrSystemError, err.Error())
	}

	// Validate ID
	v.validateID(t, valCtx, multiErr)

	// Validate Equipment Class
	v.validateEquipmentClass(ctx, t, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) ValidateUniqueness(ctx context.Context, valCtx *validator.ValidationContext, t *trailer.Trailer, multiErr *errors.MultiError) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	vb := queryutils.NewUniquenessValidator(t.GetTableName()).
		WithTenant(t.OrganizationID, t.BusinessUnitID).
		WithModelName("Tractor").
		WithFieldAndTemplate("code", t.Code,
			"Tractor with code ':value' already exists in the organization.",
			map[string]string{
				"value": t.Code,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", t.GetID())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}

func (v *Validator) validateID(t *trailer.Trailer, valCtx *validator.ValidationContext, multiErr *errors.MultiError) {
	if valCtx.IsCreate && t.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}

func (v *Validator) validateEquipmentClass(ctx context.Context, t *trailer.Trailer, multiErr *errors.MultiError) {
	et, err := v.equipTypeRepo.GetByID(ctx, repositories.GetEquipmentTypeByIDOptions{
		ID:    t.EquipmentTypeID,
		OrgID: t.OrganizationID,
		BuID:  t.BusinessUnitID,
	})
	if err != nil {
		multiErr.Add("equipmentTypeId", errors.ErrSystemError, err.Error())
		// * Immediately return to avoid further validation
		return
	}

	// Class cannot be Tractor
	if et.Class == equipmenttype.ClassTractor {
		multiErr.Add("equipmentTypeId", errors.ErrInvalid, "Equipment type cannot have a subclass of `Tractor`")
	}
}
