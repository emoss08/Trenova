package assistantservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var errWaitsUnavailable = errors.New("waits cannot be read here")

// waitNote is the input of the turn that picks up work the agent parked on a
// wait of this conversation: what it waited for, what came of it and what it
// said it would do then.
// resumedWaitNote is what a turn that picks a wait up reads, and whether that
// note carries text written outside the organization: a portal's answer is,
// so every write the turn asks for is proposed.
type resumedWaitNote struct {
	note    string
	outside bool
}

func (s *Service) waitNote(
	ctx context.Context,
	threadID, waitID pulid.ID,
	tenant pagination.TenantInfo,
) (resumedWaitNote, error) {
	if s.waits == nil {
		return resumedWaitNote{}, errWaitsUnavailable
	}
	wait, err := s.waits.Get(ctx, &agentwaitports.GetAgentWaitRequest{
		ID:         waitID,
		TenantInfo: pagination.TenantInfo{OrgID: tenant.OrgID, BuID: tenant.BuID},
		ThreadID:   threadID,
	})
	if err != nil {
		return resumedWaitNote{}, err
	}
	if wait.Status != agentwait.StatusMet && wait.Status != agentwait.StatusTimedOut {
		return resumedWaitNote{}, errortypes.NewBusinessError(
			"This wait has not ended, so there is nothing to pick up",
		)
	}

	return resumedWaitNote{note: wait.ResumeNote(), outside: wait.CarriesOutsideText()}, nil
}
