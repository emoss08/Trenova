package carrierintelservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

type VetCarrierRequest struct {
	TenantInfo pagination.TenantInfo
	CarrierID  pulid.ID
	Depth      carrierintel.LookupDepth
	Force      bool
}

func (s *Service) loadCarrierSubject(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) (repositories.CarrierIntelSubject, error) {
	subjects, err := s.subjectRepo.ListCarrierSubjects(
		ctx,
		&repositories.ListCarrierIntelSubjectsRequest{
			TenantInfo: tenantInfo,
			CarrierIDs: []pulid.ID{carrierID},
			Limit:      1,
		},
	)
	if err != nil {
		return repositories.CarrierIntelSubject{}, err
	}
	if len(subjects) == 0 {
		if _, getErr := s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
			ID:         carrierID,
			TenantInfo: tenantInfo,
		}); getErr != nil {
			return repositories.CarrierIntelSubject{}, getErr
		}
		return repositories.CarrierIntelSubject{}, errortypes.NewBusinessError(
			"Add the carrier's DOT number before running carrier intelligence",
		)
	}
	return subjects[0], nil
}

func (s *Service) VetCarrier(ctx context.Context, req *VetCarrierRequest) (*FetchResult, error) {
	subject, err := s.loadCarrierSubject(ctx, req.TenantInfo, req.CarrierID)
	if err != nil {
		return nil, err
	}
	purpose := carrierintel.PurposeVet
	depth := req.Depth
	if depth.IsValid() && depth != carrierintel.LookupDepthFull {
		purpose = carrierintel.PurposeRefresh
	}
	return s.Fetch(ctx, &FetchRequest{
		TenantInfo: req.TenantInfo,
		Subject:    subject,
		MinDepth:   depth,
		Purpose:    purpose,
		Force:      req.Force,
		Source:     carrierintel.EventSourceSnapshotDiff,
	})
}

type LookupProspectRequest struct {
	TenantInfo   pagination.TenantInfo
	DOTNumber    string
	DocketNumber string
	Depth        carrierintel.LookupDepth
}

func (s *Service) LookupProspect(
	ctx context.Context,
	req *LookupProspectRequest,
) (*FetchResult, map[string]pulid.ID, error) {
	dot := stringutils.DigitsOnly(req.DOTNumber)
	docket := stringutils.DigitsOnly(req.DocketNumber)
	if dot == "" && docket == "" {
		return nil, nil, errortypes.NewValidationError("dotNumber", errortypes.ErrRequired,
			"Enter a DOT or MC number")
	}

	subjectID := carrierintel.ProspectSubjectID(dot)
	if dot == "" {
		subjectID = "MC" + docket
	}
	depth := req.Depth
	if !depth.IsValid() {
		depth = carrierintel.LookupDepthLite
	}

	result, err := s.Fetch(ctx, &FetchRequest{
		TenantInfo: req.TenantInfo,
		Subject: repositories.CarrierIntelSubject{
			SubjectType:  carrierintel.SubjectTypeProspect,
			SubjectID:    subjectID,
			DOTNumber:    dot,
			DocketNumber: docket,
		},
		MinDepth: depth,
		Purpose:  carrierintel.PurposeSourcing,
		Source:   carrierintel.EventSourceSnapshotDiff,
	})
	if err != nil {
		return nil, nil, err
	}

	existing := map[string]pulid.ID{}
	if dotFound := result.Snapshot.DOTNumber; dotFound != "" {
		existing, err = s.subjectRepo.ListExistingCarrierDOTs(
			ctx,
			req.TenantInfo,
			[]string{dotFound},
		)
		if err != nil {
			return nil, nil, err
		}
	}
	return result, existing, nil
}

type SelfIntelligenceResult struct {
	Configured bool
	DOTNumber  string
	Snapshot   *carrierintel.CarrierIntelSnapshot
}

func (s *Service) MyIntelligence(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	refresh bool,
) (*SelfIntelligenceResult, error) {
	org, err := s.subjectRepo.GetOrganizationSubject(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if org == nil || org.DOTNumber == "" {
		return &SelfIntelligenceResult{}, nil
	}
	result := &SelfIntelligenceResult{Configured: true, DOTNumber: org.DOTNumber}
	if !refresh {
		snapshot, getErr := s.snapshotRepo.GetCurrent(
			ctx,
			tenantInfo,
			repositories.CarrierIntelSubjectRef{
				SubjectType: carrierintel.SubjectTypeOrganization,
				SubjectID:   org.SubjectID,
			},
		)
		if getErr != nil {
			return nil, getErr
		}
		result.Snapshot = snapshot
		return result, nil
	}

	fetched, err := s.Fetch(ctx, &FetchRequest{
		TenantInfo: tenantInfo,
		Subject:    *org,
		Purpose:    carrierintel.PurposeSelfMonitor,
		Force:      true,
		Source:     carrierintel.EventSourceSnapshotDiff,
	})
	if err != nil {
		return nil, err
	}
	result.Snapshot = fetched.Snapshot
	return result, nil
}

func (s *Service) VetCustomerBroker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
	force bool,
) (*FetchResult, error) {
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
	return s.Fetch(ctx, &FetchRequest{
		TenantInfo: tenantInfo,
		Subject:    *subject,
		MinDepth:   carrierintel.LookupDepthFMCSA,
		Purpose:    carrierintel.PurposeVet,
		Force:      force,
		Source:     carrierintel.EventSourceSnapshotDiff,
	})
}

func (s *Service) SyncPlan(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
) (*carrierintel.SyncPlan, error) {
	control, err := s.controlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	entity, err := s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
		ID:         carrierID,
		TenantInfo: tenantInfo,
		CarrierFilterOptions: repositories.CarrierFilterOptions{
			IncludeInsurancePolicies: true,
		},
	})
	if err != nil {
		return nil, err
	}
	snapshot, err := s.snapshotRepo.GetCurrent(ctx, tenantInfo, repositories.CarrierIntelSubjectRef{
		SubjectType: carrierintel.SubjectTypeCarrier,
		SubjectID:   carrierID.String(),
	})
	if err != nil {
		return nil, err
	}
	if snapshot == nil || snapshot.NotFound {
		empty := carrierintel.PlanCarrierSync(nil, nil, control.SyncSettings(), s.now())
		return &empty, nil
	}
	plan := carrierintel.PlanCarrierSync(
		entity,
		snapshot.Profile,
		carrierintel.SyncSettings{},
		s.now(),
	)
	return &plan, nil
}

type ApplySuggestionsRequest struct {
	TenantInfo pagination.TenantInfo
	CarrierID  pulid.ID
	Fields     []carrierintel.SyncField
	PolicyIDs  []pulid.ID
}

func (s *Service) ApplySuggestions(
	ctx context.Context,
	req *ApplySuggestionsRequest,
) (int, error) {
	if len(req.Fields) == 0 && len(req.PolicyIDs) == 0 {
		return 0, errortypes.NewValidationError("fields", errortypes.ErrRequired,
			"Select at least one suggestion to apply")
	}
	for _, field := range req.Fields {
		if !field.IsValid() {
			return 0, errortypes.NewValidationError("fields", errortypes.ErrInvalid,
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
		return 0, err
	}
	if snapshot == nil || snapshot.NotFound {
		return 0, errortypes.NewBusinessError("Run carrier intelligence for this carrier first")
	}

	entity, err := s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
		ID:         req.CarrierID,
		TenantInfo: req.TenantInfo,
		CarrierFilterOptions: repositories.CarrierFilterOptions{
			IncludeContacts:          true,
			IncludeInsurancePolicies: true,
			IncludeEDIChannels:       true,
		},
	})
	if err != nil {
		return 0, err
	}

	plan := carrierintel.PlanCarrierSync(
		entity,
		snapshot.Profile,
		carrierintel.SyncSettings{},
		s.now(),
	)
	selected := make([]carrierintel.FieldUpdate, 0, len(req.Fields))
	for _, suggestion := range plan.Suggestions {
		if slices.Contains(req.Fields, suggestion.Field) {
			selected = append(selected, suggestion)
		}
	}
	selectedInsurance := make([]carrierintel.InsuranceChange, 0, len(req.PolicyIDs))
	for _, change := range plan.InsuranceSuggestions {
		if change.PolicyID.IsNotNil() && slices.Contains(req.PolicyIDs, change.PolicyID) {
			selectedInsurance = append(selectedInsurance, change)
		}
	}
	if len(selected) == 0 && len(selectedInsurance) == 0 {
		return 0, errortypes.NewBusinessError(
			"The selected suggestions no longer apply. Refresh the carrier and try again",
		)
	}

	applied := len(carrierintel.ApplyFieldUpdates(entity, selected))
	applied += carrierintel.ApplyInsuranceChanges(entity, selectedInsurance)
	if _, err = s.carrierService.Update(ctx, entity, &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    req.TenantInfo.UserID,
		UserID:         req.TenantInfo.UserID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}); err != nil {
		return 0, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:      permission.ResourceCarrierIntelligence,
		ResourceID:    req.CarrierID.String(),
		Operation:     permission.OpUpdate,
		UserID:        req.TenantInfo.UserID,
		PrincipalType: services.PrincipalTypeUser,
		PrincipalID:   req.TenantInfo.UserID,
		CurrentState: jsonutils.MustToJSON(map[string]any{
			"fields":    selected,
			"insurance": selectedInsurance,
		}),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}, auditservice.WithComment("Carrier intelligence suggestions applied")); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}
	return applied, nil
}
