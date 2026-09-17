package agentshadow

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Control     repositories.AgentControlRepository
	Runs        repositories.AgentRunRepository
	Definitions repositories.AgentDefinitionRepository
}

type Resolver struct {
	control     repositories.AgentControlRepository
	runs        repositories.AgentRunRepository
	definitions repositories.AgentDefinitionRepository
}

func New(p Params) *Resolver {
	return &Resolver{
		control:     p.Control,
		runs:        p.Runs,
		definitions: p.Definitions,
	}
}

func (r *Resolver) Organization(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error) {
	control, err := r.control.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return false, err
	}

	return control.ShadowMode, nil
}

func (r *Resolver) ForRun(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	runID pulid.ID,
) (bool, error) {
	orgShadow, err := r.Organization(ctx, tenantInfo)
	if err != nil {
		return false, err
	}
	if orgShadow {
		return true, nil
	}

	run, err := r.runs.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         runID,
		TenantInfo: &tenantInfo,
	})
	if err != nil {
		return false, err
	}
	if run.AgentDefinitionID.IsNil() {
		return false, nil
	}

	definition, err := r.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         run.AgentDefinitionID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return false, nil
		}

		return false, err
	}

	return definition.EffectiveShadow(orgShadow), nil
}
