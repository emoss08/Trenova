package tractorvalidator

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// ValidatorParams defines the dependencies required for initializing the Validator.
// This includes the database connection and validation engine factory, equipment type repository, and logger.
type ValidatorParams struct {
	fx.In

	DB                      db.Connection
	EquipTypeRepo           repositories.EquipmentTypeRepository
	ValidationEngineFactory framework.ValidationEngineFactory
	Logger                  *logger.Logger
}

// Validator is a struct that contains the database connection and the validator.
// It provides methods to validate tractors and other related entities.
type Validator struct {
	db            db.Connection
	equipTypeRepo repositories.EquipmentTypeRepository
	vef           framework.ValidationEngineFactory
	l             *zerolog.Logger
}

// NewValidator initializes a new Validator with the provided dependencies.
//
// Parameters:
//   - p: ValidatorParams containing dependencies.
//
// Returns:
//   - *Validator: A new Validator instance.
func NewValidator(p ValidatorParams) *Validator {
	log := p.Logger.With().
		Str("validator", "tractor").
		Logger()

	return &Validator{
		db:            p.DB,
		equipTypeRepo: p.EquipTypeRepo,
		vef:           p.ValidationEngineFactory,
		l:             &log,
	}
}

func (v *Validator) Validate(ctx context.Context, valCtx *validator.ValidationContext, t *tractor.Tractor,
) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// * Basic validation rules (field presence, format, etc.)
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageBasic, framework.ValidationPriorityHigh,
		func(ctx context.Context, multiErr *errors.MultiError) error {
			t.Validate(ctx, multiErr)
			return nil
		}))

	// * Data integrity validation (uniqueness, references, etc.)
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageDataIntegrity, framework.ValidationPriorityHigh,
		func(ctx context.Context, multiErr *errors.MultiError) error {
			return v.ValidateUniqueness(ctx, valCtx, t, multiErr)
		}))

	// * Business rules validation (domain-specific rules)
	engine.AddRule(framework.NewValidationRule(framework.ValidationStageBusinessRules, framework.ValidationPriorityHigh,
		func(ctx context.Context, multiErr *errors.MultiError) error {
			// Validate equipment class
			v.validateEquipmentClass(ctx, t, multiErr)

			// Validate secondary worker
			v.validateSecondaryWorker(t, multiErr)

			// Validate worker assignment
			v.validateWorkerAssignment(ctx, valCtx, t, multiErr)

			return nil
		}))

	return engine.Validate(ctx)
}

func (v *Validator) ValidateUniqueness(ctx context.Context, valCtx *validator.ValidationContext, t *tractor.Tractor, multiErr *errors.MultiError) error {
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

func (v *Validator) validateEquipmentClass(ctx context.Context, t *tractor.Tractor, multiErr *errors.MultiError) {
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

	if et.Class != equipmenttype.ClassTractor {
		multiErr.Add("equipmentTypeId", errors.ErrInvalid, "Equipment type must have subclass 'Tractor'")
	}
}

func (v *Validator) validateSecondaryWorker(t *tractor.Tractor, multiErr *errors.MultiError) {
	// Validate the secondary worker is not the same as the primary worker
	if t.SecondaryWorkerID != nil && pulid.Equals(*t.SecondaryWorkerID, t.PrimaryWorkerID) {
		multiErr.Add("secondaryWorkerId", errors.ErrInvalid, "Secondary worker cannot be the same as the primary worker")
	}
}

func (v *Validator) validateWorkerAssignment(ctx context.Context, valCtx *validator.ValidationContext, t *tractor.Tractor, multiErr *errors.MultiError) {
	dba, err := v.db.DB(ctx)
	if err != nil {
		multiErr.Add("database", errors.ErrSystemError, err.Error())
		return
	}

	v.l.Debug().
		Str("tractorID", t.ID.String()).
		Msg("Validating worker assignment")

	checkWorker := func(workerID pulid.ID, fieldName string) {
		if workerID.IsNil() {
			return
		}

		existingTractor := new(tractor.Tractor)
		q := dba.NewSelect().Model(existingTractor)

		if valCtx.IsCreate {
			q.Where("(tr.primary_worker_id = ? OR tr.secondary_worker_id = ?)", workerID, workerID)
		} else {
			// * Exclude the current tractor from the query just incase that worker is already assigned to it
			q.Where("(tr.primary_worker_id = ? OR tr.secondary_worker_id = ?) AND id != ?", workerID, workerID, t.ID)
		}

		err = q.Scan(ctx)
		if err != nil {
			if eris.Is(err, sql.ErrNoRows) {
				v.l.Debug().
					Str("workerID", workerID.String()).
					Str("tractorID", t.ID.String()).
					Msg("Worker is not assigned to tractor")
				return
			}

			v.l.Error().
				Str("workerID", workerID.String()).
				Str("tractorID", t.ID.String()).
				Err(err).
				Msg("Error checking worker assignment")
			return
		}

		multiErr.Add(fieldName, errors.ErrInvalid, fmt.Sprintf("Worker is already assigned to tractor '%s' as %s", existingTractor.Code, getWorkerRole(t, workerID)))
	}

	checkWorker(t.PrimaryWorkerID, "primaryWorkerId")

	if t.SecondaryWorkerID != nil {
		checkWorker(*t.SecondaryWorkerID, "secondaryWorkerId")
	}
}

func getWorkerRole(t *tractor.Tractor, workerID pulid.ID) string {
	if t.PrimaryWorkerID == workerID {
		return "primary"
	}

	return "secondary"
}
