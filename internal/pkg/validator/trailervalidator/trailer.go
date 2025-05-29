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
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"github.com/rotisserie/eris"
	"go.uber.org/fx"
)

// ValidatorParams defines the dependencies required for initializing the Validator.
// This includes the database connection and validation engine factory, equipment type repository, and logger.
type ValidatorParams struct {
	fx.In

	DB                      db.Connection
	EquipTypeRepo           repositories.EquipmentTypeRepository
	ValidationEngineFactory framework.ValidationEngineFactory
}

// Validator is a struct that contains the database connection and the validator.
// It provides methods to validate trailers and other related entities.
type Validator struct {
	db            db.Connection
	equipTypeRepo repositories.EquipmentTypeRepository
	vef           framework.ValidationEngineFactory
}

// NewValidator initializes a new Validator with the provided dependencies.
//
// Parameters:
//   - p: ValidatorParams containing dependencies.
//
// Returns:
//   - *Validator: A new Validator instance.
func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		db:            p.DB,
		equipTypeRepo: p.EquipTypeRepo,
		vef:           p.ValidationEngineFactory,
	}
}

// Validate validates a trailer.
//
// Parameters
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - t: The trailer to validate.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	t *trailer.Trailer,
) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// * Basic validation rules (field presence, format, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				t.Validate(ctx, multiErr)
				return nil
			},
		),
	)

	// * Data integrity validation (uniqueness, references, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageDataIntegrity,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				return v.ValidateUniqueness(ctx, valCtx, t, multiErr)
			},
		),
	)

	// * Business rules validation (domain-specific rules)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBusinessRules,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				v.validateID(t, valCtx, multiErr)
				v.validateEquipmentClass(ctx, t, multiErr)

				return nil
			},
		),
	)

	return engine.Validate(ctx)
}

// ValidateUniqueness validates the uniqueness of a trailer.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - t: The trailer to validate.
//   - multiErr: The MultiError to add validation errors to.
//
// Returns:
//   - error: An error if the validation fails.
func (v *Validator) ValidateUniqueness(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	t *trailer.Trailer,
	multiErr *errors.MultiError,
) error {
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

// validateID validates the ID of a trailer.
//
// Parameters:
//   - t: The trailer to validate.
//   - valCtx: The validation context.
//   - multiErr: The MultiError to add validation errors to.
//
// Returns:
//   - error: An error if the validation fails.
func (v *Validator) validateID(
	t *trailer.Trailer,
	valCtx *validator.ValidationContext,
	multiErr *errors.MultiError,
) {
	if valCtx.IsCreate && t.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}

// validateEquipmentClass validates the equipment class of a trailer.
//
// Parameters:
//   - ctx: The context of the request.
//   - t: The trailer to validate.
//   - multiErr: The MultiError to add validation errors to.
//
// Returns:
//   - error: An error if the validation fails.
func (v *Validator) validateEquipmentClass(
	ctx context.Context,
	t *trailer.Trailer,
	multiErr *errors.MultiError,
) {
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
		multiErr.Add(
			"equipmentTypeId",
			errors.ErrInvalid,
			"Equipment type cannot have a subclass of `Tractor`",
		)
	}
}
