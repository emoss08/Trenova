package carrierintelreviewservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelsnapshot"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// SnapshotChange is a carrier's intelligence snapshot before and after a
// review is recorded on it.
type SnapshotChange struct {
	Before *carrierintelsnapshot.CarrierIntelSnapshot
	After  *carrierintelsnapshot.CarrierIntelSnapshot
}

func (s *Service) PlanMarkReviewed(
	ctx context.Context,
	req *MarkReviewedRequest,
) (*SnapshotChange, error) {
	note := strings.TrimSpace(req.Note)
	if note == "" {
		return nil, errortypes.NewValidationError("note", errortypes.ErrRequired,
			"Record what was reviewed")
	}
	if len(note) > 2000 {
		return nil, errortypes.NewValidationError("note", errortypes.ErrInvalid,
			"Note cannot exceed 2000 characters")
	}

	snapshot, err := s.snapshotRepo.GetCurrent(
		ctx,
		req.TenantInfo,
		carrierintelports.CarrierIntelSubjectRef{
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   req.CarrierID.String(),
		},
	)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, errortypes.NewBusinessError("Run carrier intelligence for this carrier first")
	}

	after := *snapshot
	now := s.now()
	after.ReviewedAt = &now
	after.ReviewedByID = req.TenantInfo.UserID
	after.ReviewNote = note

	return &SnapshotChange{Before: snapshot, After: &after}, nil
}
