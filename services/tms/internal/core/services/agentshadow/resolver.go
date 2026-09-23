package agentshadow

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

// Cause names the switch that put a run in shadow. There are two, on
// different pages, and a refusal that does not say which one is on sends
// people to the wrong page: every organization starts with the pause on, so
// the agent's own switch is usually the one that is already off.
type Cause string

const (
	// CauseOrganization is the organization-wide pause on the AI Control
	// overview. It wins over every agent's own switch.
	CauseOrganization = Cause("OrganizationPaused")
	// CauseDefinition is the shadow switch on the agent that made the run.
	CauseDefinition = Cause("AgentShadow")
)

// Verdict says whether a run is in shadow and, when it is, why. A zero
// Verdict is live.
type Verdict struct {
	Cause Cause
	// AgentName is set when the cause is the agent's own switch, so the
	// message can name the agent to go and change.
	AgentName string
}

func (v Verdict) Shadow() bool {
	return v.Cause != ""
}

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

func (r *Resolver) Organization(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	control, err := r.control.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return false, err
	}

	return control.ShadowMode, nil
}

// ForDefinition decides for a definition already in hand, so a caller that
// just made a run from it does not read it back.
func (r *Resolver) ForDefinition(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	definition *agentdefinition.Definition,
) (Verdict, error) {
	orgShadow, err := r.Organization(ctx, tenantInfo)
	if err != nil {
		return Verdict{}, err
	}

	return verdictFor(orgShadow, definition), nil
}

func (r *Resolver) ForRun(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	runID pulid.ID,
) (Verdict, error) {
	verdicts, err := r.ForRuns(ctx, tenantInfo, []pulid.ID{runID})
	if err != nil {
		return Verdict{}, err
	}

	return verdicts[runID], nil
}

// ForRuns decides for several runs in one pass: the organization is read
// once and each definition once, however many runs share it. A thread has
// one run per turn, so this is what the assistant asks.
func (r *Resolver) ForRuns(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	runIDs []pulid.ID,
) (map[pulid.ID]Verdict, error) {
	verdicts := make(map[pulid.ID]Verdict, len(runIDs))
	if len(runIDs) == 0 {
		return verdicts, nil
	}

	orgShadow, err := r.Organization(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if orgShadow {
		for _, id := range runIDs {
			verdicts[id] = Verdict{Cause: CauseOrganization}
		}

		return verdicts, nil
	}

	definitions := make(map[pulid.ID]*agentdefinition.Definition)
	for _, id := range runIDs {
		if _, seen := verdicts[id]; seen {
			continue
		}

		run, err := r.runs.GetByID(ctx, repositories.GetAgentRunByIDRequest{
			ID:         id,
			TenantInfo: &tenantInfo,
		})
		if err != nil {
			return nil, err
		}
		if run.AgentDefinitionID.IsNil() {
			verdicts[id] = Verdict{}
			continue
		}

		definition, seen := definitions[run.AgentDefinitionID]
		if !seen {
			definition, err = r.definition(ctx, tenantInfo, run.AgentDefinitionID)
			if err != nil {
				return nil, err
			}
			definitions[run.AgentDefinitionID] = definition
		}

		verdicts[id] = verdictFor(orgShadow, definition)
	}

	return verdicts, nil
}

// definition reads one definition, treating a missing one as live: a run
// whose agent was deleted follows the organization, as it always has.
func (r *Resolver) definition(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*agentdefinition.Definition, error) {
	definition, err := r.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	return definition, nil
}

func verdictFor(orgShadow bool, definition *agentdefinition.Definition) Verdict {
	if orgShadow {
		return Verdict{Cause: CauseOrganization}
	}
	if definition != nil && definition.EffectiveShadow(false) {
		return Verdict{Cause: CauseDefinition, AgentName: definition.Name}
	}

	return Verdict{}
}
