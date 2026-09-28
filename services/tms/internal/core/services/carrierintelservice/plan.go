package carrierintelservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// VetPlan is what vetting would ask the provider for: the subject and the
// provider that answers, with nothing fetched.
type VetPlan struct {
	Subject  repositories.CarrierIntelSubject
	Provider integration.Type
	Depth    carrierintel.LookupDepth
	Current  *carrierintel.CarrierIntelSnapshot
}

func (s *Service) PlanVetCarrier(ctx context.Context, req *VetCarrierRequest) (*VetPlan, error) {
	subject, err := s.loadCarrierSubject(ctx, req.TenantInfo, req.CarrierID)
	if err != nil {
		return nil, err
	}
	depth := req.Depth
	if !depth.IsValid() {
		depth = carrierintel.LookupDepthFull
	}

	return s.planFetch(ctx, req.TenantInfo, &subject, depth)
}

func (s *Service) PlanVetCustomerBroker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (*VetPlan, error) {
	subject, err := s.subjectRepo.GetCustomerSubject(ctx, tenantInfo, customerID)
	if err != nil {
		return nil, err
	}
	if subject.DOTNumber == "" {
		return nil, errortypes.NewBusinessError(
			"Add the customer's DOT number before vetting them as a broker",
		)
	}
	subject.Broker = true

	return s.planFetch(ctx, tenantInfo, subject, carrierintel.LookupDepthFMCSA)
}

func (s *Service) planFetch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	subject *repositories.CarrierIntelSubject,
	depth carrierintel.LookupDepth,
) (*VetPlan, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return nil, errortypes.NewBusinessError(
			"Enable a carrier intelligence provider before vetting carriers",
		)
	}

	current, err := s.snapshotRepo.GetCurrent(ctx, tenantInfo, repositories.CarrierIntelSubjectRef{
		SubjectType: subject.SubjectType,
		SubjectID:   subject.SubjectID,
	})
	if err != nil {
		return nil, err
	}

	return &VetPlan{Subject: *subject, Provider: provider, Depth: depth, Current: current}, nil
}

// EnrollmentChange is one carrier's monitoring as it stands and as the call
// would leave it.
type EnrollmentChange struct {
	CarrierID pulid.ID
	Name      string
	DOTNumber string
	Before    carrierintel.DesiredState
	After     carrierintel.DesiredState
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
	connector, ok := s.connectors[provider]
	if !ok || !connector.Descriptor().Capabilities.SupportsMonitoring() {
		return nil, errortypes.NewBusinessError(
			"{0} does not support carrier monitoring", provider.String(),
		)
	}

	subjects, err := s.subjectRepo.ListCarrierSubjects(
		ctx,
		&repositories.ListCarrierIntelSubjectsRequest{
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
	rows, err := s.enrollmentRepo.ListBySubjects(ctx, &repositories.ListEnrollmentsBySubjectRequest{
		TenantInfo:  req.TenantInfo,
		Provider:    provider,
		SubjectType: carrierintel.SubjectTypeCarrier,
		SubjectIDs:  ids,
	})
	if err != nil {
		return nil, err
	}
	existing := make(map[string]carrierintel.DesiredState, len(rows))
	for _, row := range rows {
		existing[row.SubjectID] = row.DesiredState
	}

	after := carrierintel.DesiredStateNotEnrolled
	if req.Enroll {
		after = carrierintel.DesiredStateEnrolled
	}
	changes := make([]EnrollmentChange, 0, len(subjects))
	for _, subject := range subjects {
		before, found := existing[subject.SubjectID]
		if !found {
			before = carrierintel.DesiredStateNotEnrolled
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

// SnapshotChange is a carrier's intelligence snapshot before and after a
// review is recorded on it.
type SnapshotChange struct {
	Before *carrierintel.CarrierIntelSnapshot
	After  *carrierintel.CarrierIntelSnapshot
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
		repositories.CarrierIntelSubjectRef{
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

// SuggestionPlan is the carrier as applying the chosen suggestions would
// leave it, and the suggestions that still apply.
type SuggestionPlan struct {
	Before    *carrier.Carrier
	After     *carrier.Carrier
	Fields    []carrierintel.FieldUpdate
	Insurance []carrierintel.InsuranceChange
	Applied   int
}

func (s *Service) PlanApplySuggestions(
	ctx context.Context,
	req *ApplySuggestionsRequest,
) (*SuggestionPlan, error) {
	if len(req.Fields) == 0 && len(req.PolicyIDs) == 0 {
		return nil, errortypes.NewValidationError("fields", errortypes.ErrRequired,
			"Select at least one suggestion to apply")
	}
	for _, field := range req.Fields {
		if !field.IsValid() {
			return nil, errortypes.NewValidationError("fields", errortypes.ErrInvalid,
				"Suggestion field is invalid")
		}
	}

	snapshot, err := s.snapshotRepo.GetCurrent(
		ctx,
		req.TenantInfo,
		repositories.CarrierIntelSubjectRef{
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   req.CarrierID.String(),
		},
	)
	if err != nil {
		return nil, err
	}
	if snapshot == nil || snapshot.NotFound {
		return nil, errortypes.NewBusinessError("Run carrier intelligence for this carrier first")
	}

	load := func() (*carrier.Carrier, error) {
		return s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
			ID:         req.CarrierID,
			TenantInfo: req.TenantInfo,
			CarrierFilterOptions: repositories.CarrierFilterOptions{
				IncludeContacts:          true,
				IncludeInsurancePolicies: true,
				IncludeEDIChannels:       true,
			},
		})
	}
	before, err := load()
	if err != nil {
		return nil, err
	}
	entity, err := load()
	if err != nil {
		return nil, err
	}

	plan := carrierintel.PlanCarrierSync(
		entity,
		snapshot.Profile,
		carrierintel.SyncSettings{},
		s.now(),
	)
	selected := make([]carrierintel.FieldUpdate, 0, len(req.Fields))
	for idx := range plan.Suggestions {
		if slices.Contains(req.Fields, plan.Suggestions[idx].Field) {
			selected = append(selected, plan.Suggestions[idx])
		}
	}
	selectedInsurance := make([]carrierintel.InsuranceChange, 0, len(req.PolicyIDs))
	for idx := range plan.InsuranceSuggestions {
		change := &plan.InsuranceSuggestions[idx]
		if change.PolicyID.IsNotNil() && slices.Contains(req.PolicyIDs, change.PolicyID) {
			selectedInsurance = append(selectedInsurance, *change)
		}
	}
	if len(selected) == 0 && len(selectedInsurance) == 0 {
		return nil, errortypes.NewBusinessError(
			"The selected suggestions no longer apply. Refresh the carrier and try again",
		)
	}

	applied := len(carrierintel.ApplyFieldUpdates(entity, selected))
	applied += carrierintel.ApplyInsuranceChanges(entity, selectedInsurance)

	return &SuggestionPlan{
		Before:    before,
		After:     entity,
		Fields:    selected,
		Insurance: selectedInsurance,
		Applied:   applied,
	}, nil
}

// ImportPlan is the carrier importing a prospect would make, as far as it can
// be known without asking the provider: the DOT number, checked for a carrier
// already holding it.
type ImportPlan struct {
	DOTNumber        string
	Code             string
	Provider         integration.Type
	EnrollMonitoring bool
}

func (s *Service) PlanImportProspect(
	ctx context.Context,
	req *ImportProspectRequest,
) (*ImportPlan, error) {
	dot := stringutils.DigitsOnly(req.DOTNumber)
	if dot == "" {
		return nil, errortypes.NewValidationError("dotNumber", errortypes.ErrRequired,
			"A DOT number is required to import a carrier")
	}
	existing, err := s.subjectRepo.ListExistingCarrierDOTs(ctx, req.TenantInfo, []string{dot})
	if err != nil {
		return nil, err
	}
	if id, found := existing[dot]; found {
		return nil, errortypes.NewBusinessError(
			"A carrier with DOT {0} already exists", dot,
		).WithParam("carrierId", id.String())
	}
	control, err := s.controlRepo.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	provider, ok := control.PrimaryType()
	if !ok {
		return nil, errortypes.NewBusinessError(
			"Enable a carrier intelligence provider before importing carriers",
		)
	}

	return &ImportPlan{
		DOTNumber:        dot,
		Code:             strings.TrimSpace(req.Code),
		Provider:         provider,
		EnrollMonitoring: req.EnrollMonitoring,
	}, nil
}

// PlanVerifyEquipment is the verification record VerifyEquipment would make
// before the provider is asked: everything but the result.
func (s *Service) PlanVerifyEquipment(
	ctx context.Context,
	req *VerifyEquipmentRequest,
) (*carrierintel.CarrierEquipmentVerification, *carrier.Carrier, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, &repositories.GetCarrierAssignmentByIDRequest{
		TenantInfo:          req.TenantInfo,
		CarrierAssignmentID: req.CarrierAssignmentID,
	})
	if err != nil {
		return nil, nil, err
	}

	carrierEntity, err := s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
		ID:         assignment.CarrierID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}

	entity := &carrierintel.CarrierEquipmentVerification{
		OrganizationID:      req.TenantInfo.OrgID,
		BusinessUnitID:      req.TenantInfo.BuID,
		CarrierAssignmentID: assignment.ID,
		ShipmentMoveID:      assignment.ShipmentMoveID,
		CarrierID:           carrierEntity.ID,
		ExpectedDOTNumber:   carrierEntity.DOTNumber,
		UnitType:            req.UnitType,
		VIN:                 req.VIN,
		PlateNumber:         req.PlateNumber,
		PlateState:          req.PlateState,
		UnitNumber:          req.UnitNumber,
		VerifiedByID:        req.TenantInfo.UserID,
		VerifiedAt:          s.now(),
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, nil, multiErr
	}

	return entity, carrierEntity, nil
}
