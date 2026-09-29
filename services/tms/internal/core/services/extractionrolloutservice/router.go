package extractionrolloutservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

func (s *Service) AssignExtraction(
	ctx context.Context,
	req *services.AssignExtractionRolloutRequest,
) (pulid.ID, error) {
	rollout, err := s.Get(ctx, req.TenantInfo)
	if err != nil {
		return pulid.Nil, err
	}
	candidate := rollout.CandidateID()
	if candidate.IsNil() {
		return pulid.Nil, nil
	}

	arm := extractionrollout.ArmControl
	if rollout.Assigns(req.DocumentID) {
		arm = extractionrollout.ArmCandidate
	}

	entity := &extractionrollout.RolloutAssignment{
		OrganizationID:      req.TenantInfo.OrgID,
		BusinessUnitID:      req.TenantInfo.BuID,
		DocumentID:          req.DocumentID,
		ExtractedAt:         req.ExtractedAt,
		Arm:                 arm,
		CandidateProviderID: candidate,
		Outcome:             extractionrollout.OutcomePending,
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return pulid.Nil, multiErr
	}

	assignment, _, err := s.assignments.Create(ctx, entity)
	if err != nil {
		return pulid.Nil, err
	}
	if assignment.CandidateProviderID != candidate {
		return pulid.Nil, nil
	}

	return assignment.PreferredProviderID(), nil
}

func (s *Service) SettleExtraction(
	ctx context.Context,
	req *services.SettleExtractionRolloutRequest,
) error {
	if !req.Outcome.IsSettled() {
		return errortypes.NewValidationError(
			"outcome", errortypes.ErrInvalid, "A rollout extraction settles with a final outcome",
		)
	}

	assignment, err := s.assignments.GetByExtraction(ctx, repositories.GetRolloutAssignmentRequest{
		TenantInfo:  req.TenantInfo,
		DocumentID:  req.DocumentID,
		ExtractedAt: req.ExtractedAt,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	if assignment.Outcome == req.Outcome {
		return nil
	}

	assignment.Settle(
		req.Outcome,
		req.ServedProviderID,
		stringutils.TruncateRunes(req.ServedModel, extractionrollout.MaxModelRunes),
		s.now(),
	)
	if _, err = s.assignments.Save(ctx, assignment); err != nil {
		return err
	}

	if assignment.Arm != extractionrollout.ArmCandidate {
		return nil
	}
	if err = s.enforceGuards(ctx, req.TenantInfo); err != nil {
		s.l.Error("failed to check the extraction rollout guards",
			zap.String("documentId", req.DocumentID.String()),
			zap.Error(err),
		)
	}

	return nil
}
