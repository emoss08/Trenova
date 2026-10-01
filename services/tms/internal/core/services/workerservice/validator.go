package workerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB                  *postgres.Connection
	DispatchControlRepo repositories.DispatchControlRepository
}

type Validator struct {
	validator *validationframework.TenantedValidator[*worker.Worker]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*worker.Worker]().
			WithModelName("Worker").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerScoped(p.DB)).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerScoped(p.DB)).
			WithCustomReferenceCheck(
				"stateId",
				"State does not exist",
				func(w *worker.Worker) pulid.ID { return w.StateID },
				validationframework.NewUSStateReferenceCheck(p.DB),
			).
			WithOptionalCustomReferenceCheck(
				"fleetCodeId",
				"Fleet code does not exist in your organization",
				func(w *worker.Worker) pulid.ID { return w.FleetCodeID },
				createFleetCodeCheck(p.DB),
			).
			// A worker holds a driving title. Front-office titles are held by
			// users through their membership, so offering one here would put a
			// driver on the wrong roster.
			WithOptionalCustomReferenceCheck(
				"positionId",
				"Position must be an open driving position in your organization",
				func(w *worker.Worker) pulid.ID { return w.PositionID },
				createDrivingPositionCheck(p.DB),
			).
			WithCustomRule(createWorkerComplianceRule(p.DispatchControlRepo)).
			Build(),
	}
}

func createDrivingPositionCheck(
	db *postgres.Connection,
) validationframework.CustomReferenceCheckFunc {
	return func(ctx context.Context, orgID, buID pulid.ID, refID pulid.ID) (bool, error) {
		return dbtx.Read(ctx, db, func(ctx context.Context) (bool, error) {
			if refID.IsNil() {
				return true, nil
			}

			exists, err := db.DBForContext(ctx).NewSelect().
				TableExpr("job_positions").
				ColumnExpr("1").
				Where("id = ?", refID).
				Where("organization_id = ?", orgID).
				Where("business_unit_id = ?", buID).
				Where("status = 'Active'").
				Where("is_driving_position").
				Exists(ctx)
			if err != nil {
				return false, err
			}
			return exists, nil
		})
	}
}

func createFleetCodeCheck(
	db *postgres.Connection,
) validationframework.CustomReferenceCheckFunc {
	return func(ctx context.Context, orgID, buID pulid.ID, refID pulid.ID) (bool, error) {
		return dbtx.Read(ctx, db, func(ctx context.Context) (bool, error) {
			if refID.IsNil() {
				return true, nil
			}

			exists, err := db.DBForContext(ctx).NewSelect().
				TableExpr("fleet_codes").
				ColumnExpr("1").
				Where("id = ?", refID).
				Where("organization_id = ?", orgID).
				Where("business_unit_id = ?", buID).
				Exists(ctx)
			if err != nil {
				return false, err
			}
			return exists, nil
		})
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *worker.Worker,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *worker.Worker,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
