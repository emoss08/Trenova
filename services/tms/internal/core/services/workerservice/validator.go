package workerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithCustomReferenceCheck(
				"stateId",
				"State does not exist",
				func(w *worker.Worker) pulid.ID { return w.StateID },
				createStateCheck(p.DB),
			).
			WithOptionalCustomReferenceCheck(
				"fleetCodeId",
				"Fleet code does not exist in your organization",
				func(w *worker.Worker) pulid.ID { return w.FleetCodeID },
				createFleetCodeCheck(p.DB),
			).
			WithCustomRule(createAgeComplianceRule(p.DispatchControlRepo)).
			WithCustomRule(createCDLComplianceRule(p.DispatchControlRepo)).
			WithCustomRule(createMedicalCertComplianceRule(p.DispatchControlRepo)).
			WithCustomRule(createMVRComplianceRule(p.DispatchControlRepo)).
			WithCustomRule(createDrugTestComplianceRule(p.DispatchControlRepo)).
			WithCustomRule(createHazmatComplianceRule(p.DispatchControlRepo)).
			Build(),
	}
}

func createStateCheck(
	db *postgres.Connection,
) validationframework.CustomReferenceCheckFunc {
	return func(ctx context.Context, _, _ pulid.ID, refID pulid.ID) (bool, error) {
		if refID.IsNil() {
			return true, nil
		}

		exists, err := db.DB().NewSelect().
			TableExpr("us_states").
			ColumnExpr("1").
			Where("id = ?", refID).
			Exists(ctx)
		if err != nil {
			return false, err
		}
		return exists, nil
	}
}

func createFleetCodeCheck(
	db *postgres.Connection,
) validationframework.CustomReferenceCheckFunc {
	return func(ctx context.Context, orgID, buID pulid.ID, refID pulid.ID) (bool, error) {
		if refID.IsNil() {
			return true, nil
		}

		exists, err := db.DB().NewSelect().
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
