package workerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker

	DB                  *postgres.Connection
	DispatchControlRepo dispatchcontrolports.DispatchControlRepository
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*worker.Worker]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*worker.Worker]().
			WithModelName("Worker").
			WithIntegrityChecker(p.Integrity).
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
			WithReference(tenantedvalidation.ReferenceConfig[*worker.Worker]{
				FieldName: "positionId",
				Table:     "job_positions",
				Message:   "Position must be an open driving position in your organization",
				Optional:  true,
				Conditions: []validationframework.Condition{
					{Column: "status", Value: string(domaintypes.StatusActive)},
					{Column: "is_driving_position", Value: true},
				},
				GetID: func(w *worker.Worker) pulid.ID { return w.PositionID },
			}).
			WithCustomRule(createWorkerComplianceRule(p.DispatchControlRepo)).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *worker.Worker,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *worker.Worker,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
