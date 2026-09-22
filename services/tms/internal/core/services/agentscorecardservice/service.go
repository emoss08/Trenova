// Package agentscorecardservice answers "is this agent worth keeping".
//
// An organization that lets an agent write to its records is owed a straight
// answer about how that has been going, and until this existed the answer was
// spread across three pages: the runs list said how often it ran, the
// decisions queue said what people did with its proposals, and nothing said
// what any of it cost or how much work it took off anybody.
//
// Every figure is counted at read time from the runs, the proposals and the
// usage records. There is no stored copy, so there is nothing to drift out of
// step with the ledger a person can audit.
package agentscorecardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const secondsPerDay = 86_400

type Params struct {
	fx.In

	Logger      *zap.Logger
	Scorecards  repositories.AgentScorecardRepository
	Trust       repositories.AgentToolTrustRepository
	Definitions repositories.AgentDefinitionRepository
	Permissions services.PermissionEngine
}

type Service struct {
	l           *zap.Logger
	scorecards  repositories.AgentScorecardRepository
	trust       repositories.AgentToolTrustRepository
	definitions repositories.AgentDefinitionRepository
	permissions services.PermissionEngine
	now         func() int64
}

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.agentscorecard"),
		scorecards:  p.Scorecards,
		trust:       p.Trust,
		definitions: p.Definitions,
		permissions: p.Permissions,
		now:         timeutils.NowUnix,
	}
}

func AsService(s *Service) services.AgentScorecardService { return s }

// Get counts one agent's record over the window.
//
// Reading a scorecard needs what reading the agent needs and nothing more:
// it is a summary of rows the same person could already list one by one.
func (s *Service) Get(
	ctx context.Context,
	req services.AgentScorecardRequest,
	actor *services.RequestActor,
) (*services.AgentScorecardResult, error) {
	window := req.Window
	if !window.IsValid() {
		window = agent.ScorecardWindow30d
	}

	result, err := s.permissions.Check(ctx, &services.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       permission.ResourceAgentDefinition.String(),
		Operation:      permission.OpRead,
	})
	if err != nil {
		return nil, err
	}
	if result == nil || !result.Allowed {
		return nil, errortypes.NewAuthorizationError("You cannot read this agent")
	}

	// Read through the definition first, so an id from another organization
	// is not found rather than silently counted as an agent with no history.
	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	since := s.now() - int64(window.Days())*secondsPerDay
	totals, err := s.scorecards.Aggregate(ctx, repositories.ScorecardRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: definition.ID,
		Since:             since,
	})
	if err != nil {
		return nil, err
	}

	ladder, err := s.trust.ListByDefinition(ctx, repositories.ListToolTrustRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: definition.ID,
	})
	if err != nil {
		return nil, err
	}

	return &services.AgentScorecardResult{
		Scorecard: agent.NewScorecard(definition.ID.String(), window, since, totals),
		ToolTrust: ladder,
	}, nil
}
