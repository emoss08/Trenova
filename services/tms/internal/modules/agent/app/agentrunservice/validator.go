package agentrunservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*agent.AgentRun]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*agent.AgentRun]().WithModelName("AgentRun").
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *agent.AgentRun,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *agent.AgentRun,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
