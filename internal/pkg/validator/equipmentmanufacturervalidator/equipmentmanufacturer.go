package equipmentmanufacturervalidator

import (
	"context"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	"github.com/trenova-app/transport/internal/core/domain/equipmentmanufacturer"
	"github.com/trenova-app/transport/internal/core/ports/db"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/utils/queryutils"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB db.Connection
}

type Validator struct {
	db db.Connection
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		db: p.DB,
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	em *equipmentmanufacturer.EquipmentManufacturer,
) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// Basic Equipment Manufacturer validation
	em.Validate(ctx, multiErr)

	// Validate uniqueness
	if err := v.ValidateUniqueness(ctx, valCtx, em, multiErr); err != nil {
		multiErr.Add("uniqueness", errors.ErrSystemError, err.Error())
	}

	// Validate ID
	v.validateID(em, valCtx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) ValidateUniqueness(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	em *equipmentmanufacturer.EquipmentManufacturer,
	multiErr *errors.MultiError,
) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	vb := queryutils.NewUniquenessValidator(em.GetTableName()).
		WithTenant(em.OrganizationID, em.BusinessUnitID).
		WithModelName("EquipmentManufacturer").
		WithFieldAndTemplate("name", em.Name,
			"Equipment Manufacturer with name ':value' already exists in the organization.",
			map[string]string{
				"value": em.Name,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		log.Debug().Msg("We're hitting this currently.")
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", em.ID.String())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}

func (v *Validator) validateID(em *equipmentmanufacturer.EquipmentManufacturer, valCtx *validator.ValidationContext, multiErr *errors.MultiError) {
	if valCtx.IsCreate && em.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}
