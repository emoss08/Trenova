package agentdefinitionservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// validateDelegates checks that every agent on the allowlist is one this
// agent may ask: an agent in the same organization and business unit that a
// person can talk to. One this save adds must also be enabled. One already on
// the list that has been disabled since stays, and is refused when asked, so
// disabling an agent does not leave every agent that delegates to it unable
// to be saved.
//
// previous is the definition as stored, nil for a new one. An error is a
// failure to read the delegates; what is wrong with them goes on multiErr.
func (s *Service) validateDelegates(
	ctx context.Context,
	definition, previous *agentdefinition.Definition,
	multiErr *errortypes.MultiError,
) error {
	if len(definition.DelegateIDs) == 0 {
		return nil
	}

	found, err := s.repo.ListByIDs(ctx, repositories.ListAgentDefinitionsByIDsRequest{
		IDs: definition.DelegateIDs,
		TenantInfo: pagination.TenantInfo{
			OrgID: definition.OrganizationID,
			BuID:  definition.BusinessUnitID,
		},
	})
	if err != nil {
		return fmt.Errorf("read the agents this agent may hand work to: %w", err)
	}

	byID := make(map[pulid.ID]*agentdefinition.Definition, len(found))
	for _, delegate := range found {
		byID[delegate.ID] = delegate
	}

	for idx, id := range definition.DelegateIDs {
		// An empty id and the agent itself are the domain's to report.
		if id.IsNil() || id == definition.ID {
			continue
		}

		field := fmt.Sprintf("delegateIds[%d]", idx)
		delegate, ok := byID[id]
		switch {
		case !ok:
			multiErr.Add(field, errortypes.ErrInvalid,
				"This agent does not exist in this organization")
		case delegate.IsBackground():
			multiErr.Add(field, errortypes.ErrInvalid, fmt.Sprintf(
				"%s runs on its own and cannot be handed work", delegate.Name,
			))
		case delegate.IsPageAgent():
			multiErr.Add(field, errortypes.ErrInvalid, fmt.Sprintf(
				"%s works on its own page and cannot be handed work", delegate.Name,
			))
		case !delegate.Enabled && (previous == nil || !previous.MayDelegateTo(id)):
			multiErr.Add(field, errortypes.ErrInvalid, fmt.Sprintf(
				"%s is disabled. Enable it before letting another agent hand it work",
				delegate.Name,
			))
		}
	}

	return nil
}
