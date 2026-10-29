package workerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB                  *postgres.Connection
	DispatchControlRepo dispatchcontrolports.DispatchControlRepository
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*worker.Worker]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := keelcheck.NewChecker(p.DB, workerdb.WorkersSchema())

	return &Validator{
		validator: tenantedvalidation.NewBuilder[*worker.Worker]().
			WithModelName("Worker").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
			WithReferenceCheck(
				"stateId",
				"us_states",
				"State does not exist",
				func(w *worker.Worker) pulid.ID { return w.StateID },
			).
			WithOptionalReferenceCheck(
				"fleetCodeId",
				"fleet_codes",
				"Fleet code does not exist in your organization",
				func(w *worker.Worker) pulid.ID { return w.FleetCodeID },
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
		if refID.IsNil() {
			return true, nil
		}

		return dbtx.Read(ctx, db, func(ctx context.Context) (bool, error) {
			jp := workerdb.JobPositions
			q := jp.Where(
				jp.ID.Eq(refID),
				jp.Status.Eq(domaintypes.StatusActive),
				jp.IsDrivingPosition.IsTrue(),
			)
			if dbscope.IsSystem(ctx) {
				q = q.AllTenants().Where(jp.OrganizationID.Eq(orgID), jp.BusinessUnitID.Eq(buID))
			}

			return q.Exists(ctx, db.Keel(ctx))
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
