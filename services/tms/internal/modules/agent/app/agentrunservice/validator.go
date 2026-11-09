package agentrunservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

type Validator struct {
	validator tenantedvalidation.EntityValidator[*agent.AgentRun]
}

func NewValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*agent.AgentRun]().WithModelName("AgentRun").
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *agent.AgentRun,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *agent.AgentRun,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
