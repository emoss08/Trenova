package assistantservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var errWaitsUnavailable = errors.New("waits cannot be read here")

// waitNote is the input of the turn that picks up work the agent parked on a
// wait of this conversation: what it waited for, what came of it and what it
// said it would do then.
func (s *Service) waitNote(
	ctx context.Context,
	threadID, waitID pulid.ID,
	tenant pagination.TenantInfo,
) (string, error) {
	if s.waits == nil {
		return "", errWaitsUnavailable
	}
	wait, err := s.waits.Get(ctx, &repositories.GetAgentWaitRequest{
		ID:         waitID,
		TenantInfo: pagination.TenantInfo{OrgID: tenant.OrgID, BuID: tenant.BuID},
		ThreadID:   threadID,
	})
	if err != nil {
		return "", err
	}
	if wait.Status != agentwait.StatusMet && wait.Status != agentwait.StatusTimedOut {
		return "", errortypes.NewBusinessError(
			"This wait has not ended, so there is nothing to pick up",
		)
	}

	return wait.ResumeNote(), nil
}
