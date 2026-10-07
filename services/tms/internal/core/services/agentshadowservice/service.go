package agentshadowservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentshadow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Definitions repositories.AgentDefinitionRepository
	Shadow      repositories.AgentShadowRepository
}

type Service struct {
	definitions repositories.AgentDefinitionRepository
	shadow      repositories.AgentShadowRepository
	now         func() int64
}

var _ services.AgentShadowService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{definitions: p.Definitions, shadow: p.Shadow, now: timeutils.NowUnix}
}

// Report sets the writes an agent recorded in shadow over the last days
// beside what people did to the same records.
func (s *Service) Report(ctx context.Context, req *services.AgentShadowReportRequest) (*agentshadow.Report, error) {
	if _, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.AgentID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	days := agentshadow.ClampDays(req.Days)
	now := s.now()
	since := now - int64(days)*24*60*60

	proposals, err := s.shadow.ListShadowProposals(ctx, &repositories.ListShadowProposalsRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentID,
		Since:             since,
		Limit:             agentshadow.MaxProposals,
	})
	if err != nil {
		return nil, err
	}

	changes, err := s.shadow.ListPersonChanges(ctx, &repositories.ListPersonChangesRequest{
		TenantInfo:  req.TenantInfo,
		ResourceIDs: targets(proposals),
		Since:       since,
		Until:       now,
	})
	if err != nil {
		return nil, err
	}

	return agentshadow.Build(days, proposals, changes), nil
}

func targets(proposals []agentshadow.Proposal) []string {
	seen := make(map[string]struct{}, len(proposals))
	ids := make([]string, 0, len(proposals))
	for idx := range proposals {
		id := proposals[idx].TargetID
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}
