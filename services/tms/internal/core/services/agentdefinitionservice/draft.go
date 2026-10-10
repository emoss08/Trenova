package agentdefinitionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
)

// Draft builds the agent a save would store, checked as a save would check
// it, without saving it. An ID is an edit of that agent; none is a new one.
func (s *Service) Draft(
	ctx context.Context,
	req *agentdefinitionports.SaveAgentDefinitionRequest,
) (*agentdefinition.Definition, error) {
	if req.ID.IsNil() {
		draft := &agentdefinition.Definition{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
		}
		apply(draft, req)
		if err := s.validate(ctx, draft, nil); err != nil {
			return nil, err
		}
		return draft, nil
	}

	existing, err := s.repo.GetByID(ctx, agentdefinitionports.GetAgentDefinitionByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	draft := *existing
	apply(&draft, req)
	if err = s.validate(ctx, &draft, existing); err != nil {
		return nil, err
	}
	return &draft, nil
}
