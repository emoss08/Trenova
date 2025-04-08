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
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB            db.Connection
	EquipTypeRepo repositories.EquipmentTypeRepository
	Logger        *logger.Logger
}

type Validator struct {
	db            db.Connection
	equipTypeRepo repositories.EquipmentTypeRepository
	l             *zerolog.Logger
}

func NewValidator(p ValidatorParams) *Validator {
	log := p.Logger.With().
		Str("validator", "tractor").
		Logger()

	return &Validator{
		db:            p.DB,
		equipTypeRepo: p.EquipTypeRepo,
		l:             &log,
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	t *tractor.Tractor,
) *errors.MultiError {
	multiErr := errors.NewMultiError()

	dba, err := v.db.DB(ctx)
	if err != nil {
		multiErr.Add("database", errors.ErrSystemError, err.Error())
		return multiErr
	}

	// Basic Location validation
	t.Validate(ctx, multiErr)

	// Validate uniqueness
	if err = v.ValidateUniqueness(ctx, dba, valCtx, t, multiErr); err != nil {
		multiErr.Add("uniqueness", errors.ErrSystemError, err.Error())
	}

	// Validate ID
	v.validateID(t, valCtx, multiErr)

	// Validate equipment class
	v.validateEquipmentClass(ctx, t, multiErr)

	// Validate secondary worker
	v.validateSecondaryWorker(t, multiErr)

	// Validate worker assignment
	v.validateWorkerAssignment(ctx, dba, valCtx, t, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) ValidateUniqueness(ctx context.Context, dba bun.IDB, valCtx *validator.ValidationContext, t *tractor.Tractor, multiErr *errors.MultiError) error {
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

func (v *Validator) validateID(t *tractor.Tractor, valCtx *validator.ValidationContext, multiErr *errors.MultiError) {
	if valCtx.IsCreate && t.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
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

func (v *Validator) validateWorkerAssignment(ctx context.Context, dba bun.IDB, valCtx *validator.ValidationContext, t *tractor.Tractor, multiErr *errors.MultiError) {
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

		err := q.Scan(ctx)
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
