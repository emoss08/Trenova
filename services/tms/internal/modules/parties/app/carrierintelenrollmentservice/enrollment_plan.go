package carrierintelenrollmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelenrollment"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// EnrollmentChange is one carrier's monitoring as it stands and as the call
// would leave it.
type EnrollmentChange struct {
	CarrierID pulid.ID
	Name      string
	DOTNumber string
	Before    carrierintelenrollment.DesiredState
	After     carrierintelenrollment.DesiredState
}

func (s *Service) PlanManualEnrollment(
	ctx context.Context,
	req *EnrollSubjectsRequest,
) ([]EnrollmentChange, error) {
	if len(req.CarrierIDs) == 0 {
		return nil, errortypes.NewValidationError("carrierIds", errortypes.ErrRequired,
			"Select at least one carrier")
	}
	control, err := s.controlRepo.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return nil, errortypes.NewBusinessError(
			"Enable a carrier intelligence provider before monitoring carriers",
		)
	}
	connector, ok := s.intel.Connector(provider)
	if !ok || !connector.Descriptor().Capabilities.SupportsMonitoring() {
		return nil, errortypes.NewBusinessError(
			"{0} does not support carrier monitoring", provider.String(),
		)
	}

	subjects, err := s.subjectRepo.ListCarrierSubjects(
		ctx,
		&carrierintelports.ListCarrierIntelSubjectsRequest{
			TenantInfo: req.TenantInfo,
			CarrierIDs: req.CarrierIDs,
			Limit:      len(req.CarrierIDs),
		},
	)
	if err != nil {
		return nil, err
	}
	if len(subjects) == 0 {
		return nil, errortypes.NewBusinessError(
			"None of the selected carriers have a DOT number to monitor",
		)
	}

	ids := make([]string, 0, len(subjects))
	for _, subject := range subjects {
		ids = append(ids, subject.SubjectID)
	}
	rows, err := s.enrollmentRepo.ListBySubjects(
		ctx,
		&carrierintelports.ListEnrollmentsBySubjectRequest{
			TenantInfo:  req.TenantInfo,
			Provider:    provider,
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectIDs:  ids,
		},
	)
	if err != nil {
		return nil, err
	}
	existing := make(map[string]carrierintelenrollment.DesiredState, len(rows))
	for _, row := range rows {
		existing[row.SubjectID] = row.DesiredState
	}

	after := carrierintelenrollment.DesiredStateNotEnrolled
	if req.Enroll {
		after = carrierintelenrollment.DesiredStateEnrolled
	}
	changes := make([]EnrollmentChange, 0, len(subjects))
	for _, subject := range subjects {
		before, found := existing[subject.SubjectID]
		if !found {
			before = carrierintelenrollment.DesiredStateNotEnrolled
		}
		changes = append(changes, EnrollmentChange{
			CarrierID: subject.CarrierID,
			Name:      subject.Name,
			DOTNumber: subject.DOTNumber,
			Before:    before,
			After:     after,
		})
	}

	return changes, nil
}
